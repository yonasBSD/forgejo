// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"code.gitea.io/gitea/models/db"
)

func init() {
	db.RegisterModel(new(FederationInfo))
}
