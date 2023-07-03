// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"net/http"

	"code.gitea.io/gitea/modules/context"
)

const securityTxtContent = `Contact: https://codeberg.org/forgejo/forgejo/src/branch/forgejo/CONTRIBUTING.md
Contact: mailto:security@forgejo.org
Expires: 2025-06-25T00:00:00Z
Policy: https://codeberg.org/forgejo/forgejo/src/branch/forgejo/CONTRIBUTING.md
Preferred-Languages: en
`

// returns /.well-known/security.txt content
// RFC 9116, https://www.rfc-editor.org/rfc/rfc9116
// https://securitytxt.org/
func securityTxt(ctx *context.Context) {
	ctx.PlainText(http.StatusOK, securityTxtContent)
}
