package sources

import (
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/sources"
	"code.gitea.io/gitea/services/migrations"
	"context"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

type MockMigrator struct {
	Repos []sources.RemoteRepo
}

// Migrate in this case do a reverse engineering to retrieve form opt the
// struct in form of RemoteRepos
func (m *MockMigrator) Migrate(doer, u *user_model.User, opt migrations.MigrateOptions) error {
	if doer != nil && u != nil && opt.RepoName != "" {
		m.Repos = append(m.Repos, sources.RemoteRepo{
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
	ss := SourceSyncer{
		Context:  context.Background(),
		Doer:     user,
		User:     user,
		Migrator: &mm,
	}

	t.Run("Unmatched repo tests", func(t *testing.T) {
		// Create 10 repo names, 5 of which are in the sources.
		existingRepoNames := []string{"repo1", "repo2", "repo3", "repo4", "repo5", "repo6", "repo7", "repo8", "repo9", "repo10"}

		// Create 5 mock RemoteRepos.
		testSources := sources.RemoteRepos{
			sources.RemoteRepo{Name: "repo1"},
			sources.RemoteRepo{Name: "repo2"},
			sources.RemoteRepo{Name: "repo3"},
			sources.RemoteRepo{Name: "repo4"},
			sources.RemoteRepo{Name: "repo5"},
		}

		unmatched := ss.getUnmatchedRepos(testSources, existingRepoNames)
		assert.Len(t, unmatched, 5) // Expecting 5 unmatched repos (repo6 to repo10).

		existingRepoNames = []string{"repo2", "repo3", "repo4"}
		testSources = sources.RemoteRepos{
			sources.RemoteRepo{Name: "repo2"},
			sources.RemoteRepo{Name: "repo5"},
		}

		unmatched = ss.getUnmatchedRepos(testSources, existingRepoNames)
		assert.Len(t, unmatched, 1) // Expecting 5 unmatched repos (repo6 to repo10).
	})

	t.Run("Integration", func(t *testing.T) {
		err := ss.Sync()
		assert.NoError(t, err)

		expected := DummyData
		assert.Equal(t, expected, mm.Repos)
	})
}
