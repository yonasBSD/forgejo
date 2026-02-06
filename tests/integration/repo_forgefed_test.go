// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgefedRepositoryCreateHostValid(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		// Arrange
		ctx := t.Context()

		// Act
		err := forgefed.CreateFederationHost(ctx, &forgefed.FederationHost{
			HostFqdn:   "forgejo.example.com",
			HostPort:   80,
			HostSchema: "http",
			NodeInfo: forgefed.NodeInfo{
				SoftwareName: "forgejo",
			},
		})

		// Assert
		require.NoError(t, err)
		unittest.AssertExistsAndLoadBean(t, &forgefed.FederationHost{HostFqdn: "forgejo.example.com"})
	})
}

func TestForgefedRepositoryCreateHostInvalid(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		// Arrange
		ctx := t.Context()

		// Act
		err := forgefed.CreateFederationHost(ctx, &forgefed.FederationHost{
			// invalid
		})

		// Assert
		require.Error(t, err)
		unittest.AssertNotExistsBean(t, &forgefed.FederationHost{HostFqdn: "forgejo.example.com"})
	})
}

func TestForgefedRepositoryCreateUserValid(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		// Arrange
		ctx := t.Context()

		err := forgefed.CreateFederationHost(ctx, &forgefed.FederationHost{
			HostFqdn:   "forgejo.example.com",
			HostPort:   80,
			HostSchema: "http",
			NodeInfo: forgefed.NodeInfo{
				SoftwareName: "forgejo",
			},
		})
		require.NoError(t, err)

		// Act
		err = user.CreateFederatedUser(ctx, &user.User{
			Name:  "Bob",
			Email: "bob@forgejo.example.com",
		}, &user.FederatedUser{
			ExternalID:       "1",
			FederationHostID: 1,
			InboxPath:        "/inbox",
		})

		// Assert
		require.NoError(t, err)
		localUser := unittest.AssertExistsAndLoadBean(t, &user.User{Email: "bob@forgejo.example.com"})
		unittest.AssertExistsAndLoadBean(t, &user.FederatedUser{UserID: localUser.ID, FederationHostID: 1})
	})
}

func TestForgefedRepositoryCreateUserInvalid(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		// Arrange
		ctx := t.Context()

		err := forgefed.CreateFederationHost(ctx, &forgefed.FederationHost{
			HostFqdn:   "forgejo.example.com",
			HostPort:   80,
			HostSchema: "http",
			NodeInfo: forgefed.NodeInfo{
				SoftwareName: "forgejo",
			},
		})
		require.NoError(t, err)

		// Act
		err = user.CreateFederatedUser(ctx, &user.User{
			Name:  "Bob",
			Email: "bob@forgejo.example.com",
		}, &user.FederatedUser{
			// invalid
		})

		// Assert
		require.Error(t, err)
		unittest.AssertNotExistsBean(t, &user.User{Email: "bob@forgejo.example.com"})
		unittest.AssertNotExistsBean(t, &user.FederatedUser{FederationHostID: 1})
	})
}

func TestForgefedRepositoryFindHostsAndUsers(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		// Arrange
		ctx := t.Context()

		err := forgefed.CreateFederationHost(ctx, &forgefed.FederationHost{
			HostFqdn:   "bob.example.com",
			HostPort:   80,
			HostSchema: "http",
			NodeInfo: forgefed.NodeInfo{
				SoftwareName: "forgejo",
			},
		})
		require.NoError(t, err)

		err = forgefed.CreateFederationHost(ctx, &forgefed.FederationHost{
			HostFqdn:   "alice.example.com",
			HostPort:   443,
			HostSchema: "https",
			NodeInfo: forgefed.NodeInfo{
				SoftwareName: "gitea",
			},
		})
		require.NoError(t, err)

		err = user.CreateFederatedUser(ctx, &user.User{
			Name:  "Bob",
			Email: "bob@bob.example.com",
		}, &user.FederatedUser{
			ExternalID:       "1",
			FederationHostID: 1,
			InboxPath:        "/inbox",
		})
		require.NoError(t, err)

		err = user.CreateFederatedUser(ctx, &user.User{
			Name:  "Alice",
			Email: "alice@alice.example.com",
		}, &user.FederatedUser{
			ExternalID:       "1",
			FederationHostID: 2,
			InboxPath:        "/inbox",
		})
		require.NoError(t, err)

		err = user.CreateFederatedUser(ctx, &user.User{
			Name:  "Eve",
			Email: "eve@alice.example.com",
		}, &user.FederatedUser{
			ExternalID:       "2",
			FederationHostID: 2,
			InboxPath:        "/inbox",
		})
		require.NoError(t, err)

		// Act & Assert
		hosts, err := forgefed.FindFederationHosts(ctx)
		require.NoError(t, err)
		assert.Len(t, hosts, 2)
		assert.Equal(t, "bob.example.com", hosts[0].HostFqdn)
		assert.Equal(t, "alice.example.com", hosts[1].HostFqdn)

		users, err := user.FindFederatedUsers(ctx)
		require.NoError(t, err)
		assert.Len(t, users, 3) // Bob, Alice and Eve

		users, err = user.FindFederatedUsersByHostID(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, users, 1) // Only Bob belongs to the host with ID 1
		assert.Equal(t, int64(1), users[0].FederationHostID)

		users, err = user.FindFederatedUsersByHostID(ctx, 2)
		require.NoError(t, err)
		assert.Len(t, users, 2) // Alice and Eve belong to the host with ID 2
		assert.Equal(t, int64(2), users[0].FederationHostID)
		assert.Equal(t, int64(2), users[1].FederationHostID)
	})
}
