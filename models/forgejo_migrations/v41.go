// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func AddAlternates(x *xorm.Engine) error {
	type Alternate struct {
		ID          int64              `xorm:"pk autoincr"`
		Name        string             `xorm:"NOT NULL UNIQUE"`
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	}

	type Repository struct {
		AlternateID int64 `xorm:"INDEX NOT NULL DEFAULT 0"`
	}

	if err := x.Sync(&Alternate{}); err != nil {
		return err
	}

	return x.Sync(&Repository{})
}
