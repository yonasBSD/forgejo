// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/lfs"
)

func BatchHandler(ctx *context.PrivateContext) {
	lfs.CheckAcceptMediaType(&context.Context{Base: ctx.Base})
	if ctx.Written() {
		return
	}

	ctx.Data["IsInternalLFS"] = true
	lfs.BatchHandler(&context.Context{Base: ctx.Base})
}

func DownloadHandler(ctx *context.PrivateContext) {
	ctx.Data["IsInternalLFS"] = true
	lfs.DownloadHandler(&context.Context{Base: ctx.Base})
}
