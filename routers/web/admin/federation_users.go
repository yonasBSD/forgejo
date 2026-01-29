// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	"forgejo.org/models/forgefed"
	users_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
)

const (
	tplFederationUsers base.TplName = "admin/federation/users"
)

func FederationUsers(ctx *context.Context) {
	federationHostID := ctx.ParamsInt64("id")

	if federationHostID < 1 {
		users, err := users_model.FindFederatedUsers(ctx)
		if err != nil {
			ctx.ServerError("GetFederatedUsers", err)
			return
		}
		ctx.Data["Users"] = users
		ctx.Data["TotalCount"] = len(users)
	} else {
		host, err := forgefed.GetFederationHost(ctx, federationHostID)
		if err != nil {
			ctx.ServerError("GetFederationHost", err)
			return
		}
		ctx.Data["Host"] = host

		users, err := users_model.FindFederatedUsersByHostID(ctx, federationHostID)
		if err != nil {
			ctx.ServerError("GetFederatedUsersByHostID", err)
			return
		}
		ctx.Data["Users"] = users
		ctx.Data["TotalCount"] = len(users)
	}

	ctx.Data["Title"] = "Federation users"
	ctx.Data["PageIsAdminFederationUsers"] = true

	ctx.HTML(http.StatusOK, tplFederationUsers)
}
