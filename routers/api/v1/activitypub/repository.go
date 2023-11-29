// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/activitypub"
	api "code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"

	ap "github.com/go-ap/activitypub"
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

	log.Info("RepositoryInbox: Activity.Source %v", opt.Source)
	log.Info("RepositoryInbox: Activity.Actor %v", opt.Actor)

	// assume actor is: "actor": "https://codeberg.org/api/v1/activitypub/user-id/12345" - NB: This might be actually the ID? Maybe check vocabulary.
	// parse actor
	actor, err := activitypub.ParseActorIDFromStarActivity(opt)

	// Is the actor IRI well formed?
	if err != nil {
		panic(err)
	}

	// Is the ActorData Struct valid?
	actor.PanicIfInvalid()

	log.Info("RepositoryInbox: Actor parsed. %v", actor)

	/*
		Make http client, this should make a get request on given url
		We then need to parse the answer and put it into a person-struct
		fill the person struct using some kind of unmarshall function given in
		activitypub package/actor.go
	*/

	// make http client
	host := opt.To.GetID().String()
	client, err := api.NewClient(ctx, ctx.Doer, host) // ToDo: This is hacky, we need a hostname from somewhere
	if err != nil {
		panic(err)
	}

	// get_person_by_rest
	bytes := []byte{0}                   // no body needed for getting user actor
	target := opt.Actor.GetID().String() // target is the person actor that originally performed the star activity
	response, err := client.Get(bytes, target)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	// parse response
	person, err := forgefed.ParsePersonJson(body)
	if err != nil {
		panic(err)
	}

	log.Info("target: %v", target)
	log.Info("http client. %v", client)
	log.Info("response: %v\n error: ", response, err)
	log.Info("Person is: %v", person)

	// create_user_from_person (if not alreaydy present)

	// wait 15 sec.

	ctx.Status(http.StatusNoContent)

}
