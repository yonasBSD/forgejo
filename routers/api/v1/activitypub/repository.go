// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

// ToDo: Fix linting
// ToDo: Maybe do a request for the node info
//			Then maybe save the node info in a DB table	- this could be useful for validation
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
	log.Info("RepositoryInbox: repo: %v", repository)

	activity := web.GetForm(ctx).(*forgefed.Star)
	log.Info("RepositoryInbox: activity:%v", activity)

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

	actorAsLoginId := actorId.AsLoginName() // used as LoginName in newly created user
	log.Info("RepositoryInbox: remoteStargazer: %v", actorAsLoginId)

	// Check if user already exists
	users, err := SearchUsersByLoginName(actorAsLoginId)
	if err != nil {
		ctx.ServerError("Searching for user failed", err)
		return
	}
	log.Info("RepositoryInbox: local found users: %v", len(users))

	switch len(users) {
	case 0:
		{
			user, err = createUserFromAP(ctx, actorId)
			if err != nil {
				ctx.ServerError("Creating user failed", err)
				return
			}
			log.Info("RepositoryInbox: created user from ap: %v", user)
		}
	case 1:
		{
			user = users[0]
			log.Info("RepositoryInbox: found user: %v", user)
		}
	default:
		{
			ctx.Error(http.StatusInternalServerError, "StarRepo",
				fmt.Errorf("found more than one matches for federated users"))
			return
		}
	}

	// execute the activity if the repo was not stared already
	alreadyStared := repo_model.IsStaring(ctx, user.ID, repository.ID)
	if !alreadyStared {
		err = repo_model.StarRepo(ctx, user.ID, repository.ID, true)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "StarRepo", err)
			return
		}
	}

	// wait 5 sec.
	time.Sleep(5 * time.Second)

	ctx.Status(http.StatusNoContent)
}

// TODO: Move this to model.user.search ? or to model.user.externalLoginUser ?
func SearchUsersByLoginName(loginName string) ([]*user_model.User, error) {
	var actionsUser = user_model.NewActionsUser()
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

	return users, nil

}

// ToDo: Maybe use externalLoginUser
func createUserFromAP(ctx *context.APIContext, personId forgefed.PersonId) (*user_model.User, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	var actionsUser = user_model.NewActionsUser()
	client, err := api.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return &user_model.User{}, err
	}

	response, err := client.Get(personId.AsUri())
	if err != nil {
		return &user_model.User{}, err
	}

	// validate response; ToDo: Should we widen the restrictions here?
	if response.StatusCode != 200 {
		err = fmt.Errorf("got non 200 status code for id: %v", personId.Id)
		return &user_model.User{}, err
	}
	log.Info("RepositoryInbox: got status: %v", response.Status)

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return &user_model.User{}, err
	}
	log.Info("RepositoryInbox: got body: %v", string(body))
	person := ap.Person{}
	if strings.Contains(string(body), "user does not exist") {
		err = fmt.Errorf("the requested user id did not exist on the remote server: %v", personId.Id)
	} else {
		err = person.UnmarshalJSON(body)
	}
	if err != nil {
		return &user_model.User{}, err
	}
	log.Info("RepositoryInbox: got person by ap: %v", person)
	email := fmt.Sprintf("%v@%v", uuid.New().String(), personId.Host)
	loginName := personId.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personId.HostSuffix())
	log.Info("RepositoryInbox: person.Name: %v", person.Name)
	fullName := person.Name.String()
	if len(person.Name) == 0 {
		fullName = name
	}
	password, err := pwd_gen.Generate(32, 10, 10, false, true)
	if err != nil {
		return &user_model.User{}, err
	}
	user := &user_model.User{
		LowerName:                    strings.ToLower(person.PreferredUsername.String()),
		Name:                         name,
		FullName:                     fullName,
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
