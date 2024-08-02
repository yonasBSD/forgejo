package user

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	forgefed_model "code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	actorID := "https://example.com/foo"

	ctx := db.DefaultContext

	federationHost := forgefed_model.FederationHost{
		HostFqdn: "example.com",
		NodeInfo: forgefed_model.NodeInfo{
			SoftwareName: "forgejo",
		},
	}
	require.NoError(t, forgefed_model.CreateFederationHost(ctx, &federationHost))

	user := User{
		LowerName:                    "foo",
		Name:                         "foo",
		FullName:                     "foo",
		Email:                        "foo@example.com",
		EmailNotificationsPreference: "disabled",
		Passwd:                       "foo",
		MustChangePassword:           false,
		LoginName:                    "foo",
		Type:                         UserTypeRemoteUser,
		IsAdmin:                      false,
		NormalizedFederatedURI:       actorID,
	}
	federatedUser := FederatedUser{
		ExternalID:       actorID,
		FederationHostID: federationHost.ID,
	}
	require.NoError(t, CreateFederatedUser(ctx, &user, &federatedUser))

	u, fedU, _ := FindFederatedUser(ctx, actorID, federationHost.ID)

	assert.Equal(t, u.Name, user.Name)
	assert.Equal(t, u.ID, fedU.UserID)
	assert.Equal(t, fedU.ExternalID, federatedUser.ExternalID)

	users, _ := FindFederatedUsers(ctx, 0)
	assert.Len(t, users, 1)
	var foundUser FederatedUser
	for _, v := range users {
		if v.UserID == u.ID {
			foundUser = *v
		}
	}

	assert.Equal(t, foundUser, federatedUser)
}
