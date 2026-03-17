// Copyright 2021 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package doctor

import (
	"context"
	"image"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/avatarstore"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"

	"xorm.io/builder"
)

func init() {
	Register(&Check{
		Title:                      "Generate resized versions of user avatars",
		Name:                       "resize_user_avatars",
		IsDefault:                  false,
		Run:                        GenerateResizedUserAvatars(storage.Avatars, setting.Avatar.MaxOriginSize),
		AbortIfFailed:              false,
		SkipDatabaseInitialization: false,
		Priority:                   1,
	})
	Register(&Check{
		Title:                      "Generate resized versions of repository avatars",
		Name:                       "resize_repo_avatars",
		IsDefault:                  false,
		Run:                        GenerateResizedRepoAvatars(storage.RepoAvatars, setting.Avatar.MaxOriginSize),
		AbortIfFailed:              false,
		SkipDatabaseInitialization: false,
		Priority:                   1,
	})
}

// GenerateResizedUserAvatars makes sure all resized versions of user avatars are stored in the cache
func GenerateResizedUserAvatars(imgStorage storage.ObjectStorage, maxOriginSize int64) func(ctx context.Context, logger log.Logger, autofix bool) error {
	return func(ctx context.Context, logger log.Logger, autofix bool) error {
		if err := storage.Init(); err != nil {
			logger.Error("storage.Init failed: %v", err)
			return err
		}

		err := db.Iterate(
			ctx,
			builder.Neq{"avatar": ""},
			func(ctx context.Context, user *user_model.User) error {
				return precomputeResizedAvatars(imgStorage, user.Avatar, maxOriginSize)
			},
		)
		return err
	}
}

// GenerateResizedRepoAvatars makes sure all resized versions of user avatars are stored in the cache
func GenerateResizedRepoAvatars(imgStorage storage.ObjectStorage, maxOriginSize int64) func(ctx context.Context, logger log.Logger, autofix bool) error {
	return func(ctx context.Context, logger log.Logger, autofix bool) error {
		if err := storage.Init(); err != nil {
			logger.Error("storage.Init failed: %v", err)
			return err
		}

		err := db.Iterate(
			ctx,
			builder.Neq{"avatar": ""},
			func(ctx context.Context, repo *repo_model.Repository) error {
				return precomputeResizedAvatars(imgStorage, repo.Avatar, maxOriginSize)
			},
		)
		return err
	}
}

func precomputeResizedAvatars(imgStorage storage.ObjectStorage, imgPath string, maxOriginSize int64) error {
	// Load the avatar
	avatarBytes, err := imgStorage.Open(imgPath)
	if err != nil {
		return err
	}
	meta, err := avatarBytes.Stat()
	if err != nil {
		return err
	}
	// if the avatar is small enough, don't compute resized versions for it.
	// This makes it possible to preserve animated avatars when they are small enough.
	if meta.Size() < maxOriginSize {
		return nil
	}
	img, _, err := image.Decode(avatarBytes)
	if err != nil {
		return err
	}
	return avatarstore.PrecomputeResizedAvatars(imgStorage, img, imgPath)
}
