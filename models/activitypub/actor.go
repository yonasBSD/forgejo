package activitypub

import (
	"fmt"
	"net/url"
	"strings"
)

type Validatable interface {
	Validate() error
}

type ActorID struct {
	schema string
	userId string
	path   string
	host   string
	port   string // optional
}

// TODO: Align validation-api to example from dda-devops-build
func (a ActorID) Validate() error {

	if a.schema == "" {
		return fmt.Errorf("the actor ID was not valid: Invalid Schema")
	}

	if a.host == "" {
		return fmt.Errorf("the actor ID was not valid: Invalid Host")
	}

	if !strings.Contains(a.path, "api/v1/activitypub/user-id") {
		return fmt.Errorf("the Path to the API was invalid: %v", a.path)
	}

	return nil

}

func ParseActorID(actor string) (ActorID, error) {
	u, err := url.Parse(actor)

	// check if userID IRI is well formed url
	if err != nil {
		return ActorID{}, fmt.Errorf("the actor ID was not a valid IRI: %v", err)
	}

	pathWithUserID := strings.Split(u.Path, "/")
	userId := pathWithUserID[len(pathWithUserID)-1]

	return ActorID{
		schema: u.Scheme,
		userId: userId,
		host:   u.Host,
		path:   u.Path,
		port:   u.Port(),
	}, nil
}
