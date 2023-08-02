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
	user_service "code.gitea.io/gitea/services/user"

	"github.com/stretchr/testify/assert"
	config_types "lab.forgefriends.org/friendlyforgeformat/gof3/config/types"
	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/tests"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
)

type forgejoInstance struct {
	g       *Forgejo
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
		creator: tests.NewCreator(t),
	}
}

func (o *forgejoInstance) createUser(ctx context.Context, t testing.TB) (*format.User, func()) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	fixtureUser := o.creator.CreateUser(int64(rand.Uint64()))
	user := &User{}
	user.FromFormat(fixtureUser)
	user.User.MustChangePassword = false
	user.User.LowerName = strings.ToLower(user.User.Name)

	assert.NoError(t, db.Insert(ctx, user.User))

	return fixtureUser, func() {
		assert.NoError(t, user_service.DeleteUser(ctx, &user.User, true))
	}
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
