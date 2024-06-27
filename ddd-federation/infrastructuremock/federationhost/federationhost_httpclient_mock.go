// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federationhost

import (
	"context"
	"errors"
	"time"

	domain "code.gitea.io/gitea/ddd-federation/domain/federationhost"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/timeutil"
)

type FederationHostHttpClientMock struct{}

const (
	MockFederationHost1ID int64 = 1
	MockFederationHost2ID int64 = 2
)

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

var MockFederationHost2 domain.FederationHost = domain.FederationHost{
	ID:       MockFederationHost2ID,
	HostFqdn: "https://www.existingFederationHost.com/",
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

// review jem 2024-06-27: do not hide the test-domain object szenarios deep in mock
var MockActorID forgefed.ActorID = forgefed.ActorID{
	ID:               "30",
	Schema:           "https",
	Host:             "www.example.com",
	Path:             "api/v1/activitypub/user-id",
	Port:             "",
	UnvalidatedInput: "https://www.example.com/api/v1/activitypub/user-id/30",
}

func (FederationHostHttpClientMock) GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (domain.FederationHost, error) {
	if actorID == MockActorID {
		return MockFederationHost1, nil
	}
	return domain.FederationHost{}, errors.New("actorID not found")
}
