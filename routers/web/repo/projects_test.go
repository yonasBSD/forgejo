// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"testing"

	project_model "forgejo.org/models/project"
	"forgejo.org/models/unittest"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckProjectColumnChangePermissions(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "user2/repo1/projects/1/2")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	ctx.SetParams(":id", "1")
	ctx.SetParams(":columnID", "2")

	project, column := checkProjectColumnChangePermissions(ctx)
	assert.NotNil(t, project)
	assert.NotNil(t, column)
	assert.False(t, ctx.Written())
}

func TestUpdateIssueProjectColumn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		unittest.PrepareTestEnv(t)
		// Issue 1 is in repo 1, project 1, column 1 ("To Do"). Move to column 2 ("In Progress").
		ctx, resp := contexttest.MockContext(t, "user2/repo1/issues/projects/column")
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, 1)
		ctx.Req.Form.Set("issue_id", "1")
		ctx.Req.Form.Set("column_id", "2")

		UpdateIssueProjectColumn(ctx)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Verify the card was moved to column 2
		card, err := project_model.GetProjectCard(ctx, 1, 1)
		require.NoError(t, err)
		assert.EqualValues(t, 2, card.ProjectColumnID)
	})

	t.Run("WrongRepo", func(t *testing.T) {
		unittest.PrepareTestEnv(t)
		// Issue 4 is in repo 2, but we load repo 1 — should 404
		ctx, resp := contexttest.MockContext(t, "user2/repo1/issues/projects/column")
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, 1)
		ctx.Req.Form.Set("issue_id", "4")
		ctx.Req.Form.Set("column_id", "2")

		UpdateIssueProjectColumn(ctx)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("NoProject", func(t *testing.T) {
		unittest.PrepareTestEnv(t)
		// Issue 11 is in repo 1 but has no project assigned
		ctx, resp := contexttest.MockContext(t, "user2/repo1/issues/projects/column")
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, 1)
		ctx.Req.Form.Set("issue_id", "11")
		ctx.Req.Form.Set("column_id", "2")

		UpdateIssueProjectColumn(ctx)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("CrossProjectColumn", func(t *testing.T) {
		unittest.PrepareTestEnv(t)
		// Issue 1 is in project 1. Column 4 belongs to project 4 — should 404
		ctx, resp := contexttest.MockContext(t, "user2/repo1/issues/projects/column")
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, 1)
		ctx.Req.Form.Set("issue_id", "1")
		ctx.Req.Form.Set("column_id", "4")

		UpdateIssueProjectColumn(ctx)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}
