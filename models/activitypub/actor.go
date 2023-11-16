package activitypub

import (
	"fmt"
	"net/url"
	"strings"
)

type ActorData struct {
	schema string
	userId string
	path   string
	host   string
	port   string // optional
}

func (a ActorData) ValidateActor() error {

	if a.schema == "" || a.host == "" {
		return fmt.Errorf("the actor ID was not valid: Invalid Schema or Host")
	}

	if !strings.Contains(a.path, "api/v1/activitypub/user-id") {
		return fmt.Errorf("the Path to the API was invalid: %v", a.path)
	}

	return nil

}

func ParseActor(actor string) (ActorData, error) {
	u, err := url.Parse(actor)

	// check if userID IRI is well formed url
	if err != nil {
		return ActorData{}, fmt.Errorf("the actor ID was not a valid IRI: %v", err)
	}

	pathWithUserID := strings.Split(u.Path, "/")
	userId := pathWithUserID[len(pathWithUserID)-1]

	return ActorData{
		schema: u.Scheme,
		userId: userId,
		host:   u.Host,
		path:   u.Path,
		port:   u.Port(),
	}, nil
}
