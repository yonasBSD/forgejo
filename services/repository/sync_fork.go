package repository

import (
	"context"
	"fmt"
	"slices"

	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	repo_module "code.gitea.io/gitea/modules/repository"
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
	defer repo_module.RemoveTemporaryPath(tmpPath)

	err = git.NewCommand(ctx, "clone", "-b").AddDynamicArguments(branch, repo.RepoPath(), tmpPath).Run(&git.RunOpts{Dir: tmpPath})
	if err != nil {
		return fmt.Errorf("Clone: %v", err)
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

// CanSyncFork returns if a branch of the fork can be synced with the base repo
func CanSyncFork(ctx context.Context, repo *repo_model.Repository, branch string) (bool, error) {
	forkBranch, err := git_model.GetBranch(ctx, repo.ID, branch)
	if err != nil {
		return false, err
	}

	baseBranch, err := git_model.GetBranch(ctx, repo.BaseRepo.ID, branch)
	if err != nil {
		if git_model.IsErrBranchNotExist(err) {
			// If the base repo don't have the branch, we don't need to continue
			return false, nil
		}
		return false, err
	}

	// If both branches has the same latest commit, we don't need to sync
	if forkBranch.CommitID == baseBranch.CommitID {
		return false, nil
	}

	// If the fork has newer commits, we can't sync
	if forkBranch.CommitTime >= baseBranch.CommitTime {
		return false, nil
	}

	// Check if the latest commit of the fork is also in the base
	gitRepo, err := git.OpenRepository(ctx, repo.BaseRepo.RepoPath())
	if err != nil {
		return false, err
	}
	defer gitRepo.Close()

	commit, err := gitRepo.GetCommit(forkBranch.CommitID)
	if err != nil {
		if git.IsErrNotExist(err) {
			return false, nil
		}
		return false, err
	}

	branchList, err := commit.GetAllBranches()
	if err != nil {
		return false, err
	}

	return slices.Contains(branchList, branch), nil
}
