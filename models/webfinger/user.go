// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"github.com/oleiade/gomme"
)

// WebfingerUserActor represents the user and host portions of a WebFinger request.
//
// The expected format follows the [`acct` URI RFC](https://datatracker.ietf.org/doc/rfc7565/):
//
// ```
// resource="acct:@user@host.tld"
// ```
type WebfingerUserActor struct {
	User string
	Host string
}

// ParseWebfingerUserActor parses a [WebfingerUserActor] from a WebFinger `resource` component.
func ParseWebfingerUserActor(input string) (WebfingerUserActor, error) {
	parser := gomme.Preceded(
		gomme.Token[string]("acct:"),
		gomme.Map(
			gomme.Count(ParseWebfingerAccount(), 2),
			func(components []string) (WebfingerUserActor, error) {
				return WebfingerUserActor{components[0], components[1]}, nil
			},
		),
	)

	result := parser(input)
	if result.Err != nil {
		return WebfingerUserActor{}, result.Err
	}

	return result.Output, nil
}
