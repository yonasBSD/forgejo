package activitypub

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/validation"
)

type Validatable interface { // ToDo: What is the right package for this interface?
	validate_is_not_nil() error
	validate_is_not_empty() error
	Validate() error
	IsValid() (bool, error)
	PanicIfInvalid()
}

type ActorId struct {
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
func (a ActorId) Validate() []string {

	var err = []string{}

	if res := validation.ValidateNotEmpty(a.schema, "schema"); res != nil {
		err = append(err, res.Error())
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
func (a ActorId) IsValid() (bool, error) {
	if err := a.Validate(); len(err) > 0 {
		errString := strings.Join(err, "\n")
		return false, fmt.Errorf(errString)
	}
	return true, nil
}

func (a ActorId) PanicIfInvalid() {
	if valid, err := a.IsValid(); !valid {
		panic(err)
	}
}

func (a ActorId) GetUserId() int {
	result, err := strconv.Atoi(a.userId)

	if err != nil {
		panic(err)
	}

	return result
}

func (a ActorId) GetNormalizedUri() string {
	result := fmt.Sprintf("%s://%s:%s/%s/%s", a.schema, a.host, a.port, a.path, a.userId)
	return result
}

// Returns the combination of host:port if port exists, host otherwise
func (a ActorId) GetHostAndPort() string {

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

func ValidateAndParseIRI(unvalidatedIRI string) (url.URL, error) { // ToDo: Validate that it is not the same host as ours.
	err := validate_is_not_empty(unvalidatedIRI) // url.Parse seems to accept empty strings?
	if err != nil {
		return url.URL{}, err
	}

	validatedURL, err := url.Parse(unvalidatedIRI)
	if err != nil {
		return url.URL{}, err
	}

	if len(validatedURL.Path) <= 1 {
		return url.URL{}, fmt.Errorf("path was empty")
	}

	return *validatedURL, nil
}

// TODO: This parsing is very Person-Specific. We should adjust the name & move to a better location (maybe forgefed package?)
func ParseActorID(validatedURL url.URL, source string) ActorId { // ToDo: Turn this into a factory function and do not split parsing and validation rigurously

	pathWithUserID := strings.Split(validatedURL.Path, "/")

	if containsEmptyString(pathWithUserID) {
		pathWithUserID = removeEmptyStrings(pathWithUserID)
	}

	length := len(pathWithUserID)
	pathWithoutUserID := strings.Join(pathWithUserID[0:length-1], "/")
	userId := pathWithUserID[length-1]

	log.Info("Actor: pathWithUserID: %s", pathWithUserID)
	log.Info("Actor: pathWithoutUserID: %s", pathWithoutUserID)
	log.Info("Actor: UserID: %s", userId)

	return ActorId{ // ToDo: maybe keep original input to validate against (maybe extra method)
		userId: userId,
		source: source,
		schema: validatedURL.Scheme,
		host:   validatedURL.Hostname(), // u.Host returns hostname:port
		path:   pathWithoutUserID,
		port:   validatedURL.Port(),
	}
}

func NewActorId(uri string, source string) (ActorId, error) {
	if !validation.IsValidExternalURL(uri) {
		return ActorId{}, fmt.Errorf("uri %s is not a valid external url", uri)
	}
	return ActorId{}, nil
}
