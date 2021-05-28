// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"fmt"

	"xorm.io/xorm"
)

func addCustomRepoButtonsConfigRepositoryColumn(x *xorm.Engine) error {
	type Repository struct {
		CustomRepoButtonsConfig string `xorm:"TEXT"`
	}

	if err := x.Sync2(new(Repository)); err != nil {
		return fmt.Errorf("sync2: %v", err)
	}
	return nil
}
