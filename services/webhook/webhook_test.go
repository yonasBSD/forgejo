// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	webhook_model "code.gitea.io/gitea/models/webhook"
	api "code.gitea.io/gitea/modules/structs"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
)

func TestWebhook_GetSlackHook(t *testing.T) {
	w := &webhook_model.Webhook{
		Meta: `{"channel": "foo", "username": "username", "color": "blue"}`,
	}
	slackHook := GetSlackHook(w)
	assert.Equal(t, *slackHook, SlackMeta{
		Channel:  "foo",
		Username: "username",
		Color:    "blue",
	})
}

func TestPrepareWebhooks(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	hookTasks := []*webhook_model.HookTask{
		{HookID: 1, EventType: webhook_module.HookEventPush},
	}
	for _, hookTask := range hookTasks {
		unittest.AssertNotExistsBean(t, hookTask)
	}
	assert.NoError(t, PrepareWebhooks(db.DefaultContext, EventSource{Repository: repo}, webhook_module.HookEventPush, &api.PushPayload{Commits: []*api.PayloadCommit{{}}}))
	for _, hookTask := range hookTasks {
		unittest.AssertExistsAndLoadBean(t, hookTask)
	}
}

func TestPrepareWebhooksBranchFilterMatch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	hookTasks := []*webhook_model.HookTask{
		{HookID: 4, EventType: webhook_module.HookEventPush},
	}
	for _, hookTask := range hookTasks {
		unittest.AssertNotExistsBean(t, hookTask)
	}
	// this test also ensures that * doesn't handle / in any special way (like shell would)
	assert.NoError(t, PrepareWebhooks(db.DefaultContext, EventSource{Repository: repo}, webhook_module.HookEventPush, &api.PushPayload{Ref: "refs/heads/feature/7791", Commits: []*api.PayloadCommit{{}}}))
	for _, hookTask := range hookTasks {
		unittest.AssertExistsAndLoadBean(t, hookTask)
	}
}

func TestPrepareWebhooksBranchFilterNoMatch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	hookTasks := []*webhook_model.HookTask{
		{HookID: 4, EventType: webhook_module.HookEventPush},
	}
	for _, hookTask := range hookTasks {
		unittest.AssertNotExistsBean(t, hookTask)
	}
	assert.NoError(t, PrepareWebhooks(db.DefaultContext, EventSource{Repository: repo}, webhook_module.HookEventPush, &api.PushPayload{Ref: "refs/heads/fix_weird_bug"}))

	for _, hookTask := range hookTasks {
		unittest.AssertNotExistsBean(t, hookTask)
	}
}

func TestPrepareWebhookViaRequest(t *testing.T) {
	w := &webhook_model.Webhook{
		ID:   42,
		Type: webhook_module.MATRIX,
		URL:  "https://matrix.example.com/_matrix/client/r0/rooms/ROOM_ID/send/m.room.message",
		Meta: `{"message_type":0}`, // text
		HookEvent: &webhook_module.HookEvent{
			SendEverything: true,
		},
	}
	p := createTestPayload()
	err := PrepareWebhook(db.DefaultContext, w, webhook_module.HookEventCreate, p)

	assert.NoError(t, err)

	task := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{HookID: 42})

	assert.Equal(t, int64(42), task.HookID)
	assert.Equal(t, webhook_module.HookEventCreate, task.EventType)
	assert.Equal(t, "PUT", task.RequestMethod)
	assert.Equal(t, "https://matrix.example.com/_matrix/client/r0/rooms/ROOM_ID/send/m.room.message/f8c147d78b4a12bbfad7d59d857495ca4950b7da", task.RequestURL)
	assert.Equal(t, "Content-Type: application/json\r\n", task.RequestHeader)
	assert.Equal(t, "{\n  \"body\": \"[[test/repo](http://localhost:3000/test/repo):[test](http://localhost:3000/test/repo/src/branch/test)] branch created by user1\",\n  \"msgtype\": \"\",\n  \"format\": \"org.matrix.custom.html\",\n  \"formatted_body\": \"[\\u003ca href=\\\"http://localhost:3000/test/repo\\\"\\u003etest/repo\\u003c/a\\u003e:\\u003ca href=\\\"http://localhost:3000/test/repo/src/branch/test\\\"\\u003etest\\u003c/a\\u003e] branch created by user1\"\n}", task.PayloadContent)
}
