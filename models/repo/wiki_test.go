// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"

	"github.com/stretchr/testify/assert"
)

func TestRepository_WikiCloneLink(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	cloneLink := repo.WikiCloneLink()
	assert.Equal(t, "ssh://sshuser@try.gitea.io:3000/user2/repo1.wiki.git", cloneLink.SSH)
	assert.Equal(t, "https://try.gitea.io/user2/repo1.wiki.git", cloneLink.HTTPS)
}

func TestWikiPath(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	expected := filepath.Join(setting.RepoRootPath, "user2/repo1.wiki.git")
	assert.Equal(t, expected, repo_model.WikiPath("user2", "repo1"))
}

func TestRepository_WikiPath(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	expected := filepath.Join(setting.RepoRootPath, "user2/repo1.wiki.git")
	assert.Equal(t, expected, repo.WikiPath())
}

func TestRepository_HasWiki(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.True(t, repo1.HasWiki())
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	assert.False(t, repo2.HasWiki())
}

func checkRepoWatchersEvent(t *testing.T, event repo_model.WatchEventType, isWatching bool) {
	watchers, err := repo_model.GetRepoWatchersEventIDs(db.DefaultContext, 1, event)
	assert.NoError(t, err)

	if isWatching {
		assert.Len(t, watchers, 1)
		assert.Equal(t, int64(12), watchers[0])
	} else {
		assert.Len(t, watchers, 0)
	}
}

func TestWatchRepoCustom(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1}, 0)

	// Make sure nobody is watching this repo
	watchers, err := repo_model.GetRepoWatchersIDs(db.DefaultContext, 1)
	assert.NoError(t, err)
	for _, watcher := range watchers {
		assert.NoError(t, repo_model.WatchRepoMode(db.DefaultContext, watcher, 1, repo_model.WatchModeNone))
	}

	assert.NoError(t, repo_model.WatchRepoCustom(db.DefaultContext, 12, 1, api.RepoCustomWatchOptions{Issues: true}))
	checkRepoWatchersEvent(t, repo_model.WatchEventTypeIssue, true)
	checkRepoWatchersEvent(t, repo_model.WatchEventTypePullRequest, false)
	checkRepoWatchersEvent(t, repo_model.WatchEventTypeRelease, false)

	assert.NoError(t, repo_model.WatchRepoCustom(db.DefaultContext, 12, 1, api.RepoCustomWatchOptions{PullRequests: true}))
	checkRepoWatchersEvent(t, repo_model.WatchEventTypeIssue, false)
	checkRepoWatchersEvent(t, repo_model.WatchEventTypePullRequest, true)
	checkRepoWatchersEvent(t, repo_model.WatchEventTypeRelease, false)

	assert.NoError(t, repo_model.WatchRepoCustom(db.DefaultContext, 12, 1, api.RepoCustomWatchOptions{Releases: true}))
	checkRepoWatchersEvent(t, repo_model.WatchEventTypeIssue, false)
	checkRepoWatchersEvent(t, repo_model.WatchEventTypePullRequest, false)
	checkRepoWatchersEvent(t, repo_model.WatchEventTypeRelease, true)
}
