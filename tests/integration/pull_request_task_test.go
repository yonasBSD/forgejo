// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	auth_model "code.gitea.io/gitea/models/auth"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	pull_service "code.gitea.io/gitea/services/pull"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestSynchronized(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo10 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})
	owner10 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo10.OwnerID})

	session := loginUser(t, owner10.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner10.Name, repo10.Name), &api.CreatePullRequestOption{
		Head:  "develop",
		Base:  "master",
		Title: "create a success pr",
	}).AddTokenAuth(token)
	pull := new(api.PullRequest)
	resp := MakeRequest(t, req, http.StatusCreated)
	DecodeJSON(t, resp, pull)
	assert.EqualValues(t, "master", pull.Base.Name)

	t.Run("AddTestPullRequestTask", func(t *testing.T) {
		logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
		logChecker.Filter("Updating PR").StopMark("TestPullRequest")
		defer cleanup()

		opt := &repo_module.PushUpdateOptions{
			PusherID:     owner10.ID,
			PusherName:   owner10.Name,
			RepoUserName: owner10.Name,
			RepoName:     repo10.Name,
			RefFullName:  git.RefName("refs/heads/develop"),
			OldCommitID:  pull.Head.Sha,
			NewCommitID:  pull.Head.Sha,
		}
		require.NoError(t, repo_service.PushUpdate(opt))
		logFiltered, logStopped := logChecker.Check(5 * time.Second)
		assert.True(t, logStopped)
		assert.True(t, logFiltered[0])
	})

	for _, testCase := range []struct {
		name     string
		maxPR    int64
		expected bool
	}{
		{
			name:     "TestPullRequest process PR",
			maxPR:    pull.Index,
			expected: true,
		},
		{
			name:     "TestPullRequest skip PR",
			maxPR:    pull.Index - 1,
			expected: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
			logChecker.Filter("Updating PR").StopMark("TestPullRequest")
			defer cleanup()

			pull_service.TestPullRequest(context.Background(), owner10, repo10.ID, testCase.maxPR, "develop", true, pull.Head.Sha, pull.Head.Sha)
			logFiltered, logStopped := logChecker.Check(5 * time.Second)
			assert.True(t, logStopped)
			assert.Equal(t, testCase.expected, logFiltered[0])
		})
	}
}
