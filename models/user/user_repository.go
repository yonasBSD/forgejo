// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederatedUser))
}

func CreateFederationUser(ctx context.Context, user FederatedUser) error {
	if res, err := validation.IsValid(user); !res {
		return fmt.Errorf("FederatedUser is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).Insert(user)
	return err
}
