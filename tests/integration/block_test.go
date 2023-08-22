// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"testing"

	"code.gitea.io/gitea/models/db"
	issue_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	forgejo_context "code.gitea.io/gitea/modules/context"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/translation"
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
	assert.True(t, unittest.BeanExists(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID}))
}

func TestBlockUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 8})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	session := loginUser(t, doer.Name)

	t.Run("Block", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		BlockUser(t, doer, blockedUser)
	})

	// Unblock user.
	t.Run("Unblock", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequestWithValues(t, "POST", "/"+blockedUser.Name, map[string]string{
			"_csrf":  GetCSRF(t, session, "/"+blockedUser.Name),
			"action": "unblock",
		})
		resp := session.MakeRequest(t, req, http.StatusSeeOther)

		loc := resp.Header().Get("Location")
		assert.EqualValues(t, "/"+blockedUser.Name, loc)
		unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID})
	})

	t.Run("Organization as target", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		targetOrg := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})

		t.Run("Block", func(t *testing.T) {
			req := NewRequestWithValues(t, "POST", "/"+targetOrg.Name, map[string]string{
				"_csrf":  GetCSRF(t, session, "/"+targetOrg.Name),
				"action": "block",
			})
			resp := session.MakeRequest(t, req, http.StatusBadRequest)

			assert.Contains(t, resp.Body.String(), "Action \\\"block\\\" failed")
		})

		t.Run("Unblock", func(t *testing.T) {
			req := NewRequestWithValues(t, "POST", "/"+targetOrg.Name, map[string]string{
				"_csrf":  GetCSRF(t, session, "/"+targetOrg.Name),
				"action": "unblock",
			})
			resp := session.MakeRequest(t, req, http.StatusBadRequest)

			assert.Contains(t, resp.Body.String(), "Action \\\"unblock\\\" failed")
		})
	})
}

func TestBlockIssueCreation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: doer.ID})
	BlockUser(t, doer, blockedUser)

	session := loginUser(t, blockedUser.Name)
	req := NewRequest(t, "GET", "/"+repo.OwnerName+"/"+repo.Name+"/issues/new")
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find("form.ui.form").Attr("action")
	assert.True(t, exists)
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf":   htmlDoc.GetCSRF(),
		"title":   "Title",
		"content": "Hello!",
	})

	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assert.Contains(t,
		htmlDoc.doc.Find(".ui.negative.message").Text(),
		translation.NewLocale("en-US").Tr("repo.issues.blocked_by_user"),
	)
}

func TestBlockCommentCreation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	expectedFlash := "error%3DYou%2Bcannot%2Bcreate%2Ba%2Bcomment%2Bon%2Bthis%2Bissue%2Bbecause%2Byou%2Bare%2Bblocked%2Bby%2Bthe%2Brepository%2Bowner%2Bor%2Bthe%2Bposter%2Bof%2Bthe%2Bissue."
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	BlockUser(t, doer, blockedUser)

	t.Run("Blocked by repository owner", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: doer.ID})
		issue := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 4, RepoID: repo.ID})
		issueURL := fmt.Sprintf("/%s/%s/issues/%d", url.PathEscape(repo.OwnerName), url.PathEscape(repo.Name), issue.Index)

		session := loginUser(t, blockedUser.Name)
		req := NewRequest(t, "GET", issueURL)
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		req = NewRequestWithValues(t, "POST", path.Join(issueURL, "/comments"), map[string]string{
			"_csrf":   htmlDoc.GetCSRF(),
			"content": "Not a kind comment",
		})
		session.MakeRequest(t, req, http.StatusOK)

		flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.EqualValues(t, expectedFlash, flashCookie.Value)
	})

	t.Run("Blocked by issue poster", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 5})
		issue := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 15, RepoID: repo.ID, PosterID: doer.ID})
		issueURL := fmt.Sprintf("/%s/%s/issues/%d", url.PathEscape(repo.OwnerName), url.PathEscape(repo.Name), issue.Index)

		session := loginUser(t, blockedUser.Name)
		req := NewRequest(t, "GET", issueURL)
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		req = NewRequestWithValues(t, "POST", path.Join(issueURL, "/comments"), map[string]string{
			"_csrf":   htmlDoc.GetCSRF(),
			"content": "Not a kind comment",
		})
		session.MakeRequest(t, req, http.StatusOK)

		flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.EqualValues(t, expectedFlash, flashCookie.Value)
	})
}

func TestBlockIssueReaction(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	issue := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 4, PosterID: doer.ID, RepoID: repo.ID})
	issueURL := fmt.Sprintf("/%s/%s/issues/%d", url.PathEscape(repo.OwnerName), url.PathEscape(repo.Name), issue.Index)

	BlockUser(t, doer, blockedUser)

	session := loginUser(t, blockedUser.Name)
	req := NewRequest(t, "GET", issueURL)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	req = NewRequestWithValues(t, "POST", path.Join(issueURL, "/reactions/react"), map[string]string{
		"_csrf":   htmlDoc.GetCSRF(),
		"content": "eyes",
	})
	resp = session.MakeRequest(t, req, http.StatusOK)
	type reactionResponse struct {
		Empty bool `json:"empty"`
	}

	var respBody reactionResponse
	DecodeJSON(t, resp, &respBody)

	assert.EqualValues(t, true, respBody.Empty)
}

