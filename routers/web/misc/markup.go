// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
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
		ctx.Repo.BranchName = branchName
	}

	common.RenderMarkup(ctx.Base, ctx.Repo, form.Mode, form.Text, relativePath, form.FilePath, form.Wiki)
}
