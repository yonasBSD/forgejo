// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package httplib

import (
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestIsRiskyRedirectURL(t *testing.T) {
	defer test.MockVariableValue(&setting.AppURL, "http://localhost:3000/sub/")()
	defer test.MockVariableValue(&setting.AppSubURL, "/sub")()

	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"foo", false},
		{"./", false},
		{"?key=val", false},
		{"/sub/", false},
		{"http://localhost:3000/sub/", false},
		{"/sub/foo", false},
		{"http://localhost:3000/sub/foo", false},
		// should probably be true (would requires resolving references using setting.appURL.ResolveReference(u))
		{"/sub/../", false},
		{"http://localhost:3000/sub/../", false},
		{"/sUb/", false},
		{"http://localhost:3000/sUb/foo", false},
		{"/sub", false},
		{"/foo?k=%20#abc", false},
		{"/", false},

		{"//", true},
		{"\\\\", true},
		{"/\\", true},
		{"\\/", true},
		{"mail:a@b.com", true},
		{"https://test.com", true},
		{"http://localhost:3000/foo", true},
		{"http://localhost:3000/sub", true},
		{"http://localhost:3000/sub?key=val", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRiskyRedirectURL(tt.input))
		})
	}
}
