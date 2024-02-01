// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"
	"fmt"
	"sort"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"

	"xorm.io/builder"
)

// IssueAssignees saves all issue assignees
type IssueAssignees struct {
	ID         int64 `xorm:"pk autoincr"`
	AssigneeID int64 `xorm:"INDEX"`
	IssueID    int64 `xorm:"INDEX"`
}

func init() {
	db.RegisterModel(new(IssueAssignees))
}

// UserCanBeAssigned validate that the assignee has not been blocked, and that
// the assignee has not blocked the doer (if the doer is provided).
func UserCanBeAssigned(ctx context.Context, issue *Issue, doer, assignee *user_model.User) bool {
	// first check that the user has not been blocked by someone that can do so
	// for this issue.
	blocked := user_model.IsBlockedMultiple(ctx, []int64{issue.PosterID, issue.Repo.OwnerID}, assignee.ID)
	if blocked {
		return false
	}
	// Has the doer been blocked by the assignee?
	if doer != nil {
		blocked := user_model.IsBlocked(ctx, assignee.ID, doer.ID)
		if blocked {
			return false
		}
	}
	return true // user can bee assigned.
}

// GetIssueLikelyAssignees returns all users that could be assigned
// to the specific issue. This includes users that may not part of
// the org/teams.
// Note. This is similar to repo.GetRepoAssignees(), but also returns
// users that have posted to the issue in some way (comment, PR etc...)
// Blocked users are not part of the returned list.
func GetIssueLikelyAssignees(ctx context.Context, issue *Issue, doer *user_model.User) ([]*user_model.User, error) {
	repoUsers, err := repo_model.GetRepoAssignees(ctx, issue.Repo)
	if err != nil {
		return nil, err
	}
	e := db.GetEngine(ctx)

	repoUserIDs := make([]int64, 0, len(repoUsers))
	for _, repoUser := range repoUsers {
		repoUserIDs = append(repoUserIDs, repoUser.ID)
	}
	log.Debug("Repo users ids: %v", repoUserIDs)

	// Comment of any type. Grab the poster_id for each relevant comment.
	// Note: A poster may not have write access to the repo.
	haveCommentUserIDs := make([]int64, 0, 20)
	if err := e.Table("comment").
		Join("INNER", "user", "`user`.id = `comment`.poster_id").
		Distinct("`comment`.poster_id").
		Where("`comment`.issue_id = ?", issue.ID).
		And("`user`.type = 0").
		NotIn("`comment`.poster_id", repoUserIDs).
		Find(&haveCommentUserIDs); err != nil {
		return nil, err
	}
	log.Debug("Posters user ids: %v", haveCommentUserIDs)

	// Also grab users that have been assigned by someone else, via
	// the new functionality (add a new user)
	posteeUserIDs := make([]int64, 0, 10)
	notInUserIDs := make([]int64, 0, 20)
	notInUserIDs = append(notInUserIDs, repoUserIDs...)
	notInUserIDs = append(notInUserIDs, haveCommentUserIDs...)
	if err := e.Table("comment").
		Join("INNER", "user", "`user`.id = `comment`.assignee_id").
		Distinct("`comment`.assignee_id").
		Where("`comment`.issue_id = ?", issue.ID).
		And("`comment`.type = ?", CommentTypeAssignees).
		And("`user`.type = 0").
		NotIn("`comment`.assignee_id", notInUserIDs).
		Find(&posteeUserIDs); err != nil {
		return nil, err
	}
	log.Debug("Postees: %v", posteeUserIDs)

	// Add the postees to the haveCommentUserIDs slice.
	haveCommentUserIDs = append(haveCommentUserIDs, posteeUserIDs...)

	posters := make([]*user_model.User, 0, len(haveCommentUserIDs))
	if len(haveCommentUserIDs) > 0 {
		if err = e.In("id", haveCommentUserIDs).OrderBy(user_model.GetOrderByName()).Find(&posters); err != nil {
			return nil, err
		}
	}
	// Make one user list out of the 2 user lists.
	users := make([]*user_model.User, 0, len(repoUsers)+len(posters))
	users = append(users, repoUsers...)
	users = append(users, posters...)
	sort.Slice(users, func(p, q int) bool {
		return users[p].LowerName < users[q].LowerName
	})

	// Remove any of the blocked users.
	blockedUsers, err := user_model.ListBlockedUsers(ctx, []int64{issue.PosterID, issue.Repo.OwnerID}, db.ListOptions{})
	if err != nil {
		return nil, err
	}
	okUsers := make([]*user_model.User, 0, len(users))
LOOP:
	for _, aUser := range users {
		for _, blocked := range blockedUsers {
			if blocked.ID == aUser.ID {
				continue LOOP
			}
		}
		okUsers = append(okUsers, aUser)
	}

	// and also remove any users that have blocked the doer.
	if doer != nil {
		blockerIDs, err := user_model.ListBlockedByUsersID(ctx, doer.ID)
		if err != nil {
			return nil, err
		}
		if len(blockerIDs) > 0 {
			newList := make([]*user_model.User, 0, len(okUsers))
		BLOCKED_USERS:
			for _, u := range okUsers {
				for _, buid := range blockerIDs {
					if buid == u.ID {
						continue BLOCKED_USERS
					}
				}
				newList = append(newList, u)
			}
			// re-assign
			okUsers = newList
		}
	}
	return okUsers, nil
}

