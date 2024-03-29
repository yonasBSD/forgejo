// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: GPL-3.0-or-later

package forgefed

import (
	"context"
	"io"
	"net/http"
	"time"

	"code.gitea.io/gitea/models/federation"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	user_service "code.gitea.io/gitea/services/user"

	ap "github.com/go-ap/activitypub"
)

func GetActor(id string) (*ap.Actor, error) {
	client := http.Client{}
	req, err := http.NewRequest("GET", id, nil)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"Content-Type": {"application/activity+json"},
	}
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	actorObj := new(ap.Actor)
	err = json.Unmarshal(body, &actorObj)
	if err != nil {
		return nil, err
	}
	return actorObj, nil
}

func GetPersonAvatar(ctx context.Context, person *ap.Person) ([]byte, error) {
	avatarObj := new(ap.Image)
	_, err := ap.CopyItemProperties(avatarObj, person.Icon)
	if err != nil {
		return nil, err
	}

	r, err := http.Get(avatarObj.URL.GetLink().String())
	if err != nil {
		log.Error("Got error while fetching avatar fn: %w", err)
		return nil, err
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func SavePerson(ctx context.Context, person *ap.Person) (*user.User, error) {
	hostname, err := GetHostnameFromResource(person.ID.String())
	if err != nil {
		return nil, err
	}

	exists, err := federation.FederatedHostExists(ctx, hostname)
	if err != nil {
		return nil, err
	}

	var federatedHost federation.FederatedHost
	if exists {
		x, err := federation.GetFederatdHost(ctx, hostname)
		federatedHost = *x
		if err != nil {
			return nil, err
		}
	} else {
		federatedHost := new(federation.FederatedHost)
		federatedHost.HostFqdn = hostname
		if err = federatedHost.Save(ctx); err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	u := new(user.User)
	u.Name = "@" + person.PreferredUsername.String() + "@" + hostname
	u.Email = person.PreferredUsername.String() + "@" + hostname
	u.Website = person.URL.GetID().String()
	u.KeepEmailPrivate = true

	exist, err := user.GetUser(ctx, u)
	if err != nil {
		return nil, err
	}
	if exist {
		return u, nil // TODO: must also check for federatedUser existence
	}

	if err = federation.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	avatar, err := GetPersonAvatar(ctx, person)
	if err != nil {
		log.Error("Got error while fetching avatar: %w", err)
		return nil, err
	}

	if u.IsUploadAvatarChanged(avatar) {
		_ = user_service.UploadAvatar(ctx, u, avatar)
	}

	if err = federation.CreateFederatedUser(ctx, u, &federatedHost); err != nil {
		return nil, err
	}

	return u, nil
}

func GetActorFromUser(ctx context.Context, u *user.User) (*ap.Actor, error) {
	alias := u.Name

	webfingerRes, err := WebFingerLookup(alias)
	if err != nil {
		return nil, err
	}

	actorID := webfingerRes.GetActorLink().Href

	return GetActor(actorID)
}

// Clean up remote actors (persons) without any followers in local instance
func CleanUpRemotePersons(ctx context.Context, olderThan time.Duration) error {
	page := 0
	for {
		users, err := federation.GetRemoteUsersWithNoLocalFollowers(ctx, olderThan, page)
		if len(users) == 0 {
			break
		}
		if err != nil {
			log.Trace("Error: CleanUpRemotePersons: %v", err)
			return err
		}

		for _, u := range users {
			err = user_service.DeleteUser(ctx, &u, false)
			if err != nil {
				log.Trace("Error: CleanUpRemotePersons: %v", err)
				return err
			}
		}
		page++
	}
	return nil
}

func UpdatePersonActor(ctx context.Context) error {
	// NOTE: change of any of these don't matter at this point since we are
	// ignoring actor's PreferredUsername and using their address to generate
	// username and email. Ask suggestions from other devs.
	//
	//
	//	u := new(user.User)
	//	u.Name = "@" + person.PreferredUsername.String() + "@" + hostname
	//	//panic(u.Name)
	//	u.Email = person.PreferredUsername.String() + "@" + hostname
	//	u.Website = person.URL.GetID().String()
	//	u.KeepEmailPrivate = true

	page := 0
	for {
		federatedUsers, err := federation.GetRemotePersons(ctx, page)
		if len(federatedUsers) == 0 {
			break
		}
		if err != nil {
			log.Trace("Error: UpdatePersonActor: %v", err)
			return err
		}

		for _, f := range federatedUsers {
			log.Info("Updating users, got %s", f.ExternalID)
			u, err := user.GetUserByName(ctx, f.ExternalID)
			if err != nil {
				log.Error("Got error while getting user: %w", err)
				return err
			}

			person, err := GetActorFromUser(ctx, u)
			if err != nil {
				log.Error("Got error while fetching actor: %w", err)
				return err
			}

			avatar, err := GetPersonAvatar(ctx, person)
			if err != nil {
				log.Error("Got error while fetching avatar: %w", err)
				return err
			}

			if u.IsUploadAvatarChanged(avatar) {
				_ = user_service.UploadAvatar(ctx, u, avatar)
			}
		}
		page++
	}
	return nil
}
