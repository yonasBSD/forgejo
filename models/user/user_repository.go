// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederatedUser))
}

func CreateFederationUser(ctx context.Context, user *FederatedUser) error {
	if res, err := validation.IsValid(user); !res {
		return fmt.Errorf("FederatedUser is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).Insert(user)
	return err
}

func FindFederatedUser(ctx context.Context, externalID string,
	federationHostID int64) (*User, *FederatedUser, error) {
	federatedUser := new(FederatedUser)
	user := new(User)
	has, err := db.GetEngine(ctx).Where("external_id=? and federation_host_id=?", externalID, federationHostID).Get(federatedUser)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, nil
	}
	has, err = db.GetEngine(ctx).ID(federatedUser.UserID).Get(user)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, fmt.Errorf("User %v for federated user is missing.", federatedUser.UserID)
	}

	if res, err := validation.IsValid(*user); !res {
		return nil, nil, fmt.Errorf("FederatedUser is not valid: %v", err)
	}
	if res, err := validation.IsValid(*federatedUser); !res {
		return nil, nil, fmt.Errorf("FederatedUser is not valid: %v", err)
	}
	return user, federatedUser, nil
}
