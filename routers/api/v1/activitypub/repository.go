// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/gitea/models/activitypub"
	"code.gitea.io/gitea/models/db"
	api "code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"github.com/google/uuid"

	user_model "code.gitea.io/gitea/models/user"
	ap "github.com/go-ap/activitypub"
	pwd_gen "github.com/sethvargo/go-password/password"
	//f3 "lab.forgefriends.org/friendlyforgeformat/gof3"
)

var actionsUser = user_model.NewActionsUser()

func generateUUIDMail(person ap.Actor) (string, error) {
	// UUID@remote.host
	id := uuid.New().String()

	//url, err := url.Parse(person.URL.GetID().String())

	//host := url.Host

	return strings.Join([]string{id, "example.com"}, "@"), nil
}

func generateRemoteUserName(person ap.Actor) (string, error) {
	u, err := url.Parse(person.URL.GetID().String())
	if err != nil {
		return "", err
	}

	host := strings.Split(u.Host, ":")[0] // no port in username

	if name := person.PreferredUsername.String(); name != "" {
		return strings.Join([]string{name, host}, "_"), nil
	}
	if name := person.Name.String(); name != "" {
		return strings.Join([]string{name, host}, "_"), nil
	}

	return "", fmt.Errorf("empty name, preferredUsername field")
}

func generateRandomPassword() (string, error) {
	// Generate a password that is 64 characters long with 10 digits, 10 symbols,
	// allowing upper and lower case letters, disallowing repeat characters.
	res, err := pwd_gen.Generate(32, 10, 10, false, false)
	if err != nil {
		return "", err
	}
	return res, err
}

func searchUsersByPerson(actorId string) ([]*user_model.User, error) {

	actionsUser.IsAdmin = true

	options := &user_model.SearchUserOptions{
		LoginName:       actorId,
		Actor:           actionsUser,
		Type:            user_model.UserTypeRemoteUser,
		OrderBy:         db.SearchOrderByAlphabetically,
		ListOptions:     db.ListOptions{PageSize: 1},
		IsActive:        util.OptionalBoolFalse,
		IncludeReserved: true,
	}
	users, _, err := user_model.SearchUsers(db.DefaultContext, options)
	if err != nil {
		return []*user_model.User{}, fmt.Errorf("search failed: %v", err)
	}

	log.Info("local found users: %v", len(users))

	return users, nil

}

func getPersonByRest(remoteStargazer, starReceiver string, ctx *context.APIContext) (ap.Person, error) {

	client, err := api.NewClient(ctx, actionsUser, starReceiver) // The star receiver signs the http get request
	if err != nil {
		return ap.Person{}, err
	}

	// get_person_by_rest
	bytes := []byte{0} // no body needed for getting user actor
	response, err := client.Get(bytes, remoteStargazer)
	if err != nil {
		return ap.Person{}, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return ap.Person{}, err
	}

	// parse response
	person := ap.Person{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return ap.Person{}, err
	}

	log.Info("remoteStargazer: %v", remoteStargazer)
	log.Info("http client. %v", client)
	log.Info("response: %v\n error: ", response, err)
	log.Info("Person is: %v", person)
	log.Info("Person Name is: %v", person.PreferredUsername)
	log.Info("Person URL is: %v", person.URL)

	return person, nil

}

// Repository function returns the Repository actor for a repo
func Repository(ctx *context.APIContext) {
	// swagger:operation GET /activitypub/repository-id/{repository-id} activitypub activitypubRepository
	// ---
	// summary: Returns the Repository actor for a repo
	// produces:
	// - application/json
	// parameters:
	// - name: repository-id
	//   in: path
	//   description: repository ID of the repo
	//   type: integer
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActivityPub"

	link := fmt.Sprintf("%s/api/v1/activitypub/repository-id/%d", strings.TrimSuffix(setting.AppURL, "/"), ctx.Repo.Repository.ID)
	repo := forgefed.RepositoryNew(ap.IRI(link))

	repo.Name = ap.NaturalLanguageValuesNew()
	err := repo.Name.Set("en", ap.Content(ctx.Repo.Repository.Name))
	if err != nil {
		ctx.ServerError("Set Name", err)
		return
	}

	response(ctx, repo)
}

