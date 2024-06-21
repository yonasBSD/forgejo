// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package infrastructuremock

import (
	"context"
)

type RepoRepositoryMock struct{}

func (RepoRepositoryMock) IsStaring(ctx context.Context, userID, repoID int64) bool {
	return false
}

func (RepoRepositoryMock) StarRepo(ctx context.Context, userID, repoID int64, star bool) error {
	return nil
}
