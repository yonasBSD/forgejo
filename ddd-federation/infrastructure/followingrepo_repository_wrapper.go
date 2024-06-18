// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package infrastructure

import (
	"context"

	"code.gitea.io/gitea/models/repo"
)

// ToDo: For now this is a wrapper for models/repo/repo_repository
type FollowingRepoRepositoryWrapper struct{}

func (FollowingRepoRepositoryWrapper) StoreFollowingRepos(ctx context.Context, localRepoID int64, followingRepoList []*repo.FollowingRepo) error {
	return repo.StoreFollowingRepos(ctx, localRepoID, followingRepoList)
}

func (FollowingRepoRepositoryWrapper) FindFollowingReposByRepoID(ctx context.Context, repoID int64) ([]*repo.FollowingRepo, error) {
	return repo.FindFollowingReposByRepoID(ctx, repoID)
}
