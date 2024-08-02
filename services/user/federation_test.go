package user

import (
	"bufio"
	"bytes"
	"fmt"
	"image/png"
	"testing"

	"code.gitea.io/gitea/models/db"
	forgefed_model "code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/avatar"
	"code.gitea.io/gitea/modules/setting"

	ap "github.com/go-ap/activitypub"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateFederationUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx := db.DefaultContext
	setting.AppURL = "http://localhost"
	oldUsername := "john-example.com"
	username := "john"
	actorID := "http://example.com/api/v1/activitypub/user-id/1"

	defer gock.Off()

	federationHost := forgefed_model.FederationHost{
		HostFqdn: "example.com",
		NodeInfo: forgefed_model.NodeInfo{
			SoftwareName: "forgejo",
		},
	}
	require.NoError(t, forgefed_model.CreateFederationHost(ctx, &federationHost))

	img, _ := avatar.RandomImage([]byte(actorID))
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	r := bufio.NewReader(&b)
	png.Encode(w, img)
	gock.New("http://example.com").Get("/api/v1/activitypub/user-id/1/avatar").Times(1).Reply(200).Body(r)

	person := ap.PersonNew(ap.IRI(actorID))
	_ = person.PreferredUsername.Set("en", ap.Content(username))

	person.Icon = ap.Image{
		Type:      ap.ImageType,
		MediaType: "image/png",
		URL:       ap.IRI("http://example.com/api/v1/activitypub/user-id/1/avatar"),
	}

	gock.New("http://example.com").
		Get("/api/v1/activitypub/user-id/1").Times(1).Reply(200).JSON(person)

	user := user_model.User{
		LowerName:                    oldUsername,
		Name:                         oldUsername,
		FullName:                     oldUsername,
		Email:                        "foo@example.com",
		EmailNotificationsPreference: "disabled",
		Passwd:                       oldUsername,
		MustChangePassword:           false,
		LoginName:                    oldUsername,
		Type:                         user_model.UserTypeRemoteUser,
		IsAdmin:                      false,
		NormalizedFederatedURI:       actorID,
	}
	federatedUser := user_model.FederatedUser{
		ExternalID:       actorID,
		FederationHostID: federationHost.ID,
	}
	require.NoError(t, user_model.CreateFederatedUser(ctx, &user, &federatedUser))
	assert.Equal(t, federatedUser.UserID, user.ID)
	u, _ := user_model.GetUserByID(ctx, user.ID)
	assert.Equal(t, u.Name, oldUsername)

	require.NoError(t,
		UpdatePersonActor(ctx))

	updatedUser, _, _ := user_model.FindFederatedUser(ctx, actorID, federationHost.ID)
	assert.Equal(t, updatedUser.Name, fmt.Sprintf("@%s@example.com", username))

	DeleteUser(ctx, updatedUser, false)
}
