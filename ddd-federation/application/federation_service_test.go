// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"testing"
)

func Test_GetFederationHostForURI(t *testing.T) {
	fhr := FederationHostRepositoryMock{}
	frr := FollowingRepoRepositoryMock{}
	ur := UserRepositoryMock{}
	rr := RepoRepositoryMock{}
	hca := HttpClientAPIMock{}
	sut := NewFederationService(fhr, frr, ur, rr, hca)

	sut.GetFederationHostForURI(context.Background(), "https://www.example.com/api/v1/activitypub/user-id/30")
	//TODO: Complete this unit test
	t.Fail()
}
