// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"net/url"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/services/migrations"

	"github.com/stretchr/testify/assert"
	f3_forges "lab.forgefriends.org/friendlyforgeformat/gof3/forges"
	f3_util "lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

func Test_CmdF3(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		defer test.MockVariable(&setting.F3.Enabled, true)()
		defer test.MockVariable(&setting.Migrations.AllowLocalNetworks, true)()
		// Gitea SDK (go-sdk) need to parse the AppVer from server response, so we must set it to a valid version string.
		defer test.MockVariable(&setting.AppVer, "1.16.0")
		// without migrations.Init() AllowLocalNetworks = true is not effective and
		// a http call fails with "...migration can only call allowed HTTP servers..."
		migrations.Init()

		//
		// Step 1: create a fixture
		//
		fixture := f3_forges.NewFixture(t, f3_forges.FixtureF3Factory)
		fixture.NewUser(1234)
		fixture.NewMilestone()
		fixture.NewLabel()
		fixture.NewIssue()
		fixture.NewTopic()
		fixture.NewRepository()
		fixture.NewRelease()
		fixture.NewAsset()
		fixture.NewIssueComment(nil)
		fixture.NewIssueReaction()

		//
		// Step 2: import the fixture into Gitea
		//
		{
			output, err := cmdForgejoCaptureOutput(t, []string{"forgejo", "forgejo-cli", "f3", "--import", "--directory", fixture.ForgeRoot.GetDirectory()})
			assert.NoError(t, err)
			assert.EqualValues(t, "imported\n", output)
		}

		//
		// Step 3: export Gitea into F3
		//
		directory := t.TempDir()
		{
			output, err := cmdForgejoCaptureOutput(t, []string{"forgejo", "forgejo-cli", "f3", "--export", "--no-pull-request", "--user", fixture.UserFormat.UserName, "--repository", fixture.ProjectFormat.Name, "--directory", directory})
			assert.NoError(t, err)
			assert.EqualValues(t, "exported\n", output)
		}

		//
		// Step 4: verify the export and import are equivalent
		//
		files := f3_util.Command(context.Background(), "find", directory)
		assert.Contains(t, files, "/label/")
		assert.Contains(t, files, "/issue/")
		assert.Contains(t, files, "/milestone/")
		assert.Contains(t, files, "/topic/")
		assert.Contains(t, files, "/release/")
		assert.Contains(t, files, "/asset/")
		assert.Contains(t, files, "/reaction/")
	})
}
