// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: AGPL-3.0-or-later

package v1_22 //nolint

import (
	"code.gitea.io/gitea/models/federation"
	"xorm.io/xorm"
)

func AddFederatedHost(x *xorm.Engine) error {
	return x.Sync(new(federation.FederatedHost))
}
