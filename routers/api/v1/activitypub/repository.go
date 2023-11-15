// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"

	ap "github.com/go-ap/activitypub"
	//f3 "lab.forgefriends.org/friendlyforgeformat/gof3"
)

type (
	Schema string
	UserID string
	Host   string
	Port   string
)

type ActorData struct {
	schema string
	userId string
	host   string
	port   string // optional
}

func parseActor(actor string) (ActorData, error) {
	u, err := url.Parse(actor)

	// check if userID IRI is well formed url
	if err != nil {
		return ActorData{}, fmt.Errorf("the actor ID was not valid: %v", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return ActorData{}, fmt.Errorf("the actor ID was not valid: Invalid Schema or Host")
	}

	if !strings.Contains(u.Path, "api/v1/activitypub/user-id") {
		return ActorData{}, fmt.Errorf("the Path to the API was invalid: %v\n the full URL is: %v", u.Path, actor)
	}

	pathWithUserID := strings.Split(u.Path, "/")
	userId := pathWithUserID[len(pathWithUserID)-1]

	return ActorData{
		schema: u.Scheme,
		userId: userId,
		host:   u.Host,
		port:   u.Port(),
	}, nil
}

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
	actor, err := parseActor(opt.Actor.GetID().String())

	// if not actor.isValid() then exit_with_error
	if err != nil {
		panic(err)
	}

	log.Info("RepositoryInbox: Actor parsed. %v", actor)

	// get_person_by_rest
	// create_user_from_person (if not alreaydy present)

	// wait 15 sec.

	ctx.Status(http.StatusNoContent)

}
