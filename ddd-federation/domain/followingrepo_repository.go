// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package domain

import (
	"context"

	"code.gitea.io/gitea/models/repo"
)

type FollowingRepoRepository interface {
	StoreFollowingRepos(ctx context.Context, localRepoID int64, followingRepoList []*repo.FollowingRepo) error
	FindFollowingReposByRepoID(ctx context.Context, repoID int64) ([]*repo.FollowingRepo, error)
}
