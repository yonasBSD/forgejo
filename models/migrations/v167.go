// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"xorm.io/builder"
	"xorm.io/xorm"
)

func recreateUserTableToFixDefaultValues(x *xorm.Engine) error {
	type User struct {
		ID                  int64 `xorm:"pk autoincr"`
		KeepActivityPrivate bool  `xorm:"NOT NULL DEFAULT false"`
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	var activityPrivateUsers []int64
	if err := sess.Select("id").Table("user").Where(builder.Eq{"keep_activity_private": true}).Find(&activityPrivateUsers); err != nil {
		return err
	}

	if err := dropTableColumns(sess, "user", "keep_activity_private"); err != nil {
		return err
	}

	if err := sess.Sync2(new(User)); err != nil {
		return err
	}

	for _, uid := range activityPrivateUsers {
		if _, err := sess.ID(uid).Cols("keep_activity_private").Update(&User{KeepActivityPrivate: true}); err != nil {
			return err
		}
	}

	return sess.Commit()
}
