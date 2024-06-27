// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federationhost

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/gitea/ddd-federation/domain/federationhost"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(federationhost.FederationHost))
}

type FederationHostRepositoryImpl struct{}

func (FederationHostRepositoryImpl) GetFederationHost(ctx context.Context, ID int64) (*federationhost.FederationHost, error) {
	host := new(federationhost.FederationHost)
	has, err := db.GetEngine(ctx).Where("id=?", ID).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationInfo record %v does not exist", ID)
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	return host, nil
}

func (FederationHostRepositoryImpl) FindFederationHostByFqdn(ctx context.Context, fqdn string) (*federationhost.FederationHost, error) {
	host := new(federationhost.FederationHost)
	has, err := db.GetEngine(ctx).Where("host_fqdn=?", strings.ToLower(fqdn)).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	return host, nil
}

func (FederationHostRepositoryImpl) CreateFederationHost(ctx context.Context, host *federationhost.FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return err
	}
	_, err := db.GetEngine(ctx).Insert(host)
	return err
}

func (FederationHostRepositoryImpl) UpdateFederationHost(ctx context.Context, host *federationhost.FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return err
	}
	_, err := db.GetEngine(ctx).ID(host.ID).Update(host)
	return err
}
