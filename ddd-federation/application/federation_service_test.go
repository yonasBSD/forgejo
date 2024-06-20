// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetFederationHostForURI(t *testing.T) {
	fhr := FederationHostRepositoryMock{}
	frr := FollowingRepoRepositoryMock{}
	ur := UserRepositoryMock{}
	rr := RepoRepositoryMock{}
	hca := HttpClientAPIMock{}
	sut := NewFederationService(fhr, frr, ur, rr, hca)

	host, err := sut.GetFederationHostForURI(context.Background(), "https://www.example.com/api/v1/activitypub/user-id/30")

	assert.Nil(t, err)
	assert.Equal(t, host, &MockFederationHost1)
}

func Test_GetFederationUserForID(t *testing.T) {
	fhr := FederationHostRepositoryMock{}
	frr := FollowingRepoRepositoryMock{}
	ur := UserRepositoryMock{}
	rr := RepoRepositoryMock{}
	hca := HttpClientAPIMock{}
	sut := NewFederationService(fhr, frr, ur, rr, hca)

	MockPersonID.Source = "forgejo"
	fedUser, err := sut.GetFederationUserForID(context.Background(), MockPersonID, &MockFederationHost1)

	assert.Nil(t, err)
	assert.Equal(t, fedUser, &MockUser)

}
