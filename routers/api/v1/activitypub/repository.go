// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/forgefed"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"github.com/google/uuid"

	ap "github.com/go-ap/activitypub"
	pwd_gen "github.com/sethvargo/go-password/password"
)

// TODO: remove this global var!
var actionsUser = user_model.NewActionsUser()

// TODO: Move this to model.user.search ? or to model.user.externalLoginUser ?
func SearchUsersByLoginName(loginName string) ([]*user_model.User, error) {

	actionsUser.IsAdmin = true

	options := &user_model.SearchUserOptions{
		LoginName:       loginName,
		Actor:           actionsUser,
		Type:            user_model.UserTypeRemoteUser,
		OrderBy:         db.SearchOrderByAlphabetically,
		ListOptions:     db.ListOptions{PageSize: 1},
		IsActive:        util.OptionalBoolFalse,
		IncludeReserved: true,
	}
	users, _, err := user_model.SearchUsers(db.DefaultContext, options)
	if err != nil {
		return []*user_model.User{}, fmt.Errorf("search failed: %v", err)
	}

	log.Info("local found users: %v", len(users))

	return users, nil

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

	var user *user_model.User

	repository := ctx.Repo.Repository
	log.Info("RepositoryInbox: repo: %v, owned by:  %v", repository.Name, repository.OwnerName)

	activity := web.GetForm(ctx).(*forgefed.Star)
	log.Info("RepositoryInbox: Activity.Source: %v, Activity.Actor %v, Activity.Actor.Id %v", activity.Source, activity.Actor, activity.Actor.GetID().String())

	// parse actorId (person)
	actorId, err := forgefed.NewPersonId(activity.Actor.GetID().String(), string(activity.Source))
	if err != nil {
		ctx.ServerError("Validate actorId", err)
		return
	}
	log.Info("RepositoryInbox: actorId validated: %v", actorId)
	// parse objectId (repository)
	objectId, err := forgefed.NewRepositoryId(activity.Object.GetID().String(), string(activity.Source))
	if err != nil {
		ctx.ServerError("Validate objectId", err)
		return
	}
	if objectId.Id != fmt.Sprint(repository.ID) {
		ctx.ServerError("Validate objectId", err)
		return
	}
	log.Info("RepositoryInbox: objectId validated: %v", objectId)

	adctorAsWebfinger := actorId.AsWebfinger() // used as LoginName in newly created user
	log.Info("remotStargazer: %v", adctorAsWebfinger)

	// Check if user already exists
	users, err := SearchUsersByLoginName(adctorAsWebfinger)
	if err != nil {
		ctx.ServerError(fmt.Sprintf("Searching for user failed: %v"), err)
		return
	}

	switch len(users) {
	case 0:
		{
			user, err := createUserFromAP(ctx, actorId)
			if err != nil {
				ctx.ServerError(fmt.Sprintf("Searching for user failed: %v"), err)
				return
			}
		}
	case 1:
		{
			user = users[0]
			log.Info("Found user full name was: %v", user.FullName)
			log.Info("Found user name was: %v", user.Name)
			log.Info("Found user loginname was: %v", user.LoginName)
			log.Info("%v", user)
		}
	default:
		{
			panic(fmt.Errorf("found more than one matches for federated users"))
		}
	}

	// TODO: why should we search user for a second time from db?
	remoteUser, err := user_model.GetUserByEmail(ctx, user.Email)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "StarRepo", err)
		return
	}

	// check if already starred by this user
	alreadyStared := repo_model.IsStaring(ctx, remoteUser.ID, repository.ID)
	switch alreadyStared {
	case true: // execute unstar action
		{
			err = repo_model.StarRepo(ctx, remoteUser.ID, repository.ID, false)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, "StarRepo", err)
				return
			}
		}
	case false: // execute star action
		{
			err = repo_model.StarRepo(ctx, remoteUser.ID, repository.ID, true)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, "StarRepo", err)
				return
			}
		}
	}

	// wait 5 sec.
	time.Sleep(5 * time.Second)

	ctx.Status(http.StatusNoContent)
}

func createUserFromAP(ctx *context.APIContext, actorId forgefed.PersonId) (*user_model.User, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	client, err := api.NewClient(ctx, actionsUser, "no ide where to get key material.")
	if err != nil {
		return &user_model.User{}, err
	}
	response, err := client.Get(actorId.AsUri())
	if err != nil {
		return &user_model.User{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return &user_model.User{}, err
	}
	person := ap.Person{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return &user_model.User{}, err
	}
	email := fmt.Sprintf("%v@%v", uuid.New().String(), actorId.Host)
	loginName := actorId.AsWebfinger()
	password, err := pwd_gen.Generate(32, 10, 10, false, true)
	if err != nil {
		return &user_model.User{}, err
	}
	user := &user_model.User{
		LowerName:                    strings.ToLower(person.Name.String()),
		Name:                         person.Name.String(),
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    loginName,
		Type:                         user_model.UserTypeRemoteUser,
		IsAdmin:                      false,
	}
	overwrite := &user_model.CreateUserOverwriteOptions{
		IsActive:     util.OptionalBoolFalse,
		IsRestricted: util.OptionalBoolFalse,
	}
	if err := user_model.CreateUser(ctx, user, overwrite); err != nil {
		return &user_model.User{}, err
	}

	return user, nil
}
