package task

import (
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/services/migrations"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// For communication between lower and high level func.
var ch chan string

type MockMigrator struct{}

func (m MockMigrator) Migrate(doer, u *user_model.User, opt migrations.MigrateOptions) error {
	if doer != nil || u != nil || opt.RepoName != "" {
		ch <- opt.RepoName
	}
	return fmt.Errorf("missing values")
}

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
	})
}

func TestSyncSources(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
	ch = make(chan string)

	ss := SourceSyncer{
		Doer:     user, //todo: add timeout context
		User:     user,
		Migrator: MockMigrator{},
	}
	err := ss.SyncSources()
	assert.NoError(t, err)

	for e := range <-ch {
		fmt.Println(e)
		// Todo: of course not log this.
	}
}
