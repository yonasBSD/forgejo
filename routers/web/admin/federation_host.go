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
	tplFederationHost base.TplName = "admin/federation/host"
)

func FederationHost(ctx *context.Context) {
	federationHostID := ctx.ParamsInt64("id")

	host, err := forgefed.GetFederationHost(ctx, federationHostID)
	if err != nil {
		ctx.ServerError("GetFederationHost", err)
		return
	}

	users_total, err := users_model.CountFederatedUsersByHostID(ctx, federationHostID)
	if err != nil {
		ctx.ServerError("CountFederatedUsersByHostID", err)
		return
	}

	ctx.Data["Host"] = host
	ctx.Data["UsersTotal"] = users_total
	ctx.Data["Title"] = host.HostFqdn
	ctx.Data["PageIsAdminFederationHosts"] = true

	ctx.HTML(http.StatusOK, tplFederationHost)
}
