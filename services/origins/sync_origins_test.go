package origins

import (
	_ "code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/origin"
	"code.gitea.io/gitea/services/migrations"
	"context"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
	"time"
)

type MockMigrator struct {
	Repos origin.RemoteRepos
}

// Migrate in this case do a reverse engineering to retrieve form opt the
// struct in form of RemoteRepos
func (m *MockMigrator) Migrate(doer, u *user_model.User, opt migrations.MigrateOptions) error {
	if doer != nil && u != nil && opt.RepoName != "" {
		m.Repos = append(m.Repos, origin.RemoteRepo{
			CloneURL: opt.CloneAddr,
			Name:     opt.RepoName,
			Type:     opt.GitServiceType,
		})
		return nil
	}
	return fmt.Errorf("values missing")
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
	ss := OriginSyncer{
		Context:  context.Background(),
		Doer:     user,
		User:     user,
		Migrator: &mm,
		Limit:    5,
	}

	t.Run("Unmatched repo tests", func(t *testing.T) {
		// Create 10 repo names, 5 of which are in the sources.
		existingRepoNames := []string{"repo1", "repo2", "repo3", "repo4", "repo5", "repo6", "repo7", "repo8", "repo9", "repo10"}

		// Create 5 mock RemoteRepos.
		testSources := origin.RemoteRepos{
			origin.RemoteRepo{Name: "repo1"},
			origin.RemoteRepo{Name: "repo2"},
			origin.RemoteRepo{Name: "repo3"},
			origin.RemoteRepo{Name: "repo4"},
			origin.RemoteRepo{Name: "repo5"},
		}

		unmatched := ss.getUnmatchedRepos(testSources, existingRepoNames)
		assert.Len(t, unmatched, 0) // Expecting no unmatched repos.

		existingRepoNames = []string{"repo2", "repo3", "repo4"}
		testSources = origin.RemoteRepos{
			origin.RemoteRepo{Name: "repo2"},
			origin.RemoteRepo{Name: "repo5"},
		}

		unmatched = ss.getUnmatchedRepos(testSources, existingRepoNames)
		assert.Len(t, unmatched, 1) // Expecting one unmatched repo.
	})

	t.Run("Integration", func(t *testing.T) {
		err := ss.Fetch()
		assert.NoError(t, err)

		expected := DummyData
		assert.Equal(t, expected, ss.GetIncomingRepos())

		ss.Sync()
		time.Sleep(MIGRATIONS_DELAY * 2) // After this time, at two repos will have reached at Migrator
		assert.Len(t, mm.Repos, 2)
	})
}
