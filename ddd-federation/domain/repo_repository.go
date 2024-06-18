// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package domain

import (
	"context"
)

type RepoRepository interface {
	IsStaring(ctx context.Context, userID, repoID int64) bool
	StarRepo(ctx context.Context, userID, repoID int64, star bool) error
}
