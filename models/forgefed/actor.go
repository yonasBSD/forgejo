// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/validation"
)

type Validateables interface {
	validation.Validateable
	ActorID | PersonID | RepositoryID
}

type ActorID struct {
	ID               string
	Source           string
	Schema           string
	Path             string
	Host             string
	Port             string
	UnvalidatedInput string
}

type PersonID struct {
	ActorID
}

type RepositoryID struct {
	ActorID
}

// newActorID receives already validated inputs
func newActorID(validatedURI *url.URL, source string) (ActorID, error) {
	pathWithActorID := strings.Split(validatedURI.Path, "/")
	if containsEmptyString(pathWithActorID) {
		pathWithActorID = removeEmptyStrings(pathWithActorID)
	}
	length := len(pathWithActorID)
	pathWithoutActorID := strings.Join(pathWithActorID[0:length-1], "/")
	id := pathWithActorID[length-1]

	result := ActorID{}
	result.ID = id
	result.Source = source
	result.Schema = validatedURI.Scheme
	result.Host = validatedURI.Hostname()
	result.Path = pathWithoutActorID
	result.Port = validatedURI.Port()
	result.UnvalidatedInput = validatedURI.String()

	if valid, err := IsValid(result); !valid {
		return ActorId{}, err
	}

	return result, nil
}

func NewPersonID(uri string, source string) (PersonID, error) {
	// TODO: remove after test
	//if !validation.IsValidExternalURL(uri) {
	//	return PersonId{}, fmt.Errorf("uri %s is not a valid external url", uri)
	//}
	validatedURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return PersonID{}, err
	}

	actorID, err := newActorID(validatedURI, source)
	if err != nil {
		return PersonID{}, err
	}

	// validate Person specific path
	personID := PersonID{actorID}
	if valid, outcome := validation.IsValid(personID); !valid {
		return PersonID{}, outcome
	}

	return personID, nil
}

func NewRepositoryId(uri string, source string) (RepositoryID, error) {

	if !validation.IsAPIURL(uri) {
		return RepositoryID{}, fmt.Errorf("uri %s is not a valid repo url on this host %s", uri, setting.AppURL+"api")
	}

	validatedURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return RepositoryID{}, err
	}

	actorID, err := newActorID(validatedURI, source)
	if err != nil {
		return RepositoryID{}, err
	}

	// validate Person specific path
	repoID := RepositoryID{actorID}
	if valid, outcome := validation.IsValid(repoID); !valid {
		return RepositoryID{}, outcome
	}

	return repoID, nil
}

func (id ActorID) AsURI() string {
	var result string
	if id.Port == "" {
		result = fmt.Sprintf("%s://%s/%s/%s", id.Schema, id.Host, id.Path, id.ID)
	} else {
		result = fmt.Sprintf("%s://%s:%s/%s/%s", id.Schema, id.Host, id.Port, id.Path, id.ID)
	}
	return result
}

func (id PersonID) AsWebfinger() string {
	result := fmt.Sprintf("@%s@%s", strings.ToLower(id.ID), strings.ToLower(id.Host))
	return result
}

func (id PersonID) AsLoginName() string {
	result := fmt.Sprintf("%s%s", strings.ToLower(id.ID), id.HostSuffix())
	return result
}

func (id PersonID) HostSuffix() string {
	result := fmt.Sprintf("-%s", strings.ToLower(id.Host))
	return result
}

// Validate collects error strings in a slice and returns this
func (id ActorID) Validate() []string {
	var result = []string{}
	result = append(result, validation.ValidateNotEmpty(id.ID, "userId")...)
	result = append(result, validation.ValidateNotEmpty(id.Source, "source")...)
	result = append(result, validation.ValidateNotEmpty(id.Schema, "schema")...)
	result = append(result, validation.ValidateNotEmpty(id.Path, "path")...)
	result = append(result, validation.ValidateNotEmpty(id.Host, "host")...)
	result = append(result, validation.ValidateNotEmpty(id.UnvalidatedInput, "unvalidatedInput")...)
	result = append(result, validation.ValidateOneOf(id.Source, []string{"forgejo", "gitea"})...)

	if id.UnvalidatedInput != id.AsURI() {
		result = append(result, fmt.Sprintf("not all input: %q was parsed: %q", id.UnvalidatedInput, id.AsURI()))
	}

	return result
}

func (id PersonID) Validate() []string {
	var result = id.ActorID.Validate()
	switch id.Source {
	case "forgejo", "gitea":
		if strings.ToLower(id.Path) != "api/v1/activitypub/user-id" && strings.ToLower(id.Path) != "api/activitypub/user-id" {
			result = append(result, fmt.Sprintf("path: %q has to be a person specific api path", id.Path))
		}
	}
	return result
}

func (id RepositoryID) Validate() []string {
	var result = id.ActorID.Validate()
	switch id.Source {
	case "forgejo", "gitea":
		if strings.ToLower(id.Path) != "api/v1/activitypub/repository-id" && strings.ToLower(id.Path) != "api/activitypub/repository-id" {
			result = append(result, fmt.Sprintf("path: %q has to be a repo specific api path", id.Path))
		}
	}
	return result
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

func IsValid[T Validateables](value T) (bool, error) {
	if err := value.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}
	return true, nil
}

/*
func (a RepositoryId) IsValid() (bool, error) {
	if err := a.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}

	return true, nil
}

func (a PersonId) IsValid() (bool, error) {
	if err := a.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}

	return true, nil
}
*/
