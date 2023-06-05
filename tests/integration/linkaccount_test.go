// SPDX-FileCopyrightText: Copyright the Forgejo contributors
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	user_model "code.gitea.io/gitea/models/user"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/tests"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
)

const codebergURL = "https://codeberg.org"

func TestLinkAccountChoose(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	ctx := context.Background()

	// Create a OIDC source and a known OAuth2 source
	codebergName := "codeberg"
	codeberg := addAuthSource(t, authSourcePayloadOIDC(codebergName))
	gitlabName := "gitlab"
	gitlab := addAuthSource(t, authSourcePayloadGitLabCustom(gitlabName))

	//
	// A local user
	//
	localUser := &user_model.User{
		Name:   "linkaccountuser",
		Email:  "linkaccountuser@example.com",
		Passwd: "linkaccountuser",
		Type:   user_model.UserTypeIndividual,
	}
	defer createUser(ctx, t, localUser)()

	//
	// A Codeberg user via OIDC
	//
	userCodebergUserID := "1234"
	userCodeberg := &user_model.User{
		Name:        "linkaccountcodeberguser",
		Email:       "linkaccountcodeberguser@example.com",
		Passwd:      "linkaccountcodeberguser",
		Type:        user_model.UserTypeIndividual,
		LoginType:   auth_model.OAuth2,
		LoginSource: codeberg.ID,
		LoginName:   userCodebergUserID,
	}
	defer createUser(ctx, t, userCodeberg)()

	//
	// A Gitlab user
	//
	userGitLabUserID := "5678"
	userGitLab := &user_model.User{
		Name:        "linkaccountgitlabuser",
		Email:       "linkaccountgitlabuser@example.com",
		Passwd:      "linkaccountgitlabuser",
		Type:        user_model.UserTypeIndividual,
		LoginType:   auth_model.OAuth2,
		LoginSource: gitlab.ID,
		LoginName:   userGitLabUserID,
	}
	defer createUser(ctx, t, userGitLab)()

	defer func() {
		testMiddlewareHook = nil
	}()

	for _, testCase := range []struct {
		title     string
		gothUser  goth.User
		signupTab string
		signinTab string
	}{
		{
			title: "No existing user",
			gothUser: goth.User{
				Provider: codebergName,
			},
			signupTab: "item active",
			signinTab: "item ",
		},
		{
			title: "Matched local user",
			gothUser: goth.User{
				Provider: codebergName,
				Email:    localUser.Email,
			},
			signupTab: "item ",
			signinTab: "item active",
		},
		{
			title: "Matched Codeberg local user",
			gothUser: goth.User{
				Provider: codebergName,
				UserID:   userCodebergUserID,
				Email:    userCodeberg.Email,
			},
			signupTab: "item ",
			signinTab: "item active",
		},
		{
			title: "Matched GitLab local user",
			gothUser: goth.User{
				Provider: gitlabName,
				UserID:   userGitLabUserID,
				Email:    userGitLab.Email,
			},
			signupTab: "item ",
			signinTab: "item active",
		},
	} {
		t.Run(testCase.title, func(t *testing.T) {
			testMiddlewareHook = func(ctx *gitea_context.Context) {
				ctx.Session.Set("linkAccountGothUser", testCase.gothUser)
			}

			req := NewRequest(t, "GET", "/user/link_account")
			resp := MakeRequest(t, req, http.StatusOK)
			if assert.Equal(t, resp.Code, http.StatusOK, resp.Body) {
				doc := NewHTMLParser(t, resp.Body)

				class, exists := doc.Find(`.new-menu-inner .item[data-tab="auth-link-signup-tab"]`).Attr("class")
				assert.True(t, exists, resp.Body)
				assert.Equal(t, testCase.signupTab, class)

				class, exists = doc.Find(`.new-menu-inner .item[data-tab="auth-link-signin-tab"]`).Attr("class")
				assert.True(t, exists)
				assert.Equal(t, testCase.signinTab, class)
			}
		})
	}
}
