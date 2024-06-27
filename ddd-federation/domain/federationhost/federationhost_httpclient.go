// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federationhost

import (
	"context"

	"code.gitea.io/gitea/modules/forgefed"
)

type FederationHostHttpClient interface {
	GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (FederationHost, error)
}
