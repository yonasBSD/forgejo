// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"net/url"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/f3/util"

	"github.com/stretchr/testify/assert"
	"lab.forgefriends.org/friendlyforgeformat/gof3"
	f3_forges "lab.forgefriends.org/friendlyforgeformat/gof3/forges"
	f3_common "lab.forgefriends.org/friendlyforgeformat/gof3/forges/common"
	f3_f3 "lab.forgefriends.org/friendlyforgeformat/gof3/forges/f3"
	f3_forgejo "lab.forgefriends.org/friendlyforgeformat/gof3/forges/forgejo"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
	f3_util "lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

func TestF3(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		AllowLocalNetworks := setting.Migrations.AllowLocalNetworks
		setting.F3.Enabled = true
		setting.Migrations.AllowLocalNetworks = true
		AppVer := setting.AppVer
		// Gitea SDK (go-sdk) need to parse the AppVer from server response, so we must set it to a valid version string.
		setting.AppVer = "1.16.0"
		defer func() {
			setting.Migrations.AllowLocalNetworks = AllowLocalNetworks
			setting.AppVer = AppVer
		}()

		//
		// Step 1: create a fixture
		//
		fixtureNewF3Forge := func(t *testing.T, user *format.User, tmpDir string) *f3_forges.ForgeRoot {
			root := f3_forges.NewForgeRoot(&f3_f3.Options{
				Options: gof3.Options{
					Configuration: gof3.Configuration{
						Directory: tmpDir,
					},
					Features: gof3.AllFeatures,
				},
				Remap: true,
			})
			return root
		}
		fixture := f3_forges.NewFixture(t, f3_forges.FixtureForgeFactory{Fun: fixtureNewF3Forge, AdminRequired: false})
		fixture.NewUser(5432)
		fixture.NewMilestone()
		fixture.NewLabel()
		fixture.NewIssue()
		fixture.NewTopic()
		fixture.NewRepository()
		fixture.NewPullRequest()
		fixture.NewRelease()
		fixture.NewAsset()
		fixture.NewIssueComment(nil)
		fixture.NewPullRequestComment()
		fixture.NewReview()
		fixture.NewIssueReaction()
		fixture.NewCommentReaction()

		//
		// Step 2: mirror the fixture into Forgejo
		//
		doer, err := user_model.GetAdminUser()
		assert.NoError(t, err)
		forgejoLocal := util.ForgejoForgeRoot(gof3.AllFeatures, doer)
		options := f3_common.NewMirrorOptionsRecurse()
		forgejoLocal.Forge.Mirror(context.Background(), fixture.Forge, options)

		//
		// Step 3: mirror Forgejo into F3
		//
		adminUsername := "user1"
		forgejoAPI := f3_forges.NewForgeRootFromDriver(&f3_forgejo.Forgejo{}, &f3_forgejo.Options{
			Options: gof3.Options{
				Configuration: gof3.Configuration{
					URL:       setting.AppURL,
					Directory: t.TempDir(),
				},
				Features: gof3.AllFeatures,
			},
			AuthToken: getUserToken(t, adminUsername, auth_model.AccessTokenScopeSudo, auth_model.AccessTokenScopeAll),
		})

		f3 := f3_forges.FixtureNewF3Forge(t, nil, t.TempDir())
		apiForge := forgejoAPI.Forge
		apiUser := apiForge.Users.GetFromFormat(context.Background(), &format.User{UserName: fixture.UserFormat.UserName})
		apiProject := apiUser.Projects.GetFromFormat(context.Background(), &format.Project{Name: fixture.ProjectFormat.Name})
		options = f3_common.NewMirrorOptionsRecurse(apiUser, apiProject)
		f3.Forge.Mirror(context.Background(), apiForge, options)

		//
		// Step 4: verify the fixture and F3 are equivalent
		//
		files := f3_util.Command(context.Background(), "find", f3.GetDirectory())
		assert.Contains(t, files, "/repository/git/hooks")
		assert.Contains(t, files, "/label/")
		assert.Contains(t, files, "/issue/")
		assert.Contains(t, files, "/milestone/")
		assert.Contains(t, files, "/topic/")
		assert.Contains(t, files, "/pull_request/")
		assert.Contains(t, files, "/release/")
		assert.Contains(t, files, "/asset/")
		assert.Contains(t, files, "/comment/")
		assert.Contains(t, files, "/review/")
		assert.Contains(t, files, "/reaction/")
		//		f3_util.Command(context.Background(), "cp", "-a", f3.GetDirectory(), "abc")
	})
}
