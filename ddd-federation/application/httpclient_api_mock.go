// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"errors"
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

func (HTTPClientAPIMock) GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (domain.FederationHost, error) {
	if actorID == MockActorID {
		return MockFederationHost1, nil
	}
	return domain.FederationHost{}, errors.New("actorID not found")
}

func (HTTPClientAPIMock) GetForgePersonFromAP(ctx context.Context, personID forgefed.PersonID) (forgefed.ForgePerson, error) {
	var MockForgePerson1 = forgefed.ForgePerson{}
	err := MockForgePerson1.UnmarshalJSON([]byte(`{"type":"Person","preferredUsername":"MaxMuster"}`))
	if err != nil {
		return forgefed.ForgePerson{}, err
	}
	return MockForgePerson1, nil
}

func (HTTPClientAPIMock) PostLikeActivities(ctx context.Context, doer user.User, activityList []forgefed.ForgeLike) error {
	return nil
}