// LoadAssignees load assignees of this issue.
func (issue *Issue) LoadAssignees(ctx context.Context) (err error) {
	// Reset maybe preexisting assignees
	issue.Assignees = []*user_model.User{}
	issue.Assignee = nil

	err = db.GetEngine(ctx).Table("`user`").
		Join("INNER", "issue_assignees", "assignee_id = `user`.id").
		Where("issue_assignees.issue_id = ?", issue.ID).
		Find(&issue.Assignees)
	if err != nil {
		return err
	}

	// Check if we have at least one assignee and if yes put it in as `Assignee`
	if len(issue.Assignees) > 0 {
		issue.Assignee = issue.Assignees[0]
	}
	return err
}

// GetAssigneeIDsByIssue returns the IDs of users assigned to an issue
// but skips joining with `user` for performance reasons.
// User permissions must be verified elsewhere if required.
func GetAssigneeIDsByIssue(ctx context.Context, issueID int64) ([]int64, error) {
	userIDs := make([]int64, 0, 5)
	return userIDs, db.GetEngine(ctx).
		Table("issue_assignees").
		Cols("assignee_id").
		Where("issue_id = ?", issueID).
		Distinct("assignee_id").
		Find(&userIDs)
}

// IsUserAssignedToIssue returns true when the user is assigned to the issue
func IsUserAssignedToIssue(ctx context.Context, issue *Issue, user *user_model.User) (isAssigned bool, err error) {
	return db.Exist[IssueAssignees](ctx, builder.Eq{"assignee_id": user.ID, "issue_id": issue.ID})
}

// ToggleIssueAssignee changes a user between assigned and not assigned for this issue, and make issue comment for it.
func ToggleIssueAssignee(ctx context.Context, issue *Issue, doer *user_model.User, assigneeID int64) (removed bool, comment *Comment, err error) {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return false, nil, err
	}
	defer committer.Close()

	removed, comment, err = toggleIssueAssignee(ctx, issue, doer, assigneeID, false)
	if err != nil {
		return false, nil, err
	}

	if err := committer.Commit(); err != nil {
		return false, nil, err
	}

	return removed, comment, nil
}

func toggleIssueAssignee(ctx context.Context, issue *Issue, doer *user_model.User, assigneeID int64, isCreate bool) (removed bool, comment *Comment, err error) {
	removed, err = toggleUserAssignee(ctx, issue, assigneeID)
	if err != nil {
		return false, nil, fmt.Errorf("UpdateIssueUserByAssignee: %w", err)
	}

	// Repo infos
	if err = issue.LoadRepo(ctx); err != nil {
		return false, nil, fmt.Errorf("loadRepo: %w", err)
	}

	opts := &CreateCommentOptions{
		Type:            CommentTypeAssignees,
		Doer:            doer,
		Repo:            issue.Repo,
		Issue:           issue,
		RemovedAssignee: removed,
		AssigneeID:      assigneeID,
	}
	// Comment
	comment, err = CreateComment(ctx, opts)
	if err != nil {
		return false, nil, fmt.Errorf("createComment: %w", err)
	}

	// if pull request is in the middle of creation - don't call webhook
	if isCreate {
		return removed, comment, err
	}

	return removed, comment, nil
}

// toggles user assignee state in database
func toggleUserAssignee(ctx context.Context, issue *Issue, assigneeID int64) (removed bool, err error) {
	// Check if the user exists
	assignee, err := user_model.GetUserByID(ctx, assigneeID)
	if err != nil {
		return false, err
	}

	// Check if the submitted user is already assigned, if yes delete him otherwise add him
	found := false
	i := 0
	for ; i < len(issue.Assignees); i++ {
		if issue.Assignees[i].ID == assigneeID {
			found = true
			break
		}
	}

	assigneeIn := IssueAssignees{AssigneeID: assigneeID, IssueID: issue.ID}
	if found {
		issue.Assignees = append(issue.Assignees[:i], issue.Assignees[i+1:]...)
		_, err = db.DeleteByBean(ctx, &assigneeIn)
		if err != nil {
			return found, err
		}
	} else {
		issue.Assignees = append(issue.Assignees, assignee)
		if err = db.Insert(ctx, &assigneeIn); err != nil {
			return found, err
		}
	}

	return found, nil
}

// MakeIDsFromAPIAssigneesToAdd returns an array with all assignee IDs
func MakeIDsFromAPIAssigneesToAdd(ctx context.Context, oneAssignee string, multipleAssignees []string) (assigneeIDs []int64, err error) {
	var requestAssignees []string

	// Keeping the old assigning method for compatibility reasons
	if oneAssignee != "" && !util.SliceContainsString(multipleAssignees, oneAssignee) {
		requestAssignees = append(requestAssignees, oneAssignee)
	}

	// Prevent empty assignees
	if len(multipleAssignees) > 0 && multipleAssignees[0] != "" {
		requestAssignees = append(requestAssignees, multipleAssignees...)
	}

	// Get the IDs of all assignees
	assigneeIDs, err = user_model.GetUserIDsByNames(ctx, requestAssignees, false)

	return assigneeIDs, err
}
