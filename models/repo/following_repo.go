// Copyright 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"code.gitea.io/gitea/modules/validation"
)

// FollowingRepo represents a federated Repository Actor connected with a local Repo
// ToDo: We currently get database errors if different repos on the same server want to save the same federated repos in their list
type FollowingRepo struct {
	ID               int64  `xorm:"pk autoincr"`
	RepoID           int64  `xorm:"NOT NULL"`
	ExternalID       string `xorm:"TEXT UNIQUE(federation_repo_mapping) NOT NULL"`
	FederationHostID int64  `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	ExternalOwner    string `xorm:"TEXT UNIQUE(federation_repo_mapping) NOT NULL"`
	ExternalRepoName string `xorm:"TEXT UNIQUE(federation_repo_mapping) NOT NULL"`
	Uri              string
}

func NewFollowingRepo(repoID int64, externalID string, federationHostID int64, externalOwner string, externalRepoName string, uri string) (FollowingRepo, error) {
	result := FollowingRepo{
		RepoID:           repoID,
		ExternalID:       externalID,
		FederationHostID: federationHostID,
		ExternalOwner:    externalOwner,
		ExternalRepoName: externalRepoName,
		Uri:              uri,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FollowingRepo{}, err
	}
	return result, nil
}

func (user FollowingRepo) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.RepoID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalOwner, "ExternalOwner")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalRepoName, "ExternalRepoName")...)
	result = append(result, validation.ValidateNotEmpty(user.Uri, "Uri")...)
	return result
}
