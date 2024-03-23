// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
// ToDo: Is this package the right place for federated repo? May need to diskuss this.
package repo

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederatedRepo))
}

// TODO: do we need this?
func GetFederatedRepo(ctx context.Context, ID int64) (*FederatedRepo, error) {
	repo := new(FederatedRepo)
	has, err := db.GetEngine(ctx).Where("id=?", ID).Get(repo)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationInfo record %v does not exist", ID)
	}
	if res, err := validation.IsValid(repo); !res {
		return nil, fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	return repo, nil
}

// TODO: do we need this?
func FindFederatedRepoByFQDN(ctx context.Context, fqdn string) (*FederatedRepo, error) {
	repo := new(FederatedRepo)
	has, err := db.GetEngine(ctx).Where("host_fqdn=?", strings.ToLower(fqdn)).Get(repo)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(repo); !res {
		return nil, fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	return repo, nil
}

// TODO: do we need this?
func CreateFederatedRepo(ctx context.Context, repo *FederatedRepo) error {
	if res, err := validation.IsValid(repo); !res {
		return fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).Insert(repo)
	return err
}

func UpdateFederatedRepo(ctx context.Context, localRepoId int64, federatedRepoList []*FederatedRepo) error {
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
