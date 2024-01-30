// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/tests"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
)

func TestF3_MaybePromoteUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	//
	// OAuth2 authentication source GitLab
	//
	gitlabName := "gitlab"
	_ = addAuthSource(t, authSourcePayloadGitLabCustom(gitlabName))
	//
	// F3 authentication source matching the GitLab authentication source
	//
	f3Name := "f3"
	f3 := createF3AuthSource(t, f3Name, "http://mygitlab.eu", gitlabName)

	//
	// Create a user as if it had been previously been created by the F3
	// authentication source.
	//
	gitlabUserID := "5678"
	gitlabEmail := "gitlabuser@example.com"
	userBeforeSignIn := &user_model.User{
		Name:        "gitlabuser",
		Type:        user_model.UserTypeF3,
		LoginType:   auth_model.F3,
		LoginSource: f3.ID,
		LoginName:   gitlabUserID,
	}
	defer createUser(context.Background(), t, userBeforeSignIn)()

	//
	// A request for user information sent to Goth will return a
	// goth.User exactly matching the user created above.
	//
	defer mockCompleteUserAuth(func(res http.ResponseWriter, req *http.Request) (goth.User, error) {
		return goth.User{
			Provider: gitlabName,
			UserID:   gitlabUserID,
			Email:    gitlabEmail,
		}, nil
	})()
	req := NewRequest(t, "GET", fmt.Sprintf("/user/oauth2/%s/callback?code=XYZ&state=XYZ", gitlabName))
	resp := MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/", test.RedirectURL(resp))
	userAfterSignIn := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userBeforeSignIn.ID})

	// both are about the same user
	assert.Equal(t, userAfterSignIn.ID, userBeforeSignIn.ID)
	// the login time was updated, proof the login succeeded
	assert.Greater(t, userAfterSignIn.LastLoginUnix, userBeforeSignIn.LastLoginUnix)
	// the login type was promoted from F3 to OAuth2
	assert.Equal(t, userBeforeSignIn.LoginType, auth_model.F3)
	assert.Equal(t, userAfterSignIn.LoginType, auth_model.OAuth2)
	// the OAuth2 email was used to set the missing user email
	assert.Equal(t, userBeforeSignIn.Email, "")
	assert.Equal(t, userAfterSignIn.Email, gitlabEmail)
}
