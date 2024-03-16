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
	api "code.gitea.io/gitea/modules/structs"
)

// SyncFork syncs a branch of a fork with the base repo
func SyncFork(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, branch string) error {
	err := repo.MustNotBeArchived()
	if err != nil {
		return err
	}

	err = repo.GetBaseRepo(ctx)
	if err != nil {
		return err
	}

	repo.RepoPath()

	err = git.NewCommand(ctx, "fetch", "--force").AddDynamicArguments(repo.BaseRepo.RepoPath(), fmt.Sprintf("%s:%s", branch, branch)).Run(&git.RunOpts{Dir: repo.RepoPath()})
	if err != nil {
		return err
	}

	gitRepo, err := git.OpenRepository(ctx, repo.RepoPath())
	if err != nil {
		return err
	}
	defer gitRepo.Close()

	forkBranch, err := gitRepo.GetBranch(branch)
	if err != nil {
		return err
	}

	commit, err := forkBranch.GetCommit()
	if err != nil {
		return err
	}

	_, err = git_model.UpdateBranch(ctx, repo.ID, doer.ID, branch, commit)
	if err != nil {
		return err
	}

	return nil
}

// CanSyncFork returns information about syncing a fork
func GetSyncForkInfo(ctx context.Context, repo *repo_model.Repository, branch string) (*api.SyncForkInfo, error) {
	info := new(api.SyncForkInfo)

	if !repo.IsFork {
		return info, nil
	}

	if repo.IsArchived {
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
