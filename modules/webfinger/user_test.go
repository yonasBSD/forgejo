// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger_test

import (
	"testing"

	"forgejo.org/modules/webfinger"

	"github.com/stretchr/testify/require"
)

func Test_WebfingerUserActor(t *testing.T) {
	sut := webfinger.UserActor{
		User: "user",
		Host: "host.tld",
	}

	userActor, err := webfinger.ParseUserActor("acct:@user@host.tld")
	require.NoError(t, err)

	require.Equal(t, userActor, sut)

	sut = webfinger.UserActor{
		User: "user_with_pct-encode%20",
		Host: "host.tld",
	}

	userActor, err = webfinger.ParseUserActor("acct:@user_with_pct-encode%20@host.tld")
	require.NoError(t, err)

	require.Equal(t, userActor, sut)

	sut = webfinger.UserActor{
		User: "user_with_pct-encode%20",
		Host: "host.tld_sub-delim;",
	}

	userActor, err = webfinger.ParseUserActor("acct:@user_with_pct-encode%20@host.tld_sub-delim;")
	require.NoError(t, err)

	require.Equal(t, userActor, sut)
}
