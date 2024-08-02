package user

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	forgefed_model "code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/optional"

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

	assert.Equal(t, u, user)
	assert.Equal(t, u.ID, fedU.UserID)
	assert.Equal(t, fedU, federatedUser)

	users, _ := FindFederatedUsers(ctx, 0)
	assert.Equal(t, len(users), 1)
	foundUser := optional.Option[FederatedUser]{}
	for _, v := range users {
		if v.UserID == u.ID {
			foundUser = optional.Option[FederatedUser]{*v}
		}
	}

	assert.Equal(t, foundUser, federatedUser)
}
