// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetOrCreateFederationHostForURI(t *testing.T) {
	fhr := FederationHostRepositoryMock{}
	frr := FollowingRepoRepositoryMock{}
	ur := UserRepositoryMock{}
	rr := RepoRepositoryMock{}
	hca := HTTPClientAPIMock{}
	sut := NewFederationService(fhr, frr, ur, rr, hca)

	host, err := sut.GetOrCreateFederationHostForURI(context.Background(), "https://www.example.com/api/v1/activitypub/user-id/30")

	assert.Nil(t, err)
	assert.Equal(t, &MockFederationHost1, host)
}

func Test_GetOrCreateFederationUserForID(t *testing.T) {
	fhr := FederationHostRepositoryMock{}
	frr := FollowingRepoRepositoryMock{}
	ur := UserRepositoryMock{}
	rr := RepoRepositoryMock{}
	hca := HTTPClientAPIMock{}
	sut := NewFederationService(fhr, frr, ur, rr, hca)

	MockPersonID.Source = "forgejo"
	fedUser, err := sut.GetOrCreateFederationUserForID(context.Background(), MockPersonID, &MockFederationHost1)

	assert.Nil(t, err)
	assert.Equal(t, &MockUser, fedUser)
}
