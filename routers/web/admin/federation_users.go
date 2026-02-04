// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
)

const (
	tplFederationUsers base.TplName = "admin/federation/users"
)

func FederationUsers(ctx *context.Context) {
	users, err := user_model.FindFederatedUsers(ctx)
	if err != nil {
		ctx.ServerError("GetFederatedUsers", err)
		return
	}

	ctx.Data["Users"] = users
	ctx.Data["TotalCount"] = len(users)
	ctx.Data["Title"] = "Federation users"
	ctx.Data["PageIsAdminFederationUsers"] = true

	ctx.HTML(http.StatusOK, tplFederationUsers)
}
