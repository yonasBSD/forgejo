// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	actions_model "code.gitea.io/gitea/models/actions"
	repo_model "code.gitea.io/gitea/models/repo"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/test"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestBadges(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		prep := func(t *testing.T) (*repo_model.Repository, func()) {
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

			repo, _, f := CreateDeclarativeRepo(t, owner, "",
				[]unit_model.Type{unit_model.TypeActions},
				[]unit_model.Type{unit_model.TypeIssues, unit_model.TypePullRequests, unit_model.TypeReleases},
				[]*files_service.ChangeRepoFile{
					{
						Operation:     "create",
						TreePath:      ".gitea/workflows/pr.yml",
						ContentReader: strings.NewReader("name: test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
					},
				},
			)

			assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))

			return repo, f
		}

		assertBadge := func(t *testing.T, resp *httptest.ResponseRecorder, badge string) {
			t.Helper()

			assert.Equal(t, fmt.Sprintf("https://img.shields.io/badge/%s", badge), test.RedirectURL(resp))
		}

		t.Run("Workflows", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			// Actions disabled
			req := NewRequest(t, "GET", "/user2/repo1/badges/workflows/test.yaml/badge.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "test.yaml-Not%20found-crimson")

			req = NewRequest(t, "GET", "/user2/repo1/badges/workflows/test.yaml/badge.svg?branch=no-such-branch")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "test.yaml-Not%20found-crimson")

			// Actions enabled
			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg?branch=main", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg?branch=no-such-branch", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg?event=cron", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")

			// GitHub compatibility
			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg?branch=main", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg?branch=no-such-branch", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg?event=cron", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")
		})

		t.Run("Stars", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2/repo1/badges/stars.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)

			assertBadge(t, resp, "stars-0-blue")
		})

		t.Run("Issues", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			// Issues enabled
			req := NewRequest(t, "GET", "/user2/repo1/badges/issues.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-2-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/issues/open.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-1%20open-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/issues/closed.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-1%20closed-blue")

			// Issues disabled
			req = NewRequestf(t, "GET", "/user2/%s/badges/issues.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/issues/open.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/issues/closed.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-Not%20found-crimson")
		})

		t.Run("Pulls", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			// Pull requests enabled
			req := NewRequest(t, "GET", "/user2/repo1/badges/pulls.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-3-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/pulls/open.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-3%20open-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/pulls/closed.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-0%20closed-blue")

			// Pull requests disabled
			req = NewRequestf(t, "GET", "/user2/%s/badges/pulls.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/pulls/open.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/pulls/closed.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-Not%20found-crimson")
		})

		t.Run("Release", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			req := NewRequest(t, "GET", "/user2/repo1/badges/release.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "release-v1.1-blue")

			req = NewRequestf(t, "GET", "/user2/%s/badges/release.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "release-Not%20found-crimson")
		})
	})
}
