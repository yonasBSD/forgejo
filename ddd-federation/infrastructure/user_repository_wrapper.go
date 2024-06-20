// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package infrastructure

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/auth/password"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/setting"
	"github.com/google/uuid"
)

// ToDo: For now this is a wrapper for repository functionality of models/user
type UserRepositoryWrapper struct{}

func (UserRepositoryWrapper) FindFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	return user.FindFederatedUser(ctx, externalID, federationHostID)
}

func (UserRepositoryWrapper) CreateFederatedUser(ctx context.Context, newUser *user.User, federatedUser *user.FederatedUser) error {
	return user.CreateFederatedUser(ctx, newUser, federatedUser)
}

// ToDo: These are indirections to make CreateUserFromAP testable
func (UserRepositoryWrapper) GetRepresentativeUser(ctx context.Context, person forgefed.ForgePerson, personID forgefed.PersonID) (user.User, error) {

	localFqdn, err := url.ParseRequestURI(setting.AppURL)
	if err != nil {
		return user.User{}, err
	}

	email := fmt.Sprintf("f%v@%v", uuid.New().String(), localFqdn.Hostname())
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
	fullName := person.Name.String()
	if len(person.Name) == 0 {
		fullName = name
	}

	password, err := password.Generate(32)
	if err != nil {
		return user.User{}, err
	}

	return user.User{
		LowerName:                    strings.ToLower(name),
		Name:                         name,
		FullName:                     fullName,
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    personID.AsLoginName(),
		Type:                         user.UserTypeRemoteUser,
		IsAdmin:                      false,
		NormalizedFederatedURI:       personID.AsURI(),
	}, nil
}

func (UserRepositoryWrapper) GetFederatedUser(personID forgefed.PersonID, federationHostID int64) user.FederatedUser {
	return user.FederatedUser{
		ExternalID:       personID.ID,
		FederationHostID: federationHostID,
	}
}
