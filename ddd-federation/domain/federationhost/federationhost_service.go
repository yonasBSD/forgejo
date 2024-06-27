// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federationhost

import (
	"context"

	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
)

// review gec 2024-06-27: should we keep this interface?
type FederationHostService interface {
	CreateFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*FederationHost, error)
	GetOrCreateFederationHostForURI(ctx context.Context, actorURI string) (*FederationHost, error)
}

type FederationHostServiceImpl struct {
	FederationHostRepository FederationHostRepository
	FederationhostHttpClient FederationHostHttpClient
}

func (s FederationHostServiceImpl) CreateFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*FederationHost, error) {
	result, err := s.FederationhostHttpClient.GetFederationHostFromAP(ctx, actorID)
	if err != nil {
		return nil, err
	}
	err = s.FederationHostRepository.CreateFederationHost(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s FederationHostServiceImpl) GetOrCreateFederationHostForURI(ctx context.Context, actorURI string) (*FederationHost, error) {
	log.Info("Input was: %v", actorURI)
	rawActorID, err := fm.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := s.FederationHostRepository.FindFederationHostByFqdn(ctx, rawActorID.Host)
	if err != nil {
		return nil, err
	}
	if federationHost == nil {
		result, err := s.CreateFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return nil, err
		}
		federationHost = result
	}
	return federationHost, nil
}
