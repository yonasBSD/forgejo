// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/tests"
	"github.com/stretchr/testify/assert"
)

func TestRepoSettingsHookHistory(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	// Request repository hook page with history
	req := NewRequest(t, "GET", "/user2/repo1/settings/hooks/1")
	resp := session.MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)

	t.Run("1/delivered", func(t *testing.T) {
		html, err := doc.doc.Find(".webhook div[data-tab='request-1']").Html()
		assert.NoError(t, err)
		assert.Contains(t, html, "PUT <strong>/matrix-delivered</strong>")
		assert.Contains(t, html, "X-Head: 42</pre>")
	})

	t.Run("2/undelivered", func(t *testing.T) {
		html, err := doc.doc.Find(".webhook div[data-tab='request-2']").Html()
		assert.NoError(t, err)
		assert.Contains(t, html, "PUT <strong>/matrix-undelivered</strong>")
		assert.Contains(t, html, "X-Head: 42</pre>")
	})
}
