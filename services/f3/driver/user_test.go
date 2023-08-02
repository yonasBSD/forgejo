// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/tests"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
)

func TestF3Driver_Users(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	ctx := context.Background()
	tf := newTestForgejo(t)
	provider := &UserProvider{BaseProvider: BaseProvider{g: tf.g}}
	assert.NotNil(t, provider)
	user, deleteUser := tf.createUser(ctx, t)
	defer deleteUser()

	tests.ProviderMethods[User, format.User](tests.ProviderOptions{T: t, Retry: false}, provider, user, nil)
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
