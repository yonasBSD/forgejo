// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	forgefed_model "code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"github.com/google/uuid"

	api "code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/validation"

	pwd_gen "github.com/sethvargo/go-password/password"
)

func LikeActivity(ctx *context.APIContext, form any, repositoryId int64) (error, int, string) {
	activity := form.(*forgefed_model.ForgeLike)
	if res, err := validation.IsValid(activity); !res {
		return err, http.StatusNotAcceptable, "RepositoryInbox: Validate activity"
	}
	log.Info("RepositoryInbox: activity validated:%v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	rawActorID, err := forgefed_model.NewActorID(actorURI)
	if err != nil {
		return err, http.StatusInternalServerError, "RepositoryInbox: Validating ActorID"
	}
	federationHost, err := forgefed_model.FindFederationHostByFqdn(ctx, rawActorID.Host)
	if err != nil {
		return err, http.StatusInternalServerError, "RepositoryInbox: Error while loading FederationInfo"
	}
	if federationHost == nil {
		result, err := CreateFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return err, http.StatusNotAcceptable, "RepositoryInbox: Validate actorId"
		}
		federationHost = result
		log.Info("RepositoryInbox: federationInfo validated: %v", federationHost)
	}
	if !activity.IsNewer(federationHost.LatestActivity) {
		return fmt.Errorf("Activity already processed"), http.StatusNotAcceptable, "RepositoryInbox: Validate Activity"
	}

	actorID, err := forgefed_model.NewPersonID(actorURI, string(federationHost.NodeInfo.Source))
	if err != nil {
		return err, http.StatusNotAcceptable, "RepositoryInbox: Validate actorId"
	}
	log.Info("RepositoryInbox: actorId validated: %v", actorID)
	// parse objectID (repository)
	objectID, err := forgefed_model.NewRepositoryID(activity.Object.GetID().String(), string(forgefed_model.ForgejoSourceType))
	if err != nil {
		return err, http.StatusNotAcceptable, "RepositoryInbox: Validate objectId"
	}
	if objectID.ID != fmt.Sprint(repositoryId) {
		return err, http.StatusNotAcceptable, "RepositoryInbox: Validate objectId"
	}
	log.Info("RepositoryInbox: objectId validated: %v", objectID)

	actorAsLoginID := actorID.AsLoginName() // used as LoginName in newly created user
	log.Info("RepositoryInbox: remoteStargazer: %v", actorAsLoginID)

	// Check if user already exists
	user, _, err := user_model.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
		return err, http.StatusInternalServerError, "RepositoryInbox: Searching for user failed"
	}
	if user != nil {
		log.Info("RepositoryInbox: found user: %v", user)
	} else {
		user, _, err = CreateUserFromAP(ctx, actorID, federationHost.ID)
		if err != nil {
			return err, http.StatusInternalServerError,
				"RepositoryInbox: Creating federated user failed"
		}
		log.Info("RepositoryInbox: created user from ap: %v", user)
	}

	// execute the activity if the repo was not stared already
	alreadyStared := repo.IsStaring(ctx, user.ID, repositoryId)
	if !alreadyStared {
		err = repo.StarRepo(ctx, user.ID, repositoryId, true)
		if err != nil {
			return err, http.StatusNotAcceptable, "RepositoryInbox: Star operation"
		}
	}
	federationHost.LatestActivity = activity.StartTime
	err = forgefed_model.UpdateFederationHost(ctx, *federationHost)
	if err != nil {
		return err, http.StatusNotAcceptable, "RepositoryInbox: error updateing federateionInfo"
	}

	return nil, 0, ""
}

func CreateFederationHostFromAP(ctx *context.APIContext, actorID forgefed_model.ActorID) (*forgefed_model.FederationHost, error) {
	actionsUser := user_model.NewActionsUser()
	client, err := api.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, err
	}
	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return nil, err
	}
	nodeInfoWellKnown, err := forgefed_model.NewNodeInfoWellKnown(body)
	if err != nil {
		return nil, err
	}
	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return nil, err
	}
	nodeInfo, err := forgefed_model.NewNodeInfo(body)
	if err != nil {
		return nil, err
	}
	result, err := forgefed_model.NewFederationHost(nodeInfo, actorID.Host)
	if err != nil {
		return nil, err
	}
	err = forgefed_model.CreateFederationHost(ctx, result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func CreateUserFromAP(ctx *context.APIContext, personID forgefed_model.PersonID, federationHostID int64) (*user_model.User, *user_model.FederatedUser, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	actionsUser := user_model.NewActionsUser()
	client, err := api.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, nil, err
	}

	body, err := client.GetBody(personID.AsURI())
	if err != nil {
		return nil, nil, err
	}

	person := forgefed_model.ForgePerson{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return nil, nil, err
	}
	if res, err := validation.IsValid(person); !res {
		return nil, nil, err
	}
	log.Info("RepositoryInbox: validated person: %q", person)

	localFqdn, err := url.ParseRequestURI(setting.AppURL)
	if err != nil {
		return nil, nil, err
	}
	email := fmt.Sprintf("f%v@%v", uuid.New().String(), localFqdn.Hostname())
	loginName := personID.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
	log.Info("RepositoryInbox: person.Name: %v", person.Name)
	fullName := person.Name.String()
	if len(person.Name) == 0 {
		fullName = name
	}
	password, err := pwd_gen.Generate(32, 10, 10, false, true)
	if err != nil {
		return nil, nil, err
	}
	user := user_model.User{
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

	federatedUser := user_model.FederatedUser{
		ExternalID:       personID.ID,
		FederationHostID: federationHostID,
	}

	err = user_model.CreateFederatedUser(ctx, &user, &federatedUser)
	if err != nil {
		return nil, nil, err
	}

	return &user, &federatedUser, nil
}
