package activitypub

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/forgefed"
)

type Validatable interface { // ToDo: What is the right package for this interface?
	validate_is_not_nil() error
	validate_is_not_empty() error
	Validate() error
	IsValid() (bool, error)
	PanicIfInvalid()
}

type ActorID struct {
	userId string
	source forgefed.SourceType
	schema string
	path   string
	host   string
	port   string // optional
}

// ToDo: validate_is_not_empty maybe not as an extra method
func (a ActorID) validate_is_not_empty(str string, field string) error {

	if str == "" {
		return fmt.Errorf("field %v was empty", field)
	}

	return nil
}

/*
Validate collects error strings, concatenates and returns them

TODO: Align validation-api to example from dda-devops-build
*/
func (a ActorID) Validate() []string {

	err := make([]string, 0, 3) // ToDo: Solve this dynamically

	if res := a.validate_is_not_empty(a.schema, "schema"); res != nil {
		err = append(err, res.Error())
	}

	if res := a.validate_is_not_empty(a.host, "host"); res != nil {
		err = append(err, res.Error())
	}

	switch a.source {
	case "forgejo", "gitea":
		if !strings.Contains(a.path, "api/v1/activitypub/user-id") {
			err = append(err, fmt.Errorf("the Path to the API was invalid: %v", a.path).Error())
		}
	default:
		err = append(err, fmt.Errorf("currently only forgeo and gitea sources are allowed from actor id").Error())
	}

	return err

}

func (a ActorID) IsValid() (bool, error) {
	if err := a.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}
	return true, nil
}

func (a ActorID) PanicIfInvalid() {
	if valid, err := a.IsValid(); !valid {
		panic(err)
	}
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

func ParseActorFromStarActivity(star *forgefed.Star) (ActorID, error) {
	u, err := url.Parse(star.Actor.GetID().String())

	// check if userID IRI is well formed url
	if err != nil {
		return ActorID{}, fmt.Errorf("the actor ID was not a valid IRI: %v", err)
	}

	pathWithUserID := strings.Split(u.Path, "/")
	userId := pathWithUserID[len(pathWithUserID)-1]

	return ActorID{ // ToDo: maybe keep original input to validate against (maybe extra method)
		userId: userId,
		source: star.Source,
		schema: u.Scheme, // ToDo: Add source type field
		host:   u.Host,
		path:   u.Path,
		port:   u.Port(),
	}, nil
}
