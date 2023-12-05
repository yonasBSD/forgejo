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

func searchUsersByPerson(remoteStargazer string, person ap.Actor) ([]*user_model.User, error) {

	actionsUser.IsAdmin = true

	options := &user_model.SearchUserOptions{
		LoginName:       remoteStargazer,
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
	//    "https:443//codeberg.org/api/v1/activitypub/user-id/12345"
	//    "https://codeberg.org/api/v1/activitypub/../activitypub/user-id/12345"
	// parse actor
	actor, err := activitypub.ParseActorIDFromStarActivity(activity)

	// Is the actor IRI well formed?
	if err != nil {
		panic(err)
	}

	// Is the ActorData Struct valid?
	actor.PanicIfInvalid()

	log.Info("RepositoryInbox: Actor parsed. %v", actor)

	/*
		Make http client, this should make a get request on given url
		We then need to parse the answer and put it into a person-struct
		fill the person struct using some kind of unmarshall function given in
		activitypub package/actor.go
	*/

	// make http client
	// TODO: this should also work without autorizing the api call // doer might be empty
	host := activity.To.GetID().String()
	client, err := api.NewClient(ctx, ctx.Doer, host) // ToDo: This is hacky, we need a hostname from somewhere
	if err != nil {
		panic(err)
	}

	// get_person_by_rest
	bytes := []byte{0}                        // no body needed for getting user actor
	target := activity.Actor.GetID().String() // target is the person actor that originally performed the star activity
	response, err := client.Get(bytes, target)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	// parse response
	person := ap.Person{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		panic(err)
	}

	log.Info("target: %v", target)
	log.Info("http client. %v", client)
	log.Info("response: %v\n error: ", response, err)
	log.Info("Person is: %v", person)
	log.Info("Person Name is: %v", person.PreferredUsername)
	log.Info("Person URL is: %v", person.URL)

	// Check if user already exists
	// TODO: If we where able to search for federated id there would be no need to get the remote person.
	// Search for login_name
	options := &user_model.SearchUserOptions{
		Keyword: target,
		Actor:   ctx.Doer,
		Type:    user_model.UserTypeRemoteUser,
		OrderBy: db.SearchOrderByAlphabetically,
		ListOptions: db.ListOptions{
			Page:     0,
			PageSize: 1,
			ListAll:  true,
		},
	}
	users, usersCount, err := user_model.SearchUsers(db.DefaultContext, options)
	if err != nil {
		fmt.Errorf("Search failed: %v", err)
	}

	log.Info("local found users: %v", usersCount)

	if usersCount == 0 {
		// create user

		/*
			ToDo: Make user

			Fill in user There is a usertype remote in models/user/user.go
			In Location maybe the federated user ID
			isActive to false
			isAdmin to false
			maybe static email as userid@hostname.tld
			- maybe test if we can do user without e-mail
			- then test if two users can have the same adress
			-	otherwise uuid@hostname.tld
			fill in names correctly
			etc

			We need a remote server with federation enabled to test this

			The "if not already present" part might be easy:
			Check the user database for given user id.
			This could happen with something like: user_model.SearchUsers() as seen in routers/api/v1/user.go
			SearchUsers is defined in models/user/search.go
			And depending on implementation check if the person already exists in federated user db.
		*/
		email, err := generateUUIDMail(person)
		if err != nil {
			fmt.Errorf("Generate user failed: %v", err)
		}

		username, err := getUserName(person)
		if err != nil {
			fmt.Errorf("Generate user failed: %v", err)
		}

		password, err := generateRandomPassword()
		if err != nil {
			fmt.Errorf("Generate password failed: %v", err)
		}

		user := &user_model.User{
			LowerName:                    strings.ToLower(username),
			Name:                         username,
			Email:                        email,
			EmailNotificationsPreference: "disabled",
			Passwd:                       password,
			MustChangePassword:           false,
			LoginName:                    target,
			Type:                         user_model.UserTypeRemoteUser,
			IsAdmin:                      false,
		}

		overwriteDefault := &user_model.CreateUserOverwriteOptions{
			IsActive:     util.OptionalBoolFalse,
			IsRestricted: util.OptionalBoolFalse,
		}

		if err := user_model.CreateUser(ctx, user, overwriteDefault); err != nil {
			panic(fmt.Errorf("CreateUser: %w", err))
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
