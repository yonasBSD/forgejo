// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"testing"
)

func TestActorParserEmpty(t *testing.T) {
	item := ""
	want := ActorID{}

	got, _ := ParseActorID(item)

	if got != want {
		t.Errorf("ParseActorID returned non empty actor id for empty input.")
	}
}

func TestActorParserValid(t *testing.T) {
	item := "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"
	want := ActorID{
		schema: "https",
		userId: "1",
		path:   "/api/v1/activitypub/user-id/1",
		host:   "repo.prod.meissa.de",
		port:   "",
	}

	got, _ := ParseActorID(item)

	if got != want {
		t.Errorf("ParseActorID did not return want: %v.", want)
	}
}

func TestValidateValid(t *testing.T) {
	item := ActorID{
		schema: "https",
		userId: "1",
		path:   "/api/v1/activitypub/user-id/1",
		host:   "repo.prod.meissa.de",
		port:   "",
	}

	if err := item.Validate(); err != nil {
		t.Errorf("Validating actor returned non nil with valid input.")
	}
}

func TestValidateInvalid(t *testing.T) {
	item := "123456"

	actor, _ := ParseActorID(item)

	if err := actor.Validate(); err == nil {
		t.Errorf("Validating actor returned nil with false input.")
	}
}

func TestGetHostAndPort(t *testing.T) {
	item := ActorID{
		schema: "https",
		userId: "1",
		path:   "/api/v1/activitypub/user-id/1",
		host:   "repo.prod.meissa.de",
		port:   "80",
	}
	want := "repo.prod.meissa.de:80"

	hostAndPort := item.GetHostAndPort()

	if hostAndPort != want {
		t.Errorf("GetHostAndPort did not return correct host and port combination: %v", hostAndPort)
	}

}
