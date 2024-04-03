// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/auth/password"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/validation"

	"github.com/google/uuid"
)

// ProcessLikeActivity receives a ForgeLike activity and does the following:
// Validation of the activity
// Creation of a (remote) federationHost if not existing
// Creation of a forgefed Person if not existing
// Validation of incoming RepositoryID against Local RepositoryID
// Star the repo if it wasn't already stared
// Do some mitigation against out of order attacks
func ProcessLikeActivity(ctx context.Context, form any, repositoryID int64) (int, string, error) {
	activity := form.(*forgefed.ForgeLike)
	if res, err := validation.IsValid(activity); !res {
		return http.StatusNotAcceptable, "Invalid activity", err
	}
	log.Info("Activity validated:%v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	log.Info("actorURI was: %v", actorURI)
	federationHost, err := GetFederationHostFromUri(ctx, actorURI)
	if err != nil {
		return http.StatusInternalServerError, "Wrong FederationHost", err
	}
	if !activity.IsNewer(federationHost.LatestActivity) {
		return http.StatusNotAcceptable, "Activity out of order.", fmt.Errorf("Activity already processed")
	}
	actorID, err := forgefed.NewPersonID(actorURI, string(federationHost.NodeInfo.Source))
	if err != nil {
		return http.StatusNotAcceptable, "Invalid PersonID", err
	}
	log.Info("Actor accepted:%v", actorID)

	// parse objectID (repository)
	objectID, err := forgefed.NewRepositoryID(activity.Object.GetID().String(), string(forgefed.ForgejoSourceType))
	if err != nil {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	if objectID.ID != fmt.Sprint(repositoryID) {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	log.Info("Object accepted:%v", objectID)

	// Check if user already exists
	user, _, err := user.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
		return http.StatusInternalServerError, "Searching for user failed", err
	}
	if user != nil {
		log.Info("Found local federatedUser: %v", user)
	} else {
		user, _, err = CreateUserFromAP(ctx, actorID, federationHost.ID)
		if err != nil {
			return http.StatusInternalServerError, "Error creating federatedUser", err
		}
		log.Info("Created federatedUser from ap: %v", user)
	}
	log.Info("Got user:%v", user.Name)

	// execute the activity if the repo was not stared already
	alreadyStared := repo.IsStaring(ctx, user.ID, repositoryID)
	if !alreadyStared {
		err = repo.StarRepo(ctx, user.ID, repositoryID, true)
		if err != nil {
			return http.StatusNotAcceptable, "Error staring", err
		}
	}
	federationHost.LatestActivity = activity.StartTime
	err = forgefed.UpdateFederationHost(ctx, federationHost)
	if err != nil {
		return http.StatusNotAcceptable, "Error updating federatedHost", err
	}

	return 0, "", nil
}

func CreateFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (*forgefed.FederationHost, error) {
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

func GetFederationHostFromUri(ctx context.Context, actorURI string) (*forgefed.FederationHost, error) {
	// parse actorID (person)
	log.Info("Input was: %v", actorURI)
	rawActorID, err := forgefed.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := forgefed.FindFederationHostByFqdn(ctx, rawActorID.Host)
	if err != nil {
		return nil, err
	}
	if federationHost == nil {
		result, err := CreateFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return nil, err
		}
		federationHost = result
	}
	return federationHost, nil
}

func CreateUserFromAP(ctx context.Context, personID forgefed.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
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
	password, err := password.Generate(32)
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

// TODO: This needs probably an own struct/value object
// ====================================================
func GetRepoOwnerAndNameFromRepoUri(uri string) (string, string, error) {
	path, err := getUriPathFromRepoUri(uri)
	if err != nil {
		return "", "", err
	}
	pathSplit := strings.Split(path, "/")

	return pathSplit[0], pathSplit[1], nil
}

func GetRepoAPIUriFromRepoUri(uri string) (string, error) {
	path, err := getUriPathFromRepoUri(uri)
	if err != nil {
		return "", err
	}

	parsedUri, err := url.ParseRequestURI(uri)
	if err != nil {
		return "", err
	}
	parsedUri.Path = fmt.Sprintf("%v/%v", "api/v1/repos", path)

	return parsedUri.String(), nil
}

func getUriPathFromRepoUri(uri string) (string, error) {
	if !IsValidRepoUri(uri) {
		return "", errors.New("malformed repository uri")
	}
	parsedUri, err := url.ParseRequestURI(uri)
	if err != nil {
		return "", err
	}
	path := strings.TrimPrefix(parsedUri.Path, "/")
	return path, nil
}

// TODO: This belongs into some value object
func IsValidRepoUri(uri string) bool {
	parsedUri, err := url.ParseRequestURI(uri)
	if err != nil {
		return false
	}
	path := strings.TrimPrefix(parsedUri.Path, "/")
	pathSplit := strings.Split(path, "/")
	if len(pathSplit) != 2 {
		return false
	}
	for _, part := range pathSplit {
		if strings.TrimSpace(part) == "" {
			return false
		}
	}
	return true
}

// ====================================================

// Create or update a list of FollowingRepo structs
func StoreFollowingRepoList(ctx context.Context, localRepoId int64, followingRepoList []string) (int, string, error) {
	followingRepos := make([]*repo.FollowingRepo, 0, len(followingRepoList))
	for _, uri := range followingRepoList {
		owner, repoName, err := GetRepoOwnerAndNameFromRepoUri(uri)
		if err != nil {
			return http.StatusBadRequest, "Malformed Repository URI", err
		}
		repoApiUri, err := GetRepoAPIUriFromRepoUri(uri)
		if err != nil {
			return http.StatusBadRequest, "Malformed Repository URI", err
		}
		log.Info("RepoApiUri: %q", repoApiUri)

		// TODO: derive host from repoApiUri or derive APAPI URI from repoApiUri
		federationHost, err := GetFederationHostFromUri(ctx, repoApiUri)
		if err != nil {
			return http.StatusInternalServerError, "Wrong FederationHost", err
		}
		// TODO: derive id from repoApiUri
		followingRepoID, err := forgefed.NewRepositoryID(repoApiUri, string(federationHost.NodeInfo.Source))
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepo, err := repo.NewFollowingRepo(localRepoId, followingRepoID.ID, federationHost.ID, owner, repoName, uri)
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepos = append(followingRepos, &followingRepo)
	}

	repo.StoreFollowingRepos(ctx, localRepoId, followingRepos)

	return 0, "", nil
}

func DeleteFollowingRepos(ctx context.Context, localRepoId int64) error {
	return repo.StoreFollowingRepos(ctx, localRepoId, []*repo.FollowingRepo{})
}

func SendLikeActivities(ctx context.Context, doer user.User, repoID int64) error {
	followingRepos, err := repo.FindFollowingReposByRepoID(ctx, repoID)
	log.Info("Federated Repos is: %v", followingRepos)
	if err != nil {
		return err
	}

	likeActivityList := make([]forgefed.ForgeLike, 0)
	for _, followingRepo := range followingRepos {
		target := followingRepo.Uri
		likeActivity, err := forgefed.NewForgeLike(doer.APAPIURL(), target, time.Now())
		if err != nil {
			return err
		}
		likeActivityList = append(likeActivityList, likeActivity)
	}

	apclient, err := activitypub.NewClient(ctx, &doer, doer.APAPIURL())
	if err != nil {
		return err
	}
	for _, activity := range likeActivityList {
		json, err := activity.MarshalJSON()
		if err != nil {
			return err
		}

		_, err = apclient.Post([]byte(json), fmt.Sprintf("%v/inbox/", activity.Object))
		if err != nil {
			log.Error("error %v while sending activity: %q", err, activity)
		}
	}

	return nil
}
