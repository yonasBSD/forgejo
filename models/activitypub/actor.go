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

type PersonId struct {
	userId           string
	source           string
	schema           string
	path             string
	host             string
	port             string
	unvalidatedInput string
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

func (a PersonId) GetUserId() int {
	result, err := strconv.Atoi(a.userId)

	if err != nil {
		panic(err)
	}

	return result
}

func (a PersonId) GetNormalizedUri() string {
	result := fmt.Sprintf("%s://%s:%s/%s/%s", a.schema, a.host, a.port, a.path, a.userId)
	return result
}

// Returns the combination of host:port if port exists, host otherwise
func (a PersonId) GetHostAndPort() string {

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
func ParseActorID(validatedURL url.URL, source string) PersonId { // ToDo: Turn this into a factory function and do not split parsing and validation rigurously

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

	return PersonId{ // ToDo: maybe keep original input to validate against (maybe extra method)
		userId: userId,
		source: source,
		schema: validatedURL.Scheme,
		host:   validatedURL.Hostname(), // u.Host returns hostname:port
		path:   pathWithoutUserID,
		port:   validatedURL.Port(),
	}
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
