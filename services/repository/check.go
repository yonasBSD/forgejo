// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	system_model "forgejo.org/models/system"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/container"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

// GitFsckRepos calls 'git fsck' to check repository health.
func GitFsckRepos(ctx context.Context, timeout time.Duration, args git.TrustedCmdArgs) error {
	log.Trace("Doing: GitFsck")

	if err := db.Iterate(
		ctx,
		builder.Expr("id>0 AND is_fsck_enabled=?", true),
		func(ctx context.Context, repo *repo_model.Repository) error {
			select {
			case <-ctx.Done():
				return db.ErrCancelledf("before fsck of %s", repo.FullName())
			default:
			}
			return GitFsckRepo(ctx, repo, timeout, args)
		},
	); err != nil {
		log.Trace("Error: GitFsck: %v", err)
		return err
	}

	log.Trace("Finished: GitFsck")
	return nil
}

// GitFsckRepo calls 'git fsck' to check an individual repository's health.
func GitFsckRepo(ctx context.Context, repo *repo_model.Repository, timeout time.Duration, args git.TrustedCmdArgs) error {
	log.Trace("Running health check on repository %-v", repo)
	repoPath := repo.RepoPath()
	if err := git.Fsck(ctx, repoPath, timeout, args); err != nil {
		log.Warn("Failed to health check repository (%-v): %v", repo, err)
		if err = system_model.CreateRepositoryNotice("Failed to health check repository (%s): %v", repo.FullName(), err); err != nil {
			log.Error("CreateRepositoryNotice: %v", err)
		}
	}
	return nil
}

// GitGcRepos calls 'git gc' to remove unnecessary files and optimize the local repository
func GitGcRepos(ctx context.Context, timeout time.Duration, args git.TrustedCmdArgs) error {
	log.Trace("Doing: GitGcRepos")

	if err := db.Iterate(
		ctx,
		builder.Gt{"id": 0},
		func(ctx context.Context, repo *repo_model.Repository) error {
			select {
			case <-ctx.Done():
				return db.ErrCancelledf("before GC of %s", repo.FullName())
			default:
			}
			// we can ignore the error here because it will be logged in GitGCRepo
			_ = GitGcRepo(ctx, repo, timeout, args)
			return nil
		},
	); err != nil {
		return err
	}

	log.Trace("Finished: GitGcRepos")
	return nil
}

func syncRepoToAlternate(ctx context.Context, repo *repo_model.Repository, timeout time.Duration) error {
	if repo.AlternateID == 0 {
		return nil
	}

	if err := repo.GetAlternate(ctx); err != nil {
		return fmt.Errorf("failed to get alternate for repo %s: %w", repo.FullName(), err)
	}

	altPath := repo.Alternate.GetPath()

	// Potentially the main repo or its location could have changed, so make sure origin actually points to it
	command := git.NewCommand(ctx, "remote", "set-url", "origin").
		AddDynamicArguments(repo.RepoPath()).
		SetDescription(fmt.Sprintf("Update alternate %s origin url: %s", altPath, repo.RepoPath()))
	stdout, _, err := command.RunStdString(&git.RunOpts{Timeout: timeout, Dir: altPath})
	if err != nil {
		return fmt.Errorf("failed to update alternate origin url for %s -> %s. Stdout: %s\nError: %w", altPath, repo.RepoPath(), stdout, err)
	}

	command = git.NewCommand(ctx, "remote", "update", "--prune", "origin").
		SetDescription(fmt.Sprintf("Sync alternate %s from its main repo", altPath))
	stdout, _, err = command.RunStdString(&git.RunOpts{Timeout: timeout, Dir: altPath})
	if err != nil {
		return fmt.Errorf("failed to sync alternate %s from its main repo. Stdout: %s\nError: %w", altPath, stdout, err)
	}

	return nil
}

