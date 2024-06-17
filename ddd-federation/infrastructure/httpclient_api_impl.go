// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package infrastructure

import (
	"context"
	"fmt"
	"time"

	"code.gitea.io/gitea/ddd-federation/domain"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/validation"
)

type HttpClientAPIImpl struct {
}

func (_ HttpClientAPIImpl) GetFederationHostFromAP(ctx context.Context, actorID forgefed.ActorID) (domain.FederationHost, error) {
	actionsUser := user.NewActionsUser()
	client, err := activitypub.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return domain.FederationHost{}, err
	}
	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return domain.FederationHost{}, err
	}
	nodeInfoWellKnown, err := domain.NewNodeInfoWellKnown(body)
	if err != nil {
		return domain.FederationHost{}, err
	}
	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return domain.FederationHost{}, err
	}
	nodeInfo, err := domain.NewNodeInfo(body)
	if err != nil {
		return domain.FederationHost{}, err
	}
	result, err := domain.NewFederationHost(nodeInfo, actorID.Host)
	if err != nil {
		return domain.FederationHost{}, err
	}

	return result, nil
}

func (_ HttpClientAPIImpl) GetForgePersonFromAP(ctx context.Context, personID forgefed.PersonID) (forgefed.ForgePerson, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	actionsUser := user.NewActionsUser()
	client, err := activitypub.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return forgefed.ForgePerson{}, err
	}
	body, err := client.GetBody(personID.AsURI())
	if err != nil {
		return forgefed.ForgePerson{}, err
	}
	person := forgefed.ForgePerson{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return forgefed.ForgePerson{}, err
	}
	if res, err := validation.IsValid(person); !res {
		return forgefed.ForgePerson{}, err
	}
	log.Info("Fetched valid person:%q", person)

	return person, nil
}

func (_ HttpClientAPIImpl) PostLikeActivities(ctx context.Context, doer user.User, activityList []forgefed.ForgeLike) error {
	apclient, err := activitypub.NewClient(ctx, &doer, doer.APActorID())
	if err != nil {
		return err
	}
	for i, activity := range activityList {
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
