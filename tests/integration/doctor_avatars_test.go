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
	"forgejo.org/tests"

	"github.com/stretchr/testify/require"
)

func TestPrecomputeUserAvatars(t *testing.T) {
	defer tests.PrepareTestEnv(t, 1)()
	var err error
	tmpDir := t.TempDir()
	defer test.MockVariableValue(&setting.Avatar.Storage.Type, setting.LocalStorageType)()
	defer test.MockVariableValue(&setting.Avatar.Storage.Path, tmpDir)()
	avatarStorage, err := storage.NewLocalStorage(t.Context(), &setting.Storage{Path: tmpDir})
	require.NoError(t, err)
	storage.Avatars = avatarStorage

	// make the maximum uncached image size small, so that our test image is bigger than that
	defer test.MockVariableValue(&setting.Avatar.MaxOriginSize, 3)()

	ctx := db.DefaultContext

	u := unittest.AssertExistsAndLoadBean(t, &user.User{ID: 2})
	// generate an avatar for this user
	myImage := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)
	// store the avatar, but only the original (not the resized versions)
	avatarPath := "some_id"
	storage.Avatars.Save(avatarPath, bytes.NewReader(buff.Bytes()), -1)
	u.UseCustomAvatar = true
	u.Avatar = avatarPath
	err = user.UpdateUserCols(ctx, u, "use_custom_avatar", "avatar")
	require.NoError(t, err)

	// run the doctor command
	err = doctor.GenerateResizedUserAvatars(storage.Avatars, setting.Avatar.MaxOriginSize)(ctx, log.GetLogger("doctor"), true)
	require.NoError(t, err)

	// the resized version of the avatar is now stored in the cache
	_, err = storage.Avatars.Stat(fmt.Sprintf("resized/64/%s", avatarPath))
	require.NoError(t, err)
}

func TestPrecomputeRepoAvatars(t *testing.T) {
	defer tests.PrepareTestEnv(t, 1)()
	var err error
	tmpDir := t.TempDir()
	defer test.MockVariableValue(&setting.RepoAvatar.Storage.Type, setting.LocalStorageType)()
	defer test.MockVariableValue(&setting.RepoAvatar.Storage.Path, tmpDir)()
	avatarStorage, err := storage.NewLocalStorage(t.Context(), &setting.Storage{Path: tmpDir})
	require.NoError(t, err)
	storage.RepoAvatars = avatarStorage
	// make the maximum uncached image size small, so that our test image is bigger than that
	defer test.MockVariableValue(&setting.Avatar.MaxOriginSize, 3)()

	ctx := db.DefaultContext

	u := unittest.AssertExistsAndLoadBean(t, &repo.Repository{ID: 2})
	// generate an avatar for this repo
	myImage := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)
	// store the avatar, but only the original (not the resized versions)
	avatarPath := "some_id"
	storage.RepoAvatars.Save(avatarPath, bytes.NewReader(buff.Bytes()), -1)
	u.Avatar = avatarPath
	err = repo.UpdateRepositoryCols(ctx, u, "avatar")
	require.NoError(t, err)

	// run the doctor command
	err = doctor.GenerateResizedRepoAvatars(storage.RepoAvatars, setting.Avatar.MaxOriginSize)(ctx, log.GetLogger("doctor"), true)
	require.NoError(t, err)

	// the resized version of the avatar is now stored in the cache
	_, err = storage.RepoAvatars.Stat(fmt.Sprintf("resized/64/%s", avatarPath))
	require.NoError(t, err)
}
