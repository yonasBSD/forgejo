// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"fmt"
	"strings"

	"github.com/oleiade/gomme"
)

// joinStrings is a helper function to join an array of strings with no delimiter.
func joinStrings(s []string) (string, error) {
	return strings.Join(s, ""), nil
}

// runeToString is a helper function to convert a rune to a string.
func runeToString(r rune) (string, error) {
	return string(r), nil
}

// ParseWebfingerAccount parses a WebFinger `resource` component using the `acct` format for ActivityPub accounts.
func ParseWebfingerAccount() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Preceded(
			gomme.Optional(gomme.Token[string]("@")),
			gomme.Map(
				gomme.Many1(gomme.Alternative(
					URIUnreserved(),
					URIPercentEncoded(),
					URISubDelims(),
				)),
				joinStrings,
			),
		)(input)
	}
}

// URIUnreserved returns a gomme parser for
// [URI unreserved](https://datatracker.ietf.org/doc/rfc3986/) characters.
func URIUnreserved() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Alternative(
			gomme.Alphanumeric1[string](),
			gomme.Map(
				gomme.OneOf[string]('-', '.', '_', '~', ':'),
				runeToString,
			),
		)(input)
	}
}

// URIPercentEncoded returns a gomme parser for
// [URI pct-encoded](https://datatracker.ietf.org/doc/rfc3986/) characters.
func URIPercentEncoded() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map[string, []string, string](
			gomme.Preceded(
				gomme.Token[string]("%"),
				gomme.Count(gomme.HexDigit0[string](), 2),
			),
			func(hex []string) (string, error) {
				return fmt.Sprintf("%%%s%s", hex[0], hex[1]), nil
			},
		)(input)
	}
}

// URISubDelims returns a gomme parser for
// [URI sub-delims](https://datatracker.ietf.org/doc/rfc3986/) characters.
func URISubDelims() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map[string, rune, string](
			gomme.OneOf[string]('!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '='),
			runeToString,
		)(input)
	}
}
