// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	forgefed_model "code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers"

	"github.com/stretchr/testify/assert"
)

func TestActivityPubRepository(t *testing.T) {
	setting.Federation.Enabled = true
	testWebRoutes = routers.NormalRoutes()
	defer func() {
		setting.Federation.Enabled = false
		testWebRoutes = routers.NormalRoutes()
	}()

	onGiteaRun(t, func(*testing.T, *url.URL) {
		repositoryID := 2
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/activitypub/repository-id/%v", repositoryID))
		resp := MakeRequest(t, req, http.StatusOK)
		body := resp.Body.Bytes()
		assert.Contains(t, string(body), "@context")

		var repository forgefed_model.Repository
		err := repository.UnmarshalJSON(body)
		assert.NoError(t, err)

		assert.Regexp(t, fmt.Sprintf("activitypub/repository-id/%v$", repositoryID), repository.GetID().String())
	})
}

func TestActivityPubMissingRepository(t *testing.T) {
	setting.Federation.Enabled = true
	testWebRoutes = routers.NormalRoutes()
	defer func() {
		setting.Federation.Enabled = false
		testWebRoutes = routers.NormalRoutes()
	}()

	onGiteaRun(t, func(*testing.T, *url.URL) {
		repositoryID := 9999999
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/activitypub/repository-id/%v", repositoryID))
		resp := MakeRequest(t, req, http.StatusNotFound)
		assert.Contains(t, resp.Body.String(), "repository does not exist")
	})
}
