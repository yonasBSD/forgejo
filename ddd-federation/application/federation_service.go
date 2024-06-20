// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"code.gitea.io/gitea/ddd-federation/domain"
	"code.gitea.io/gitea/ddd-federation/infrastructure"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/validation"
)

// TODO: Is it allowed to create/use objects/entities/aggregates from outside in domain?
//		 Or only in application/infra?

type FederationService struct {
	federationHostRepository domain.FederationHostRepository
	followingRepoRepository  domain.FollowingRepoRepository
	userRepository           domain.UserRepository
	repoRepository           domain.RepoRepository
	httpClientAPI            domain.HttpClientAPI
}

// NewFederationService returns a FederationService.
// If no FederationHostRepository is passed as param, then `infrastructure.FederationHostRepositoryImpl` is used.
// If no HttpClientAPI is passed as param, then `infrastructure.HttpClientAPIImpl` is used.
// If a FederationHostRepository is passed as param, a FederationService using the passed repo is returned.
// If a HttpClientAPI is passed as param, a FederationService using the passed api is returned.
func NewFederationService(params ...interface{}) FederationService {
	var federationHostRepository domain.FederationHostRepository = nil
	var followingRepoRepository domain.FollowingRepoRepository = nil
	var userRepository domain.UserRepository = nil
	var repoRepository domain.RepoRepository = nil
	var httpClientAPI domain.HttpClientAPI = nil

	for _, param := range params {
		switch v := param.(type) {
		case domain.FederationHostRepository:
			federationHostRepository = v
		case domain.FollowingRepoRepository:
			followingRepoRepository = v
		case domain.UserRepository:
			userRepository = v
		case domain.RepoRepository:
			repoRepository = v
		case domain.HttpClientAPI:
			httpClientAPI = v
		}
	}

	if federationHostRepository == nil {
		federationHostRepository = domain.FederationHostRepository(infrastructure.FederationHostRepositoryImpl{})
	}
	if followingRepoRepository == nil {
		followingRepoRepository = domain.FollowingRepoRepository(infrastructure.FollowingRepoRepositoryWrapper{})
	}
	if userRepository == nil {
		userRepository = domain.UserRepository(infrastructure.UserRepositoryWrapper{})
	}
	if repoRepository == nil {
		repoRepository = domain.RepoRepository(infrastructure.RepoRepositoryWrapper{})
	}
	if httpClientAPI == nil {
		httpClientAPI = domain.HttpClientAPI(infrastructure.HttpClientAPIImpl{})
	}

	return FederationService{
		federationHostRepository: federationHostRepository,
		followingRepoRepository:  followingRepoRepository,
		userRepository:           userRepository,
		repoRepository:           repoRepository,
		httpClientAPI:            httpClientAPI,
	}
}

