// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/validation"
	context_service "code.gitea.io/gitea/services/context"
)

// ProcessLikeActivity receives a ForgeLike activity and does the following:
// Validation of the activity
// Creation of a (remote) federationHost if not existing
// Creation of a forgefed Person if not existing
// Validation of incoming RepositoryID against Local RepositoryID
// Star the repo if it wasn't already stared
// Do some mitigation against out of order attacks
func ProcessLikeActivity(ctx *context_service.APIContext, form any, repositoryID int64) (int, string, error) {
	activity := form.(*fm.ForgeLike)
	if res, err := validation.IsValid(activity); !res {
		return http.StatusNotAcceptable, "Invalid activity", err
	}
	log.Info("Activity validated:%v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	user, _, federationHost, err := findOrCreateFederatedUser(ctx, actorURI)
	if err != nil {
		return http.StatusInternalServerError, "findOrCreateFederatedUser", err
	}

	if !activity.IsNewer(federationHost.LatestActivity) {
		return http.StatusNotAcceptable, "Activity out of order.", fmt.Errorf("Activity already processed")
	}

	// parse objectID (repository)
	objectID, err := fm.NewRepositoryID(activity.Object.GetID().String(), string(forgefed.ForgejoSourceType))
	if err != nil {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	if objectID.ID != fmt.Sprint(repositoryID) {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	log.Info("Object accepted:%v", objectID)

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

// Create or update a list of FollowingRepo structs
func StoreFollowingRepoList(ctx context.Context, localRepoID int64, followingRepoList []string) (int, string, error) {
	followingRepos := make([]*repo.FollowingRepo, 0, len(followingRepoList))
	for _, uri := range followingRepoList {
		federationHost, err := getFederationHostForURI(ctx.Base, uri)
		if err != nil {
			return http.StatusInternalServerError, "Wrong FederationHost", err
		}
		followingRepoID, err := fm.NewRepositoryID(uri, string(federationHost.NodeInfo.SoftwareName))
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepo, err := repo.NewFollowingRepo(localRepoID, followingRepoID.ID, federationHost.ID, uri)
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepos = append(followingRepos, &followingRepo)
	}

	if err := repo.StoreFollowingRepos(ctx, localRepoID, followingRepos); err != nil {
		return 0, "", err
	}

	return 0, "", nil
}

func DeleteFollowingRepos(ctx context.Context, localRepoID int64) error {
	return repo.StoreFollowingRepos(ctx, localRepoID, []*repo.FollowingRepo{})
}

func SendLikeActivities(ctx context.Context, doer user.User, repoID int64) error {
	followingRepos, err := repo.FindFollowingReposByRepoID(ctx, repoID)
	log.Info("Federated Repos is: %v", followingRepos)
	if err != nil {
		return err
	}

	likeActivityList := make([]fm.ForgeLike, 0)
	for _, followingRepo := range followingRepos {
		log.Info("Found following repo: %v", followingRepo)
		target := followingRepo.URI
		likeActivity, err := fm.NewForgeLike(doer.APActorID(), target, time.Now())
		if err != nil {
			return err
		}
		likeActivityList = append(likeActivityList, likeActivity)
	}

	apclientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return err
	}
	apclient, err := apclientFactory.WithKeys(ctx, &doer, doer.APActorID() + "#main-key")
	if err != nil {
		return err
	}
	for i, activity := range likeActivityList {
		activity.StartTime = activity.StartTime.Add(time.Duration(i) * time.Second)
		json, err := activity.MarshalJSON()
		if err != nil {
			return err
		}

		_, err = apclient.Post(json, fmt.Sprintf("%v/inbox/", activity.Object))
		if err != nil {
			log.Error("error %v while sending activity: %q", err, activity)
		}
	}

	return nil
}
