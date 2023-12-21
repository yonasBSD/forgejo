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

type Validateable interface {
	Validate() []string
}

type ActorId struct {
	Validateable
	Id               string
	Source           string
	Schema           string
	Path             string
	Host             string
	Port             string
	UnvalidatedInput string
}

type PersonId struct {
	ActorId
}

type RepositoryId struct {
	ActorId
}

func newActorId(uri, source string) (ActorId, error) {
	validatedUri, _ := url.Parse(uri) // ToDo: Why no err treatment at this place?
	pathWithActorID := strings.Split(validatedUri.Path, "/")
	if containsEmptyString(pathWithActorID) {
		pathWithActorID = removeEmptyStrings(pathWithActorID)
	}
	length := len(pathWithActorID)
	pathWithoutActorID := strings.Join(pathWithActorID[0:length-1], "/")
	id := pathWithActorID[length-1]

	result := ActorId{}
	result.Id = id
	result.Source = source
	result.Schema = validatedUri.Scheme
	result.Host = validatedUri.Hostname()
	result.Path = pathWithoutActorID
	result.Port = validatedUri.Port()
	result.UnvalidatedInput = uri

	if valid, err := result.IsValid(); !valid {
		return ActorId{}, err
	}

	return result, nil
}

func NewPersonId(uri string, source string) (PersonId, error) {
	// TODO: remove after test
	//if !validation.IsValidExternalURL(uri) {
	//	return PersonId{}, fmt.Errorf("uri %s is not a valid external url", uri)
	//}

	actorId, err := newActorId(uri, source)
	if err != nil {
		return PersonId{}, err
	}

	// validate Person specific path
	personId := PersonId{actorId}
	if valid, outcome := personId.IsValid(); !valid {
		return PersonId{}, outcome
	}

	return personId, nil
}

// TODO: tbd how an which parts can be generalized
func NewRepositoryId(uri string, source string) (RepositoryId, error) {
	if !validation.IsAPIURL(uri) {
		return RepositoryId{}, fmt.Errorf("uri %s is not a valid repo url on this host %s", uri, setting.AppURL+"api")
	}

	actorId, err := newActorId(uri, source)
	if err != nil {
		return RepositoryId{}, err
	}

	// validate Person specific path
	repoId := RepositoryId{actorId}
	if valid, outcome := repoId.IsValid(); !valid {
		return RepositoryId{}, outcome
	}

	return repoId, nil
}

func (id ActorId) AsUri() string {
	result := ""
	if id.Port == "" {
		result = fmt.Sprintf("%s://%s/%s/%s", id.Schema, id.Host, id.Path, id.Id)
	} else {
		result = fmt.Sprintf("%s://%s:%s/%s/%s", id.Schema, id.Host, id.Port, id.Path, id.Id)
	}
	return result
}

func (id PersonId) AsWebfinger() string {
	result := fmt.Sprintf("@%s@%s", strings.ToLower(id.Id), strings.ToLower(id.Host))
	return result
}

func (id PersonId) AsLoginName() string {
	result := fmt.Sprintf("%s%s", strings.ToLower(id.Id), id.HostSuffix())
	return result
}

func (id PersonId) HostSuffix() string {
	result := fmt.Sprintf("-%s", strings.ToLower(id.Host))
	return result
}

/*
Validate collects error strings in a slice and returns this
*/
func (value ActorId) Validate() []string {
	var result = []string{}
	result = append(result, validation.ValidateNotEmpty(value.Id, "userId")...)
	result = append(result, validation.ValidateNotEmpty(value.Source, "source")...)
	result = append(result, validation.ValidateNotEmpty(value.Schema, "schema")...)
	result = append(result, validation.ValidateNotEmpty(value.Path, "path")...)
	result = append(result, validation.ValidateNotEmpty(value.Host, "host")...)
	result = append(result, validation.ValidateNotEmpty(value.UnvalidatedInput, "unvalidatedInput")...)
	result = append(result, validation.ValidateOneOf(value.Source, []string{"forgejo", "gitea"})...)

	if value.UnvalidatedInput != value.AsUri() {
		result = append(result, fmt.Sprintf("not all input: %q was parsed: %q", value.UnvalidatedInput, value.AsUri()))
	}

	return result
}

// TODO: Move valid-parts to valid package
/*
IsValid concatenates the error messages with newlines and returns them if there are any
*/
func (a ActorId) IsValid() (bool, error) {
	if err := a.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}

	return true, nil
}

func (a PersonId) IsValid() (bool, error) {
	switch a.Source {
	case "forgejo", "gitea":
		if strings.ToLower(a.Path) != "api/v1/activitypub/user-id" && strings.ToLower(a.Path) != "api/activitypub/user-id" {
			err := fmt.Errorf("path: %q has to be an api path", a.Path)
			return false, err
		}
	}
	return true, nil
}

func (a RepositoryId) IsValid() (bool, error) {
	switch a.Source {
	case "forgejo", "gitea":
		if strings.ToLower(a.Path) != "api/v1/activitypub/repository-id" && strings.ToLower(a.Path) != "api/activitypub/repository-id" {
			err := fmt.Errorf("path: %q has to be an api path", a.Path)
			return false, err
		}
	}
	return true, nil
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
