package v1_21

import (
	"code.gitea.io/gitea/models"
	"xorm.io/xorm"
)

func AddSourceTable(x *xorm.Engine) error {
	type Source struct {
		ID             int64  `xorm:"pk autoincr"`
		UserID         int64  `xorm:"UNIQUE NOT NULL"`
		Type           string `xorm:"NOT NULL"`
		RemoteUsername string `xorm:"NOT NULL"`
		Token          string
	}

	return x.Sync(
		new(models.Source),
	)
}
