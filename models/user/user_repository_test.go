package user

import (
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	forgefed_model "code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/log"

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

	assert.Equal(t, u.ID, fedU.UserID)
	assert.Equal(t, fedU.ID, federatedUser.ID)

	users, _ := FindFederatedUsers(ctx, 0)
	assert.Equal(t, len(users), 1)
	var foundUser *FederatedUser
	for _, v := range users {
		if v.UserID == u.ID {
			foundUser = v
		}
	}

	assert.Equal(t, foundUser.ID, federatedUser.ID)
}

func TestUserCleanUp(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ctx := db.DefaultContext

	federationHost := forgefed_model.FederationHost{
		HostFqdn: "testusercleanup.example.com",
		NodeInfo: forgefed_model.NodeInfo{
			SoftwareName: "forgejo",
		},
	}
	require.NoError(t, forgefed_model.CreateFederationHost(ctx, &federationHost))

	user := User{
		LowerName:                    "ffoo",
		Name:                         "ffoo",
		FullName:                     "ffoo",
		Email:                        "ffoo@example.com",
		EmailNotificationsPreference: "disabled",
		Passwd:                       "ffoo",
		MustChangePassword:           false,
		LoginName:                    "ffoo",
		NumFollowers:                 1,
		Type:                         UserTypeRemoteUser,
		IsAdmin:                      false,
		NormalizedFederatedURI:       "https://testusercleanup.example.com/foo",
	}
	federatedUser := FederatedUser{
		ExternalID:       user.NormalizedFederatedURI,
		FederationHostID: federationHost.ID,
	}
	require.NoError(t, CreateFederatedUser(ctx, &user, &federatedUser))

	userNoFollower := User{
		LowerName:                    "bar",
		Name:                         "bar",
		FullName:                     "bar",
		Email:                        "bar@example.com",
		EmailNotificationsPreference: "disabled",
		Passwd:                       "bar",
		MustChangePassword:           false,
		LoginName:                    "bar",
		Type:                         UserTypeRemoteUser,
		IsAdmin:                      false,
		NormalizedFederatedURI:       "https://testusercleanup.example.com/bar",
	}
	federatedUserNoFollower := FederatedUser{
		ExternalID:       userNoFollower.NormalizedFederatedURI,
		FederationHostID: federationHost.ID,
	}
	require.NoError(t, CreateFederatedUser(ctx, &userNoFollower, &federatedUserNoFollower))

	userNoFollowerNoStar := User{
		LowerName:                    "baz",
		Name:                         "baz",
		FullName:                     "baz",
		Email:                        "baz@example.com",
		EmailNotificationsPreference: "disabled",
		Passwd:                       "baz",
		MustChangePassword:           false,
		LoginName:                    "baz",
		Type:                         UserTypeRemoteUser,
		IsAdmin:                      false,
		NormalizedFederatedURI:       "https://testusercleanup.example.com/baz",
	}
	federatedUserNoFollowerNoStar := FederatedUser{
		ExternalID:       userNoFollowerNoStar.NormalizedFederatedURI,
		FederationHostID: federationHost.ID,
	}
	require.NoError(t, CreateFederatedUser(ctx, &userNoFollowerNoStar, &federatedUserNoFollowerNoStar))

	_, err := db.Exec(ctx, "INSERT INTO star (uid, repo_id) values (?, 1)", userNoFollower.ID)
	if err != nil {
		log.Info("err: %v", err)
		panic(err)
	}

	time.Sleep(time.Second)

	users, _ := FindFederatedUserWithNoFollowersAndNoStars(ctx, time.Microsecond, 0)
	assert.Len(t, users, 1)

	var foundUser *User

	for _, u := range users {
		log.Info("username: %s", u.Name)
		if u.ID == userNoFollowerNoStar.ID {
			foundUser = u
		}
	}

	u, _ := GetUserByID(ctx, userNoFollowerNoStar.ID)

	assert.Equal(t, foundUser, u)
}
