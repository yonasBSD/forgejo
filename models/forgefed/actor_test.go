// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"
)

func TestNewPersonId(t *testing.T) {
	expected := PersonId{
		userId:           "1",
		source:           "forgejo",
		schema:           "https",
		path:             "api/v1/activitypub/user-id",
		host:             "an.other.host",
		port:             "",
		unvalidatedInput: "https://an.other.host/api/v1/activitypub/user-id/1",
	}
	sut, _ := NewPersonId("https://an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}

	expected = PersonId{
		userId:           "1",
		source:           "forgejo",
		schema:           "https",
		path:             "api/v1/activitypub/user-id",
		host:             "an.other.host",
		port:             "443",
		unvalidatedInput: "https://an.other.host:443/api/v1/activitypub/user-id/1",
	}
	sut, _ = NewPersonId("https://an.other.host:443/api/v1/activitypub/user-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}
}

func TestPersonIdValidation(t *testing.T) {
	sut := PersonId{
		source:           "forgejo",
		schema:           "https",
		path:             "api/v1/activitypub/user-id",
		host:             "an.other.host",
		port:             "",
		unvalidatedInput: "https://an.other.host/api/v1/activitypub/user-id/",
	}
	if sut.Validate()[0] != "Field userId may not be empty" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}

	sut = PersonId{
		userId:           "1",
		source:           "forgejox",
		schema:           "https",
		path:             "api/v1/activitypub/user-id",
		host:             "an.other.host",
		port:             "",
		unvalidatedInput: "https://an.other.host/api/v1/activitypub/user-id/1",
	}
	if sut.Validate()[0] != "Value forgejox is not contained in allowed values [[forgejo gitea]]" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}

	sut = PersonId{
		userId:           "1",
		source:           "forgejo",
		schema:           "https",
		path:             "api/v1/activitypub/user-idx",
		host:             "an.other.host",
		port:             "",
		unvalidatedInput: "https://an.other.host/api/v1/activitypub/user-id/1",
	}
	if sut.Validate()[0] != "path has to be a api path" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}

	sut = PersonId{
		userId:           "1",
		source:           "forgejo",
		schema:           "https",
		path:             "api/v1/activitypub/user-id",
		host:             "an.other.host",
		port:             "",
		unvalidatedInput: "https://an.other.host/api/v1/activitypub/user-id/1?illegal=action",
	}
	if sut.Validate()[0] != "not all input: \"https://an.other.host/api/v1/activitypub/user-id/1?illegal=action\" was parsed: \"https://an.other.host/api/v1/activitypub/user-id/1\"" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}
}

func TestWebfingerId(t *testing.T) {
	sut, _ := NewPersonId("https://codeberg.org/api/v1/activitypub/user-id/12345", "forgejo")
	if sut.AsWebfinger() != "@12345@codeberg.org" {
		t.Errorf("wrong webfinger: %v", sut.AsWebfinger())
	}

	sut, _ = NewPersonId("https://Codeberg.org/api/v1/activitypub/user-id/12345", "forgejo")
	if sut.AsWebfinger() != "@12345@codeberg.org" {
		t.Errorf("wrong webfinger: %v", sut.AsWebfinger())
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
	_, err = NewPersonId("https://myuser@an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if err == nil {
		t.Errorf("uri may not contain unparsed elements")
	}

	_, err = NewPersonId("https://an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if err != nil {
		t.Errorf("this uri should be valid but was: %v", err)
	}
}