func TestBlockCommentReaction(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 1, RepoID: repo.ID})
	comment := unittest.AssertExistsAndLoadBean(t, &issue_model.Comment{ID: 3, PosterID: doer.ID, IssueID: issue.ID})
	_ = comment.LoadIssue(db.DefaultContext)
	issueURL := fmt.Sprintf("/%s/%s/issues/%d", url.PathEscape(repo.OwnerName), url.PathEscape(repo.Name), issue.Index)

	BlockUser(t, doer, blockedUser)

	session := loginUser(t, blockedUser.Name)
	req := NewRequest(t, "GET", issueURL)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	req = NewRequestWithValues(t, "POST", path.Join(repo.Link(), "/comments/", strconv.FormatInt(comment.ID, 10), "/reactions/react"), map[string]string{
		"_csrf":   htmlDoc.GetCSRF(),
		"content": "eyes",
	})
	resp = session.MakeRequest(t, req, http.StatusOK)
	type reactionResponse struct {
		Empty bool `json:"empty"`
	}

	var respBody reactionResponse
	DecodeJSON(t, resp, &respBody)

	assert.EqualValues(t, true, respBody.Empty)
}

// TestBlockFollow ensures that the doer and blocked user cannot follow each other.
func TestBlockFollow(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	BlockUser(t, doer, blockedUser)

	// Doer cannot follow blocked user.
	t.Run("Doer follow blocked user", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		session := loginUser(t, doer.Name)

		req := NewRequestWithValues(t, "POST", "/"+blockedUser.Name, map[string]string{
			"_csrf":  GetCSRF(t, session, "/"+blockedUser.Name),
			"action": "follow",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		unittest.AssertNotExistsBean(t, &user_model.Follow{UserID: doer.ID, FollowID: blockedUser.ID})
	})

	// Blocked user cannot follow doer.
	t.Run("Blocked user follow doer", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		session := loginUser(t, blockedUser.Name)

		req := NewRequestWithValues(t, "POST", "/"+doer.Name, map[string]string{
			"_csrf":  GetCSRF(t, session, "/"+doer.Name),
			"action": "follow",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		unittest.AssertNotExistsBean(t, &user_model.Follow{UserID: blockedUser.ID, FollowID: doer.ID})
	})
}

// TestBlockUserFromOrganization ensures that an organisation can block and unblock an user.
func TestBlockUserFromOrganization(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 17, Type: user_model.UserTypeOrganization})
	unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: org.ID})
	session := loginUser(t, doer.Name)

	t.Run("Block user", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/block", map[string]string{
			"_csrf": GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
			"uname": blockedUser.Name,
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		assert.True(t, unittest.BeanExists(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: org.ID}))
	})

	t.Run("Unblock user", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/unblock", map[string]string{
			"_csrf":   GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
			"user_id": strconv.FormatInt(blockedUser.ID, 10),
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: org.ID})
	})

	t.Run("Organization as target", func(t *testing.T) {
		targetOrg := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})

		t.Run("Block", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/block", map[string]string{
				"_csrf": GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
				"uname": targetOrg.Name,
			})
			session.MakeRequest(t, req, http.StatusInternalServerError)
			unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: targetOrg.ID})
		})

		t.Run("Unblock", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/unblock", map[string]string{
				"_csrf":   GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
				"user_id": strconv.FormatInt(targetOrg.ID, 10),
			})
			session.MakeRequest(t, req, http.StatusInternalServerError)
		})
	})
}

// TestBlockAddCollaborator ensures that the doer and blocked user cannot add each each other as collaborators.
func TestBlockAddCollaborator(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	BlockUser(t, user1, user2)

	t.Run("BlockedUser Add Doer", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: user2.ID})

		session := loginUser(t, user2.Name)
		req := NewRequestWithValues(t, "POST", path.Join(repo.Link(), "/settings/collaboration"), map[string]string{
			"_csrf":        GetCSRF(t, session, path.Join(repo.Link(), "/settings/collaboration")),
			"collaborator": user1.Name,
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.EqualValues(t, "error%3DCannot%2Badd%2Bthe%2Bcollaborator%252C%2Bbecause%2Bthey%2Bhave%2Bblocked%2Bthe%2Brepository%2Bowner.", flashCookie.Value)
	})

	t.Run("Doer Add BlockedUser", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 7, OwnerID: user1.ID})

		session := loginUser(t, user1.Name)
		req := NewRequestWithValues(t, "POST", path.Join(repo.Link(), "/settings/collaboration"), map[string]string{
			"_csrf":        GetCSRF(t, session, path.Join(repo.Link(), "/settings/collaboration")),
			"collaborator": user2.Name,
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.EqualValues(t, "error%3DCannot%2Badd%2Bthe%2Bcollaborator%252C%2Bbecause%2Bthe%2Brepository%2Bowner%2Bhas%2Bblocked%2Bthem.", flashCookie.Value)
	})
}
