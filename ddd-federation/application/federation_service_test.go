// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"strings"
	"testing"

	mock "code.gitea.io/gitea/ddd-federation/infrastructure_mock"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/setting"
	"github.com/stretchr/testify/assert"
)

func Test_GetOrCreateFederationHostForURI(t *testing.T) {
	fhr := mock.FederationHostRepositoryMock{}
	frr := mock.FollowingRepoRepositoryMock{}
	ur := mock.UserRepositoryMock{}
	rr := mock.RepoRepositoryMock{}
	hca := mock.HTTPClientAPIMock{}
	sut := NewFederationService(fhr, frr, ur, rr, hca)

	host1, err1 := sut.GetOrCreateFederationHostForURI(context.Background(), "https://www.example.com/api/v1/activitypub/user-id/30")
	host2, err2 := sut.GetOrCreateFederationHostForURI(context.Background(), "https://www.existingFederationHost.com/api/v1/activitypub/user-id/30")

	assert.Nil(t, err1)
	assert.Nil(t, err2)
	assert.Equal(t, &mock.MockFederationHost1, host1)
	assert.Equal(t, &mock.MockFederationHost2, host2)
}

func Test_NewRemoteUser(t *testing.T) {
	setting.AppURL = "https://our-forgejo.com/"
	defer func() {
		setting.AppURL = ""
	}()

	fhr := mock.FederationHostRepositoryMock{}
	frr := mock.FollowingRepoRepositoryMock{}
	ur := mock.UserRepositoryMock{}
	rr := mock.RepoRepositoryMock{}
	hca := mock.HTTPClientAPIMock{}
	sut := NewFederationService(fhr, frr, ur, rr, hca)

	var mockPersonID forgefed.PersonID = forgefed.PersonID{
		ActorID: mock.MockActorID,
	}

	sutUser, err := sut.NewRemoteUser(mockPersonID, "", "MaxMuster")

	assert.Nil(t, err)
	assert.Equal(t, "maxmuster-www.example.com", sutUser.LowerName)
	assert.Equal(t, "MaxMuster-www.example.com", sutUser.Name)
	assert.Equal(t, "MaxMuster-www.example.com", sutUser.FullName)
	assert.Equal(t, "our-forgejo.com", strings.Split(sutUser.Email, "@")[1])
	assert.Equal(t, user.UserTypeRemoteUser, sutUser.Type)
}
