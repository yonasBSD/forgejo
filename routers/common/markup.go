// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/markup/markdown"
	"code.gitea.io/gitea/services/context"
)

// RenderMarkup renders markup text for the /markup and /markdown endpoints
func RenderMarkup(ctx *context.Base, repo *context.Repository, mode, text, urlPrefix, filePath string, wiki bool) {
	var markupType string
	relativePath := ""

	if len(text) == 0 {
		_, _ = ctx.Write([]byte(""))
		return
	}

	switch mode {
	case "markdown":
		// Raw markdown
		if err := markdown.RenderRaw(&markup.RenderContext{
			Ctx: ctx,
			Links: markup.Links{
				AbsolutePrefix: true,
				Base:           urlPrefix,
			},
		}, strings.NewReader(text), ctx.Resp); err != nil {
			ctx.Error(http.StatusInternalServerError, err.Error())
		}
		return
	case "comment":
		// Comment as markdown
		markupType = markdown.MarkupName
	case "gfm":
		// Github Flavored Markdown as document
		markupType = markdown.MarkupName
	case "file":
		// File as document based on file extension
		markupType = ""
		relativePath = filePath
	default:
		ctx.Error(http.StatusUnprocessableEntity, fmt.Sprintf("Unknown mode: %s", mode))
		return
	}

	meta := map[string]string{}
	if repo != nil && repo.Repository != nil {
		if mode == "comment" {
			meta = repo.Repository.ComposeMetas(ctx)
		} else {
			meta = repo.Repository.ComposeDocumentMetas(ctx)
		}
	}
	if mode != "comment" {
		meta["mode"] = "document"
	}

	repo.IsViewBranch = true
	if err := markup.Render(&markup.RenderContext{
		Ctx: ctx,
		Links: markup.Links{
			AbsolutePrefix: true,
			Base:           repo.RepoLink,
			BranchPath:     repo.BranchNameSubURL(),
			TreePath:       path.Dir(filePath),
		},
		Metas:        meta,
		IsWiki:       wiki,
		Type:         markupType,
		RelativePath: relativePath,
	}, strings.NewReader(text), ctx.Resp); err != nil {
		if markup.IsErrUnsupportedRenderExtension(err) {
			ctx.Error(http.StatusUnprocessableEntity, err.Error())
		} else {
			ctx.Error(http.StatusInternalServerError, err.Error())
		}
		return
	}
}
