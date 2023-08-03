// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
	config_types "lab.forgefriends.org/friendlyforgeformat/gof3/config/types"
	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/tests"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
)

type forgejoInstance struct {
	g       *Forgejo
	t       tests.TestingT
	ctx     context.Context
	creator *tests.Creator
}

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", "..", ".."),
	})
}

func newTestForgejo(t tests.TestingT) forgejoInstance {
	g := Forgejo{}

	ctx := context.Background()

	doer, err := user_model.GetAdminUser(ctx)
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

	return forgejoInstance{
		g:       &g,
		t:       t,
		ctx:     ctx,
		creator: tests.NewCreator(t),
	}
}

func (o *forgejoInstance) createUser() (*format.User, *User) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	fixtureUser := o.creator.CreateUser(int64(rand.Uint64()))
	assert.NotNil(o.t, fixtureUser)

	user := &User{}
	user.FromFormat(fixtureUser)
	user.User.MustChangePassword = false
	user.User.LowerName = strings.ToLower(user.User.Name)

	assert.NoError(o.t, db.Insert(o.ctx, user.User))

	return fixtureUser, user
}

func (o *forgejoInstance) createProject(user *User) (*format.Project, *Project) {
	fixtureProject := o.creator.CreateProject(user.ToFormat())
	assert.NotNil(o.t, fixtureProject)

	project := &Project{}
	project.FromFormat(fixtureProject)

	provider := &ProjectProvider{BaseProvider: BaseProvider{g: o.g}}

	return fixtureProject, provider.Put(o.ctx, user, project, nil)
}

type BeanConstraint[Bean any, BeanFormat any, BeanFormatPtr any] interface {
	*Bean
	ToFormat() BeanFormatPtr
	FromFormat(BeanFormatPtr)
}

type BeanFormatConstraint[BeanFormat any] interface {
	*BeanFormat
}

func toFromFormat[
	Bean any,
	BeanFormat any,

	BeanPtr BeanConstraint[Bean, BeanFormat, BeanFormatPtr],
	BeanFormatPtr BeanFormatConstraint[BeanFormat],
](t tests.TestingT, b BeanPtr,
) {
	f := b.ToFormat()
	var otherB Bean
	BeanPtr(&otherB).FromFormat(f)
	assert.EqualValues(t, b, BeanPtr(&otherB))
}
