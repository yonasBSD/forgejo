// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package infrastructure

import (
	"context"

	"code.gitea.io/gitea/models/user"
)

// ToDo: For now this is a wrapper for repository functionality of models/user
type UserRepositoryWrapper struct{}

func (UserRepositoryWrapper) FindFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	return user.FindFederatedUser(ctx, externalID, federationHostID)
}

func (UserRepositoryWrapper) CreateFederatedUser(ctx context.Context, newUser *user.User, federatedUser *user.FederatedUser) error {
	return user.CreateFederatedUser(ctx, newUser, federatedUser)
}
