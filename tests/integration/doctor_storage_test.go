// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/test"
	doctor "forgejo.org/services/doctor"
	repo_service "forgejo.org/services/repository"
	user_service "forgejo.org/services/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveUnusedUserAvatars(t *testing.T) {
	defer tests.PrepareTestEnv(t, 1)()
	defer test.MockVariableValue(&setting.Avatar.Storage.Type, setting.LocalStorageType)()
	defer test.MockVariableValue(&setting.Avatar.Storage.Path, t.TempDir())()
	// make the maximum uncached image size small, so that our test image is bigger than that
	defer test.MockVariableValue(&setting.Avatar.MaxOriginSize, 3)()
	var err error

	ctx := db.DefaultContext

	u := unittest.AssertExistsAndLoadBean(t, &user.User{ID: 2})
	// generate an avatar for this user
	myImage := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)
	err = user_service.UploadAvatar(ctx, u, buff.Bytes())
	require.NoError(t, err)
	assert.NotEmpty(t, u.Avatar)
	avatarPath := u.CustomAvatarRelativePath()

	// disconnect the avatar from the user
	u.Avatar = ""
	err = user.UpdateUserCols(ctx, u, "avatar")
	require.NoError(t, err)

	// make sure the avatar is still stored as a file
	_, err = storage.Avatars.Stat(avatarPath)
	require.NoError(t, err)
	// the downscaled versions are also stored
	_, err = storage.Avatars.Stat(fmt.Sprintf("resized/64/%s", avatarPath))
	require.NoError(t, err)

	doctor.CheckStorage(&doctor.CheckStorageOptions{Avatars: true})(ctx, log.GetLogger("doctor"), true)

	// the avatar is no longer stored
	_, err = storage.Avatars.Stat(avatarPath)
	require.Error(t, err)
	// the downscaled versions are not stored either
	_, err = storage.Avatars.Stat(fmt.Sprintf("resized/64/%s", avatarPath))
	require.Error(t, err)
}

func TestRemoveUnusedRepoAvatars(t *testing.T) {
	defer tests.PrepareTestEnv(t, 1)()
	defer test.MockVariableValue(&setting.RepoAvatar.Storage.Type, setting.LocalStorageType)()
	defer test.MockVariableValue(&setting.RepoAvatar.Storage.Path, t.TempDir())()
	// make the maximum uncached image size small, so that our test image is bigger than that
	defer test.MockVariableValue(&setting.Avatar.MaxOriginSize, 3)()
	var err error

	ctx := db.DefaultContext

	r := unittest.AssertExistsAndLoadBean(t, &repo.Repository{ID: 2})
	// generate an avatar for this user
	myImage := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)
	err = repo_service.UploadAvatar(ctx, r, buff.Bytes())
	require.NoError(t, err)
	assert.NotEmpty(t, r.Avatar)
	avatarPath := r.CustomAvatarRelativePath()

	// disconnect the avatar from the user
	r.Avatar = ""
	err = repo.UpdateRepositoryCols(ctx, r, "avatar")
	require.NoError(t, err)

	// make sure the avatar is still stored as a file
	_, err = storage.RepoAvatars.Stat(avatarPath)
	require.NoError(t, err)
	// the downscaled versions are also stored
	_, err = storage.RepoAvatars.Stat(fmt.Sprintf("resized/64/%s", avatarPath))
	require.NoError(t, err)

	doctor.CheckStorage(&doctor.CheckStorageOptions{RepoAvatars: true})(ctx, log.GetLogger("doctor"), true)

	// the avatar is no longer stored
	_, err = storage.RepoAvatars.Stat(avatarPath)
	require.Error(t, err)
	// the downscaled versions are not stored either
	_, err = storage.RepoAvatars.Stat(fmt.Sprintf("resized/64/%s", avatarPath))
	require.Error(t, err)
}
