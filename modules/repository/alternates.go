// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
)

func applyAlternateToRepo(ctx context.Context, repo *repo_model.Repository, alternate *repo_model.Alternate) error {
	gitRepo, err := gitrepo.OpenRepository(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to open repository %s to set up alternate: %w", repo.FullName(), err)
	}
	defer gitRepo.Close()

	altPath, err := filepath.Abs(alternate.GetPath())
	if err != nil {
		return fmt.Errorf("failed to get absolute path for alternate: %w", err)
	}

	objPath := filepath.Join(altPath, "objects")

	// Apply the alternate in the database before anything else.
	// It's much easier to recover from "Alternate set in DB, but not in the actual repo" than the other way around.
	err = repo.UpdateAlternate(ctx, alternate)
	if err != nil {
		return fmt.Errorf("failed to update repo alternate: %w", err)
	}

	// Check if the alternate does not exist yet
	if _, err := os.Stat(altPath); os.IsNotExist(err) {
		err = git.Clone(ctx, gitRepo.Path, altPath, git.CloneRepoOptions{
			Bare:   true,
			Mirror: true,
		})
		if err != nil {
			if errDelete := util.RemoveAll(altPath); errDelete != nil {
				return fmt.Errorf("failed to remove alternate path after clone failure: %w, %w", errDelete, err)
			}
			return fmt.Errorf("failed to clone repository for alternate: %w", err)
		}

		if stdout, _, err := git.NewCommand(ctx, "config", "gc.auto", "0").
			SetDescription(fmt.Sprintf("AlternateSetup(git config): %s", repo.FullName())).
			RunStdString(&git.RunOpts{Dir: altPath}); err != nil {
			if errDelete := util.RemoveAll(altPath); errDelete != nil {
				return fmt.Errorf("failed to remove alternate path after config failure: %w, %w", errDelete, err)
			}
			return fmt.Errorf("git config failed: %w, %s", err, stdout)
		}
	}

	alts, err := gitRepo.GetAlternatePaths()
	if err != nil {
		return fmt.Errorf("failed to get alternate paths: %w", err)
	}

	// This edge case can happen if a chain of 3 repos has its middle link converted to standalone and
	// then GC is run on the leaf repo.
	if (len(alts) > 0 && alts[0] != objPath) || len(alts) > 1 {
		// The repo already has different alternates, pull packs back in first
		if stdout, _, err := git.NewCommand(ctx, "repack", "-a", "-d").
			SetDescription(fmt.Sprintf("AlternateSetup(git repack): %s", repo.FullName())).
			RunStdString(&git.RunOpts{Dir: repo.RepoPath()}); err != nil {
			log.Error("git repack failed for %v:\nStdout: %s\nError: %v", repo, stdout, err)
			return err
		}
	}

	// Set the alternate path in relative mode, so that the data dir can be relocated
	if err := gitRepo.SetAlternatePaths([]string{objPath}, true); err != nil {
		return fmt.Errorf("failed to set alternate path on root repo: %w", err)
	}

	// Clean up existing objects that are now in the alternate
	if stdout, _, err := git.NewCommand(ctx, "gc").
		SetDescription(fmt.Sprintf("AlternateSetup(git gc): %s", repo.FullName())).
		RunStdString(&git.RunOpts{Dir: gitRepo.Path}); err != nil {
		return fmt.Errorf("git gc/prune after alternate setup failed: %w, %s", err, stdout)
	}

	return nil
}

