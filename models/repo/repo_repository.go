// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"code.gitea.io/gitea/models/db"
)

func init() {
	db.RegisterModel(new(FederatedRepo))
}
