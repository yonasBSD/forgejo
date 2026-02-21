// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	"forgejo.org/models/forgefed"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
)

const (
	tplFederationHosts base.TplName = "admin/federation/hosts"
)

func FederationHosts(ctx *context.Context) {
	sort := ctx.FormTrim("sort")

	hosts, err := forgefed.FindFederationHosts(ctx)
	if err != nil {
		ctx.ServerError("GetFederationHosts", err)
		return
	}
	total := len(hosts)

	ctx.Data["Title"] = ctx.Tr("admin.federation.hosts.title")
	ctx.Data["PageIsAdminFederationHosts"] = true
	ctx.Data["SortType"] = sort
	ctx.Data["TotalCount"] = total
	ctx.Data["Hosts"] = hosts

	ctx.HTML(http.StatusOK, tplFederationHosts)
}
