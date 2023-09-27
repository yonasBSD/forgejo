package origins

import (
	_ "code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	origin_module "code.gitea.io/gitea/modules/origin"
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
	Repos origin_module.RemoteRepos
}

// Migrate in this case do a reverse engineering to retrieve form opt the
// struct in form of RemoteRepos
func (m *MockMigrator) Migrate(ctx context.Context, doer, u *user_model.User, opt migrations.MigrateOptions) error {
	if doer != nil && u != nil && opt.RepoName != "" {
		m.Repos = append(m.Repos, origin_module.RemoteRepo{
			CloneURL: opt.CloneAddr,
			Name:     opt.RepoName,
			Type:     opt.GitServiceType,
		})
		return nil
	}
	return fmt.Errorf("values missing")
}

type MockMigratorErr struct {
	Repos origin_module.RemoteRepos
}

func (m *MockMigratorErr) Migrate(ctx context.Context, doer, user *user_model.User, opt migrations.MigrateOptions) error {
	return fmt.Errorf("error while migrating")
}

func NewOriginSyncerTest(ctx context.Context, doer, u *user_model.User, limit int, migrator Migrator) *OriginSyncer {
	os := &OriginSyncer{
		Doer:            doer,
		User:            u,
		Context:         ctx,
		Migrator:        migrator,
		Limit:           limit,
		actualMigration: make(chan *origin_module.RemoteRepo, limit),
		finished:        make(chan struct{}, 2),
		err:             make(chan error, limit),
	}
	return os
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
	ss := NewOriginSyncerTest(
		context.Background(),
		user,
		user,
		10,
		&mm,
	)

	t.Run("Unmatched repo tests", func(t *testing.T) {
		// Create 10 repo names, 5 of which are in the sources.
		existingRepoNames := []string{"repo1", "repo2", "repo3", "repo4", "repo5", "repo6", "repo7", "repo8", "repo9", "repo10"}

		// Create 5 mock RemoteRepos.
		testSources := origin_module.RemoteRepos{
			origin_module.RemoteRepo{Name: "repo1"},
			origin_module.RemoteRepo{Name: "repo2"},
			origin_module.RemoteRepo{Name: "repo3"},
			origin_module.RemoteRepo{Name: "repo4"},
			origin_module.RemoteRepo{Name: "repo5"},
		}

		unmatched := ss.getUnmatchedRepos(testSources, existingRepoNames)
		assert.Len(t, unmatched, 0) // Expecting no unmatched repos.

		existingRepoNames = []string{"repo2", "repo3", "repo4"}
		testSources = origin_module.RemoteRepos{
			origin_module.RemoteRepo{Name: "repo2"},
			origin_module.RemoteRepo{Name: "repo5"},
		}

		unmatched = ss.getUnmatchedRepos(testSources, existingRepoNames)
		assert.Len(t, unmatched, 1) // Expecting one unmatched repo.
	})

	t.Run("Integration", func(t *testing.T) {
		err := ss.Fetch()
		assert.NoError(t, err)

		expected := DummyRepos
		assert.Equal(t, expected, ss.GetIncomingRepos())

		err = ss.Sync()
		assert.NoError(t, err)
		assert.True(t, ss.InProgress())

		timeout := time.After(time.Duration(len(DummyRepos)) * MIGRATIONS_DELAY * 2)
		index := 0

	loop:
		for {
			select {
			case repo, ok := <-ss.GetActualMigration():
				if !ok {
					break loop // exit the loop, don't return from the function
				}
				assert.Equal(t, &DummyRepos[index], repo)
				index++
			case <-timeout:
				t.Fatal("Timeout while waiting for migrations")
				return
			}
		}

		assert.Equal(t, len(DummyRepos), len(mm.Repos))

		// Check for errors
		select {
		case err := <-ss.Error():
			t.Fatalf("Received unexpected error: %v", err) // Fail the test if there's an error
		default:
			// No errors, which is expected
		}
		assert.Equal(t, <-ss.Finished(), struct{}{})
	})

}

func TestSyncOriginsCancel(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})

	mm := MockMigrator{}
	ss := NewOriginSyncerTest(
		context.Background(),
		user,
		user,
		10,
		&mm,
	)

	err := ss.Fetch()
	assert.NoError(t, err)

	expected := DummyRepos
	assert.Equal(t, expected, ss.GetIncomingRepos())

	err = ss.Sync()
	assert.NoError(t, err)

	time.Sleep(MIGRATIONS_DELAY / 2) // Wait 1 repo be synced
	ss.Cancel()
	time.Sleep(MIGRATIONS_DELAY) // Wait a bit till context be canceled

	assert.False(t, ss.InProgress())
	assert.Equal(t, DummyRepos[:1], mm.Repos)
}

func TestSyncOriginsError(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})

	mm := MockMigratorErr{}
	ss := NewOriginSyncerTest(
		context.Background(),
		user,
		user,
		10,
		&mm,
	)

	err := ss.Fetch()
	assert.NoError(t, err)

	expected := DummyRepos
	assert.Equal(t, expected, ss.GetIncomingRepos())

	err = ss.Sync()
	assert.Error(t, <-ss.Error())
	assert.False(t, ss.InProgress())
	assert.Equal(t, <-ss.Finished(), struct{}{})
	assert.Len(t, mm.Repos, 0)
}

func TestSyncOriginsLimit(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})

	limit := 2

	mm := MockMigratorErr{}
	ss := NewOriginSyncerTest(
		context.Background(),
		user,
		user,
		limit,
		&mm,
	)

	err := ss.Fetch()
	assert.NoError(t, err)

	expected := DummyRepos[:limit]
	assert.Equal(t, expected, ss.GetIncomingRepos())
}
