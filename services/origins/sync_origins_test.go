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
func (m *MockMigrator) Migrate(ctx context.Context, doer, u *user_model.User, opt migrations.MigrateOptions) error {
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

func TestSyncOrigins(t *testing.T) {
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

		err = ss.Sync()
		assert.NoError(t, err)
		assert.True(t, ss.InProgress())

		time.Sleep(MIGRATIONS_DELAY * 3) // After this time, three repos will be reached at migrator
		assert.Equal(t, len(mm.Repos), 3)

		// Check for errors
		select {
		case err := <-ss.Error():
			t.Fatalf("Received unexpected error: %v", err) // Fail the test if there's an error
		default:
			// No errors, which is expected
		}
	})
}

func TestSyncOriginsCancel(t *testing.T) {
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

	err := ss.Fetch()
	assert.NoError(t, err)

	expected := DummyData
	assert.Equal(t, expected, ss.GetIncomingRepos())

	err = ss.Sync()
	assert.NoError(t, err)

	time.Sleep(MIGRATIONS_DELAY / 2) // Wait 1 repo be synced
	ss.Cancel()
	time.Sleep(MIGRATIONS_DELAY / 2) // Wait a bit till context be canceled

	assert.False(t, ss.InProgress())
	assert.Equal(t, mm.Repos, DummyData[:1])

}
