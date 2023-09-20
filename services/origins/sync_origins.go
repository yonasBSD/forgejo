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
	Migrate(doer, u *user.User, opt migrations.MigrateOptions) error
}

type RealMigrator struct{}

func (m RealMigrator) Migrate(doer, u *user.User, opt migrations.MigrateOptions) error {
	return task.MigrateRepository(doer, u, opt)
}

// DummyData represents a set of mock repositories for testing purposes.
var DummyData = origin_module.RemoteRepos{
	origin_module.RemoteRepo{Name: "dummy1", CloneURL: "null.com/r1.git", Type: structs.NotMigrated},
	origin_module.RemoteRepo{Name: "dummy2", CloneURL: "null.com/r2.git", Type: structs.NotMigrated},
	origin_module.RemoteRepo{Name: "dummy3", CloneURL: "null.com/r3.git", Type: structs.NotMigrated},
}

// OriginSyncer is responsible for synchronizing repositories from external origins.
type OriginSyncer struct {
	Context  context.Context
	Doer     *user.User // The logged user performing the sync operation.
	User     *user.User // The user whose new repositories are supposed to be mirrored (can be an org.).
	Migrator Migrator   // Will call this to migrate and mirror a repo
	Limit    int        // The maximum number of repositories to sync.
}

// SyncOrigins initializes an OriginSyncer and starts the synchronization process.
// You can initialize OriginSyncer by yourself if you want to test it.
// SyncOrigins will also return a cancel function which can be used to stop scheduling
// of new migrations
func SyncOrigins(ctx context.Context, doer, u *user.User, limit int) (context.CancelFunc, error) {
	os := &OriginSyncer{
		Doer:     doer,
		User:     u,
		Context:  ctx,
		Migrator: RealMigrator{},
		Limit:    limit,
	}
	return os.Sync()
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

// mirrorRepos mirrors the provided repositories.
func (s *OriginSyncer) mirrorRepos(repos origin_module.RemoteRepos) context.CancelFunc {
	ctx, cancel := context.WithCancel(s.Context)

	go func() {
		defer cancel()

		for _, r := range repos {
			select {
			case <-ctx.Done():
				log.Info("Migration from remote origins stopped")
				return
			default:
				time.Sleep(MIGRATIONS_DELAY)
				migrateOptions := migrations.MigrateOptions{
					CloneAddr:      r.CloneURL,
					RepoName:       r.Name,
					Mirror:         true,
					GitServiceType: r.Type,
				}
				err := s.Migrator.Migrate(s.Doer, s.User, migrateOptions)
				if err != nil {
					log.Error("Error while scheduling migration for repo %s: %v", r.Name, err)
					return
				}

				log.Info("Repository migration %v from %v scheduled", r.Name, r.Type)
			}
		}
	}()

	return cancel
}

// Sync orchestrates the entire process of getting origins defined by user from a database,
// identifying, check unmatched repos, and mirror them.
func (s *OriginSyncer) Sync() (context.CancelFunc, error) {
	modelSources, err := models.GetOriginsByUserID(s.Context, s.User.ID)
	if err != nil {
		return nil, err
	}

	existentRepos, err := s.getUserRepoNames()
	if err != nil {
		return nil, err
	}

	var allUnmatchedRepos origin_module.RemoteRepos

	for _, source := range modelSources {
		newSources, err := s.fetchReposBySourceType(source)
		if err != nil {
			return nil, fmt.Errorf("failed to get repos for source type %v: %w", source.Type, err)
		}

		unmatchedRepos := s.getUnmatchedRepos(newSources, existentRepos)
		if len(unmatchedRepos) != 0 {
			log.Info("New unmatched repositories found on %v: %v.", source.Type, unmatchedRepos) //todo
		}
		allUnmatchedRepos = append(allUnmatchedRepos, unmatchedRepos...)
	}

	if len(allUnmatchedRepos) > 0 {
		cancel := s.mirrorRepos(allUnmatchedRepos)
		return cancel, nil
	}

	return nil, nil
}
