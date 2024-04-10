// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import "xorm.io/xorm"

func AddHideArchiveLinksToRelease(x *xorm.Engine) error {
	type Release struct {
		HideArchiveLinks bool
	}

	return x.Sync(&Release{})
}
