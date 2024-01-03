// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

// ToDo: Fix linting
// ToDo: Maybe do a request for the node info
//			Then maybe save the node info in a DB table	- this could be useful for validation
import (
	"fmt"
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

	// parse actorID (person)
	actorUri := activity.Actor.GetID().String()
	rawActorID, err := forgefed.NewActorID(actorUri)
	nodeInfo, err := createNodeInfo(ctx, rawActorID)
	log.Info("RepositoryInbox: nodeInfo validated: %v", nodeInfo)

	actorID, err := forgefed.NewPersonID(actorUri, string(nodeInfo.Source))
	if err != nil {
		ctx.ServerError("Validate actorId", err)
		return
	}
	log.Info("RepositoryInbox: actorId validated: %v", actorID)
	// parse objectID (repository)
	objectID, err := forgefed.NewRepositoryID(activity.Object.GetID().String(), string(nodeInfo.Source))
	if err != nil {
		ctx.ServerError("Validate objectId", err)
		return
	}
	if objectID.ID != fmt.Sprint(repository.ID) {
		ctx.ServerError("Validate objectId", err)
		return
	}
	log.Info("RepositoryInbox: objectId validated: %v", objectID)

	actorAsLoginID := actorID.AsLoginName() // used as LoginName in newly created user
	log.Info("RepositoryInbox: remoteStargazer: %v", actorAsLoginID)

	// Check if user already exists
	users, err := SearchUsersByLoginName(actorAsLoginID)
	if err != nil {
		ctx.ServerError("Searching for user failed", err)
		return
	}
	log.Info("RepositoryInbox: local found users: %v", len(users))

	switch len(users) {
	case 0:
		{
			user, err = createUserFromAP(ctx, actorID)
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
	actionsUser := user_model.NewActionsUser()
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

func createNodeInfo(ctx *context.APIContext, actorID forgefed.ActorID) (forgefed.NodeInfo, error) {
	actionsUser := user_model.NewActionsUser()
	client, err := api.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return forgefed.NodeInfo{}, err
	}
	body, err := client.GetBody(actorID.AsWellKnownNodeInfoUri())
	if err != nil {
		return forgefed.NodeInfo{}, err
	}
	nodeInfoWellKnown, err := forgefed.NewNodeInfoWellKnown(body)
	if err != nil {
		return forgefed.NodeInfo{}, err
	}
	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return forgefed.NodeInfo{}, err
	}
	return forgefed.NewNodeInfo(body)
}

// ToDo: Maybe use externalLoginUser
func createUserFromAP(ctx *context.APIContext, personID forgefed.PersonID) (*user_model.User, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	actionsUser := user_model.NewActionsUser()
	client, err := api.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return &user_model.User{}, err
	}

	body, err := client.GetBody(personID.AsURI())
	if err != nil {
		return &user_model.User{}, err
	}

	person := ap.Person{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return &user_model.User{}, err
	}
	log.Info("RepositoryInbox: got person by ap: %v", person)

	// TODO: we should validate the person object here!

	email := fmt.Sprintf("%v@%v", uuid.New().String(), personID.Host)
	loginName := personID.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
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
