// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"strconv"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/migrations/base"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/timeutil"
	webhook_module "code.gitea.io/gitea/modules/webhook"
	"github.com/stretchr/testify/assert"
)

// Webhook represents a web hook object.
type Webhook struct {
	ID              int64 `xorm:"pk autoincr"`
	RepoID          int64 `xorm:"INDEX"` // An ID of 0 indicates either a default or system webhook
	OwnerID         int64 `xorm:"INDEX"`
	IsSystemWebhook bool
	URL             string `xorm:"url TEXT"`
	HTTPMethod      string `xorm:"http_method"`
	ContentType     webhook.HookContentType
	Secret          string                    `xorm:"TEXT"`
	Events          string                    `xorm:"TEXT"`
	IsActive        bool                      `xorm:"INDEX"`
	Type            webhook_module.HookType   `xorm:"VARCHAR(16) 'type'"`
	Meta            string                    `xorm:"TEXT"` // store hook-specific attributes
	LastStatus      webhook_module.HookStatus // Last delivery status

	// HeaderAuthorizationEncrypted should be accessed using HeaderAuthorization() and SetHeaderAuthorization()
	HeaderAuthorizationEncrypted string `xorm:"TEXT"`

	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

func TestUpdateHookTaskTable(t *testing.T) {
	type HookTaskMigrated HookTask

	// HookTask represents a hook task, as of before the migration
	type HookTask struct {
		ID             int64  `xorm:"pk autoincr"`
		HookID         int64  `xorm:"index"`
		UUID           string `xorm:"unique"`
		PayloadContent string `xorm:"LONGTEXT"`
		EventType      webhook_module.HookEventType
		IsDelivered    bool
		Delivered      timeutil.TimeStampNano

		// History info.
		IsSucceed       bool
		RequestContent  string `xorm:"LONGTEXT"`
		ResponseContent string `xorm:"LONGTEXT"`
	}

	// Prepare and load the testing database
	x, deferable := base.PrepareTestEnv(t, 0, new(HookTask), new(Webhook), new(HookTaskMigrated))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}
	err := unittest.InitAndLoadFixtures(x, "fixtures/v6_TestUpdateHookTaskTable")
	if err != nil {
		t.Fatal(err)
	}

	err = UpdateHookTaskTable(x)
	if err != nil {
		t.Fatal(err)
	}

	expected := []HookTaskMigrated{}
	if err := x.Table("hook_task_migrated").Asc("id").Find(&expected); !assert.NoError(t, err) {
		return
	}

	got := []HookTaskMigrated{}
	if err := x.Table("hook_task").Asc("id").Find(&got); !assert.NoError(t, err) {
		return
	}

	for i, expected := range expected {
		expected, got := expected, got[i]
		t.Run(strconv.FormatInt(expected.ID, 10), func(t *testing.T) {
			assert.Equal(t, expected.RequestMethod, got.RequestMethod)
			assert.Equal(t, expected.RequestURL, got.RequestURL)

			expectedHeader := expected.RequestHeader
			// newline in YAML are \n, while the header has \r\n
			expectedHeader = strings.ReplaceAll(expectedHeader, "\n", "\r\n")
			// trailing space is removed from YAML by most editors. Add it back
			expectedHeader = strings.ReplaceAll(expectedHeader, ":\r\n", ": \r\n")
			assert.Equal(t, expectedHeader, got.RequestHeader)

			assert.Equal(t, expected.PayloadContent, got.PayloadContent)
			assert.Equal(t, expected.AddDefaultHeaders, got.AddDefaultHeaders)
		})
	}
}
