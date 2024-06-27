// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federationhost

import "context"

type FederationHostRepository interface {
	GetFederationHost(ctx context.Context, ID int64) (*FederationHost, error)
	FindFederationHostByFqdn(ctx context.Context, fqdn string) (*FederationHost, error)
	CreateFederationHost(ctx context.Context, host *FederationHost) error
	UpdateFederationHost(ctx context.Context, host *FederationHost) error
}
