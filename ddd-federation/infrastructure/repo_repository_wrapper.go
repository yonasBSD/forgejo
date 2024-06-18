// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package infrastructure

import (
	"context"

	"code.gitea.io/gitea/models/repo"
)

// ToDo: For now this is a wrapper for repository functionality of models/repo
type RepoRepositoryWrapper struct{}

func (RepoRepositoryWrapper) IsStaring(ctx context.Context, userID, repoID int64) bool {
	return repo.IsStaring(ctx, userID, repoID)
}

func (RepoRepositoryWrapper) StarRepo(ctx context.Context, userID, repoID int64, star bool) error {
	return repo.StarRepo(ctx, userID, repoID, star)
}
