// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserFork(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User13 has repo 11 forked from repo10
	repo, err := repo_model.GetRepositoryByID(db.DefaultContext, 10)
	require.NoError(t, err)
	assert.NotNil(t, repo)
	repo, err = repo_model.GetUserFork(db.DefaultContext, repo.ID, 13)
	require.NoError(t, err)
	assert.NotNil(t, repo)

	repo, err = repo_model.GetRepositoryByID(db.DefaultContext, 9)
	require.NoError(t, err)
	assert.NotNil(t, repo)
	repo, err = repo_model.GetUserFork(db.DefaultContext, repo.ID, 13)
	require.NoError(t, err)
	assert.Nil(t, repo)
}
