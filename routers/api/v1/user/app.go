// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package user

import (
	"net/http"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/convert"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/routers/api/v1/utils"
)

// ListAccessTokens list all the access tokens
func ListAccessTokens(ctx *context.APIContext) {
	// swagger:operation GET /users/{username}/tokens user userGetTokens
	// ---
	// summary: List the authenticated user's access tokens
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results, maximum page size is 50
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/AccessTokenList"

	tokens, err := models.ListAccessTokens(ctx.User.ID, utils.GetListOptions(ctx))
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListAccessTokens", err)
		return
	}

	apiTokens := make([]*api.AccessToken, len(tokens))
	for i := range tokens {
		apiTokens[i] = &api.AccessToken{
			ID:             tokens[i].ID,
			Name:           tokens[i].Name,
			TokenLastEight: tokens[i].TokenLastEight,
		}
	}
	ctx.JSON(http.StatusOK, &apiTokens)
}

// CreateAccessToken create access tokens
func CreateAccessToken(ctx *context.APIContext, form api.CreateAccessTokenOption) {
	// swagger:operation POST /users/{username}/tokens user userCreateToken
	// ---
	// summary: Create an access token
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user
	//   type: string
	//   required: true
	// - name: accessToken
	//   in: body
	//   schema:
	//     type: object
	//     required:
	//       - name
	//     properties:
	//       name:
	//         type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/AccessToken"

	t := &models.AccessToken{
		UID:  ctx.User.ID,
		Name: form.Name,
	}
	if err := models.NewAccessToken(t); err != nil {
		ctx.Error(http.StatusInternalServerError, "NewAccessToken", err)
		return
	}
	ctx.JSON(http.StatusCreated, &api.AccessToken{
		Name:           t.Name,
		Token:          t.Token,
		ID:             t.ID,
		TokenLastEight: t.TokenLastEight,
	})
}

// DeleteAccessToken delete access tokens
func DeleteAccessToken(ctx *context.APIContext) {
	// swagger:operation DELETE /users/{username}/tokens/{token} user userDeleteAccessToken
	// ---
	// summary: delete an access token
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user
	//   type: string
	//   required: true
	// - name: token
	//   in: path
	//   description: token to be deleted
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	tokenID := ctx.ParamsInt64(":id")
	if err := models.DeleteAccessTokenByID(tokenID, ctx.User.ID); err != nil {
		if models.IsErrAccessTokenNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteAccessTokenByID", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CreateOauth2Application is the handler to create a new OAuth2 Application for the authenticated user
func CreateOauth2Application(ctx *context.APIContext, data api.CreateOAuth2ApplicationOptions) {
	// swagger:operation POST /user/applications/oauth2 user userCreateOAuth2Application
	// ---
	// summary: creates a new OAuth2 application
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/CreateOAuth2ApplicationOptions"
	// responses:
	//   "201":
	//     "$ref": "#/responses/OAuth2Application"
	app, err := models.CreateOAuth2Application(models.CreateOAuth2ApplicationOptions{
		Name:         data.Name,
		UserID:       ctx.User.ID,
		RedirectURIs: data.RedirectURIs,
	})
	if err != nil {
		ctx.Error(http.StatusBadRequest, "", "error creating oauth2 application")
		return
	}
	secret, err := app.GenerateClientSecret()
	if err != nil {
		ctx.Error(http.StatusBadRequest, "", "error creating application secret")
		return
	}
	app.ClientSecret = secret

	ctx.JSON(http.StatusCreated, convert.ToOAuth2Application(app))
}

// ListOauth2Applications list all the Oauth2 application
func ListOauth2Applications(ctx *context.APIContext) {
	// swagger:operation GET /user/applications/oauth2 user userGetOauth2Application
	// ---
	// summary: List the authenticated user's oauth2 applications
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results, maximum page size is 50
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/OAuth2ApplicationList"

	apps, err := models.ListOAuth2Applications(ctx.User.ID, utils.GetListOptions(ctx))
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListOAuth2Applications", err)
		return
	}

	apiApps := make([]*api.OAuth2Application, len(apps))
	for i := range apps {
		apiApps[i] = convert.ToOAuth2Application(apps[i])
		apiApps[i].ClientSecret = "" // Hide secret on application list
	}
	ctx.JSON(http.StatusOK, &apiApps)
}

// DeleteOauth2Application delete OAuth2 Application
func DeleteOauth2Application(ctx *context.APIContext) {
	// swagger:operation DELETE /user/applications/oauth2/{id} user userDeleteOAuth2Application
	// ---
	// summary: delete an OAuth2 Application
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: token to be deleted
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	appID := ctx.ParamsInt64(":id")
	if err := models.DeleteOAuth2Application(appID, ctx.User.ID); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteOauth2ApplicationByID", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
