// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package infrastruckture_mock

import (
	"context"

	"code.gitea.io/gitea/models/repo"
)

type FollowingRepoRepositoryMock struct{}

func (FollowingRepoRepositoryMock) StoreFollowingRepos(ctx context.Context, localRepoID int64, followingRepoList []*repo.FollowingRepo) error {
	return nil
}

func (FollowingRepoRepositoryMock) FindFollowingReposByRepoID(ctx context.Context, repoID int64) ([]*repo.FollowingRepo, error) {
	return nil, nil
}
