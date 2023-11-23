package activitypub

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Validatable interface { // ToDo: What is the right package for this interface?
	validate_is_not_nil() error
	validate_is_not_empty() error
	Validate() error
}

type ActorID struct {
	schema string
	userId string
	path   string
	host   string
	port   string // optional
}

func (a ActorID) validate_is_not_empty(str string, field string) error {

	if str == "" {
		return fmt.Errorf("field %v was empty", field)
	}

	return nil
}

func (a ActorID) GetUserId() int {
	result, err := strconv.Atoi(a.userId)

	if err != nil {
		panic(err)
	}

	return result
}

// Returns the combination of host:port if port exists, host otherwise
func (a ActorID) GetHostAndPort() string {

	if a.port != "" {
		return strings.Join([]string{a.host, a.port}, ":")
	}

	return a.host
}

// TODO: Align validation-api to example from dda-devops-build
func (a ActorID) Validate() error {

	if err := a.validate_is_not_empty(a.schema, "schema"); err != nil {
		return err
	}

	if err := a.validate_is_not_empty(a.host, "host"); err != nil {
		return err
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
