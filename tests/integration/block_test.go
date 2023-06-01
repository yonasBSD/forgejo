// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"
	"github.com/stretchr/testify/assert"
)

func BlockUser(t *testing.T, doer, blockedUser *user_model.User) {
	t.Helper()

	unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID})

	session := loginUser(t, doer.Name)
	req := NewRequestWithValues(t, "POST", "/"+blockedUser.Name, map[string]string{
		"_csrf":  GetCSRF(t, session, "/"+blockedUser.Name),
		"action": "block",
	})
	resp := session.MakeRequest(t, req, http.StatusOK)

	type redirect struct {
		Redirect string `json:"redirect"`
	}

	var respBody redirect
	DecodeJSON(t, resp, &respBody)
	assert.EqualValues(t, "/"+blockedUser.Name, respBody.Redirect)
	assert.EqualValues(t, true, unittest.BeanExists(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID}))
}

func TestBlockUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 8})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	BlockUser(t, doer, blockedUser)

	// Unblock user.
	session := loginUser(t, doer.Name)
	req := NewRequestWithValues(t, "POST", "/"+blockedUser.Name, map[string]string{
		"_csrf":  GetCSRF(t, session, "/"+blockedUser.Name),
		"action": "unblock",
	})
	resp := session.MakeRequest(t, req, http.StatusSeeOther)

	loc := resp.Header().Get("Location")
	assert.EqualValues(t, "/"+blockedUser.Name, loc)
	unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID})
}
