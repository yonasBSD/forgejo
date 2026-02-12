// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"forgejo.org/models/repo"
	"forgejo.org/modules/setting"

	"github.com/oleiade/gomme"
)

// RepoActor represents the repo, owner, and host portions of a WebFinger request.
//
// The expected format follows the [`acct` URI RFC](https://datatracker.ietf.org/doc/rfc7565/):
//
// ```
// resource="acct:@repo@owner@host.tld"
// ```
//
// The leading `@` symbol is optional, and would be percent-encoded if present (any `@` symbol is potentially percent-encoded).
// Symbols are displayed in their decoded form for clarity.
type RepoActor struct {
	Repo             string
	Owner            string
	Host             string
	ID               int64
	Email            string
	KeepEmailPrivate bool
	HTMLURL          string
	AvatarLink       string
}

// ParseWebfingerRepo parses a [RepoActor] from a WebFinger `resource` component.
func ParseRepoActor(input string) (RepoActor, error) {
	parser := gomme.Preceded(
		gomme.Token[string]("acct:"),
		gomme.Map(
			gomme.Count(ParseAcctPart(), 3),
			func(components []string) (RepoActor, error) {
				return RepoActor{
					Repo:  components[0],
					Owner: components[1],
					Host:  components[2],
				}, nil
			},
		),
	)

	result := parser(input)
	if result.Err != nil {
		return RepoActor{}, result.Err
	}

	return result.Output, nil
}

// RepoActorFromUser constructs a [RepoActor] from a [Repo] database entry.
func RepoActorFromRepo(ctx context.Context, r *repo.Repository) RepoActor {
	if r == nil {
		return RepoActor{}
	}

	w := RepoActor{
		Repo: r.Name,
		Host: setting.AppURL,
		ID:   r.ID,
	}

	if r.Owner != nil {
		w.Email = r.Owner.Email
		w.KeepEmailPrivate = r.Owner.KeepEmailPrivate
		w.AvatarLink = r.Owner.AvatarLink(ctx)
	}

	return w
}

// JRD construct a [JRD] from the [RepoActor].
func (w RepoActor) JRD() JRD {
	appURL, err := url.Parse(setting.AppURL)
	if err != nil {
		return JRD{}
	}

	aliases := []string{
		appURL.String() + "api/v1/activitypub/repo-id/" + fmt.Sprint(w.ID),
	}

	if len(w.Email) != 0 && !w.KeepEmailPrivate {
		aliases = append(aliases, fmt.Sprintf("mailto:%s", w.Email))
	}

	links := []*Link{
		{
			Rel:  "self",
			Type: "application/activity+json",
			Href: appURL.String() + "api/v1/activitypub/repo-id/" + fmt.Sprint(w.ID),
		},
		{
			Rel:  "http://openid.net/specs/connect/1.0/issuer",
			Href: strings.TrimSuffix(appURL.String(), "/"),
		},
	}

	if len(w.HTMLURL) != 0 {
		links = append(links, &Link{
			Rel:  "http://webfinger.net/rel/profile-page",
			Type: "text/html",
			Href: w.HTMLURL,
		})
	}

	if len(w.AvatarLink) != 0 {
		links = append(links, &Link{
			Rel:  "http://webfinger.net/rel/avatar",
			Href: w.AvatarLink,
		})
	}

	return JRD{
		Subject: fmt.Sprintf("acct:%s@%s@%s", url.QueryEscape(w.Repo), url.QueryEscape(w.Owner), appURL.Host),
		Aliases: aliases,
		Links:   links,
	}
}
