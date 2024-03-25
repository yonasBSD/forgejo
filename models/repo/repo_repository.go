// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
// ToDo: Is this package the right place for federated repo? May need to diskuss this.
package repo

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederatedRepo))
}

// ToDo: Validate before returning
func FindFederatedRepoByRepoID(ctx context.Context, repoId int64) ([]*FederatedRepo, error) {
	maxFederatedRepos := 10
	sess := db.GetEngine(ctx).Where("repo_id=?", repoId)
	sess = sess.Limit(maxFederatedRepos, 0)
	federatedRepoList := make([]*FederatedRepo, 0, maxFederatedRepos)
	return federatedRepoList, sess.Find(&federatedRepoList)
}

func StoreFederatedRepos(ctx context.Context, localRepoId int64, federatedRepoList []*FederatedRepo) error {
	for _, federatedRepo := range federatedRepoList {
		if res, err := validation.IsValid(*federatedRepo); !res {
			return fmt.Errorf("FederationInfo is not valid: %v", err)
		}
	}

	// Begin transaction
	ctx, committer, err := db.TxContext((ctx))
	if err != nil {
		return err
	}
	defer committer.Close()

	_, err = db.GetEngine(ctx).Where("repo_id=?", localRepoId).Delete(FederatedRepo{})
	if err != nil {
		return err
	}
	for _, federatedRepo := range federatedRepoList {
		_, err = db.GetEngine(ctx).Insert(federatedRepo)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return committer.Commit()
}
