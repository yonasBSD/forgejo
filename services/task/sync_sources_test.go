package task

import (
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/sources"
	"code.gitea.io/gitea/services/migrations"
	"context"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

type MockMigrator struct {
	Repos []sources.RemoteRepo
}

func (m *MockMigrator) Migrate(doer, u *user_model.User, opt migrations.MigrateOptions) error {
	if doer != nil || u != nil || opt.RepoName != "" {
		m.Repos = append(m.Repos, sources.RemoteRepo{
			URL:  opt.CloneAddr,
			Name: opt.RepoName,
			Type: opt.GitServiceType,
		})
		return nil
	}
	return nil
}

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
	})
}

func TestSyncSources(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})

	mm := MockMigrator{}
	ss := SourceSyncer{
		Context:  context.Background(),
		Doer:     user,
		User:     user,
		Migrator: &mm,
	}

	err := ss.SyncSources()
	assert.NoError(t, err)

	// TODO: in mm.Repos we have every new remote repo that should be mirrored (github starred, gitlab user repos...)
	// TODO: we need to check 2 things: those repos dont exist in the local instance. 2: if they really exist
	// TODO: in the remote source. That is, under the remote username those repo exists
	// assert not contain this repo locally
	assert.Equal(t, "jekyll-theme-chirpy", mm.Repos[0].Name) //the first
}
