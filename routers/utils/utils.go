// Copyright 2024 The Forgejo Authors. All rights reserved.
// Copyright 2017, 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"html"
	"strings"
)

// SanitizeFlashErrorString will sanitize a flash error string
func SanitizeFlashErrorString(x string) string {
	return strings.ReplaceAll(html.EscapeString(x), "\n", "<br>")
}
