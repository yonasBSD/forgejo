package activitypub

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/log"
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
	source string
	schema string
	path   string
	host   string
	port   string // optional
}

func validate_is_not_empty(str string) error {
	if str == "" {
		return fmt.Errorf("the given string was empty")
	}
	return nil
}

/*
Validate collects error strings in a slice and returns this
*/
func (a ActorID) Validate() []string {

	var err = []string{}

	if res := validate_is_not_empty(a.schema); res != nil {
		err = append(err, strings.Join([]string{res.Error(), "for schema field"}, " "))
	}

	if res := validate_is_not_empty(a.host); res != nil {
		err = append(err, strings.Join([]string{res.Error(), "for host field"}, " "))
	}

	switch a.source {
	case "forgejo", "gitea":
		if !strings.Contains(a.path, "api/v1/activitypub/user-id") &&
			!strings.Contains(a.path, "api/v1/activitypub/repository-id") {
			err = append(err, fmt.Errorf("the Path to the API was invalid: ---%v---", a.path).Error())
		}
	default:
		err = append(err, fmt.Errorf("currently only forgeo and gitea sources are allowed from actor id").Error())
	}

	return err

}

/*
IsValid concatenates the error messages with newlines and returns them if there are any
*/
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

func (a ActorID) GetNormalizedUri() string {
	result := fmt.Sprintf("%s://%s:%s/%s/%s", a.schema, a.host, a.port, a.path, a.userId)
	return result
}

// Returns the combination of host:port if port exists, host otherwise
func (a ActorID) GetHostAndPort() string {

	if a.port != "" {
		return strings.Join([]string{a.host, a.port}, ":")
	}

	return a.host
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

// TODO: This parsing is very Person-Specific. We should adjust the name & move to a better location (maybe forgefed package?)
func ParseActorID(unvalidatedIRI, source string) (ActorID, error) {
	if unvalidatedIRI == "" {
		return ActorID{}, fmt.Errorf("the given IRI was empty")
	}

	u, err := url.Parse(unvalidatedIRI)

	// check if userID IRI is well formed url
	if err != nil {
		return ActorID{}, fmt.Errorf("the actor ID was not a valid IRI: %v", err)
	}

	pathWithUserID := strings.Split(u.Path, "/")

	if containsEmptyString(pathWithUserID) {
		pathWithUserID = removeEmptyStrings(pathWithUserID)
	}

	length := len(pathWithUserID)
	pathWithoutUserID := strings.Join(pathWithUserID[0:length-1], "/")
	userId := pathWithUserID[length-1]

	log.Info("Actor: pathWithUserID: %s", pathWithUserID)
	log.Info("Actor: pathWithoutUserID: %s", pathWithoutUserID)
	log.Info("Actor: UserID: %s", userId)

	return ActorID{ // ToDo: maybe keep original input to validate against (maybe extra method)
		userId: userId,
		source: source,
		schema: u.Scheme,
		host:   u.Hostname(), // u.Host returns hostname:port
		path:   pathWithoutUserID,
		port:   u.Port(),
	}, nil
}
