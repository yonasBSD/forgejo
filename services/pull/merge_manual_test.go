// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package pull_test

import (
	"testing"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/repository"
	"forgejo.org/services/pull"

	"github.com/stretchr/testify/require"
)

func TestCloseManuallyMergedPRs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pushCommits := []*repository.PushCommit{
		{
			Sha1:           "abcdef1",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user4@example.com",
			AuthorName:     "User Four",
			Message:        "Merge branch 'foo' into 'main'\n\nMerges: #3",
		},
		{
			Sha1:           "abcdef2",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User Two",
			Message:        "a plain message",
		},
	}

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	repo.Owner = user

	issueBean := &issues_model.Issue{RepoID: repo.ID, Index: 3}
	prBean := &issues_model.PullRequest{HeadRepoID: repo.ID, Index: 3}

	require.NoError(t, pull.CloseManuallyMergedPRs(db.DefaultContext, user, repo, pushCommits, repo.DefaultBranch))
	unittest.AssertExistsAndLoadBean(t, issueBean, "is_closed=1")
	unittest.AssertExistsAndLoadBean(t, prBean)

	require.Equal(t, "abcdef1", prBean.MergedCommitID)
	require.True(t, prBean.HasMerged)
	unittest.CheckConsistencyFor(t, &activities_model.Action{})
}
