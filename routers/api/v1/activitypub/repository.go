// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"net/http"
	"strings"

	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	ap "github.com/go-ap/activitypub"
	//f3 "lab.forgefriends.org/friendlyforgeformat/gof3"
	"github.com/go-ap/jsonld"
)

// Repository function returns the Repository actor for a repo
func Repository(ctx *context.APIContext) {
	// swagger:operation GET /activitypub/repository-id/{repository-id} activitypub activitypubRepository
	// ---
	// summary: Returns the Repository actor for a repo
	// produces:
	// - application/json
	// parameters:
	// - name: repository-id
	//   in: path
	//   description: repository ID of the repo
	//   type: integer
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActivityPub"

	// TODO: Mabe we should use F3 Repo instead?
	link := fmt.Sprintf("%s/api/v1/activitypub/repoistory-id/%d", strings.TrimSuffix(setting.AppURL, "/"), ctx.Repo.Repository.ID)
	repository := ap.ApplicationNew(ap.IRI(link))
	repository.Name.Set("en", ap.Content(ctx.Repo.Repository.Name))

	binary, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI), jsonld.IRI(ap.SecurityContextURI)).Marshal(repository)
	if err != nil {
		ctx.ServerError("MarshalJSON", err)
		return
	}
	ctx.Resp.Header().Add("Content-Type", activitypub.ActivityStreamsContentType)
	ctx.Resp.WriteHeader(http.StatusOK)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("write to resp err: %v", err)
	}
}

// PersonInbox function handles the incoming data for a repository inbox
func RepositoryInbox(ctx *context.APIContext) {
	// swagger:operation POST /activitypub/repository-id/{repository-id}/inbox activitypub activitypubRepository
	// ---
	// summary: Send to the inbox
	// produces:
	// - application/json
	// parameters:
	// - name: repository-id
	//   in: path
	//   description: repository ID of the repo
	//   type: integer
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	log.Info("RepositoryInbox: %v, %v", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)

	ctx.Status(http.StatusNoContent)
}
