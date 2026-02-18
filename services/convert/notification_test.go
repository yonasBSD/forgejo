// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToNotificationThread(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("issue notification", func(t *testing.T) {
		// Notification 1: source=issue, issue_id=1, status=unread
		n := unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 1})
		require.NoError(t, n.LoadAttributes(db.DefaultContext))

		thread := ToNotificationThread(db.DefaultContext, n)
		assert.Equal(t, int64(1), thread.ID)
		assert.True(t, thread.Unread)
		assert.False(t, thread.Pinned)
		require.NotNil(t, thread.Subject)
		assert.Equal(t, api.NotifySubjectIssue, thread.Subject.Type)
		assert.Equal(t, api.StateOpen, thread.Subject.State)
	})

	t.Run("pinned notification", func(t *testing.T) {
		// Notification 3: status=pinned
		n := unittest.AssertExistsAndLoadBean(t, &activities_model.Notification{ID: 3})
		require.NoError(t, n.LoadAttributes(db.DefaultContext))

		thread := ToNotificationThread(db.DefaultContext, n)
		assert.False(t, thread.Unread)
		assert.True(t, thread.Pinned)
	})

	t.Run("merged pull request returns merged state", func(t *testing.T) {
		// Issue 2 is a pull request; pull_request 1 has has_merged=true.
		// Build a notification for this PR to verify the subject state
		// is "merged" using the StateMerged enum constant.
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})

		n := &activities_model.Notification{
			ID:         999,
			UserID:     2,
			RepoID:     repo.ID,
			Status:     activities_model.NotificationStatusUnread,
			Source:     activities_model.NotificationSourcePullRequest,
			IssueID:    issue.ID,
			Issue:      issue,
			Repository: repo,
		}

		thread := ToNotificationThread(db.DefaultContext, n)
		require.NotNil(t, thread.Subject)
		assert.Equal(t, api.NotifySubjectPull, thread.Subject.Type)
		assert.Equal(t, api.StateMerged, thread.Subject.State)
	})

	t.Run("open pull request returns open state", func(t *testing.T) {
		// Issue 3 is a pull request; pull_request 2 has has_merged=false.
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 3})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})

		n := &activities_model.Notification{
			ID:         998,
			UserID:     2,
			RepoID:     repo.ID,
			Status:     activities_model.NotificationStatusUnread,
			Source:     activities_model.NotificationSourcePullRequest,
			IssueID:    issue.ID,
			Issue:      issue,
			Repository: repo,
		}

		thread := ToNotificationThread(db.DefaultContext, n)
		require.NotNil(t, thread.Subject)
		assert.Equal(t, api.NotifySubjectPull, thread.Subject.Type)
		assert.Equal(t, api.StateOpen, thread.Subject.State)
	})
}
