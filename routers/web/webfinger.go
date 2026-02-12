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
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/webfinger"
	"forgejo.org/services/context"
)

// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4

// WebfingerUserActor returns a user entry from the database.
func WebfingerUserActor(ctx *context.Context, appURL *url.URL, userActor *webfinger.UserActor) (*user_model.User, error) {
	if userActor == nil {
		return nil, fmt.Errorf("nil WebfingerUserActor")
	}

	user, host := userActor.User, userActor.Host
	if host != appURL.Host {
		return nil, fmt.Errorf("invalid host, have: %v, expected: %v", host, appURL.Host)
	}

	// Instance actor
	if userActor.IsGhost() {
		jrd := userActor.JRD()
		ctx.JSON(http.StatusOK, &jrd)
		ctx.Resp.Header().Set("Content-Type", "application/jrd+json")

		return nil, nil
	}

	return user_model.GetUserByName(ctx, user)
}

// WebfingerRenderUserActor creates a JRD response for an user entry.
func WebfingerRenderUserActor(ctx *context.Context, u *user_model.User) {
	if u == nil || !user_model.IsUserVisibleToViewer(ctx, u, ctx.Doer) {
		ctx.Error(http.StatusNotFound)
		return
	}

	jrd := webfinger.UserActorFromUser(ctx, u).JRD()
	ctx.JSON(http.StatusOK, &jrd)
	ctx.Resp.Header().Set("Content-Type", "application/jrd+json")
}

// WebfingerRepoActor returns a repository entry from the database.
func WebfingerRepoActor(ctx *context.Context, appURL *url.URL, repoActor *webfinger.RepoActor) (*repo_model.Repository, error) {
	if repoActor == nil {
		return nil, fmt.Errorf("nil WebfingerRepoActor")
	}

	repo, owner, host := repoActor.Repo, repoActor.Owner, repoActor.Host
	if host != appURL.Host {
		ctx.Error(http.StatusBadRequest)
		return nil, fmt.Errorf("invalid host, have: %v, expected: %v", host, appURL.Host)
	}

	return repo_model.GetRepositoryByOwnerAndName(ctx, owner, repo)
}

// WebfingerRenderRepoActor creates a JRD response for a repository entry.
func WebfingerRenderRepoActor(ctx *context.Context, r *repo_model.Repository) {
	if r == nil || !user_model.IsUserVisibleToViewer(ctx, r.Owner, ctx.Doer) {
		ctx.Error(http.StatusNotFound)
		return
	}

	jrd := webfinger.RepoActorFromRepo(ctx, r).JRD()
	ctx.JSON(http.StatusOK, &jrd)
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
		if repo, err := webfinger.ParseRepoActor(resource.String()); err == nil {
			if r, err = WebfingerRepoActor(ctx, appURL, &repo); err != nil {
				break
			}
		} else if userActor, err := webfinger.ParseUserActor(resource.String()); err == nil {
			if u, err = WebfingerUserActor(ctx, appURL, &userActor); u == nil && err == nil {
				return
			}
		} else {
			log.Error("Error getting acct: %s Error: %v", resource.Opaque, err)
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
		WebfingerRenderUserActor(ctx, u)
		return
	} else if r != nil {
		WebfingerRenderRepoActor(ctx, r)
		return
	}

	ctx.Error(http.StatusNotFound)
}