// PersonInbox function handles the incoming data for a repository inbox
func RepositoryInbox(ctx *context.APIContext) {
	// swagger:operation POST /activitypub/repository-id/{repository-id}/inbox activitypub activitypubRepository
	// ---
	// summary: Send to the inbox
	// produces:
	// - application/json
	// parameters:
	// - name: repository-id
	//   in: path
	//   description: repository ID of the repo
	//   type: integer
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/Star"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	log.Info("RepositoryInbox: repo %v, %v", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)
	activity := web.GetForm(ctx).(*forgefed.Star)
	log.Info("RepositoryInbox: Activity.Source %v", activity.Source)
	log.Info("RepositoryInbox: Activity.Actor %v", activity.Actor)

	// assume actor is: "actor": "https://codeberg.org/api/v1/activitypub/user-id/12345" - NB: This might be actually the ID? Maybe check vocabulary.
	//    "https://Codeberg.org/api/v1/activitypub/user-id/12345"
	//    "https://codeberg.org:443/api/v1/activitypub/user-id/12345"
	//    "https://codeberg.org/api/v1/activitypub/../activitypub/user-id/12345"
	//    "https://user:password@codeberg.org/api/v1/activitypub/user-id/12345"
	//    "https://codeberg.org/api/v1/activitypub//user-id/12345"

	// parse senderActorId
	// senderActorId holds the data to construct the sender of the star
	log.Info("activity.Actor.GetID().String(): %v", activity.Actor.GetID().String())
	senderActorId, err := activitypub.ParseActorID(activity.Actor.GetID().String(), string(activity.Source))
	if err != nil {
		panic(err)
	}

	receivedRepoId, err := activitypub.ParseActorID(activity.Activity.Object.GetID().String(), string(activity.Source))
	if err != nil {
		panic(err)
	}

	// validate receiverActorId against repo owner
	repositoryID := ctx.Repo.Repository.ID
	if repositoryID != int64(receivedRepoId.GetUserId()) {
		panic(
			fmt.Errorf("received repo id and repo id were not identical:\nreceived id: %v\nrepo id:%v", receivedRepoId, repositoryID))
	}

	// Is the ActorID Struct valid?
	senderActorId.PanicIfInvalid()
	receivedRepoId.PanicIfInvalid()
	log.Info("RepositoryInbox: Actor parsed. %v", senderActorId)
	log.Info("RepositoryInbox: Actor parsed. %v", receivedRepoId)

	remoteStargazer := senderActorId.GetNormalizedUri() // used as LoginName in newly created user
	starReceiver := receivedRepoId.GetNormalizedUri()
	log.Info("remotStargazer: %v", remoteStargazer)
	log.Info("starReceiver: %v", starReceiver)

	// Check if user already exists
	// TODO: If we where able to search for federated id there would be no need to get the remote person.
	//			 N.B. We need the username as a display name from the remote host. This requires us to make another request
	//			 			We might extend the Star Activity by the username, then this request would become redundant
	users, err := searchUsersByPerson(remoteStargazer)
	if err != nil {
		panic(fmt.Errorf("searching for user failed: %v", err))
	}

	if len(users) == 0 {

		person, err := getPersonByRest(remoteStargazer, starReceiver, ctx)

		// create user
		//	ToDo:	We need a remote server with federation enabled to properly test this

		email, err := generateUUIDMail(person)
		if err != nil {
			panic(fmt.Errorf("generate user failed: %v", err))
		}

		username, err := generateRemoteUserName(person)
		if err != nil {
			panic(fmt.Errorf("generate user failed: %v", err))
		}

		password, err := generateRandomPassword()
		if err != nil {
			panic(fmt.Errorf("generate password failed: %v", err))
		}

		user := &user_model.User{
			LowerName:                    strings.ToLower(username),
			Name:                         username,
			Email:                        email,
			EmailNotificationsPreference: "disabled",
			Passwd:                       password,
			MustChangePassword:           false,
			LoginName:                    remoteStargazer,
			Type:                         user_model.UserTypeRemoteUser,
			IsAdmin:                      false,
		}

		overwriteDefault := &user_model.CreateUserOverwriteOptions{
			IsActive:     util.OptionalBoolFalse,
			IsRestricted: util.OptionalBoolFalse,
		}

		if err := user_model.CreateUser(ctx, user, overwriteDefault); err != nil {
			panic(fmt.Errorf("createUser: %w", err))
		}
		log.Info("User created!")
	} else {
		// use first user
		user := users[0]
		log.Info("%v", user)
	}

	// TODO: handle case of count > 1
	// execute star action

	// wait 15 sec.

	ctx.Status(http.StatusNoContent)
}
