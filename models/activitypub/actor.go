// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/validation"
)

type PersonId struct {
	userId           string
	source           string
	schema           string
	path             string
	host             string
	port             string
	unvalidatedInput string
}

func NewPersonId(uri string, source string) (PersonId, error) {
	if !validation.IsValidExternalURL(uri) {
		return PersonId{}, fmt.Errorf("uri %s is not a valid external url", uri)
	}

	validatedUri, _ := url.Parse(uri)
	pathWithUserID := strings.Split(validatedUri.Path, "/")

	if containsEmptyString(pathWithUserID) {
		pathWithUserID = removeEmptyStrings(pathWithUserID)
	}

	length := len(pathWithUserID)
	pathWithoutUserID := strings.Join(pathWithUserID[0:length-1], "/")
	userId := pathWithUserID[length-1]

	actorId := PersonId{
		userId:           userId,
		source:           source,
		schema:           validatedUri.Scheme,
		host:             validatedUri.Hostname(),
		path:             pathWithoutUserID,
		port:             validatedUri.Port(),
		unvalidatedInput: uri,
	}
	if valid, err := actorId.IsValid(); !valid {
		return PersonId{}, err
	}

	return actorId, nil
}

func (id PersonId) AsUri() string {
	result := ""
	if id.port == "" {
		result = fmt.Sprintf("%s://%s/%s/%s", id.schema, id.host, id.path, id.userId)
	} else {
		result = fmt.Sprintf("%s://%s:%s/%s/%s", id.schema, id.host, id.port, id.path, id.userId)
	}
	return result
}

func (id PersonId) AsWebfinger() string {
	result := fmt.Sprintf("@%s@%s", strings.ToLower(id.userId), strings.ToLower(id.host))
	return result
}

/*
Validate collects error strings in a slice and returns this
*/
func (value PersonId) Validate() []string {
	var result = []string{}
	result = append(result, validation.ValidateNotEmpty(value.userId, "userId")...)
	result = append(result, validation.ValidateNotEmpty(value.source, "source")...)
	result = append(result, validation.ValidateNotEmpty(value.schema, "schema")...)
	result = append(result, validation.ValidateNotEmpty(value.path, "path")...)
	result = append(result, validation.ValidateNotEmpty(value.host, "host")...)
	result = append(result, validation.ValidateNotEmpty(value.unvalidatedInput, "unvalidatedInput")...)

	result = append(result, validation.ValidateOneOf(value.source, []string{"forgejo", "gitea"})...)
	switch value.source {
	case "forgejo", "gitea":
		if !strings.Contains(value.path, "api/v1/activitypub/user-id") {
			result = append(result, fmt.Sprintf("path has to be a api path"))
		}
	}
	if value.unvalidatedInput != value.AsUri() {
		result = append(result, fmt.Sprintf("not all input: %q was parsed: %q", value.unvalidatedInput, value.AsUri()))
	}

	return result
}

/*
IsValid concatenates the error messages with newlines and returns them if there are any
*/
func (a PersonId) IsValid() (bool, error) {
	if err := a.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}
	return true, nil
}

func (a PersonId) PanicIfInvalid() {
	if valid, err := a.IsValid(); !valid {
		panic(err)
	}
}

func containsEmptyString(ar []string) bool {
	for _, elem := range ar {
		if elem == "" {
			return true
		}
	}
	return false
}

func removeEmptyStrings(ls []string) []string {
	var rs []string
	for _, str := range ls {
		if str != "" {
			rs = append(rs, str)
		}
	}
	return rs
}
