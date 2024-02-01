// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	user_service "code.gitea.io/gitea/services/user"

	"github.com/stretchr/testify/assert"
)

func TestUpdateAssignee(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Fake issue with assignees
	issue, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	assert.NoError(t, err)

	err = issue.LoadAttributes(db.DefaultContext)
	assert.NoError(t, err)

	// Assign multiple users
	user2, err := user_model.GetUserByID(db.DefaultContext, 2)
	assert.NoError(t, err)
	_, _, err = issues_model.ToggleIssueAssignee(db.DefaultContext, issue, &user_model.User{ID: 1}, user2.ID)
	assert.NoError(t, err)

	org3, err := user_model.GetUserByID(db.DefaultContext, 3)
	assert.NoError(t, err)
	_, _, err = issues_model.ToggleIssueAssignee(db.DefaultContext, issue, &user_model.User{ID: 1}, org3.ID)
	assert.NoError(t, err)

	user1, err := user_model.GetUserByID(db.DefaultContext, 1) // This user is already assigned (see the definition in fixtures), so running  UpdateAssignee should unassign him
	assert.NoError(t, err)
	_, _, err = issues_model.ToggleIssueAssignee(db.DefaultContext, issue, &user_model.User{ID: 1}, user1.ID)
	assert.NoError(t, err)

	// Check if he got removed
	isAssigned, err := issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, user1)
	assert.NoError(t, err)
	assert.False(t, isAssigned)

	// Check if they're all there
	err = issue.LoadAssignees(db.DefaultContext)
	assert.NoError(t, err)

	var expectedAssignees []*user_model.User
	expectedAssignees = append(expectedAssignees, user2, org3)

	for in, assignee := range issue.Assignees {
		assert.Equal(t, assignee.ID, expectedAssignees[in].ID)
	}

	// Check if the user is assigned
	isAssigned, err = issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, user2)
	assert.NoError(t, err)
	assert.True(t, isAssigned)

	// This user should not be assigned
	isAssigned, err = issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, &user_model.User{ID: 4})
	assert.NoError(t, err)
	assert.False(t, isAssigned)
}

// TestPossibleAssignees verifies that any no-blocked user is assignable
// to an issue.
// This test also checks that not-blocked repo users with write access
// or not-blocked non-repo users with a comment on the issue are in the list
// of likely assignees.
func TestPossibleAssignees(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	user2, err := user_model.GetUserByID(db.DefaultContext, 2)
	assert.NoError(t, err)

	reponame := "big_test_public_mirror_6"
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{Name: reponame})

	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		Repo:     repo,
		Title:    "Minimal issue",
		Content:  "issuecontent1",
		IsPull:   false,
		PosterID: user2.ID,
		Poster:   user2,
		IsClosed: false,
	}
	err = issues_model.InsertIssues(db.DefaultContext, issue)
	assert.NoError(t, err)

	// each user will add a comment to show in the list of likely assignees,
	// then will be blocked by a user
	// of authority for the issue, making the user not assignable anymore
	assignUserThenBlockUser(t, issue, user2, 4, issue.Repo.OwnerID)
	assignUserThenBlockUser(t, issue, user2, 5, issue.PosterID)
	// User 8 is not known to the issue, but still assignable, until
	// user 8 blocks the Doer.
	assignUserThenblockDoer(t, issue, user2, 8, issue.PosterID)
}

