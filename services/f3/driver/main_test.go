// SPDX-License-Identifier: MIT

package driver

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	config_types "lab.forgefriends.org/friendlyforgeformat/gof3/config/types"
	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/tests"
)

func TestForgeMethods(t *testing.T) {
	unittest.PrepareTestEnv(t)
	tests.TestForgeMethods(t, NewTestForgejo)
}

type forgejoInstance struct {
	tests.ForgeInstance
}

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", "..", ".."),
	})
}

func (o *forgejoInstance) Init(t tests.TestingT) {
	g := &Forgejo{}
	o.ForgeInstance.Init(t, g)

	doer, err := user_model.GetAdminUser(o.Ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %v", err))
	}

	options := &Options{
		Options: config_types.Options{
			Configuration: config_types.Configuration{
				Type: strings.ToLower(Name),
			},
			Features: config_types.AllFeatures,
			Logger:   ToF3Logger(nil),
		},
		Doer: doer,
	}
	options.SetDefaults()
	g.Init(options)
}

func NewTestForgejo(t tests.TestingT) tests.ForgeTestInterface {
	o := forgejoInstance{}
	o.Init(t)
	return &o
}
