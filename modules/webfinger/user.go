// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"forgejo.org/models/user"
	"forgejo.org/modules/setting"

	"github.com/oleiade/gomme"
)

// UserActor represents the user and host portions of a WebFinger request.
//
// The expected format follows the [`acct` URI RFC](https://datatracker.ietf.org/doc/rfc7565/):
//
// ```
// resource="acct:@user@host.tld"
// ```
//
// The leading `@` symbol is optional, and would be percent-encoded if present (any `@` symbol is potentially percent-encoded).
// Symbols are displayed in their decoded form for clarity.
type UserActor struct {
	User             string
	Host             string
	ID               int64
	Email            string
	AvatarLink       string
	KeepEmailPrivate bool
}

// ParseUserActor parses a [UserActor] from a WebFinger `resource` component.
func ParseUserActor(input string) (UserActor, error) {
	parser := gomme.Preceded(
		gomme.Token[string]("acct:"),
		gomme.Map(
			gomme.Count(ParseAcctPart(), 2),
			func(components []string) (UserActor, error) {
				return UserActor{User: components[0], Host: components[1]}, nil
			},
		),
	)

	result := parser(input)
	if result.Err != nil {
		return UserActor{}, result.Err
	}

	return result.Output, nil
}

// IsGhost returns if the [UserActor] is a "ghost" user.
func (w UserActor) IsGhost() bool {
	return w.User == "ghost"
}

// HTMLURL constructs the HTML URL for the [UserActor].
func (w UserActor) HTMLURL() string {
	if w.IsGhost() {
		return ""
	}
	return setting.AppURL + url.PathEscape(w.User)
}

// UserActorFromUser constructs a [UserActor] from a [User] database entry.
func UserActorFromUser(ctx context.Context, user *user.User) UserActor {
	var w UserActor

	if user != nil {
		w = UserActor{
			User:             user.Name,
			ID:               user.ID,
			AvatarLink:       user.AvatarLink(ctx),
			Email:            user.Email,
			KeepEmailPrivate: user.KeepEmailPrivate,
		}
	}

	return w
}

// JRD construct a [JRD] from the [UserActor].
func (w UserActor) JRD() JRD {
	var jrd JRD

	appURL, err := url.Parse(setting.AppURL)
	if err != nil {
		return jrd
	}

	if w.IsGhost() {
		aliases := []string{
			appURL.String() + "api/v1/activitypub/actor",
		}

		links := []*Link{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: appURL.String() + "api/v1/activitypub/actor",
			},
		}
		jrd = JRD{
			Subject: fmt.Sprintf("acct:%s@%s", "ghost", w.Host),
			Aliases: aliases,
			Links:   links,
		}
	} else {
		aliases := []string{
			w.HTMLURL(),
			appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(w.ID),
		}
		if !w.KeepEmailPrivate {
			aliases = append(aliases, fmt.Sprintf("mailto:%s", w.Email))
		}

		links := []*Link{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: w.HTMLURL(),
			},
			{
				Rel:  "http://webfinger.net/rel/avatar",
				Href: w.AvatarLink,
			},
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(w.ID),
			},
			{
				Rel:  "http://openid.net/specs/connect/1.0/issuer",
				Href: strings.TrimSuffix(appURL.String(), "/"),
			},
		}
		jrd = JRD{
			Subject: fmt.Sprintf("acct:%s@%s", url.QueryEscape(w.User), appURL.Host),
			Aliases: aliases,
			Links:   links,
		}
	}

	return jrd
}
