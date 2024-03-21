// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"code.gitea.io/gitea/modules/validation"
)

// FederatedRepo represents a federated Repository Actor connected with a local Repo
type FederatedRepo struct {
	ID               int64  `xorm:"pk autoincr"`
	RepositoryID     int64  `xorm:"NOT NULL"`
	ExternalID       string `xorm:"TEXT UNIQUE(federation_repo_mapping) NOT NULL"`
	FederationHostID int64  `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
}

func NewFederatedRepo(repositoryID int64, externalID string, federationHostID int64) (FederatedRepo, error) {
	result := FederatedRepo{
		RepositoryID:     repositoryID,
		ExternalID:       externalID,
		FederationHostID: federationHostID,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedRepo{}, err
	}
	return result, nil
}

func (user FederatedRepo) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.RepositoryID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	return result
}
