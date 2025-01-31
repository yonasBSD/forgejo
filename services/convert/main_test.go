// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"

	"code.gitea.io/gitea/models/unittest"

	_ "code.gitea.io/gitea/models/actions"
	_ "code.gitea.io/gitea/models/forgefed"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
