// SPDX-License-Identifier: MIT

package v1_20 //nolint

import (
	"xorm.io/xorm"
)

func CreateSemVerTable(x *xorm.Engine) error {
	type ForgejoSemVer struct {
		Version string
	}

	return x.Sync(new(ForgejoSemVer))
}