func assignUserThenBlockUser(t *testing.T, issue *issues_model.Issue, doer *user_model.User, assigneeID, blockerID int64) {
	assignee, err := user_model.GetUserByID(db.DefaultContext, assigneeID)
	assert.NoError(t, err)

	// The UI uses issues_model.GetIssueLikelyAssignees() to populate
	// the issue assignee drop down.
	// User is not in the list of likely assignees.
	assignees, err := issues_model.GetIssueLikelyAssignees(db.DefaultContext, issue, doer)
	assert.NoError(t, err)
	found := inList(assignee.ID, assignees)
	assert.False(t, found)

	// Even though the user is not a likely assignee, the user is assignable.
	valid := issues_model.UserCanBeAssigned(db.DefaultContext, issue, doer, assignee)
	assert.True(t, valid)

	// User  comments on the issue (Any type of comments would do).
	_, err = issues_model.CreateComment(db.DefaultContext, &issues_model.CreateCommentOptions{
		Type:    issues_model.CommentTypeComment,
		Doer:    assignee,
		Repo:    issue.Repo,
		Issue:   issue,
		Content: "Hello Again",
	})
	assert.NoError(t, err)

	// User is now in the list.
	assignees, err = issues_model.GetIssueLikelyAssignees(db.DefaultContext, issue, doer)
	assert.NoError(t, err)
	found = inList(assignee.ID, assignees)
	assert.True(t, found)

	// Block user.
	err = user_service.BlockUser(db.DefaultContext, blockerID, assignee.ID)
	assert.NoError(t, err)

	// User is not part of that list of likely assignees anymore.
	assignees, err = issues_model.GetIssueLikelyAssignees(db.DefaultContext, issue, doer)
	assert.NoError(t, err)
	found = inList(assignee.ID, assignees)
	assert.False(t, found)

	// User is not assignable.
	valid = issues_model.UserCanBeAssigned(db.DefaultContext, issue, doer, assignee)
	assert.False(t, valid)
}

func assignUserThenblockDoer(t *testing.T, issue *issues_model.Issue, doer *user_model.User, assigneeID, blockerID int64) {
	// Get the possible assignee.
	assignee, err := user_model.GetUserByID(db.DefaultContext, assigneeID)
	assert.NoError(t, err)

	// The UI uses issues_model.GetIssueLikelyAssignees() to populate
	// the issue assignee drop down.
	// User is not in the list of possible assignees.
	assignees, err := issues_model.GetIssueLikelyAssignees(db.DefaultContext, issue, doer)
	assert.NoError(t, err)
	found := inList(assignee.ID, assignees)
	assert.False(t, found)

	// But user is assignable nonetheless.
	valid := issues_model.UserCanBeAssigned(db.DefaultContext, issue, doer, assignee)
	assert.True(t, valid)

	// User  comments on the issue (Any type of comments would do).
	_, err = issues_model.CreateComment(db.DefaultContext, &issues_model.CreateCommentOptions{
		Type:    issues_model.CommentTypeComment,
		Doer:    assignee,
		Repo:    issue.Repo,
		Issue:   issue,
		Content: "Hello Again",
	})
	assert.NoError(t, err)

	// User is now in the list.
	assignees, err = issues_model.GetIssueLikelyAssignees(db.DefaultContext, issue, doer)
	assert.NoError(t, err)
	found = inList(assignee.ID, assignees)
	assert.True(t, found)

	// Block Doer.
	// The assignee has blocked the Doer
	err = user_service.BlockUser(db.DefaultContext, assignee.ID, doer.ID)
	assert.NoError(t, err)

	// User is not assignable anymore.
	valid = issues_model.UserCanBeAssigned(db.DefaultContext, issue, doer, assignee)
	assert.False(t, valid)

	// User is also not proposed as an assignee anymore.
	assignees, err = issues_model.GetIssueLikelyAssignees(db.DefaultContext, issue, doer)
	assert.NoError(t, err)
	found = inList(assignee.ID, assignees)
	assert.False(t, found)
}

func inList(userID int64, list []*user_model.User) bool {
	for _, u := range list {
		if u.ID == userID {
			return true
		}
	}
	return false
}

func TestMakeIDsFromAPIAssigneesToAdd(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	_ = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	_ = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	IDs, err := issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "", []string{""})
	assert.NoError(t, err)
	assert.Equal(t, []int64{}, IDs)

	_, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "", []string{"none_existing_user"})
	assert.Error(t, err)

	IDs, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "user1", []string{"user1"})
	assert.NoError(t, err)
	assert.Equal(t, []int64{1}, IDs)

	IDs, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "user2", []string{""})
	assert.NoError(t, err)
	assert.Equal(t, []int64{2}, IDs)

	IDs, err = issues_model.MakeIDsFromAPIAssigneesToAdd(db.DefaultContext, "", []string{"user1", "user2"})
	assert.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, IDs)
}
