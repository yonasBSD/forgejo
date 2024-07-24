// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddSSHRegenerationTwoFactor(x *xorm.Engine) error {
	type TwoFactor struct {
		ID                       int64 `xorm:"pk autoincr"`
		AllowRegenerationOverSSH bool  `xorm:"NOT NULL DEFAULT true"`
	}

	return x.Sync(new(TwoFactor))
}
