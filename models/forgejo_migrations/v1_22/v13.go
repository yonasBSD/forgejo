// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: GPL-3.0-or-later

package v1_22 //nolint:revive

import (
	"xorm.io/xorm"
)

func AddFederatedHostAndUser(x *xorm.Engine) error {
	type FederatedHost struct {
		ID int64 `xorm:"pk autoincr"`
		//nolint:unused
		isBlocked bool
		HostFqdn  string `xorm:"UNIQUE(s) INDEX"`
	}

	type FederatedUser struct {
		ID               int64  `xorm:"pk autoincr"`
		UserID           int64  `xorm:"INDEX"`
		ExternalID       string `xorm:"UNIQUE(s) INDEX"`
		FederationHostID int64  `xorm:"INDEX"`
	}

	if err := x.Sync(new(FederatedHost)); err != nil {
		return err
	}

	return x.Sync(new(FederatedUser))
}
