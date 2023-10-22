// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestFeedRepoCommits(t *testing.T) {
	// This test verifies that feeds for repo commits are working.
	defer tests.PrepareTestEnv(t)()

	t.Run("Feed repo commits", func(t *testing.T) {
		t.Run("Atom", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2/repo1/commits.atom")
			resp := MakeRequest(t, req, http.StatusOK)

			data := resp.Body.String()
			assert.Contains(t, data, "<title>Commits for user2/repo1</title>")
		})

		t.Run("RSS", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2/repo1/commits.rss")
			resp := MakeRequest(t, req, http.StatusOK)

			data := resp.Body.String()
			assert.Contains(t, data, "<title>Commits for user2/repo1</title>")
		})
	})
}
