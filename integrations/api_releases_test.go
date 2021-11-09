// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"

	"github.com/stretchr/testify/assert"
)

func TestAPIListReleases(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := db.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	user2 := db.AssertExistsAndLoadBean(t, &models.User{ID: 2}).(*models.User)
	session := loginUser(t, user2.LowerName)
	token := getTokenForLoggedInUser(t, session)

	link, _ := url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/releases", user2.Name, repo.Name))
	link.RawQuery = url.Values{"token": {token}}.Encode()
	resp := session.MakeRequest(t, NewRequest(t, "GET", link.String()), http.StatusOK)
	var apiReleases []*api.Release
	DecodeJSON(t, resp, &apiReleases)
	if assert.Len(t, apiReleases, 3) {
		for _, release := range apiReleases {
			switch release.ID {
			case 1:
				assert.False(t, release.IsDraft)
				assert.False(t, release.IsPrerelease)
			case 4:
				assert.True(t, release.IsDraft)
				assert.False(t, release.IsPrerelease)
			case 5:
				assert.False(t, release.IsDraft)
				assert.True(t, release.IsPrerelease)
			default:
				assert.NoError(t, fmt.Errorf("unexpected release: %v", release))
			}
		}
	}

	// test filter
	testFilterByLen := func(auth bool, query url.Values, expectedLength int, msgAndArgs ...string) {
		link.RawQuery = query.Encode()
		if auth {
			query.Set("token", token)
			resp = session.MakeRequest(t, NewRequest(t, "GET", link.String()), http.StatusOK)
		} else {
			resp = MakeRequest(t, NewRequest(t, "GET", link.String()), http.StatusOK)
		}
		DecodeJSON(t, resp, &apiReleases)
		assert.Len(t, apiReleases, expectedLength, msgAndArgs)
	}

	testFilterByLen(false, url.Values{"draft": {"true"}}, 0, "anon should not see drafts")
	testFilterByLen(true, url.Values{"draft": {"true"}}, 1, "repo owner should see drafts")
	testFilterByLen(true, url.Values{"draft": {"false"}}, 2, "exclude drafts")
	testFilterByLen(true, url.Values{"draft": {"false"}, "pre-release": {"false"}}, 1, "exclude drafts and pre-releases")
	testFilterByLen(true, url.Values{"pre-release": {"true"}}, 1, "only get pre-release")
	testFilterByLen(true, url.Values{"draft": {"true"}, "pre-release": {"true"}}, 0, "there is no pre-release draft")
}

func createNewReleaseUsingAPI(t *testing.T, session *TestSession, token string, owner *models.User, repo *models.Repository, name, target, title, desc string) *api.Release {
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/releases?token=%s",
		owner.Name, repo.Name, token)
	req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateReleaseOption{
		TagName:      name,
		Title:        title,
		Note:         desc,
		IsDraft:      false,
		IsPrerelease: false,
		Target:       target,
	})
	resp := session.MakeRequest(t, req, http.StatusCreated)

	var newRelease api.Release
	DecodeJSON(t, resp, &newRelease)
	db.AssertExistsAndLoadBean(t, &models.Release{
		ID:      newRelease.ID,
		TagName: newRelease.TagName,
		Title:   newRelease.Title,
		Note:    newRelease.Note,
	})

	return &newRelease
}

func TestAPICreateAndUpdateRelease(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := db.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := db.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	gitRepo, err := git.OpenRepository(repo.RepoPath())
	assert.NoError(t, err)
	defer gitRepo.Close()

	err = gitRepo.CreateTag("v0.0.1", "master")
	assert.NoError(t, err)

	target, err := gitRepo.GetTagCommitID("v0.0.1")
	assert.NoError(t, err)

	newRelease := createNewReleaseUsingAPI(t, session, token, owner, repo, "v0.0.1", target, "v0.0.1", "test")

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/releases/%d?token=%s",
		owner.Name, repo.Name, newRelease.ID, token)
	req := NewRequest(t, "GET", urlStr)
	resp := session.MakeRequest(t, req, http.StatusOK)

	var release api.Release
	DecodeJSON(t, resp, &release)

	assert.Equal(t, newRelease.TagName, release.TagName)
	assert.Equal(t, newRelease.Title, release.Title)
	assert.Equal(t, newRelease.Note, release.Note)

	req = NewRequestWithJSON(t, "PATCH", urlStr, &api.EditReleaseOption{
		TagName:      release.TagName,
		Title:        release.Title,
		Note:         "updated",
		IsDraft:      &release.IsDraft,
		IsPrerelease: &release.IsPrerelease,
		Target:       release.Target,
	})
	resp = session.MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &newRelease)
	db.AssertExistsAndLoadBean(t, &models.Release{
		ID:      newRelease.ID,
		TagName: newRelease.TagName,
		Title:   newRelease.Title,
		Note:    newRelease.Note,
	})
}

func TestAPICreateReleaseToDefaultBranch(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := db.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := db.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	createNewReleaseUsingAPI(t, session, token, owner, repo, "v0.0.1", "", "v0.0.1", "test")
}

func TestAPICreateReleaseToDefaultBranchOnExistingTag(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := db.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := db.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	gitRepo, err := git.OpenRepository(repo.RepoPath())
	assert.NoError(t, err)
	defer gitRepo.Close()

	err = gitRepo.CreateTag("v0.0.1", "master")
	assert.NoError(t, err)

	createNewReleaseUsingAPI(t, session, token, owner, repo, "v0.0.1", "", "v0.0.1", "test")
}

