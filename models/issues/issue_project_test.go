// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/issues"
	"forgejo.org/models/organization"
	"forgejo.org/models/project"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProjectColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Issue with project column", func(t *testing.T) {
		// Issue 1 is in project 1, column 1 ("To Do")
		issue := unittest.AssertExistsAndLoadBean(t, &issues.Issue{ID: 1})
		col, err := issue.LoadProjectColumn(db.DefaultContext)
		require.NoError(t, err)
		require.NotNil(t, col)
		assert.Equal(t, "To Do", col.Title)
		assert.EqualValues(t, 1, col.ID)
	})

	t.Run("Issue with project but no column assigned", func(t *testing.T) {
		// Issue 2 is in project 1, but project_board_id=0 (no column)
		issue := unittest.AssertExistsAndLoadBean(t, &issues.Issue{ID: 2})
		col, err := issue.LoadProjectColumn(db.DefaultContext)
		require.NoError(t, err)
		assert.Nil(t, col)
	})

	t.Run("Issue not in any project", func(t *testing.T) {
		// Issue 4 is in repo 2, not assigned to any project
		issue := unittest.AssertExistsAndLoadBean(t, &issues.Issue{ID: 4})
		col, err := issue.LoadProjectColumn(db.DefaultContext)
		require.NoError(t, err)
		assert.Nil(t, col)
	})
}

func TestPrivateIssueProjects(t *testing.T) {
	defer unittest.OverrideFixtures("models/fixtures/PrivateIssueProjects")()
	require.NoError(t, unittest.PrepareTestDatabase())

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	t.Run("Organization project", func(t *testing.T) {
		org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
		orgProject := unittest.AssertExistsAndLoadBean(t, &project.Project{ID: 1001, OwnerID: org.ID})
		column := unittest.AssertExistsAndLoadBean(t, &project.Column{ID: 1001, ProjectID: orgProject.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, user2, org, optional.None[bool]())
			require.NoError(t, err)
			assert.Len(t, issueList, 2)
			assert.EqualValues(t, 16, issueList[0].ID)
			assert.EqualValues(t, 6, issueList[1].ID)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.None[bool]())
			require.NoError(t, err)
			assert.Equal(t, 2, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.Some(true))
			require.NoError(t, err)
			assert.Equal(t, 0, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.Some(false))
			require.NoError(t, err)
			assert.Equal(t, 2, issuesNum)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, nil, org, optional.None[bool]())
			require.NoError(t, err)
			assert.Len(t, issueList, 1)
			assert.EqualValues(t, 16, issueList[0].ID)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, orgProject, nil, org, optional.None[bool]())
			require.NoError(t, err)
			assert.Equal(t, 1, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, nil, org, optional.Some(true))
			require.NoError(t, err)
			assert.Equal(t, 0, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, nil, org, optional.Some(false))
			require.NoError(t, err)
			assert.Equal(t, 1, issuesNum)
		})
	})

	t.Run("User project", func(t *testing.T) {
		userProject := unittest.AssertExistsAndLoadBean(t, &project.Project{ID: 1002, OwnerID: user2.ID})
		column := unittest.AssertExistsAndLoadBean(t, &project.Column{ID: 1002, ProjectID: userProject.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, user2, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.Len(t, issueList, 2)
			assert.EqualValues(t, 7, issueList[0].ID)
			assert.EqualValues(t, 1, issueList[1].ID)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, userProject, user2, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.Equal(t, 2, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, userProject, user2, nil, optional.Some(true))
			require.NoError(t, err)
			assert.Equal(t, 0, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, userProject, user2, nil, optional.Some(false))
			require.NoError(t, err)
			assert.Equal(t, 2, issuesNum)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, nil, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.Len(t, issueList, 1)
			assert.EqualValues(t, 1, issueList[0].ID)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, userProject, nil, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.Equal(t, 1, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, userProject, nil, nil, optional.Some(true))
			require.NoError(t, err)
			assert.Equal(t, 0, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, userProject, nil, nil, optional.Some(false))
			require.NoError(t, err)
			assert.Equal(t, 1, issuesNum)
		})
	})
}

func TestPrivateRepoProjects(t *testing.T) {
	defer unittest.OverrideFixtures("models/fixtures/TestPrivateRepoProjects")()
	require.NoError(t, unittest.PrepareTestDatabase())

	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	orgProject := unittest.AssertExistsAndLoadBean(t, &project.Project{ID: 1001, OwnerID: org.ID})
	column := unittest.AssertExistsAndLoadBean(t, &project.Column{ID: 1001, ProjectID: orgProject.ID})

	t.Run("Partial access", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		user29 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 29})

		issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, user29, org, optional.None[bool]())
		require.NoError(t, err)
		assert.Len(t, issueList, 1)
		assert.EqualValues(t, 6, issueList[0].ID)

		issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, orgProject, user29, org, optional.None[bool]())
		require.NoError(t, err)
		assert.Equal(t, 1, issuesNum)

		issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user29, org, optional.Some(true))
		require.NoError(t, err)
		assert.Equal(t, 0, issuesNum)

		issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user29, org, optional.Some(false))
		require.NoError(t, err)
		assert.Equal(t, 1, issuesNum)
	})

	t.Run("Full access", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, user2, org, optional.None[bool]())
		require.NoError(t, err)
		assert.Len(t, issueList, 2)
		assert.EqualValues(t, 15, issueList[0].ID)
		assert.EqualValues(t, 6, issueList[1].ID)

		issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.None[bool]())
		require.NoError(t, err)
		assert.Equal(t, 2, issuesNum)

		issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.Some(true))
		require.NoError(t, err)
		assert.Equal(t, 0, issuesNum)

		issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.Some(false))
		require.NoError(t, err)
		assert.Equal(t, 2, issuesNum)
	})
}
