// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_WebfingerRepo(t *testing.T) {
	sut := WebfingerRepo{
		Repo: "project",
		Owner: "user",
		Host: "host.tld",
	}

	repo, err := ParseWebfingerRepo("acct:@project@user@host.tld")
	require.NoError(t, err)

	require.Equal(t, repo, sut)

	sut = WebfingerRepo{
		Repo: "project_with_pct-encode%20",
		Owner: "user_with_pct-encode%20",
		Host: "host.tld",
	}

	repo, err = ParseWebfingerRepo("acct:@project_with_pct-encode%20@user_with_pct-encode%20@host.tld")
	require.NoError(t, err)

	require.Equal(t, repo, sut)

	sut = WebfingerRepo{
		Repo: "project",
		Owner: "user_with_pct-encode%20",
		Host: "host.tld_sub-delim;",
	}

	repo, err = ParseWebfingerRepo("acct:@project@user_with_pct-encode%20@host.tld_sub-delim;")
	require.NoError(t, err)

	require.Equal(t, repo, sut)
}