func TestAPIGetReleaseByTag(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := db.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := db.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)

	tag := "v1.1"

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/releases/tags/%s",
		owner.Name, repo.Name, tag)

	req := NewRequestf(t, "GET", urlStr)
	resp := session.MakeRequest(t, req, http.StatusOK)

	var release *api.Release
	DecodeJSON(t, resp, &release)

	assert.Equal(t, "testing-release", release.Title)

	nonexistingtag := "nonexistingtag"

	urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/releases/tags/%s",
		owner.Name, repo.Name, nonexistingtag)

	req = NewRequestf(t, "GET", urlStr)
	resp = session.MakeRequest(t, req, http.StatusNotFound)

	var err *api.APIError
	DecodeJSON(t, resp, &err)
	assert.EqualValues(t, "Not Found", err.Message)
}

func TestAPIDeleteReleaseByTagName(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := db.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := db.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	createNewReleaseUsingAPI(t, session, token, owner, repo, "release-tag", "", "Release Tag", "test")

	// delete release
	req := NewRequestf(t, http.MethodDelete, fmt.Sprintf("/api/v1/repos/%s/%s/releases/tags/release-tag?token=%s", owner.Name, repo.Name, token))
	_ = session.MakeRequest(t, req, http.StatusNoContent)

	// make sure release is deleted
	req = NewRequestf(t, http.MethodDelete, fmt.Sprintf("/api/v1/repos/%s/%s/releases/tags/release-tag?token=%s", owner.Name, repo.Name, token))
	_ = session.MakeRequest(t, req, http.StatusNotFound)

	// delete release tag too
	req = NewRequestf(t, http.MethodDelete, fmt.Sprintf("/api/v1/repos/%s/%s/tags/release-tag?token=%s", owner.Name, repo.Name, token))
	_ = session.MakeRequest(t, req, http.StatusNoContent)
}

func TestAPICreateReleaseAttachment(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := models.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := models.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	releaseID := 1

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/releases/%d/assets?token=%s",
		owner.Name, repo.Name, releaseID, token)

	body := &bytes.Buffer{}
	buff := generateImg()
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("attachment", "image.png")
	assert.NoError(t, err)
	_, err = io.Copy(part, &buff)
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	req := NewRequestWithBody(t, http.MethodPost, urlStr, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	resp := session.MakeRequest(t, req, http.StatusCreated)

	var attachment *api.Attachment
	DecodeJSON(t, resp, &attachment)

	assert.Equal(t, "image.png", attachment.Name)
}

func TestAPIEditReleaseAttachmentById(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := models.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := models.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	attachID := 9

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, attachID, token)

	newAttachName := map[string]string{
		"name": "New Name",
	}
	req := NewRequestWithValues(t, http.MethodPatch, urlStr, newAttachName)
	resp := session.MakeRequest(t, req, http.StatusCreated)

	var attachment *api.Attachment
	DecodeJSON(t, resp, &attachment)

	assert.Equal(t, "New Name", attachment.Name)

	nonExistingAttachID := 12

	urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, nonExistingAttachID, token)

	req = NewRequestWithJSON(t, http.MethodPatch, urlStr, &newAttachName)
	resp = session.MakeRequest(t, req, http.StatusInternalServerError)

	var err *api.APIError
	DecodeJSON(t, resp, &err)
	assert.EqualValues(t, "", err.Message)
}

func TestAPIGetReleaseAttachmentById(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := models.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := models.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	attachID := 9

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, attachID, token)

	req := NewRequestf(t, http.MethodGet, urlStr)
	resp := session.MakeRequest(t, req, http.StatusOK)

	var attachment *api.Attachment
	DecodeJSON(t, resp, &attachment)

	assert.Equal(t, "attach1", attachment.Name)

	nonExistingAttachID := 12

	urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, nonExistingAttachID, token)

	req = NewRequestf(t, http.MethodGet, urlStr)
	_ = session.MakeRequest(t, req, http.StatusInternalServerError)

	// Test if attachment you get belongs to this repo.
	attachID = 11

	urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, attachID, token)

	req = NewRequestf(t, http.MethodGet, urlStr)
	_ = session.MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIDeleteReleaseAttachmentById(t *testing.T) {
	defer prepareTestEnv(t)()

	repo := models.AssertExistsAndLoadBean(t, &models.Repository{ID: 1}).(*models.Repository)
	owner := models.AssertExistsAndLoadBean(t, &models.User{ID: repo.OwnerID}).(*models.User)
	session := loginUser(t, owner.LowerName)
	token := getTokenForLoggedInUser(t, session)

	attachID := 9

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, attachID, token)

	req := NewRequestf(t, http.MethodDelete, urlStr)
	_ = session.MakeRequest(t, req, http.StatusNoContent)

	nonExistingAttachID := 12

	urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, nonExistingAttachID, token)

	req = NewRequestf(t, http.MethodDelete, urlStr)
	_ = session.MakeRequest(t, req, http.StatusInternalServerError)

	// Test if attachment you delete belongs to this repo.
	attachID = 11

	urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/releases/assets/%d?token=%s",
		owner.Name, repo.Name, attachID, token)

	req = NewRequestf(t, http.MethodDelete, urlStr)
	_ = session.MakeRequest(t, req, http.StatusNotFound)
}
