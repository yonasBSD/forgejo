// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.gitea.io/gitea/ddd-federation/domain"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/timeutil"
)

type HttpClientAPIMock struct{}

var MockFederationHost1 domain.FederationHost = domain.FederationHost{
	ID:       1,
	HostFqdn: "https://www.example.com/",
	NodeInfo: domain.NodeInfo{
		SoftwareName: domain.ForgejoSourceType,
	},
	LatestActivity: time.Date(2020, 01, 01, 12, 12, 12, 0, time.Now().UTC().Location()),
	Created:        timeutil.TimeStampNow(),
	Updated:        timeutil.TimeStampNow(),
}

var MockActorID forgefed.ActorID = forgefed.ActorID{
	ID:               "30",
	Schema:           "https",
	Host:             "www.example.com",
	Path:             "api/v1/activitypub/user-id",
	Port:             "",
	UnvalidatedInput: "https://www.example.com/api/v1/activitypub/user-id/30",
}

func (HttpClientAPIMock) GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (domain.FederationHost, error) {
	if actorID == MockActorID {
		fmt.Printf("actorID is: %v", actorID)
		return MockFederationHost1, nil
	}
	return domain.FederationHost{}, errors.New("actorID not found")
}

func (HttpClientAPIMock) GetForgePersonFromAP(ctx context.Context, personID forgefed.PersonID) (forgefed.ForgePerson, error) {
	return forgefed.ForgePerson{}, nil
}

func (HttpClientAPIMock) PostLikeActivities(ctx context.Context, doer user.User, activityList []forgefed.ForgeLike) error {
	return nil
}
