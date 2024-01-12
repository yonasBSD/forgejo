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

func GetFederationInfoByHostFqdn(ctx context.Context, fqdn string) (*FederationInfo, error) {
	info := new(FederationInfo)
	has, err := db.GetEngine(ctx).Where("host_fqdn=?", fqdn).Get(info)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationInfo record %v does not exist", fqdn)
	}
	if res, err := validation.IsValid(info); !res {
		return nil, fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	return info, nil
}
