// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federationhost

import (
	"context"

	domain "code.gitea.io/gitea/ddd-federation/domain/federationhost"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/forgefed"
)

type FederationHostHttpClientImpl struct{}

func (FederationHostHttpClientImpl) GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (domain.FederationHost, error) {
	actionsUser := user.NewActionsUser()
	client, err := activitypub.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return domain.FederationHost{}, err
	}
	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return domain.FederationHost{}, err
	}
	nodeInfoWellKnown, err := domain.NewNodeInfoWellKnown(body)
	if err != nil {
		return domain.FederationHost{}, err
	}
	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return domain.FederationHost{}, err
	}
	nodeInfo, err := domain.NewNodeInfo(body)
	if err != nil {
		return domain.FederationHost{}, err
	}
	result, err := domain.NewFederationHost(nodeInfo, actorID.Host)
	if err != nil {
		return domain.FederationHost{}, err
	}

	return result, nil
}
