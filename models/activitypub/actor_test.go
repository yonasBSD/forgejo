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

var invalidMockStar *forgefed.Star = &forgefed.Star{
	Source: "",
	Activity: ap.Activity{
		Actor:  ap.IRI(""),
		Type:   "Star",
		Object: ap.IRI("https://example.com/"),
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

func TestValidateAndParseIRIEmpty(t *testing.T) {
	item := emptyMockStar.Object.GetLink().String()

	_, err := ValidateAndParseIRI(item)

	if err == nil {
		t.Errorf("ValidateAndParseIRI returned no error for empty input.")
	}
}

func TestValidateAndParseIRINoPath(t *testing.T) {
	item := emptyMockStar.Object.GetLink().String()

	_, err := ValidateAndParseIRI(item)

	if err == nil {
		t.Errorf("ValidateAndParseIRI returned no error for empty path.")
	}
}

func TestActorParserValid(t *testing.T) {
	item, _ := ValidateAndParseIRI(mockStar.Actor.GetID().String())
	want := PersonId{
		userId: "1",
		source: "forgejo",
		schema: "https",
		path:   "api/v1/activitypub/user-id",
		host:   "repo.prod.meissa.de",
		port:   "",
	}

	got := ParseActorID(item, "forgejo")

	if got != want {
		t.Errorf("\nParseActorID did not return want: %v\n but %v", want, got)
	}
}

func TestValidateValid(t *testing.T) {
	item := PersonId{
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
	item, _ := ValidateAndParseIRI("https://example.org/some-path/to/nowhere/")

	actor := ParseActorID(item, "forgejo")

	if valid, _ := actor.IsValid(); valid {
		t.Errorf("Actor was valid with invalid input.")
	}
}

func TestGetHostAndPort(t *testing.T) {
	item := PersonId{
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

func TestShouldThrowErrorOnInvalidInput(t *testing.T) {
	_, err := NewPersonId("", "forgejo")
	if err == nil {
		t.Errorf("empty input should be invalid.")
	}

	_, err = NewPersonId("http://localhost:3000/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("localhost uris are not external")
	}
	_, err = NewPersonId("./api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("relative uris are not alowed")
	}
	_, err = NewPersonId("http://1.2.3.4/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("uri may not be ip-4 based")
	}
	_, err = NewPersonId("http:///[fe80::1ff:fe23:4567:890a%25eth0]/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("uri may not be ip-6 based")
	}
	_, err = NewPersonId("https://codeberg.org/api/v1/activitypub/../activitypub/user-id/12345", "forgejo")
	if err == nil {
		t.Errorf("uri may not contain relative path elements")
	}

	_, err = NewPersonId("https://an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if err != nil {
		t.Errorf("this uri should be valid but was: %v", err)
	}
}
