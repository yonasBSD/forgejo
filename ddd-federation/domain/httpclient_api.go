// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package domain

import (
	"context"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/forgefed"
)

type HttpClientAPI interface {
	GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (FederationHost, error)
	GetForgePersonFromAP(ctx context.Context, personID forgefed.PersonID) (forgefed.ForgePerson, error)
	PostLikeActivities(ctx context.Context, doer user.User, activityList []forgefed.ForgeLike) error
}
