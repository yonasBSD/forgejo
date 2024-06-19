// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"

	"code.gitea.io/gitea/models/user"
)

type UserRepositoryMock struct{}

func (UserRepositoryMock) FindFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	return nil, nil, nil
}

func (UserRepositoryMock) CreateFederatedUser(ctx context.Context, user *user.User, federatedUser *user.FederatedUser) error {
	return nil
}
