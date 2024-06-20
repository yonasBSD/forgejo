// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"code.gitea.io/gitea/ddd-federation/domain"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/timeutil"
)

type HTTPClientAPIMock struct{}

const MockFederationHost1ID int64 = 1

var MockFederationHost1 domain.FederationHost = domain.FederationHost{
	ID:       MockFederationHost1ID,
	HostFqdn: "https://www.example.com/",
	NodeInfo: domain.NodeInfo{
		SoftwareName: domain.ForgejoSourceType,
	},
	LatestActivity: time.Date(
		2020, 1, 1, 12, 12, 12, 0,
		time.Now().UTC().Location(),
	),
	Created: timeutil.TimeStampNow(),
	Updated: timeutil.TimeStampNow(),
}

var MockActorID forgefed.ActorID = forgefed.ActorID{
	ID:               "30",
	Schema:           "https",
	Host:             "www.example.com",
	Path:             "api/v1/activitypub/user-id",
	Port:             "",
	UnvalidatedInput: "https://www.example.com/api/v1/activitypub/user-id/30",
}

var MockPersonID forgefed.PersonID = forgefed.PersonID{
	ActorID: MockActorID,
}

var MockUser = user.User{
	LowerName:                    strings.ToLower("[]-www.example.com"),
	Name:                         "[]-www.example.com",
	FullName:                     "[]-www.example.com",
	Email:                        "email@example.com",
	EmailNotificationsPreference: "disabled",
	Passwd:                       "password",
	MustChangePassword:           false,
	LoginName:                    "",
	Type:                         user.UserTypeRemoteUser,
	IsAdmin:                      false,
	NormalizedFederatedURI:       "",
}

var MockFederatedUser, _ = user.NewFederatedUser(MockUser.ID, MockPersonID.ID, MockFederationHost1.ID)

func (HTTPClientAPIMock) GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (domain.FederationHost, error) {
	if actorID == MockActorID {
		return MockFederationHost1, nil
	}
	return domain.FederationHost{}, errors.New("actorID not found")
}

func (HTTPClientAPIMock) GetForgePersonFromAP(ctx context.Context, personID forgefed.PersonID) (forgefed.ForgePerson, error) {
	return forgefed.ForgePerson{}, nil
}

func (HTTPClientAPIMock) PostLikeActivities(ctx context.Context, doer user.User, activityList []forgefed.ForgeLike) error {
	return nil
}
