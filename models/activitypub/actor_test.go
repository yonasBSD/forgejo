// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"testing"

	"code.gitea.io/gitea/modules/forgefed"
	ap "github.com/go-ap/activitypub"
)

var emptyMockStar *forgefed.Star = &forgefed.Star{
	Source: "",
	Activity: ap.Activity{
		Actor:  ap.IRI(""),
		Type:   "Star",
		Object: ap.IRI(""),
	},
}

var mockStar *forgefed.Star = &forgefed.Star{
	Source: "forgejo",
	Activity: ap.Activity{
		Actor:  ap.IRI("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"),
		Type:   "Star",
		Object: ap.IRI("https://codeberg.org/api/v1/activitypub/repository-id/1"),
	},
}

func TestActorParserEmpty(t *testing.T) {
	item := emptyMockStar
	want := ActorID{}

	got, _ := ParseActorFromStarActivity(item)

	if got != want {
		t.Errorf("ParseActorID returned non empty actor id for empty input.")
	}
}

func TestActorParserValid(t *testing.T) {
	item := mockStar
	want := ActorID{
		userId: "1",
		source: "forgejo",
		schema: "https",
		path:   "/api/v1/activitypub/user-id/1",
		host:   "repo.prod.meissa.de",
		port:   "",
	}

	got, _ := ParseActorFromStarActivity(item)

	if got != want {
		t.Errorf("\nParseActorID did not return want: %v\n but %v", want, got)
	}
}

func TestValidateValid(t *testing.T) {
	item := ActorID{
		userId: "1",
		source: "forgejo",
		schema: "https",
		path:   "/api/v1/activitypub/user-id/1",
		host:   "repo.prod.meissa.de",
		port:   "",
	}

	if valid, _ := item.IsValid(); !valid {
		t.Errorf("Actor was invalid with valid input.")
	}
}

func TestValidateInvalid(t *testing.T) {
	item := emptyMockStar

	actor, _ := ParseActorFromStarActivity(item)

	if valid, _ := actor.IsValid(); valid {
		t.Errorf("Actor was valid with invalid input.")
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