func setupAlternateForRepo(ctx context.Context, repo *repo_model.Repository) error {
	gitRepo, err := gitrepo.OpenRepository(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to open repository %s to set up alternate: %w", repo.FullName(), err)
	}
	defer gitRepo.Close()

	alts, err := gitRepo.GetAlternatePaths()
	if err != nil {
		return fmt.Errorf("could not get alternates path for repo %s: %w", repo.FullName(), err)
	}

	if len(alts) > 1 {
		return fmt.Errorf("repo %s has multiple alternates", repo.FullName())
	}

	if len(alts) > 0 && repo.AlternateID == 0 {
		return fmt.Errorf("repo %s has alternates, but no AlternateID is set", repo.FullName())
	}

	if repo.AlternateID != 0 {
		err = repo.GetAlternate(ctx)
		if err != nil {
			return fmt.Errorf("failed to get alternate for repo %s: %w", repo.FullName(), err)
		}

		if len(alts) == 0 {
			// Somehow the repo lost its alternate, so set it up
			return applyAlternateToRepo(ctx, repo, repo.Alternate)
		}

		if filepath.Dir(alts[0]) != filepath.Clean(repo.Alternate.GetPath()) {
			return fmt.Errorf("repo %s has has mismatching alternates", repo.FullName())
		}

		return nil
	}

	// Update the repo first, since a set AlternateID but non-existent on-disk alternate can be handled much more gracefully.
	// This part needs to be in a transaction since a dangling entry in the Alternates table is highly problematic to deal with.
	var alt *repo_model.Alternate
	err = db.WithTx(ctx, func(ctx context.Context) error {
		var err error
		alt, err = repo_model.CreateAlternateForRepo(ctx, repo)
		if err != nil {
			return err
		}

		err = repo.UpdateAlternate(ctx, alt)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create alternate for repo: %w", err)
	}

	return applyAlternateToRepo(ctx, repo, alt)
}

// Ensures the repo has an alternate set on it.
// It will first find the root repository this repo is forked from, set up an alternate there if neccesary,
// and then apply that alternate to the provided repo.
func EnsureAlternate(ctx context.Context, repo *repo_model.Repository) (*repo_model.Alternate, error) {
	if !setting.Repository.UseAlternates {
		return nil, nil
	}

	// The very root repo of a fork chain gets turned into the original alternate
	rootRepo, err := repo.GetRootBaseRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get fork chain base repo: %w", err)
	}

	// Setup/validate the initial alternate at the base of the fork chain
	err = setupAlternateForRepo(ctx, rootRepo)
	if err != nil {
		return nil, err
	}

	// Apply the alternate from the root repo to the main repo, if they're not the same
	if rootRepo.ID != repo.ID {
		err = applyAlternateToRepo(ctx, repo, rootRepo.Alternate)
		if err != nil {
			return nil, err
		}
	}

	return repo.Alternate, nil
}

// Safely detaches an alternate from a repo, deleting the alternate if it becomes dangling
func DetachAlternate(ctx context.Context, repo *repo_model.Repository) error {
	if repo.AlternateID == 0 {
		return nil
	}

	if err := repo.GetAlternate(ctx); err != nil {
		return fmt.Errorf("failed to get alternate for %v: %v", repo, err)
	}

	if stdout, _, err := git.NewCommand(ctx, "repack", "-a", "-d").
		SetDescription(fmt.Sprintf("DetachAlternate(git repack): %s", repo.FullName())).
		RunStdString(&git.RunOpts{Dir: repo.RepoPath()}); err != nil {
		return fmt.Errorf("git repack failed for %v:\nStdout: %s\nError: %v", repo, stdout, err)
	}

	err := util.Remove(filepath.Join(repo.RepoPath(), "objects", "info", "alternates"))
	if err != nil {
		return fmt.Errorf("unlinking alternate failed for %v: %v", repo, err)
	}

	alt := repo.Alternate
	err = repo.UpdateAlternate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed clearing out repos alternate: %v", err)
	}

	has, err := alt.HasRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if alternate still has repos: %v", err)
	}

	if !has {
		_, err = db.DeleteByBean(ctx, alt)
		if err != nil {
			return fmt.Errorf("failed deleting dangling alternate from db: %v", err)
		}

		err = util.RemoveAll(alt.GetPath())
		if err != nil {
			return fmt.Errorf("failed deleting dangling alternate from disk: %v", err)
		}
	}

	return nil
}
