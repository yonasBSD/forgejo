// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"github.com/oleiade/gomme"
)

// WebfingerRepo represents the repo, owner, and host portions of a WebFinger request.
//
// The expected format follows the [`acct` URI RFC](https://datatracker.ietf.org/doc/rfc7565/):
//
// ```
// resource="acct:@repo@owner@host.tld"
// ```
type WebfingerRepo struct {
	Repo  string
	Owner string
	Host  string
}

// ParseWebfingerRepo parses a [WebfingerRepo] from a WebFinger `resource` component.
func ParseWebfingerRepo(input string) (WebfingerRepo, error) {
	parser := gomme.Preceded(
		gomme.Token[string]("acct:"),
		gomme.Map(
			gomme.Count(ParseWebfingerAccount(), 3),
			func(components []string) (WebfingerRepo, error) {
				return WebfingerRepo{components[0], components[1], components[2]}, nil
			},
		),
	)

	result := parser(input)
	if result.Err != nil {
		return WebfingerRepo{}, result.Err
	}

	return result.Output, nil
}

