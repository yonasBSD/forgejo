// Copyright 2023, 2024 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/forgefed"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/validation"
	"code.gitea.io/gitea/modules/web"

	ap "github.com/go-ap/activitypub"
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
		ctx.Error(http.StatusInternalServerError, "Set Name", err)
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
	//     "$ref": "#/definitions/ForgeLike"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	var user *user_model.User

	repository := ctx.Repo.Repository
	log.Info("RepositoryInbox: repo: %v", repository)

	activity := web.GetForm(ctx).(*forgefed.ForgeLike)
	if res, err := validation.IsValid(activity); !res {
		ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: Validate activity", err)
		return
	}
	log.Info("RepositoryInbox: activity validated:%v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	rawActorID, err := forgefed.NewActorID(actorURI)
	if err != nil {
		ctx.Error(http.StatusInternalServerError,
			"RepositoryInbox: Validating ActorID", err)
		return
	}
	federationHost, err := forgefed.FindFederationHostByFqdn(ctx, rawActorID.Host)
	if err != nil {
		ctx.Error(http.StatusInternalServerError,
			"RepositoryInbox: Error while loading FederationInfo", err)
		return
	}
	if federationHost == nil {
		result, err := createFederationHost(ctx, rawActorID)
		if err != nil {
			ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: Validate actorId", err)
			return
		}
		federationHost = &result
		log.Info("RepositoryInbox: federationInfo validated: %v", federationHost)
	}
	if !activity.IsNewer(federationHost.LatestActivity) {
		ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: Validate Activity",
			fmt.Errorf("Activity already processed"))
		return
	}

	actorID, err := forgefed.NewPersonID(actorURI, string(federationHost.NodeInfo.Source))
	if err != nil {
		ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: Validate actorId", err)
		return
	}
	log.Info("RepositoryInbox: actorId validated: %v", actorID)
	// parse objectID (repository)
	objectID, err := forgefed.NewRepositoryID(activity.Object.GetID().String(), string(forgefed.ForgejoSourceType))
	if err != nil {
		ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: Validate objectId", err)
		return
	}
	if objectID.ID != fmt.Sprint(repository.ID) {
		ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: Validate objectId", err)
		return
	}
	log.Info("RepositoryInbox: objectId validated: %v", objectID)

	actorAsLoginID := actorID.AsLoginName() // used as LoginName in newly created user
	log.Info("RepositoryInbox: remoteStargazer: %v", actorAsLoginID)

	// Check if user already exists
	// TODO: search for federation user instead
	// users, _, err := SearchFederatedUser(actorID.ID, federationHost.ID)
	users, err := SearchUsersByLoginName(actorAsLoginID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "RepositoryInbox: Searching for user failed", err)
		return
	}
	log.Info("RepositoryInbox: local found users: %v", len(users))

	switch len(users) {
	case 0:
		{
			user, err = createUserFromAP(ctx, actorID, federationHost.ID)
			if err != nil {
				ctx.Error(http.StatusInternalServerError,
					"RepositoryInbox: Creating federated user failed", err)
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
			ctx.Error(http.StatusInternalServerError, "RepositoryInbox",
				fmt.Errorf(" more than one matches for federated users"))
			return
		}
	}

	// execute the activity if the repo was not stared already
	alreadyStared := repo_model.IsStaring(ctx, user.ID, repository.ID)
	if !alreadyStared {
		err = repo_model.StarRepo(ctx, user.ID, repository.ID, true)
		if err != nil {
			ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: Star operation", err)
			return
		}
	}
	federationHost.LatestActivity = activity.StartTime
	err = forgefed.UpdateFederationHost(ctx, *federationHost)
	if err != nil {
		ctx.Error(http.StatusNotAcceptable, "RepositoryInbox: error updateing federateionInfo", err)
		return
	}

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

func createFederationHost(ctx *context.APIContext, actorID forgefed.ActorID) (forgefed.FederationHost, error) {
	actionsUser := user_model.NewActionsUser()
	client, err := api.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return forgefed.FederationHost{}, err
	}
	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return forgefed.FederationHost{}, err
	}
	nodeInfoWellKnown, err := forgefed.NewNodeInfoWellKnown(body)
	if err != nil {
		return forgefed.FederationHost{}, err
	}
	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return forgefed.FederationHost{}, err
	}
	nodeInfo, err := forgefed.NewNodeInfo(body)
	if err != nil {
		return forgefed.FederationHost{}, err
	}
	result, err := forgefed.NewFederationHost(nodeInfo, actorID.Host)
	if err != nil {
		return forgefed.FederationHost{}, err
	}
	err = forgefed.CreateFederationHost(ctx, result)
	if err != nil {
		return forgefed.FederationHost{}, err
	}
	return result, nil
}

func createUserFromAP(ctx *context.APIContext, personID forgefed.PersonID, federationHostID int64) (*user_model.User, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	actionsUser := user_model.NewActionsUser()
	client, err := api.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, err
	}

	body, err := client.GetBody(personID.AsURI())
	if err != nil {
		return nil, err
	}

	person := forgefed.ForgePerson{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	if res, err := validation.IsValid(person); !res {
		return nil, err
	}
	log.Info("RepositoryInbox: validated person: %q", person)

	user, _, err := user_model.CreateFederatedUserFromAP(ctx, person, personID, federationHostID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
