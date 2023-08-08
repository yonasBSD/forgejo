// SPDX-License-Identifier: MIT

package forgejo

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/forgejo/semver"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/log"

	"github.com/stretchr/testify/assert"
)

func TestForgejo_v1TOv5_0_1Included(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	ctx := db.DefaultContext
	e := db.GetEngine(ctx)

	logFatal = func(string, ...any) {}
	defer func() {
		logFatal = log.Fatal
	}()

	configWithSoragePath := `
[storage]
PATH = /something
`

	for _, testCase := range []struct {
		name   string
		semver string
		config string
	}{
		{
			name:   "5.0.0 with no [storage].PATH",
			semver: "5.0.0+0-gitea-1.20.1",
			config: "",
		},
		{
			name:   "5.0.1 with no [storage].PATH",
			semver: "5.0.1+0-gitea-1.20.2",
			config: "",
		},
		{
			name:   "5.0.2 with [storage].PATH",
			semver: "5.0.2+0-gitea-1.20.3",
			config: configWithSoragePath,
		},
	} {
		cfg := configFixture(t, testCase.config)
		semver.SetVersionString(ctx, testCase.semver)
		assert.NoError(t, v1TOv5_0_1Included(e, cfg))
	}

	for _, testCase := range []struct {
		name   string
		semver string
		config string
	}{
		{
			name:   "5.0.0 with  [storage].PATH",
			semver: "5.0.0+0-gitea-1.20.1",
			config: configWithSoragePath,
		},
		{
			name:   "5.0.1 with [storage].PATH",
			semver: "5.0.1+0-gitea-1.20.2",
			config: configWithSoragePath,
		},
		{
			name:   "4.2.1 with [storage].PATH",
			semver: "4.2.1+0-gitea-1.19.3",
			config: configWithSoragePath,
		},
	} {
		cfg := configFixture(t, testCase.config)
		semver.SetVersionString(ctx, testCase.semver)
		assert.ErrorContains(t, v1TOv5_0_1Included(e, cfg), "[storage].PATH is set")
	}
}
