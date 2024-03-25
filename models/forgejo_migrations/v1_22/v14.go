// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: AGPL-3.0-or-later

package v1_22 //nolint

import (
	"code.gitea.io/gitea/models/federation"
	"code.gitea.io/gitea/modules/log"
	"xorm.io/xorm"
)

func AddFederatedUser(x *xorm.Engine) error {
	log.Info("Running Add user migration")
	return x.Sync(new(federation.FederatedUser))
}
