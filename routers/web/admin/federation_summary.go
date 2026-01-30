// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	"forgejo.org/models/forgefed"
	users_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const (
	tplFederationSummary base.TplName = "admin/federation/summary"
)

func FederationSummary(ctx *context.Context) {
	hosts_total, err := forgefed.CountFederationHosts(ctx)
	if err != nil {
		ctx.ServerError("CountFederationHosts", err)
		return
	}

	users_total, err := users_model.CountFederatedUsers(ctx)
	if err != nil {
		ctx.ServerError("CountFederatedUsers", err)
		return
	}

	ctx.Data["HostsTotal"] = hosts_total
	ctx.Data["UsersTotal"] = users_total

	ctx.Data["FederationEnabled"] = setting.Federation.Enabled
	ctx.Data["FederationShareUserStatistics"] = setting.Federation.ShareUserStatistics
	ctx.Data["FederationMaxSize"] = setting.Federation.MaxSize / 1024 / 1024 // in MiB
	ctx.Data["FederationSignatureAlgorithms"] = setting.Federation.SignatureAlgorithms
	ctx.Data["FederationDigestAlgorithm"] = setting.Federation.DigestAlgorithm
	ctx.Data["FederationGetHeaders"] = setting.Federation.GetHeaders
	ctx.Data["FederationPostHeaders"] = setting.Federation.PostHeaders
	ctx.Data["FederationSignatureEnforced"] = setting.Federation.SignatureEnforced

	ctx.Data["Title"] = "Federation summary"
	ctx.Data["PageIsAdminFederationSummary"] = true

	ctx.HTML(http.StatusOK, tplFederationSummary)
}
