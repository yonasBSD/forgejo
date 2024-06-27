// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federationhost

import (
	"context"

	domain "code.gitea.io/gitea/ddd-federation/domain/federationhost"
)

type FederationHostRepositoryMock struct{}

func (FederationHostRepositoryMock) GetFederationHost(ctx context.Context, ID int64) (*domain.FederationHost, error) {
	return nil, nil
}

func (FederationHostRepositoryMock) FindFederationHostByFqdn(ctx context.Context, fqdn string) (*domain.FederationHost, error) {
	if fqdn == "www.existingFederationHost.com" {
		return &MockFederationHost2, nil
	}
	return nil, nil
}

func (FederationHostRepositoryMock) CreateFederationHost(ctx context.Context, host *domain.FederationHost) error {
	return nil
}

func (FederationHostRepositoryMock) UpdateFederationHost(ctx context.Context, host *domain.FederationHost) error {
	return nil
}
