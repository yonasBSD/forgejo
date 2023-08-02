// SPDX-License-Identifier: MIT

package driver

import (
	"testing"

	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/tests"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
	f3_util "lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

func TestF3Driver_Users(t *testing.T) {
	unittest.PrepareTestEnv(t)

	tf := newTestForgejo(t)
	provider := &UserProvider{BaseProvider: BaseProvider{g: tf.g}}
	fixtureUser := tf.creator.CreateUser(f3_util.RandInt64())

	tests.ProviderMethods[User, format.User](tests.ProviderOptions{T: t}, provider, fixtureUser, nil)
}

func TestF3Driver_UserFormat(t *testing.T) {
	user := User{
		User: user_model.User{
			ID:       1234,
			Type:     user_model.UserTypeF3,
			Name:     "username",
			FullName: "User Name",
			Email:    "username@example.com",
		},
	}
	toFromFormat[User, format.User, *User, *format.User](t, &user)
}
