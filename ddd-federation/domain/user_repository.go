// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package domain

import (
	"context"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/forgefed"
)

type UserRepository interface {
	FindFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*user.User, *user.FederatedUser, error)
	CreateFederatedUser(ctx context.Context, user *user.User, federatedUser *user.FederatedUser) error
	GetRepresentativeUser(ctx context.Context, person forgefed.ForgePerson, personID forgefed.PersonID) (user.User, error)
	GetFederatedUser(personID forgefed.PersonID, federationHostID int64) user.FederatedUser
}