func gcAlternate(ctx context.Context, repo *repo_model.Repository, timeout time.Duration, args git.TrustedCmdArgs) error {
	if repo.AlternateID == 0 {
		return nil
	}

	if err := repo.GetAlternate(ctx); err != nil {
		return fmt.Errorf("failed to get alternate for repo %s: %w", repo.FullName(), err)
	}

	altPath := repo.Alternate.GetPath()

	var usingRepos []*repo_model.Repository
	if err := db.GetEngine(ctx).Where("alternate_id = ?", repo.AlternateID).Find(&usingRepos); err != nil {
		return fmt.Errorf("failed to fetch repositories using alternate %d: %w", repo.AlternateID, err)
	}

	// fetch all refs from all repos that use this alternate and store them in a set
	refsSet := make(container.Set[string])
	for _, usingRepo := range usingRepos {
		command := git.NewCommand(ctx, "rev-parse", "--all").
			SetDescription(fmt.Sprintf("Get all refs from repo %s", usingRepo.FullName()))
		stdout, _, err := command.RunStdString(&git.RunOpts{Timeout: timeout, Dir: usingRepo.RepoPath()})
		if err != nil {
			log.Warn("Failed to get refs from repository %s: %v", usingRepo.FullName(), err)
			continue
		}
		refs := strings.Split(strings.TrimSpace(stdout), "\n")
		for _, ref := range refs {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				refsSet.Add(ref)
			}
		}
	}

	// filter those refs that aren't in the alternate to begin with
	realRefsList := make([]string, 0, len(refsSet))
	if len(refsSet) > 0 {
		command := git.NewCommand(ctx, "cat-file", "--batch-check").
			SetDescription(fmt.Sprintf("Batch check refs existence in alternate %s", altPath))
		stdout, _, err := command.RunStdString(&git.RunOpts{
			Timeout: timeout,
			Dir:     altPath,
			Stdin:   strings.NewReader(strings.Join(refsSet.Values(), "\n")),
		})
		if err != nil {
			return fmt.Errorf("failed to batch check refs existence in alternate %s. Stdout: %s\nError: %w", altPath, stdout, err)
		}

		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasSuffix(line, " missing") {
				parts := strings.SplitN(line, " ", 2)
				realRefsList = append(realRefsList, parts[0])
			}
		}
	}

	// create temporary refs for all refs other repos use
	// they're left in place, next time around the remote update --prune will clean them up again
	if len(realRefsList) > 0 {
		tempRefPrefix := "refs/forgejo-prune-protection/"
		var updateCommands strings.Builder

		for i, hash := range realRefsList {
			tempRef := fmt.Sprintf("%stemp-ref-%d", tempRefPrefix, i)
			updateCommands.WriteString(fmt.Sprintf("update %s %s\n", tempRef, hash))
		}

		command := git.NewCommand(ctx, "update-ref", "--stdin").
			SetDescription("Create temporary refs for alternate garbage collection")
		stdout, _, err := command.RunStdString(&git.RunOpts{
			Timeout: timeout,
			Dir:     altPath,
			Stdin:   strings.NewReader(updateCommands.String()),
		})
		if err != nil {
			return fmt.Errorf("failed to create temporary refs in alternate %s. Stdout: %s\nError: %w", altPath, stdout, err)
		}
	}

	// finally, it is now safe to run git gc on the alternate
	command := git.NewCommand(ctx, "gc").AddArguments(args...).
		SetDescription(fmt.Sprintf("Garbage collect alternate %s", altPath))
	stdout, _, err := command.RunStdString(&git.RunOpts{Timeout: timeout, Dir: altPath})
	if err != nil {
		return fmt.Errorf("failed to garbage collect alternate %s. Stdout: %s\nError: %w", altPath, stdout, err)
	}

	return nil
}

// GitGcRepo calls 'git gc' to remove unnecessary files and optimize the local repository
func GitGcRepo(ctx context.Context, repo *repo_model.Repository, timeout time.Duration, args git.TrustedCmdArgs) error {
	log.Trace("Running git gc on %-v", repo)

	if repo.IsFork || repo.NumForks > 0 {
		// If the repo is a fork or has forks, ensure it has an alternate set up
		if _, err := repo_module.EnsureAlternate(ctx, repo); err != nil {
			log.Error("Failed to set up alternate for repository %s: %v", repo.FullName(), err)
			desc := fmt.Sprintf("Failed to set up alternate for repository %s: %v", repo.FullName(), err)
			if err := system_model.CreateRepositoryNotice(desc); err != nil {
				log.Error("CreateRepositoryNotice: %v", err)
			}
			return fmt.Errorf("Failed to set up alternate for repository %s: %w", repo.FullName(), err)
		}
	} else {
		// If it is neither, make sure it does not have an alternate
		if err := repo_module.DetachAlternate(ctx, repo); err != nil {
			log.Error("Failed to detach alternate from repository %s: %v", repo.FullName(), err)
			desc := fmt.Sprintf("Failed to detach alternate from repository %s: %v", repo.FullName(), err)
			if err := system_model.CreateRepositoryNotice(desc); err != nil {
				log.Error("CreateRepositoryNotice: %v", err)
			}
			return fmt.Errorf("Failed to detach alternate from repository %s: %w", repo.FullName(), err)
		}
	}

	if !repo.IsFork && repo.AlternateID != 0 {
		// Sync new objects from the alternates main repository to the alternate
		if err := syncRepoToAlternate(ctx, repo, timeout); err != nil {
			log.Error("Failed to sync repository %s to its alternate: %v", repo.FullName(), err)
			desc := fmt.Sprintf("Failed to sync repository %s to its alternate: %v", repo.FullName(), err)
			if err := system_model.CreateRepositoryNotice(desc); err != nil {
				log.Error("CreateRepositoryNotice: %v", err)
			}
			return fmt.Errorf("Failed to sync repository %s to its alternate: %w", repo.FullName(), err)
		}

		if err := gcAlternate(ctx, repo, timeout, args); err != nil {
			log.Error("Failed to garbage collect alternate for repository %s: %v", repo.FullName(), err)
			desc := fmt.Sprintf("Failed to garbage collect alternate for repository %s: %v", repo.FullName(), err)
			if err := system_model.CreateRepositoryNotice(desc); err != nil {
				log.Error("CreateRepositoryNotice: %v", err)
			}
			return fmt.Errorf("Failed to garbage collect alternate for repository %s: %w", repo.FullName(), err)
		}
	}

	command := git.NewCommand(ctx, "gc").AddArguments(args...).
		SetDescription(fmt.Sprintf("Repository Garbage Collection: %s", repo.FullName()))
	var stdout string
	var err error
	stdout, _, err = command.RunStdString(&git.RunOpts{Timeout: timeout, Dir: repo.RepoPath()})
	if err != nil {
		log.Error("Repository garbage collection failed for %-v. Stdout: %s\nError: %v", repo, stdout, err)
		desc := fmt.Sprintf("Repository garbage collection failed for %s. Stdout: %s\nError: %v", repo.RepoPath(), stdout, err)
		if err := system_model.CreateRepositoryNotice(desc); err != nil {
			log.Error("CreateRepositoryNotice: %v", err)
		}
		return fmt.Errorf("Repository garbage collection failed in repo: %s: Error: %w", repo.FullName(), err)
	}

	// Now update the size of the repository
	if err := repo_module.UpdateRepoSize(ctx, repo); err != nil {
		log.Error("Updating size as part of garbage collection failed for %-v. Stdout: %s\nError: %v", repo, stdout, err)
		desc := fmt.Sprintf("Updating size as part of garbage collection failed for %s. Stdout: %s\nError: %v", repo.RepoPath(), stdout, err)
		if err := system_model.CreateRepositoryNotice(desc); err != nil {
			log.Error("CreateRepositoryNotice: %v", err)
		}
		return fmt.Errorf("Updating size as part of garbage collection failed in repo: %s: Error: %w", repo.FullName(), err)
	}

	return nil
}

