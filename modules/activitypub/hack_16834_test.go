// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package activitypub

import (
	"testing"

	_ "code.gitea.io/gitea/models" // https://discourse.gitea.io/t/testfixtures-could-not-clean-table-access-no-such-table-access/4137/4
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"github.com/stretchr/testify/assert"
)

func TestHack16834(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}).(*user_model.User)
	pub, priv, err := GetKeyPair(user1)
	assert.NoError(t, err)
	pub1, err := GetPublicKey(user1)
	assert.NoError(t, err)
	assert.Equal(t, pub, pub1)
	priv1, err := GetPrivateKey(user1)
	assert.NoError(t, err)
	assert.Equal(t, priv, priv1)
}
