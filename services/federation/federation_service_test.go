// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsValidRepoUri(t *testing.T) {
	validUri := "http://localhost:3000/me/test"
	invalidUri := "http://localhost:3000/me/test/foo"
	assert.True(t, IsValidRepoUri(validUri))
	assert.False(t, IsValidRepoUri(invalidUri))
}

func Test_GetRepoAPIUriFromRepoUri(t *testing.T) {
	uri := "http://localhost:3000/me/test"
	expectedUri := "http://localhost:3000/api/v1/repos/me/test"

	res, err := GetRepoAPIUriFromRepoUri(uri)
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, expectedUri, res)
}

func Test_GetRepoOwnerAndNameFromRepoUri(t *testing.T) {
	validUri := "http://localhost:3000/me/test"
	invalidUri := "http://localhost:3000/me/test/foo"

	owner, name, err := GetRepoOwnerAndNameFromRepoUri(validUri)
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, "me", owner)
	assert.Equal(t, "test", name)

	_, _, err = GetRepoOwnerAndNameFromRepoUri(invalidUri)
	assert.Error(t, err)
}
