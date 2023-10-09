// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integration

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/convert"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIGetCommentAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
	assert.NoError(t, comment.LoadIssue(db.DefaultContext))
	assert.NoError(t, comment.LoadAttachments(db.DefaultContext))
	attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: comment.Attachments[0].ID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: comment.Issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	t.Run("UnrelatedCommentID", func(t *testing.T) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
		repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
		token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)
		req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/comments/%d/assets/%d?token=%s", repoOwner.Name, repo.Name, comment.ID, attachment.ID, token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)
	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/comments/%d/assets/%d?token=%s", repoOwner.Name, repo.Name, comment.ID, attachment.ID, token)
	session.MakeRequest(t, req, http.StatusOK)
	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/comments/%d/assets/%d?token=%s", repoOwner.Name, repo.Name, comment.ID, attachment.ID, token)
	resp := session.MakeRequest(t, req, http.StatusOK)

	var apiAttachment api.Attachment
	DecodeJSON(t, resp, &apiAttachment)

	expect := convert.ToAPIAttachment(repo, attachment)
	assert.Equal(t, expect.ID, apiAttachment.ID)
	assert.Equal(t, expect.Name, apiAttachment.Name)
	assert.Equal(t, expect.UUID, apiAttachment.UUID)
	assert.Equal(t, expect.Created.Unix(), apiAttachment.Created.Unix())
	assert.Equal(t, expect.DownloadURL, apiAttachment.DownloadURL)
}

func TestAPIListCommentAttachments(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)
	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/comments/%d/assets?token=%s",
		repoOwner.Name, repo.Name, comment.ID, token)
	resp := session.MakeRequest(t, req, http.StatusOK)

	var apiAttachments []*api.Attachment
	DecodeJSON(t, resp, &apiAttachments)
	expectedCount := unittest.GetCount(t, &repo_model.Attachment{CommentID: comment.ID})
	assert.Len(t, apiAttachments, expectedCount)

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachments[0].ID, CommentID: comment.ID})
}

func TestAPICreateCommentAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d/assets?token=%s",
		repoOwner.Name, repo.Name, comment.ID, token)

	filename := "image.png"
	buff := generateImg()
	body := &bytes.Buffer{}

	// Setup multi-part
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("attachment", filename)
	assert.NoError(t, err)
	_, err = io.Copy(part, &buff)
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	req := NewRequestWithBody(t, "POST", urlStr, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	resp := session.MakeRequest(t, req, http.StatusCreated)

	apiAttachment := new(api.Attachment)
	DecodeJSON(t, resp, &apiAttachment)

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID, CommentID: comment.ID})
}

func TestAPICreateCommentAttachmentAutoDate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d/assets?token=%s",
		repoOwner.Name, repo.Name, comment.ID, token)

	filename := "image.png"
	buff := generateImg()
	body := &bytes.Buffer{}

	t.Run("WithAutoDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Setup multi-part
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("attachment", filename)
		assert.NoError(t, err)
		_, err = io.Copy(part, &buff)
		assert.NoError(t, err)
		err = writer.Close()
		assert.NoError(t, err)

		req := NewRequestWithBody(t, "POST", urlStr, body)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		resp := session.MakeRequest(t, req, http.StatusCreated)
		apiAttachment := new(api.Attachment)
		DecodeJSON(t, resp, &apiAttachment)

		unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID})
		// the execution of the API call supposedly lasted less than one minute
		updatedSince := time.Since(apiAttachment.Created)
		assert.LessOrEqual(t, updatedSince, time.Minute)

		commentAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID})
		updatedSince = time.Since(commentAfter.UpdatedUnix.AsTime())
		assert.LessOrEqual(t, updatedSince, time.Minute)
	})

	t.Run("WithUpdateDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		updatedAt := time.Now().Add(-time.Hour).Truncate(time.Second)
		urlStr += fmt.Sprintf("&updated_at=%s", updatedAt.UTC().Format(time.RFC3339))

		// Setup multi-part
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("attachment", filename)
		assert.NoError(t, err)
		_, err = io.Copy(part, &buff)
		assert.NoError(t, err)
		err = writer.Close()
		assert.NoError(t, err)

		req := NewRequestWithBody(t, "POST", urlStr, body)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		resp := session.MakeRequest(t, req, http.StatusCreated)
		apiAttachment := new(api.Attachment)
		DecodeJSON(t, resp, &apiAttachment)

		// dates will be converted into the same tz, in order to compare them
		utcTZ, _ := time.LoadLocation("UTC")
		unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID})
		assert.Equal(t, updatedAt.In(utcTZ), apiAttachment.Created.In(utcTZ))

		commentAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID})
		assert.Equal(t, updatedAt.In(utcTZ), commentAfter.UpdatedUnix.AsTime().In(utcTZ))
	})
}

func TestAPIEditCommentAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	const newAttachmentName = "newAttachmentName"

	attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 6})
	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: attachment.CommentID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d/assets/%d?token=%s",
		repoOwner.Name, repo.Name, comment.ID, attachment.ID, token)
	req := NewRequestWithValues(t, "PATCH", urlStr, map[string]string{
		"name": newAttachmentName,
	})
	resp := session.MakeRequest(t, req, http.StatusCreated)
	apiAttachment := new(api.Attachment)
	DecodeJSON(t, resp, &apiAttachment)

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID, CommentID: comment.ID, Name: apiAttachment.Name})
}

func TestAPIDeleteCommentAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 6})
	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: attachment.CommentID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d/assets/%d?token=%s",
		repoOwner.Name, repo.Name, comment.ID, attachment.ID, token)

	req := NewRequestf(t, "DELETE", urlStr)
	session.MakeRequest(t, req, http.StatusNoContent)

	unittest.AssertNotExistsBean(t, &repo_model.Attachment{ID: attachment.ID, CommentID: comment.ID})
}
