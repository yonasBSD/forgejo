// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger_test

import (
	"testing"

	"forgejo.org/modules/webfinger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WebfingerRepo(t *testing.T) {
	sut := webfinger.RepoActor{
		Repo:  "project",
		Owner: "user",
		Host:  "host.tld",
	}

	repo, err := webfinger.ParseRepoActor("acct:@project@user@host.tld")
	require.NoError(t, err)

	assert.Equal(t, repo, sut)

	sut = webfinger.RepoActor{
		Repo:  "project_with_pct-encode%20",
		Owner: "user_with_pct-encode%20",
		Host:  "host.tld",
	}

	repo, err = webfinger.ParseRepoActor("acct:@project_with_pct-encode%20@user_with_pct-encode%20@host.tld")
	require.NoError(t, err)

	assert.Equal(t, repo, sut)

	sut = webfinger.RepoActor{
		Repo:  "project",
		Owner: "user_with_pct-encode%20",
		Host:  "host.tld_sub-delim;",
	}

	repo, err = webfinger.ParseRepoActor("acct:@project@user_with_pct-encode%20@host.tld_sub-delim;")
	require.NoError(t, err)

	assert.Equal(t, repo, sut)
}
