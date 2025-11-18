// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_WebfingerUserActor(t *testing.T) {
	sut := WebfingerUserActor{
		User: "user",
		Host: "host.tld",
	}

	userActor, err := ParseWebfingerUserActor("acct:@user@host.tld")
	require.NoError(t, err)

	require.Equal(t, userActor, sut)

	sut = WebfingerUserActor{
		User: "user_with_pct-encode%20",
		Host: "host.tld",
	}

	userActor, err = ParseWebfingerUserActor("acct:@user_with_pct-encode%20@host.tld")
	require.NoError(t, err)

	require.Equal(t, userActor, sut)

	sut = WebfingerUserActor{
		User: "user_with_pct-encode%20",
		Host: "host.tld_sub-delim;",
	}

	userActor, err = ParseWebfingerUserActor("acct:@user_with_pct-encode%20@host.tld_sub-delim;")
	require.NoError(t, err)

	require.Equal(t, userActor, sut)
}
