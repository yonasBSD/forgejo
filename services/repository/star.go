// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"

	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/federation"
)

// ToDo: Consistent Naming, smth like StarLocalAndFollowingRepo
func StarRepoAndFederate(ctx context.Context, doer user.User, repoID int64, star bool) error {
	if err := repo.StarRepo(ctx, doer.ID, repoID, star); err != nil {
		return err
	}

	if star && setting.Federation.Enabled {
		if err := federation.SendLikeActivities(ctx, doer, repoID); err != nil {
			return err
		}
	}

	return nil
}