// ProcessLikeActivity receives a ForgeLike activity and does the following:
// Validation of the activity
// Creation of a (remote) federationHost if not existing
// Creation of a forgefed Person if not existing
// Validation of incoming RepositoryID against Local RepositoryID
// Star the repo if it wasn't already stared
// Do some mitigation against out of order attacks
func (s FederationService) ProcessLikeActivity(ctx context.Context, form any, repositoryID int64) (int, string, error) {
	activity := form.(*fm.ForgeLike)
	if res, err := validation.IsValid(activity); !res {
		// ToDo: Why do we return an additional error string?
		return http.StatusNotAcceptable, "Invalid activity", err
	}
	log.Info("Activity validated:%v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	log.Info("actorURI was: %v", actorURI)
	federationHost, err := s.GetFederationHostForURI(ctx, actorURI)
	if err != nil {
		return http.StatusInternalServerError, "Wrong FederationHost", err
	}
	if !activity.IsNewer(federationHost.LatestActivity) {
		return http.StatusNotAcceptable, "Activity out of order.", fmt.Errorf("Activity already processed")
	}
	personID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		return http.StatusNotAcceptable, "Invalid PersonID", err
	}
	log.Info("Person ActorID accepted:%v", personID)

	// parse objectID (repository)
	objectID, err := fm.NewRepositoryID(activity.Object.GetID().String(), string(domain.ForgejoSourceType))
	if err != nil {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	if objectID.ID != fmt.Sprint(repositoryID) {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	log.Info("Object accepted:%v", objectID)

	user, err := s.GetFederationUserForID(ctx, personID, federationHost)
	if err != nil {
		return http.StatusInternalServerError, "Error getting or creating federation user", err
	}

	// execute the activity if the repo was not stared already
	alreadyStared := s.repoRepository.IsStaring(ctx, user.ID, repositoryID)
	if !alreadyStared {
		err = s.repoRepository.StarRepo(ctx, user.ID, repositoryID, true)
		if err != nil {
			return http.StatusNotAcceptable, "Error staring", err
		}
	}
	federationHost.LatestActivity = activity.StartTime
	err = s.federationHostRepository.UpdateFederationHost(ctx, federationHost)
	if err != nil {
		return http.StatusNotAcceptable, "Error updating federatedHost", err
	}

	return 0, "", nil
}

func (s FederationService) CreateFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*domain.FederationHost, error) {
	result, err := s.httpClientAPI.GetFederationHostFromAP(ctx, actorID)
	if err != nil {
		return nil, err
	}
	err = s.federationHostRepository.CreateFederationHost(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Check if federation host exists on this server DB.
// If not create it from actorURI.
func (s FederationService) GetFederationHostForURI(ctx context.Context, actorURI string) (*domain.FederationHost, error) {
	log.Info("Input was: %v", actorURI)
	rawActorID, err := fm.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := s.federationHostRepository.FindFederationHostByFqdn(ctx, rawActorID.Host)
	if err != nil {
		return nil, err
	}
	if federationHost == nil {
		result, err := s.CreateFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return nil, err
		}
		federationHost = result
	}
	return federationHost, nil
}

// Check if the federated user already exists on this server DB.
// Create if it does not.
func (s FederationService) GetFederationUserForID(ctx context.Context, personID fm.PersonID, federationHost *domain.FederationHost) (*user.User, error) {
	user, _, err := s.userRepository.FindFederatedUser(ctx, personID.ID, federationHost.ID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		log.Info("Found local federatedUser: %v", user)
	} else {
		user, _, err = s.CreateUserFromAP(ctx, personID, federationHost.ID)
		if err != nil {
			return nil, err
		}
		log.Info("Created federatedUser from ap: %v", user)
	}
	log.Info("Got user:%v", user.Name)

	return user, nil
}

// ToDo: user.Federated user is not used anywhere. Do we need to return it?
func (s FederationService) CreateUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	person, err := s.httpClientAPI.GetForgePersonFromAP(ctx, personID)
	if err != nil {
		return nil, nil, err
	}

	representativeUser, err := s.userRepository.GetRepresentativeUser(ctx, person, personID)
	if err != nil {
		return nil, nil, err
	}

	federatedUser := s.userRepository.GetFederatedUser(personID, federationHostID)

	err = s.userRepository.CreateFederatedUser(ctx, &representativeUser, &federatedUser)
	if err != nil {
		return nil, nil, err
	}
	log.Info("Created federatedUser:%q", federatedUser)

	return &representativeUser, &federatedUser, nil
}

// Create or update a list of FollowingRepo structs
func (s FederationService) StoreFollowingRepoList(ctx context.Context, localRepoID int64, followingRepoList []string) (int, string, error) {
	followingRepos := make([]*repo.FollowingRepo, 0, len(followingRepoList))
	for _, uri := range followingRepoList {
		federationHost, err := s.GetFederationHostForURI(ctx, uri)
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

	if err := s.followingRepoRepository.StoreFollowingRepos(ctx, localRepoID, followingRepos); err != nil {
		return 0, "", err
	}

	return 0, "", nil
}

func (s FederationService) DeleteFollowingRepos(ctx context.Context, localRepoID int64) error {
	return s.followingRepoRepository.StoreFollowingRepos(ctx, localRepoID, []*repo.FollowingRepo{})
}

func (s FederationService) SendLikeActivities(ctx context.Context, doer user.User, repoID int64) error {
	followingRepos, err := s.followingRepoRepository.FindFollowingReposByRepoID(ctx, repoID)
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

	return s.httpClientAPI.PostLikeActivities(ctx, doer, likeActivityList)
}
