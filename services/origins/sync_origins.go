// SPDX-License-Identifier: MIT
//
// Package origins provide functionality to mirror repositories from various origin types.
// The primary component is the OriginSyncer, which fetches repositories from external origins
// and mirrors them if they don't already exist in the local instance.
//
// Terminology:
//   - Origin: The source or platform from where the repositories will be mirrored.
//     Examples include GitHub starred repos, Gitlab forked repos, etc.
//   - RemoteRepos: A representation of repositories retrieved from an external origin.
//     These are not saved to the database but are used to determine which
//     repositories need to be mirrored.
package origins

import (
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/migrations"
	"code.gitea.io/gitea/services/task"
	"context"
	"fmt"
	"sync"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/repo"
	origin_module "code.gitea.io/gitea/modules/origin"
	"code.gitea.io/gitea/modules/util"
)

const MIGRATIONS_DELAY = 1 * time.Second

// Migrator defines an interface for migrating repositories. This is useful
// because we want to test this package without making real cloning operations
type Migrator interface {
	Migrate(ctx context.Context, doer, u *user.User, opt migrations.MigrateOptions) error
}

type RealMigrator struct{}

func (m RealMigrator) Migrate(ctx context.Context, doer, u *user.User, opt migrations.MigrateOptions) error {
	return task.MigrateRepository(ctx, doer, u, opt)
}

// DummyData represents a set of mock repositories for testing purposes.
var DummyData = origin_module.RemoteRepos{
	origin_module.RemoteRepo{Name: "fake_repo1", CloneURL: "null.com/r1.git", Type: structs.NotMigrated},
	origin_module.RemoteRepo{Name: "fake_repo2", CloneURL: "null.com/r2.git", Type: structs.NotMigrated},
	origin_module.RemoteRepo{Name: "fake_repo3", CloneURL: "null.com/r3.git", Type: structs.NotMigrated},
}

// OriginSyncer is responsible for synchronizing repositories from external origins.
type OriginSyncer struct {
	Context  context.Context
	Doer     *user.User // The logged user performing the sync operation.
	User     *user.User // The user whose new repositories are supposed to be mirrored (can be an org.).
	Migrator Migrator   // Func to migrate a repo
	Limit    int        // The maximum number of repositories to sync.

	incomingRepos origin_module.RemoteRepos
	cancel        context.CancelFunc

	inProgress      bool
	inProgressMu    sync.Mutex
	actualMigration *origin_module.RemoteRepo

	err chan error
}

// NewOriginSyncer initializes an OriginSyncer and starts the synchronization process.
// You can initialize OriginSyncer by yourself if you want to test it.
// NewOriginSyncer will also return a cancel function which can be used to stop scheduling
// of new migrations
func NewOriginSyncer(ctx context.Context, doer, u *user.User, limit int) *OriginSyncer {
	os := &OriginSyncer{
		Doer:     doer,
		User:     u,
		Context:  ctx,
		Migrator: RealMigrator{},
		Limit:    limit,
	}
	return os
}

// getUserRepoNames retrieves the names of all repositories owned by the user.
func (s *OriginSyncer) getUserRepoNames() ([]string, error) {
	userRepos, _, err := repo.GetUserRepositories(&repo.SearchRepoOptions{
		Actor: s.User,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user repositories: %w", err)
	}
	var repoNames []string
	for _, r := range userRepos {
		repoNames = append(repoNames, r.Name)
	}
	return repoNames, nil
}

// fetchReposBySourceType fetches repositories based on the provided source type.
func (s *OriginSyncer) fetchReposBySourceType(source models.Origin) (origin_module.RemoteRepos, error) {
	switch source.Type {
	case models.GithubStarred:
		return origin_module.GithubStars(source.RemoteUsername, source.Token)
	case models.Dummy:
		// Create a dummy set of RemoteRepos, for tests
		return DummyData, nil
	default:
		return nil, fmt.Errorf("unsupported source type: %v", source.Type)
	}
}

// getUnmatchedRepos identifies repositories from the new origin that doesn't yet
// exist in the user's local repositories.
func (s *OriginSyncer) getUnmatchedRepos(newSources origin_module.RemoteRepos,
	existingRepoNames []string) origin_module.RemoteRepos {

	unmatchedRepoNames := util.LeftDiff(newSources.GetNames(), existingRepoNames)
	if len(unmatchedRepoNames) > s.Limit {
		unmatchedRepoNames = unmatchedRepoNames[:s.Limit]
	}

	return newSources.FilterBy(unmatchedRepoNames)
}

func (s *OriginSyncer) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *OriginSyncer) Error() chan error {
	return s.err
}

// GetIncomingRepos will return every new repository found after fetching remote origins
func (s *OriginSyncer) GetIncomingRepos() origin_module.RemoteRepos {
	return s.incomingRepos
}

func (s *OriginSyncer) InProgress() bool {
	s.inProgressMu.Lock()
	defer s.inProgressMu.Unlock()
	return s.inProgress
}

// GetActualMigration will return which repository Sync is currently migrating.
func (s *OriginSyncer) GetActualMigration() *origin_module.RemoteRepo {
	return s.actualMigration
}

// Sync mirrors the fetched repositories.
func (s *OriginSyncer) Sync() error {
	if len(s.incomingRepos) == 0 {
		return nil
	}

	s.inProgressMu.Lock()
	if s.inProgress {
		return fmt.Errorf("origins synchronization already in progress")
	}
	s.inProgress = true
	s.inProgressMu.Unlock()

	ctx, cancel := context.WithCancel(s.Context)
	s.cancel = cancel

	go func() {
		defer cancel()

		for _, r := range s.incomingRepos {
			select {
			case <-ctx.Done():
				log.Info("Migration from remote origins stopped")
				s.inProgressMu.Lock()
				s.inProgress = false
				s.inProgressMu.Unlock()
				return
			default:
				migrateOptions := migrations.MigrateOptions{
					CloneAddr:      r.CloneURL,
					RepoName:       r.Name,
					Mirror:         true,
					GitServiceType: r.Type,
				}
				s.actualMigration = &r
				err := s.Migrator.Migrate(ctx, s.Doer, s.User, migrateOptions)
				if err != nil {
					log.Error("Error while adding migration for repo %s: %v", r.Name, err)
					s.err <- err
					return
				}
				log.Info("Repository migration %v from %v started/scheduled", r.Name, r.Type)
				time.Sleep(MIGRATIONS_DELAY) // The user must have the right to cancel
			}
		}
		s.inProgressMu.Lock()
		s.inProgress = false
		s.inProgressMu.Unlock()
	}()
	return nil
}

// Fetch orchestrates the entire process of getting origins defined by user from a database,
// identifying and check unmatched repos
func (s *OriginSyncer) Fetch() error {
	modelSources, err := models.GetOriginsByUserID(s.Context, s.User.ID)
	if err != nil {
		return err
	}

	existentRepos, err := s.getUserRepoNames()
	if err != nil {
		return err
	}

	for _, source := range modelSources {
		newSources, err := s.fetchReposBySourceType(source)
		if err != nil {
			return fmt.Errorf("failed to get repos for source type %v: %w", source.Type.GetName(), err)
		}

		unmatchedRepos := s.getUnmatchedRepos(newSources, existentRepos)
		if len(unmatchedRepos) != 0 {
			log.Info("New unmatched repositories found on %v: %v.", source.Type.GetName(), unmatchedRepos)
		}
		s.incomingRepos = append(s.incomingRepos, unmatchedRepos...)
	}

	return nil
}
