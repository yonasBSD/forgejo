// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederatedUser))
}

type FederatedUser struct {
	ID               int64          `xorm:"pk NOT NULL"`
	UserID           int64          `xorm:"NOT NULL"`
	ExternalID       string         `xorm:"TEXT UNIQUE(federation_mapping) NOT NULL"`
	FederationHostID int64          `xorm:"UNIQUE(federation_mapping) NOT NULL"`
	RawData          map[string]any `xorm:"TEXT JSON"`
}

func NewFederatedUser(userID int64, externalID string, federationHostID int64, rawData map[string]any) (FederatedUser, error) {
	result := FederatedUser{
		UserID:           userID,
		ExternalID:       externalID,
		FederationHostID: federationHostID,
		RawData:          rawData,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUser{}, err
	}
	return result, nil
}

func (user FederatedUser) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	return result
}
