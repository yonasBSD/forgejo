// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"xorm.io/xorm"
)

func addRequireSignedCommits(x *xorm.Engine) error {

	type ProtectedBranch struct {
		RequireSignedCommits      bool     `xorm:"NOT NULL DEFAULT false"`
	}

	return x.Sync2(new(ProtectedBranch))
}
