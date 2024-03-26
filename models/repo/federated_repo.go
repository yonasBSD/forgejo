// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"code.gitea.io/gitea/modules/validation"
)

// FederatedRepo represents a federated Repository Actor connected with a local Repo
// ToDo: We currently get database errors if different repos on the same server want to save the same federated repos in their list
type FederatedRepo struct {
	ID               int64  `xorm:"pk autoincr"`
	RepoID           int64  `xorm:"NOT NULL"`
	ExternalID       string `xorm:"TEXT UNIQUE(federation_repo_mapping) NOT NULL"`
	FederationHostID int64  `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	Uri              string
}

func NewFederatedRepo(repoID int64, externalID string, federationHostID int64, uri string) (FederatedRepo, error) {
	result := FederatedRepo{
		RepoID:           repoID,
		ExternalID:       externalID,
		FederationHostID: federationHostID,
		Uri:              uri,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedRepo{}, err
	}
	return result, nil
}

func (user FederatedRepo) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.RepoID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	result = append(result, validation.ValidateNotEmpty(user.Uri, "Uri")...)
	return result
}
