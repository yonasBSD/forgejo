// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/models/webfinger"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4

// WebfingerUserActor returns a user entry from the database.
func WebfingerUserActor(ctx *context.Context, appURL *url.URL, parts []string) (*user_model.User, error) {
	user, host := parts[0], parts[1]
	if host != appURL.Host {
		ctx.Error(http.StatusBadRequest)
		return nil, fmt.Errorf("invalid host, have: %v, expected: %v", host, appURL.Host)
	}

	// Instance actor
	if user == "ghost" {
		aliases := []string{
			appURL.String() + "api/v1/activitypub/actor",
		}

		links := []*webfinger.Link{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: appURL.String() + "api/v1/activitypub/actor",
			},
		}

		ctx.Resp.Header().Add("Access-Control-Allow-Origin", "*")
		ctx.JSON(http.StatusOK, &webfinger.JRD{
			Subject: fmt.Sprintf("acct:%s@%s", "ghost", appURL.Host),
			Aliases: aliases,
			Links:   links,
		})
		ctx.Resp.Header().Set("Content-Type", "application/jrd+json")

		return nil, nil
	}

	return user_model.GetUserByName(ctx, user)
}

// WebfingerRenderUserActor creates a JRD response for an user entry.
func WebfingerRenderUserActor(ctx *context.Context, appURL *url.URL, u *user_model.User) {
	if !user_model.IsUserVisibleToViewer(ctx, u, ctx.Doer) {
		ctx.Error(http.StatusNotFound)
		return
	}

	aliases := []string{
		u.HTMLURL(),
		appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(u.ID),
	}
	if !u.KeepEmailPrivate {
		aliases = append(aliases, fmt.Sprintf("mailto:%s", u.Email))
	}

	links := []*webfinger.Link{
		{
			Rel:  "http://webfinger.net/rel/profile-page",
			Type: "text/html",
			Href: u.HTMLURL(),
		},
		{
			Rel:  "http://webfinger.net/rel/avatar",
			Href: u.AvatarLink(ctx),
		},
		{
			Rel:  "self",
			Type: "application/activity+json",
			Href: appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(u.ID),
		},
		{
			Rel:  "http://openid.net/specs/connect/1.0/issuer",
			Href: strings.TrimSuffix(appURL.String(), "/"),
		},
	}

	ctx.Resp.Header().Add("Access-Control-Allow-Origin", "*")
	ctx.JSON(http.StatusOK, &webfinger.JRD{
		Subject: fmt.Sprintf("acct:%s@%s", url.QueryEscape(u.Name), appURL.Host),
		Aliases: aliases,
		Links:   links,
	})
	ctx.Resp.Header().Set("Content-Type", "application/jrd+json")
}

// WebfingerRepoActor returns a repository entry from the database.
func WebfingerRepoActor(ctx *context.Context, appURL *url.URL, parts []string) (*repo_model.Repository, error) {
	repo, owner, host := parts[0], parts[1], parts[2]
	if host != appURL.Host {
		ctx.Error(http.StatusBadRequest)
		return nil, fmt.Errorf("invalid host, have: %v, expected: %v", host, appURL.Host)
	}

	return repo_model.GetRepositoryByOwnerAndName(ctx, owner, repo)
}

// WebfingerRenderRepoActor creates a JRD response for a repository entry.
func WebfingerRenderRepoActor(ctx *context.Context, appURL *url.URL, r *repo_model.Repository) {
	if !user_model.IsUserVisibleToViewer(ctx, r.Owner, ctx.Doer) {
		ctx.Error(http.StatusNotFound)
		return
	}
	aliases := []string{
		appURL.String() + "api/v1/activitypub/repo-id/" + fmt.Sprint(r.ID),
	}
	if r.Owner != nil && !r.Owner.KeepEmailPrivate {
		aliases = append(aliases, fmt.Sprintf("mailto:%s", r.Owner.Email))
	}

	links := []*webfinger.Link{
		{
			Rel:  "self",
			Type: "application/activity+json",
			Href: appURL.String() + "api/v1/activitypub/repo-id/" + fmt.Sprint(r.ID),
		},
		{
			Rel:  "http://openid.net/specs/connect/1.0/issuer",
			Href: strings.TrimSuffix(appURL.String(), "/"),
		},
	}
	if r.Owner != nil {
		links = append(links, []*webfinger.Link{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: r.Owner.HTMLURL(),
			},
			{
				Rel:  "http://webfinger.net/rel/avatar",
				Href: r.Owner.AvatarLink(ctx),
			},
		}...)
	}

	ctx.Resp.Header().Add("Access-Control-Allow-Origin", "*")
	ctx.JSON(http.StatusOK, &webfinger.JRD{
		Subject: fmt.Sprintf("acct:%s@%s", url.QueryEscape(r.OwnerName), appURL.Host),
		Aliases: aliases,
		Links:   links,
	})
	ctx.Resp.Header().Set("Content-Type", "application/jrd+json")
}

// WebfingerQuery returns information about a resource
// https://datatracker.ietf.org/doc/html/rfc7565
func WebfingerQuery(ctx *context.Context) {
	appURL, _ := url.Parse(setting.AppURL)

	resource, err := url.Parse(ctx.FormTrim("resource"))
	if err != nil {
		ctx.Error(http.StatusBadRequest)
		return
	}

	var u *user_model.User
	var r *repo_model.Repository

	switch resource.Scheme {
	case "acct":
		// allow only the current host
		parts := strings.SplitN(resource.Opaque, "@", 4)

		switch len(parts) {
		case 2:
			u, err = WebfingerUserActor(ctx, appURL, parts)
			if u == nil && err == nil {
				return
			}
		case 4:
			r, err = WebfingerRepoActor(ctx, appURL, parts[1:])

		default:
			ctx.Error(http.StatusBadRequest)
			return
		}
	case "mailto":
		u, err = user_model.GetUserByEmail(ctx, resource.Opaque)
		if u != nil && u.KeepEmailPrivate {
			err = user_model.ErrUserNotExist{}
		}
	case "https", "http":
		if resource.Host != appURL.Host {
			ctx.Error(http.StatusBadRequest)
			return
		}

		p := strings.Trim(resource.Path, "/")
		if len(p) == 0 {
			ctx.Error(http.StatusNotFound)
			return
		}

		parts := strings.Split(p, "/")

		switch len(parts) {
		case 1: // user
			u, err = user_model.GetUserByName(ctx, parts[0])
		case 2: // repository
			ctx.Error(http.StatusNotFound)
			return

		case 3:
			switch parts[2] {
			case "issues":
				ctx.Error(http.StatusNotFound)
				return

			case "pulls":
				ctx.Error(http.StatusNotFound)
				return

			case "projects":
				ctx.Error(http.StatusNotFound)
				return

			default:
				ctx.Error(http.StatusNotFound)
				return
			}

		default:
			ctx.Error(http.StatusNotFound)
			return
		}

	default:
		ctx.Error(http.StatusBadRequest)
		return
	}
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusNotFound)
		} else {
			log.Error("Error getting user: %s Error: %v", resource.Opaque, err)
			ctx.Error(http.StatusInternalServerError)
		}
		return
	} else if u != nil {
		WebfingerRenderUserActor(ctx, appURL, u)
		return
	} else if r != nil {
		WebfingerRenderRepoActor(ctx, appURL, r)
		return
	}

	ctx.Error(http.StatusNotFound)
}
