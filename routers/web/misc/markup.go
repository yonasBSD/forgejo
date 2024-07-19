// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"
	"strings"

	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/services/context"
)

// Markup render markup document to HTML
func Markup(ctx *context.Context) {
	form := web.GetForm(ctx).(*api.MarkupOption)

	relativePath := form.Context

	if !form.Wiki {
		branchName := relativePath[strings.LastIndex(relativePath, "/")+1:]

		// Check write access for the current branch
		if !ctx.Repo.CanWriteToBranch(ctx, ctx.Doer, branchName) && !ctx.IsUserRepoAdmin() {
			ctx.Error(http.StatusForbidden, "reqRepoBranchWriter", "user should have a permission to write to this branch")
			return
		}
		ctx.Repo.BranchName = branchName
	}

	common.RenderMarkup(ctx.Base, ctx.Repo, form.Mode, form.Text, relativePath, form.FilePath, form.Wiki)
}
