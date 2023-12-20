// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"fmt"
	"html"
	"io"
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/setting"
)

// SanitizeFlashErrorString will sanitize a flash error string
func SanitizeFlashErrorString(x string) string {
	return strings.ReplaceAll(html.EscapeString(x), "\n", "<br>")
}

// IsExternalURL checks if rawURL points to an external URL like http://example.com
func IsExternalURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	appURL, _ := url.Parse(setting.AppURL)
	if len(parsed.Host) != 0 && strings.Replace(parsed.Host, "www.", "", 1) != strings.Replace(appURL.Host, "www.", "", 1) {
		return true
	}
	return false
}

// Limit number of characters in a string (useful to prevent log injection attacks and overly long log outputs)
// Thanks to https://www.socketloop.com/tutorials/golang-characters-limiter-example
func CharLimiter(s string, limit int) string {

	reader := strings.NewReader(s)

	buff := make([]byte, limit)

	n, _ := io.ReadAtLeast(reader, buff, limit)

	if n != 0 {
		return fmt.Sprint(string(buff), "...")
	} else {
		return s
	}
}
