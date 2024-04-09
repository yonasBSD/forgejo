// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: GPL-3.0-or-later

package forgefed

import (
	"context"
	"io"
	"net/http"

	"code.gitea.io/gitea/models/federation"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"

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

	if err = federation.CreateFederatedUser(ctx, u, &federatedHost); err != nil {
		return nil, err
	}

	return u, nil
}
