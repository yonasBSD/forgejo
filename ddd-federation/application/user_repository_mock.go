// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/forgefed"
)

type UserRepositoryMock struct{}

func (UserRepositoryMock) FindFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	return nil, nil, nil
}

func (UserRepositoryMock) CreateFederatedUser(ctx context.Context, user *user.User, federatedUser *user.FederatedUser) error {
	return nil
}

func (UserRepositoryMock) GetRepresentativeUser(ctx context.Context, person forgefed.ForgePerson, personID forgefed.PersonID) (user.User, error) {
	if personID == MockPersonID {
		return MockUser, nil
	}
	return user.User{}, nil
}

func (UserRepositoryMock) GetFederatedUser(personID forgefed.PersonID, federationHostID int64) user.FederatedUser {
	return user.FederatedUser{
		ExternalID:       personID.ID,
		FederationHostID: federationHostID,
	}
}
