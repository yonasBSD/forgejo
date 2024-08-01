// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: GPL-3.0-or-later

package federation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"code.gitea.io/gitea/modules/log"

	ap "github.com/go-ap/activitypub"
)

//	user_service "code.gitea.io/gitea/services/user"

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
