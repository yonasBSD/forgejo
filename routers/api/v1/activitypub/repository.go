// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/activitypub"
	api "code.gitea.io/gitea/modules/activitypub2"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"

	apiPerson "code.gitea.io/gitea/modules/activitypub2/activitypub"
	ap "github.com/go-ap/activitypub"
	"github.com/go-openapi/strfmt"
	//f3 "lab.forgefriends.org/friendlyforgeformat/gof3"
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

	link := fmt.Sprintf("%s/api/v1/activitypub/repository-id/%d", strings.TrimSuffix(setting.AppURL, "/"), ctx.Repo.Repository.ID)
	repo := forgefed.RepositoryNew(ap.IRI(link))

	repo.Name = ap.NaturalLanguageValuesNew()
	err := repo.Name.Set("en", ap.Content(ctx.Repo.Repository.Name))
	if err != nil {
		ctx.ServerError("Set Name", err)
		return
	}

	response(ctx, repo)
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
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/Star"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	log.Info("RepositoryInbox: repo %v, %v", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)
	opt := web.GetForm(ctx).(*forgefed.Star)

	err := opt.ValidateStar()

	if err != nil {
		panic(err)
	}

	log.Info("RepositoryInbox: Activity.Source %v", opt.Source)
	log.Info("RepositoryInbox: Activity.Actor %v", opt.Actor)

	// assume actor is: "actor": "https://codeberg.org/api/v1/activitypub/user-id/12345" - NB: This might be actually the ID? Maybe check vocabulary.
	// TODO: validate input in front of parsing.
	// parse actor
	actor, err := activitypub.ParseActorID(opt.Actor.GetID().String())

	// Is the actor IRI well formed?
	if err != nil {
		panic(err)
	}

	// Is the ActorData Struct valid?
	err = actor.Validate()

	if err != nil {
		panic(err)
	}

	log.Info("RepositoryInbox: Actor parsed. %v", actor)

	// get_person_by_rest
	c := api.NewHTTPClientWithConfig(strfmt.Default,
		api.DefaultTransportConfig().WithHost("localhost:3000").WithBasePath("/api/v1/").WithSchemes([]string{"http"}))
	//c := client.Default
	person, err := c.Activitypub.ActivitypubPerson(
		apiPerson.NewActivitypubPersonParams().WithUserID(1), nil)

	log.Info("http client. %v", c)
	log.Info("person. %v", person)

	// create_user_from_person (if not alreaydy present)

	// wait 15 sec.

	ctx.Status(http.StatusNoContent)

}
