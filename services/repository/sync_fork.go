// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"fmt"
	"slices"

	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	api "code.gitea.io/gitea/modules/structs"
)

// SyncFork syncs a branch of a fork with the base repo
func SyncFork(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, branch string) error {
	err := repo.GetBaseRepo(ctx)
	if err != nil {
		return err
	}

	tmpPath, err := repo_module.CreateTemporaryPath("sync")
	if err != nil {
		return err
	}
	defer func() {
		if err := repo_module.RemoveTemporaryPath(tmpPath); err != nil {
			log.Error("SyncFork: RemoveTemporaryPath: %s", err)
		}
	}()

	err = git.Clone(ctx, repo.RepoPath(), tmpPath, git.CloneRepoOptions{})
	if err != nil {
		return err
	}

	gitRepo, err := git.OpenRepository(ctx, tmpPath)
	if err != nil {
		return err
	}
	defer gitRepo.Close()

	command := git.NewCommand(gitRepo.Ctx, "remote", "add", "upstream")
	command = command.AddDynamicArguments(repo.BaseRepo.RepoPath())
	err = command.Run(&git.RunOpts{Dir: gitRepo.Path})
	if err != nil {
		return fmt.Errorf("RemoteAdd: %v", err)
	}

	err = git.NewCommand(gitRepo.Ctx, "fetch", "upstream").Run(&git.RunOpts{Dir: gitRepo.Path})
	if err != nil {
		return fmt.Errorf("FetchUpstream: %v", err)
	}

	command = git.NewCommand(gitRepo.Ctx, "checkout")
	command = command.AddDynamicArguments(branch)
	err = command.Run(&git.RunOpts{Dir: gitRepo.Path})
	if err != nil {
		return fmt.Errorf("Checkout: %v", err)
	}

	command = git.NewCommand(gitRepo.Ctx, "rebase")
	command = command.AddDynamicArguments(fmt.Sprintf("upstream/%s", branch))
	err = command.Run(&git.RunOpts{Dir: gitRepo.Path})
	if err != nil {
		return fmt.Errorf("Rebase: %v", err)
	}

	pushEnv := repo_module.PushingEnvironment(doer, repo)
	err = git.NewCommand(ctx, "push", "origin").AddDynamicArguments(branch).Run(&git.RunOpts{Dir: tmpPath, Env: pushEnv})
	if err != nil {
		return fmt.Errorf("Push: %v", err)
	}

	return nil
}

// CanSyncFork returns inofmrtaion about syncing a fork
func GetSyncForkInfo(ctx context.Context, repo *repo_model.Repository, branch string) (*api.SyncForkInfo, error) {
	info := new(api.SyncForkInfo)
	info.Allowed = false

	if !repo.IsFork {
		return info, nil
	}

	err := repo.GetBaseRepo(ctx)
	if err != nil {
		return nil, err
	}

	forkBranch, err := git_model.GetBranch(ctx, repo.ID, branch)
	if err != nil {
		return nil, err
	}

	info.ForkCommit = forkBranch.CommitID

	baseBranch, err := git_model.GetBranch(ctx, repo.BaseRepo.ID, branch)
	if err != nil {
		if git_model.IsErrBranchNotExist(err) {
			// If the base repo don't have the branch, we don't need to continue
			return info, nil
		}
		return nil, err
	}

	info.BaseCommit = baseBranch.CommitID

	// If both branches has the same latest commit, we don't need to sync
	if forkBranch.CommitID == baseBranch.CommitID {
		return info, nil
	}

	// If the fork has newer commits, we can't sync
	if forkBranch.CommitTime >= baseBranch.CommitTime {
		return info, nil
	}

	// Check if the latest commit of the fork is also in the base
	gitRepo, err := git.OpenRepository(ctx, repo.BaseRepo.RepoPath())
	if err != nil {
		return nil, err
	}
	defer gitRepo.Close()

	commit, err := gitRepo.GetCommit(forkBranch.CommitID)
	if err != nil {
		if git.IsErrNotExist(err) {
			return info, nil
		}
		return nil, err
	}

	branchList, err := commit.GetAllBranches()
	if err != nil {
		return nil, err
	}

	info.Allowed = slices.Contains(branchList, branch)

	return info, nil
}
