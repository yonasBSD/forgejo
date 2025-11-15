// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package pull

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"forgejo.org/models"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/references"
	"forgejo.org/modules/repository"
	"forgejo.org/modules/timeutil"
	notify_service "forgejo.org/services/notify"
)

// MergedManually tries to figure out whether a PR got merged manually.
//
// If this is the case, it's marked as manually merged with a reference to the
// commit on the base repo.
func MergedManually(
	ctx context.Context,
	pr *issues_model.PullRequest,
	doer *user_model.User,
	baseGitRepo *git.Repository,
	commitID string,
) error {
	pullWorkingPool.CheckIn(fmt.Sprint(pr.ID))
	defer pullWorkingPool.CheckOut(fmt.Sprint(pr.ID))

	if err := db.WithTx(ctx, func(ctx context.Context) error {
		if err := pr.LoadBaseRepo(ctx); err != nil {
			return err
		}
		prUnit, err := pr.BaseRepo.GetUnit(ctx, unit.TypePullRequests)
		if err != nil {
			return err
		}
		prConfig := prUnit.PullRequestsConfig()

		// Check if merge style is correct and allowed
		if !prConfig.IsMergeStyleAllowed(repo_model.MergeStyleManuallyMerged) {
			return models.ErrInvalidMergeStyle{ID: pr.BaseRepo.ID, Style: repo_model.MergeStyleManuallyMerged}
		}

		objectFormat := git.ObjectFormatFromName(pr.BaseRepo.ObjectFormatName)
		if len(commitID) != objectFormat.FullLength() {
			return errors.New("Wrong commit ID")
		}

		commit, err := baseGitRepo.GetCommit(commitID)
		if err != nil {
			if git.IsErrNotExist(err) {
				return errors.New("Wrong commit ID")
			}
			return err
		}
		commitID = commit.ID.String()

		ok, err := baseGitRepo.IsCommitInBranch(commitID, pr.BaseBranch)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("Wrong commit ID")
		}

		if err := manuallyMergePR(ctx, pr.Issue, commitID, &commit.Author.When, doer); err != nil {
			return fmt.Errorf("manually merging PR: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	notify_service.MergePullRequest(baseGitRepo.Ctx, doer, pr)
	log.Info("manuallyMerged[%d]: Marked as manually merged into %s/%s by commit id: %s", pr.ID, pr.BaseRepo.Name, pr.BaseBranch, commitID)

	return handleCloseCrossReferences(ctx, pr, doer)
}

// CloseManuallyMergedPRs finds references to open PRs within the given commits
// and auto-closes all existing pull requests, if there's a match.
//
// This is the case when branchName matches the default branch for this repo
// and commits use one of the manual merge keywords to mention PRs in the commits.
//
// In comparison to [MergedManually], this is based purely on the manual
// information provided by the user. It also does not need to be activated via
// the settings, as the security model assumes that only verified users are
// allowed to push to the base branch of the repository.
//
// A mismatch of the branch comparison check is silently ignored.
func CloseManuallyMergedPRs(
	ctx context.Context,
	doer *user_model.User,
	repo *repo_model.Repository,
	commits []*repository.PushCommit,
	branchName string,
) error {
	// At first, we skip operation entirely if we're not on the default branch.
	if branchName != repo.DefaultBranch {
		return nil
	}

	// Commits are appended in the reverse order.
	for _, commit := range slices.Backward(commits) {
		for _, ref := range references.FindAllIssueReferences(commit.Message) {
			// Skip processing for other actions, we're only interested in manual merges.
			if ref.Action != references.XRefActionManuallyMerges {
				continue
			}

			// Referenced PR is maybe from another repo, we have to check that.
			if len(ref.Owner) > 0 && len(ref.Name) > 0 {
				refRepo, err := repo_model.GetRepositoryByOwnerAndName(ctx, ref.Owner, ref.Name)
				if err != nil {
					if repo_model.IsErrRepoNotExist(err) {
						log.Warn("Repository referenced in commit but does not exist: %v", err)
					} else {
						log.Error("repo_model.GetRepositoryByOwnerAndName: %v", err)
					}
					continue
				}

				// We do not want to support cross-repo manually merges right now.
				// Therefore, we make sure that the mentioned repo is actually equal to the current one.
				if refRepo.ID != repo.ID {
					continue
				}
			}

			refIssue, err := getIssueFromRef(ctx, repo, ref.Index)
			if err != nil {
				// If issues do not exist, we ignore it.
				if issues_model.IsErrIssueNotExist(err) {
					continue
				}
				return err
			}

			if err := manuallyMergePR(
				ctx,
				refIssue,
				commit.Sha1,
				&commit.Timestamp,
				doer,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

// getIssueFromRef returns the issue referenced by a ref.
//
// If the issue does not exist, a [issues_model.ErrIssueNoExist] error is returned from the underlying model.
func getIssueFromRef(ctx context.Context, repo *repo_model.Repository, index int64) (*issues_model.Issue, error) {
	issue, err := issues_model.GetIssueByIndex(ctx, repo.ID, index)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

// manuallyMergePR marks the PR behind [refIssue] as merged on behalf of the
// current user and automatically closes the cross references.
func manuallyMergePR(
	ctx context.Context,
	refIssue *issues_model.Issue,
	commitID string,
	commitTimestamp *time.Time,
	doer *user_model.User,
) error {
	// Skip all PRs that are either already closed or not a PR.
	if !refIssue.IsPull || refIssue.IsClosed {
		return nil
	}

	if err := refIssue.LoadPullRequest(ctx); err != nil {
		return fmt.Errorf("loading PR info for ID %d: %w", refIssue.ID, err)
	}

	pr := refIssue.PullRequest
	pr.MergedCommitID = commitID
	pr.MergedUnix = timeutil.TimeStamp(commitTimestamp.Unix())
	pr.Status = issues_model.PullRequestStatusManuallyMerged
	pr.Merger = doer
	pr.MergerID = doer.ID
	if _, err := pr.SetMerged(ctx); err != nil {
		return fmt.Errorf("marking PR %d as merged: %w", pr.ID, err)
	}

	notify_service.MergePullRequest(ctx, doer, pr)
	log.Info("manuallyMerged[%d]: Marked as manually merged into %s/%s by commit id: %s", pr.ID, pr.BaseRepo.Name, pr.BaseBranch, commitID)

	return handleCloseCrossReferences(ctx, pr, doer)
}

// detectManualMerges checks if a pull request got manually merged on the base branch.
//
// This is the case when the commit appears on the base branch.
//
// However, especially when working with signatures or manually squashing a pull request
// via `git merge --squash`, merge requests are not automatically closed.
//
// To cover this, the manual merge keywords were introduced as part of #10129,
// which allow users to specify the PRs that got manually merged. Said behaviour
// is not part of this function and is handled separately in merge_manual.go.
func detectManualMerges(ctx context.Context, pr *issues_model.PullRequest) bool {
	if err := pr.LoadBaseRepo(ctx); err != nil {
		log.Error("%-v LoadBaseRepo: %v", pr, err)
		return false
	}

	if unit, err := pr.BaseRepo.GetUnit(ctx, unit.TypePullRequests); err == nil {
		config := unit.PullRequestsConfig()
		if !config.AutodetectManualMerge {
			return false
		}
	} else {
		log.Error("%-v BaseRepo.GetUnit(unit.TypePullRequests): %v", pr, err)
		return false
	}

	commit, err := getMergeCommit(ctx, pr)
	if err != nil {
		log.Error("%-v getMergeCommit: %v", pr, err)
		return false
	}

	if commit == nil {
		// no merge commit found
		return false
	}

	pr.MergedCommitID = commit.ID.String()
	pr.MergedUnix = timeutil.TimeStamp(commit.Author.When.Unix())
	pr.Status = issues_model.PullRequestStatusManuallyMerged
	merger, _ := user_model.GetUserByEmail(ctx, commit.Author.Email)

	// When the commit author is unknown set the BaseRepo owner as merger
	if merger == nil {
		if pr.BaseRepo.Owner == nil {
			if err = pr.BaseRepo.LoadOwner(ctx); err != nil {
				log.Error("%-v BaseRepo.LoadOwner: %v", pr, err)
				return false
			}
		}
		merger = pr.BaseRepo.Owner
	}
	pr.Merger = merger
	pr.MergerID = merger.ID

	if merged, err := pr.SetMerged(ctx); err != nil {
		log.Error("%-v setMerged : %v", pr, err)
		return false
	} else if !merged {
		return false
	}

	notify_service.MergePullRequest(ctx, merger, pr)

	log.Info("manuallyMerged[%-v]: Marked as manually merged into %s/%s by commit id: %s", pr, pr.BaseRepo.Name, pr.BaseBranch, commit.ID.String())
	return true
}
