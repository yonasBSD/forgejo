// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"fmt"
	"strings"
	"testing"

	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/json"
	api "forgejo.org/modules/structs"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMSTeamsPayload(t *testing.T) {
	mc := msteamsConvertor{}

	// helper to find text within the adaptive card body
	findTextInBody := func(pl MSTeamsPayload, substr string) bool {
		for _, c := range pl.Body {
			for _, it := range c.Items {
				switch v := it.(type) {
				case MSTeamsTextBlock:
					if strings.Contains(v.Text, substr) {
						return true
					}
				case MSTeamsColumnSet:
					for _, col := range v.Columns {
						for _, it2 := range col.Items {
							if tb, ok := it2.(MSTeamsTextBlock); ok && strings.Contains(tb.Text, substr) {
								return true
							}
						}
					}
				case MSTeamsContainer:
					for _, it2 := range v.Items {
						if tb, ok := it2.(MSTeamsTextBlock); ok && strings.Contains(tb.Text, substr) {
							return true
						}
					}
				}
			}
		}
		return false
	}

	t.Run("Create", func(t *testing.T) {
		p := createTestPayload()

		pl, err := mc.Create(p)
		require.NoError(t, err)

		// header
		require.GreaterOrEqual(t, len(pl.Body), 2)
		header := pl.Body[0]
		hb, ok := header.Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("💬 Update | [%s](%s)", p.Repo.FullName, p.Repo.HTMLURL), hb.Text)

		// sender contains display name and action text
		sender := pl.Body[1]
		cs, ok := sender.Items[0].(MSTeamsColumnSet)
		require.True(t, ok)
		colText, ok := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Contains(t, colText.Text, "**user1**")
		assert.Contains(t, colText.Text, "created a new branch 'test'")

		// action button should point to branch
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/src/test", pl.Actions[0].URL)
	})

	t.Run("Delete", func(t *testing.T) {
		p := deleteTestPayload()

		pl, err := mc.Delete(p)
		require.NoError(t, err)

		hb, ok := pl.Body[0].Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("💬 Update | [%s](%s)", p.Repo.FullName, p.Repo.HTMLURL), hb.Text)

		sender := pl.Body[1]
		cs, ok := sender.Items[0].(MSTeamsColumnSet)
		require.True(t, ok)
		colText, ok := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Contains(t, colText.Text, "deleted branch 'test'")

		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/src/test", pl.Actions[0].URL)
	})

	t.Run("Fork", func(t *testing.T) {
		p := forkTestPayload()

		pl, err := mc.Fork(p)
		require.NoError(t, err)

		hb, ok := pl.Body[0].Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("💬 Update | [%s](%s)", p.Repo.FullName, p.Repo.HTMLURL), hb.Text)

		cs, ok := pl.Body[1].Items[0].(MSTeamsColumnSet)
		require.True(t, ok)
		colText, ok := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Contains(t, colText.Text, "forked")

		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo", pl.Actions[0].URL)
	})

	t.Run("Push", func(t *testing.T) {
		p := pushTestPayload()

		pl, err := mc.Push(p)
		require.NoError(t, err)

		// header + sender
		hb, ok := pl.Body[0].Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("💬 Update | [%s](%s)", p.Repo.FullName, p.Repo.HTMLURL), hb.Text)
		cs, ok := pl.Body[1].Items[0].(MSTeamsColumnSet)
		require.True(t, ok)
		colText, ok := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		require.True(t, ok)
		assert.Contains(t, colText.Text, "pushed 2 new commits to test")

		// commit details present in body
		require.True(t, findTextInBody(pl, "2020558"))
		require.True(t, findTextInBody(pl, "commit message"))

		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/src/test", pl.Actions[0].URL)
	})

	t.Run("Issue", func(t *testing.T) {
		p := issueTestPayload()

		p.Action = api.HookIssueOpened
		pl, err := mc.Issue(p)
		require.NoError(t, err)

		// header and sender
		hb, _ := pl.Body[0].Items[0].(MSTeamsTextBlock)
		assert.Equal(t, fmt.Sprintf("💬 Update | [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL), hb.Text)
		cs, _ := pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "opened issue #2")

		// issue title and body present
		assert.True(t, findTextInBody(pl, "Issue #2: crash"))
		assert.True(t, findTextInBody(pl, "issue body"))
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/issues/2", pl.Actions[0].URL)

		p.Action = api.HookIssueClosed
		pl, err = mc.Issue(p)
		require.NoError(t, err)
		cs, _ = pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ = cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "closed issue #2")
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/issues/2", pl.Actions[0].URL)
	})

	t.Run("IssueComment", func(t *testing.T) {
		p := issueCommentTestPayload()

		pl, err := mc.IssueComment(p)
		require.NoError(t, err)

		cs, _ := pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "commented on issue #2")
		assert.True(t, findTextInBody(pl, "more info needed"))
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/issues/2#issuecomment-4", pl.Actions[0].URL)
	})

	t.Run("PullRequest", func(t *testing.T) {
		p := pullRequestTestPayload()

		pl, err := mc.PullRequest(p)
		require.NoError(t, err)

		cs, _ := pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "opened new pull request #12")
		assert.True(t, findTextInBody(pl, "Pull request #12: Fix bug"))
		assert.True(t, findTextInBody(pl, "fixes bug #2"))
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/pulls/12", pl.Actions[0].URL)
	})

	t.Run("PullRequestComment", func(t *testing.T) {
		p := pullRequestCommentTestPayload()

		pl, err := mc.IssueComment(p)
		require.NoError(t, err)

		cs, _ := pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "commented on pull request #12")
		assert.True(t, findTextInBody(pl, "changes requested"))
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/pulls/12#issuecomment-4", pl.Actions[0].URL)
	})

	t.Run("Review", func(t *testing.T) {
		p := pullRequestTestPayload()
		p.Action = api.HookIssueReviewed

		pl, err := mc.Review(p, webhook_module.HookEventPullRequestReviewApproved)
		require.NoError(t, err)

		// review content should be present
		assert.True(t, findTextInBody(pl, "good job"))
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/pulls/12", pl.Actions[0].URL)
	})

	t.Run("Repository", func(t *testing.T) {
		p := repositoryTestPayload()

		pl, err := mc.Repository(p)
		require.NoError(t, err)

		hb, _ := pl.Body[0].Items[0].(MSTeamsTextBlock)
		assert.Equal(t, fmt.Sprintf("💬 Update | [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL), hb.Text)
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo", pl.Actions[0].URL)
	})

	t.Run("Package", func(t *testing.T) {
		p := packageTestPayload()

		pl, err := mc.Package(p)
		require.NoError(t, err)

		cs, _ := pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "created package")
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/user1/-/packages/container/GiteaContainer/latest", pl.Actions[0].URL)
	})

	t.Run("Wiki", func(t *testing.T) {
		p := wikiTestPayload()

		p.Action = api.HookWikiCreated
		pl, err := mc.Wiki(p)
		require.NoError(t, err)

		cs, _ := pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "created new wiki page 'index'")
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/wiki/index", pl.Actions[0].URL)

		p.Action = api.HookWikiEdited
		pl, err = mc.Wiki(p)
		require.NoError(t, err)
		cs, _ = pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ = cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "edited wiki page 'index'")
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/wiki/index", pl.Actions[0].URL)

		p.Action = api.HookWikiDeleted
		pl, err = mc.Wiki(p)
		require.NoError(t, err)
		cs, _ = pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ = cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "deleted wiki page 'index'")
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/wiki/index", pl.Actions[0].URL)
	})

	t.Run("Release", func(t *testing.T) {
		p := pullReleaseTestPayload()

		pl, err := mc.Release(p)
		require.NoError(t, err)

		cs, _ := pl.Body[1].Items[0].(MSTeamsColumnSet)
		colText, _ := cs.Columns[1].Items[0].(MSTeamsTextBlock)
		assert.Contains(t, colText.Text, "published release v1.0")
		require.Len(t, pl.Actions, 1)
		assert.Equal(t, "http://localhost:3000/test/repo/releases/tag/v1.0", pl.Actions[0].URL)
	})
}

