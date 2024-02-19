// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func syncForkTest(t *testing.T, forkName, urlPart string) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})

	baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	baseUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: baseRepo.OwnerID})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	/// Create a new fork
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/forks", baseUser.Name, baseRepo.LowerName), &api.CreateForkOption{Name: &forkName}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusAccepted)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var syncForkInfo *api.SyncForkInfo
	DecodeJSON(t, resp, &syncForkInfo)

	// This is a new fork, so the commits in both branches should be the same
	assert.False(t, syncForkInfo.Allowed)
	assert.Equal(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)

	// Make a commit on the base branch
	err := createOrReplaceFileInBranch(baseUser, baseRepo, "sync_fork.txt", "master", "Hello")
	require.NoError(t, err)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &syncForkInfo)

	// The commits should no longer be the same and we can sync
	assert.True(t, syncForkInfo.Allowed)
	assert.NotEqual(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)

	// Sync the fork
	req = NewRequestf(t, "POST", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusNoContent)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &syncForkInfo)

	// After the sync both commits should be the same again
	assert.False(t, syncForkInfo.Allowed)
	assert.Equal(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)
}

func TestAPIRepoSyncForkDefault(t *testing.T) {
	syncForkTest(t, "SyncForkDefault", "sync_fork")
}

func TestAPIRepoSyncForkBranch(t *testing.T) {
	syncForkTest(t, "SyncForkBranch", "sync_fork/master")
}