func gatherMissingRepoRecords(ctx context.Context) (repo_model.RepositoryList, error) {
	repos := make([]*repo_model.Repository, 0, 10)
	if err := db.Iterate(
		ctx,
		builder.Gt{"id": 0},
		func(ctx context.Context, repo *repo_model.Repository) error {
			select {
			case <-ctx.Done():
				return db.ErrCancelledf("during gathering missing repo records before checking %s", repo.FullName())
			default:
			}
			isDir, err := util.IsDir(repo.RepoPath())
			if err != nil {
				return fmt.Errorf("Unable to check dir for %s. %w", repo.FullName(), err)
			}
			if !isDir {
				repos = append(repos, repo)
			}
			return nil
		},
	); err != nil {
		if strings.HasPrefix(err.Error(), "Aborted gathering missing repo") {
			return nil, err
		}
		if err2 := system_model.CreateRepositoryNotice("gatherMissingRepoRecords: %v", err); err2 != nil {
			log.Error("CreateRepositoryNotice: %v", err2)
		}
		return nil, err
	}
	return repos, nil
}

// DeleteMissingRepositories deletes all repository records that lost Git files.
func DeleteMissingRepositories(ctx context.Context, doer *user_model.User) error {
	repos, err := gatherMissingRepoRecords(ctx)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return nil
	}

	for _, repo := range repos {
		select {
		case <-ctx.Done():
			return db.ErrCancelledf("during DeleteMissingRepositories before %s", repo.FullName())
		default:
		}
		log.Trace("Deleting %d/%d...", repo.OwnerID, repo.ID)
		if err := DeleteRepositoryDirectly(ctx, doer, repo.ID); err != nil {
			log.Error("Failed to DeleteRepository %-v: Error: %v", repo, err)
			if err2 := system_model.CreateRepositoryNotice("Failed to DeleteRepository %s [%d]: Error: %v", repo.FullName(), repo.ID, err); err2 != nil {
				log.Error("CreateRepositoryNotice: %v", err)
			}
		}
	}
	return nil
}

// ReinitMissingRepositories reinitializes all repository records that lost Git files.
func ReinitMissingRepositories(ctx context.Context) error {
	repos, err := gatherMissingRepoRecords(ctx)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return nil
	}

	for _, repo := range repos {
		select {
		case <-ctx.Done():
			return db.ErrCancelledf("during ReinitMissingRepositories before %s", repo.FullName())
		default:
		}
		log.Trace("Initializing %d/%d...", repo.OwnerID, repo.ID)
		if err := git.InitRepository(ctx, repo.RepoPath(), true, repo.ObjectFormatName); err != nil {
			log.Error("Unable (re)initialize repository %d at %s. Error: %v", repo.ID, repo.RepoPath(), err)
			if err2 := system_model.CreateRepositoryNotice("InitRepository [%d]: %v", repo.ID, err); err2 != nil {
				log.Error("CreateRepositoryNotice: %v", err2)
			}
		}
	}
	return nil
}
