// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

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

func CreateFederatedRepo(ctx context.Context, repo *FederatedRepo) error {
	if res, err := validation.IsValid(repo); !res {
		return fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).Insert(repo)
	return err
}

func UpdateFederatedRepo(ctx context.Context, repo *FederatedRepo) error {
	if res, err := validation.IsValid(repo); !res {
		return fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).ID(repo.ID).Update(repo)
	return err
}
