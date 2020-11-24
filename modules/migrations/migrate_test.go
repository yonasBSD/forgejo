// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"testing"

	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
)

func TestMigrateWhiteBlocklist(t *testing.T) {
	setting.Migrations.AllowlistedDomains = []string{"github.com"}
	assert.NoError(t, Init())

	err := isMigrateURLAllowed("https://gitlab.com/gitlab/gitlab.git")
	assert.Error(t, err)

	err = isMigrateURLAllowed("https://github.com/go-gitea/gitea.git")
	assert.NoError(t, err)

	setting.Migrations.AllowlistedDomains = []string{}
	setting.Migrations.BlocklistedDomains = []string{"github.com"}
	assert.NoError(t, Init())

	err = isMigrateURLAllowed("https://gitlab.com/gitlab/gitlab.git")
	assert.NoError(t, err)

	err = isMigrateURLAllowed("https://github.com/go-gitea/gitea.git")
	assert.Error(t, err)
}