func TestMSTeamsJSONPayload(t *testing.T) {
	p := pushTestPayload()
	data, err := p.JSONPayload()
	require.NoError(t, err)

	hook := &webhook_model.Webhook{
		RepoID:     3,
		IsActive:   true,
		Type:       webhook_module.MSTEAMS,
		URL:        "https://msteams.example.com/",
		Meta:       ``,
		HTTPMethod: "POST",
	}
	task := &webhook_model.HookTask{
		HookID:         hook.ID,
		EventType:      webhook_module.HookEventPush,
		PayloadContent: string(data),
		PayloadVersion: 2,
	}

	req, reqBody, err := msteamsHandler{}.NewRequest(t.Context(), hook, task)
	require.NotNil(t, req)
	require.NotNil(t, reqBody)
	require.NoError(t, err)

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "https://msteams.example.com/", req.URL.String())
	assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	var body MSTeamsPayload
	err = json.NewDecoder(req.Body).Decode(&body)
	require.NoError(t, err)
	// ensure commit info is present in resulting adaptive card
	assert.True(t, func() bool {
		for _, c := range body.Body {
			for _, it := range c.Items {
				if tb, ok := it.(MSTeamsTextBlock); ok && strings.Contains(tb.Text, "commit message") {
					return true
				}
			}
		}
		return false
	}())
}
