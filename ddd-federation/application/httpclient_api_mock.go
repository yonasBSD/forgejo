// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"

	"code.gitea.io/gitea/ddd-federation/domain"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/forgefed"
)

type HttpClientAPIMock struct{}

func (HttpClientAPIMock) GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (domain.FederationHost, error) {
	return domain.FederationHost{}, nil
}

func (HttpClientAPIMock) GetForgePersonFromAP(ctx context.Context, personID forgefed.PersonID) (forgefed.ForgePerson, error) {
	return forgefed.ForgePerson{}, nil
}

func (HttpClientAPIMock) PostLikeActivities(ctx context.Context, doer user.User, activityList []forgefed.ForgeLike) error {
	return nil
}
