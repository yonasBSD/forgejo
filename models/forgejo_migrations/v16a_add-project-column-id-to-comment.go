// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add project column ID fields to comment",
		Upgrade:     addProjectColumnIDToComment,
	})
}

func addProjectColumnIDToComment(x *xorm.Engine) error {
	type Comment struct {
		OldProjectColumnID int64 `xorm:"'old_project_board_id'"`
		ProjectColumnID    int64 `xorm:"'project_board_id'"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(Comment))
	return err
}
