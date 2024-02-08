// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/validation"

	"github.com/google/uuid"
	pwd_gen "github.com/sethvargo/go-password/password"
)

func LikeActivity(ctx *context.APIContext, form any, repositoryId int64) (error, int, string) {
	activity := form.(*forgefed.ForgeLike)
	if res, err := validation.IsValid(activity); !res {
		return err, http.StatusNotAcceptable, "Invalid activity"
	}
	log.Info("Activity validated:%v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	rawActorID, err := forgefed.NewActorID(actorURI)
	if err != nil {
		return err, http.StatusInternalServerError, "Invalid ActorID"
	}
	federationHost, err := forgefed.FindFederationHostByFqdn(ctx, rawActorID.Host)
	if err != nil {
		return err, http.StatusInternalServerError, "Could not loading FederationHost"
	}
	if federationHost == nil {
		result, err := CreateFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return err, http.StatusNotAcceptable, "Invalid FederationHost"
		}
		federationHost = result
	}
	if !activity.IsNewer(federationHost.LatestActivity) {
		return fmt.Errorf("Activity already processed"), http.StatusNotAcceptable, "Activity out of order."
	}
	actorID, err := forgefed.NewPersonID(actorURI, string(federationHost.NodeInfo.Source))
	if err != nil {
		return err, http.StatusNotAcceptable, "Invalid PersonID"
	}
	log.Info("Actor accepted:%v", actorID)

	// parse objectID (repository)
	objectID, err := forgefed.NewRepositoryID(activity.Object.GetID().String(), string(forgefed.ForgejoSourceType))
	if err != nil {
		return err, http.StatusNotAcceptable, "Invalid objectId"
	}
	if objectID.ID != fmt.Sprint(repositoryId) {
		return err, http.StatusNotAcceptable, "Invalid objectId"
	}
	log.Info("Object accepted:%v", objectID)

	// Check if user already exists
	user, _, err := user.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
		return err, http.StatusInternalServerError, "Searching for user failed"
	}
	if user != nil {
		log.Info("Found local federatedUser: %v", user)
	} else {
		user, _, err = CreateUserFromAP(ctx, actorID, federationHost.ID)
		if err != nil {
			return err, http.StatusInternalServerError, "Error creating federatedUser"
		}
		log.Info("Created federatedUser from ap: %v", user)
	}
	log.Info("Got user:%v", user.Name)

	// execute the activity if the repo was not stared already
	alreadyStared := repo.IsStaring(ctx, user.ID, repositoryId)
	if !alreadyStared {
		err = repo.StarRepo(ctx, user.ID, repositoryId, true)
		if err != nil {
			return err, http.StatusNotAcceptable, "Error staring"
		}
	}
	federationHost.LatestActivity = activity.StartTime
	err = forgefed.UpdateFederationHost(ctx, federationHost)
	if err != nil {
		return err, http.StatusNotAcceptable, "Error updating federatedHost"
	}

	return nil, 0, ""
}

func CreateFederationHostFromAP(ctx *context.APIContext, actorID forgefed.ActorID) (*forgefed.FederationHost, error) {
	actionsUser := user.NewActionsUser()
	client, err := activitypub.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, err
	}
	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return nil, err
	}
	nodeInfoWellKnown, err := forgefed.NewNodeInfoWellKnown(body)
	if err != nil {
		return nil, err
	}
	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return nil, err
	}
	nodeInfo, err := forgefed.NewNodeInfo(body)
	if err != nil {
		return nil, err
	}
	result, err := forgefed.NewFederationHost(nodeInfo, actorID.Host)
	if err != nil {
		return nil, err
	}
	err = forgefed.CreateFederationHost(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func CreateUserFromAP(ctx *context.APIContext, personID forgefed.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	actionsUser := user.NewActionsUser()
	client, err := activitypub.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, nil, err
	}

	body, err := client.GetBody(personID.AsURI())
	if err != nil {
		return nil, nil, err
	}

	person := forgefed.ForgePerson{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return nil, nil, err
	}
	if res, err := validation.IsValid(person); !res {
		return nil, nil, err
	}
	log.Info("Fetched valid person:%q", person)

	localFqdn, err := url.ParseRequestURI(setting.AppURL)
	if err != nil {
		return nil, nil, err
	}
	email := fmt.Sprintf("f%v@%v", uuid.New().String(), localFqdn.Hostname())
	loginName := personID.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
	fullName := person.Name.String()
	if len(person.Name) == 0 {
		fullName = name
	}
	password, err := pwd_gen.Generate(32, 10, 10, false, true)
	if err != nil {
		return nil, nil, err
	}
	newUser := user.User{
		LowerName:                    strings.ToLower(person.PreferredUsername.String()),
		Name:                         name,
		FullName:                     fullName,
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    loginName,
		Type:                         user.UserTypeRemoteUser,
		IsAdmin:                      false,
	}
	federatedUser := user.FederatedUser{
		ExternalID:       personID.ID,
		FederationHostID: federationHostID,
	}
	err = user.CreateFederatedUser(ctx, &newUser, &federatedUser)
	if err != nil {
		return nil, nil, err
	}
	log.Info("Created federatedUser:%q", federatedUser)

	return &newUser, &federatedUser, nil
}
