// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederationInfo))
}

func GetFederationInfo(ctx context.Context, ID int64) (*FederationInfo, error) {
	info := new(FederationInfo)
	has, err := db.GetEngine(ctx).Where("id=?", ID).Get(info)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationInfo record %v does not exist", ID)
	}
	if res, err := validation.IsValid(info); !res {
		return nil, fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	return info, nil
}

func FindFederationInfoByHostFqdn(ctx context.Context, fqdn string) (*FederationInfo, error) {
	info := new(FederationInfo)
	has, err := db.GetEngine(ctx).Where("host_fqdn=?", fqdn).Get(info)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(info); !res {
		return nil, fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	return info, nil
}

func CreateFederationInfo(ctx context.Context, info FederationInfo) error {
	if res, err := validation.IsValid(info); !res {
		return fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).Insert(info)
	return err
}

func UpdateFederationInfo(ctx context.Context, info FederationInfo) error {
	if res, err := validation.IsValid(info); !res {
		return fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).ID(info.ID).Update(info)
	return err
}
