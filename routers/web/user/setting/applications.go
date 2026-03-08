// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const (
	tplSettingsApplications base.TplName = "user/settings/applications"
)

// Applications render manage access token page
func Applications(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsSettingsApplications"] = true

	loadApplicationsData(ctx)

	ctx.HTML(http.StatusOK, tplSettingsApplications)
}

// DeleteApplication response for delete user access token
func DeleteApplication(ctx *context.Context) {
	if err := auth_model.DeleteAccessTokenByID(ctx, ctx.FormInt64("id"), ctx.Doer.ID); err != nil {
		ctx.Flash.Error("DeleteAccessTokenByID: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("settings.delete_token_success"))
	}

	ctx.JSONRedirect(setting.AppSubURL + "/user/settings/applications")
}

// RegenerateApplication response for regenerating user access token
func RegenerateApplication(ctx *context.Context) {
	if t, err := auth_model.RegenerateAccessTokenByID(ctx, ctx.FormInt64("id"), ctx.Doer.ID); err != nil {
		if auth_model.IsErrAccessTokenNotExist(err) {
			ctx.Flash.Error(ctx.Tr("error.not_found"))
		} else {
			ctx.Flash.Error(ctx.Tr("error.server_internal"))
			log.Error("DeleteAccessTokenByID", err)
		}
	} else {
		ctx.Flash.Success(ctx.Tr("settings.regenerate_token_success"))
		ctx.Flash.Info(t.Token)
	}

	ctx.JSONRedirect(setting.AppSubURL + "/user/settings/applications")
}

type TokenWithResources struct {
	Token        *auth_model.AccessToken
	Repositories []*repo_model.Repository
}

func loadApplicationsData(ctx *context.Context) {
	ctx.Data["AccessTokenScopePublicOnly"] = auth_model.AccessTokenScopePublicOnly
	tokens, err := db.Find[auth_model.AccessToken](ctx, auth_model.ListAccessTokensOptions{UserID: ctx.Doer.ID})
	if err != nil {
		ctx.ServerError("ListAccessTokens", err)
		return
	}

	// Load all the AccessTokenResourceRepo for the tokens that we're returning:
	reposByTokenID, err := repo_model.BulkGetRepositoriesForAccessTokens(ctx, tokens,
		func(repo *repo_model.Repository) (bool, error) {
			// Repos associated with a repo-specific access token should already be visible to the token owner, but it's
			// possible that access has changed, such as a removed collaborator on a repo -- don't provide info on that
			// repo if so.
			permission, err := access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
			if err != nil {
				return false, err
			}
			return permission.HasAccess(), nil
		})
	if err != nil {
		ctx.ServerError("BulkGetRepositoriesForAccessTokens", err)
		return
	}

	tokensWithResources := make([]*TokenWithResources, len(tokens))
	for i := range tokens {
		tokensWithResources[i] = &TokenWithResources{
			Token:        tokens[i],
			Repositories: reposByTokenID[tokens[i].ID],
		}
	}

	ctx.Data["TokensWithResources"] = tokensWithResources
	ctx.Data["EnableOAuth2"] = setting.OAuth2.Enabled
	ctx.Data["IsAdmin"] = ctx.Doer.IsAdmin
	if setting.OAuth2.Enabled {
		ctx.Data["Applications"], err = db.Find[auth_model.OAuth2Application](ctx, auth_model.FindOAuth2ApplicationsOptions{
			OwnerID: ctx.Doer.ID,
		})
		if err != nil {
			ctx.ServerError("GetOAuth2ApplicationsByUserID", err)
			return
		}
		ctx.Data["Grants"], err = auth_model.GetOAuth2GrantsByUserID(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.ServerError("GetOAuth2GrantsByUserID", err)
			return
		}
		ctx.Data["EnableAdditionalGrantScopes"] = setting.OAuth2.EnableAdditionalGrantScopes
	}
}
