// Package sources SPDX-License-Identifier: MIT
//
// Package sources provide functionality to mirror repositories from various source types.
// The primary component is the SourceSyncer, which fetches repositories from external sources
// and mirrors them if they don't already exist in the local instance.
//
// Terminology:
//   - Source: The origin of the repositories that will be mirrored.
//     Examples include GitHub starred repos, Gitlab forked repos, etc.
//   - RemoteRepos: A representation of repositories retrieved from an external source.
//     These are not saved to the database but are used to determine which
//     repositories need to be mirrored.
package sources

import (
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/migrations"
	"code.gitea.io/gitea/services/task"
	"context"
	"fmt"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/repo"
	sources_module "code.gitea.io/gitea/modules/sources"
	"code.gitea.io/gitea/modules/util"
)

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
var DummyData = sources_module.RemoteRepos{
	sources_module.RemoteRepo{Name: "dummy1", CloneURL: "null.com/r1.git", Type: structs.NotMigrated},
	sources_module.RemoteRepo{Name: "dummy2", CloneURL: "null.com/r2.git", Type: structs.NotMigrated},
	sources_module.RemoteRepo{Name: "dummy3", CloneURL: "null.com/r3.git", Type: structs.NotMigrated},
}

// SourceSyncer is responsible for synchronizing repositories from external sources.
type SourceSyncer struct {
	Context  context.Context
	Doer     *user.User // The logged user performing the sync operation.
	User     *user.User // The user whose new repositories are supposed to be mirrored (can be an org.).
	Migrator Migrator   // Will call this to migrate and mirror a repo
	Limit    int        // The maximum number of repositories to sync.
}

// SyncSources initializes a SourceSyncer and starts the synchronization process.
// You can initialize SourceSyncer by yourself if you want to test it
func SyncSources(ctx context.Context, doer, u *user.User, limit int) error {
	ss := &SourceSyncer{
		Doer:     doer,
		User:     u,
		Context:  ctx,
		Migrator: RealMigrator{},
		Limit:    limit,
	}
	return ss.Sync()
}

// getUserRepoNames retrieves the names of all repositories owned by the user.
func (s *SourceSyncer) getUserRepoNames() ([]string, error) {
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

// mirrorRepos mirrors the provided repositories.
func (s *SourceSyncer) mirrorRepos(repos sources_module.RemoteRepos) error {
	for _, r := range repos {
		migrateOptions := migrations.MigrateOptions{
			CloneAddr:      r.CloneURL,
			RepoName:       r.Name,
			Mirror:         true,
			GitServiceType: r.Type,
		}
		err := s.Migrator.Migrate(s.Doer, s.User, migrateOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

// fetchReposBySourceType fetches repositories based on the provided source type.
func (s *SourceSyncer) fetchReposBySourceType(source models.Source) (sources_module.RemoteRepos, error) {
	switch source.Type {
	case models.GithubStarred:
		return sources_module.GithubStars(source.RemoteUsername, source.Token)
	case models.Dummy:
		// Create a dummy set of RemoteRepos, for tests
		return DummyData, nil
	default:
		return nil, fmt.Errorf("unsupported source type: %v", source.Type)
	}
}

// getUnmatchedRepos identifies repositories from the new sources that don't already
// exist in the user's repositories.
func (s *SourceSyncer) getUnmatchedRepos(newSources sources_module.RemoteRepos,
	existingRepoNames []string) sources_module.RemoteRepos {

	unmatchedRepoNames := util.LeftDiff(newSources.GetNames(), existingRepoNames)
	if len(unmatchedRepoNames) > s.Limit {
		unmatchedRepoNames = unmatchedRepoNames[:s.Limit]
	}
	newSources.FilterBy(unmatchedRepoNames)
	return newSources
}

// Sync orchestrates the entire process of getting sources defined by user from a database,
// identifying, check unmatched repos and mirror them.
func (s *SourceSyncer) Sync() error {
	modelSources, err := models.GetSourcesByUserID(s.Context, s.User.ID)
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
			return fmt.Errorf("failed to get repos for source type %v: %w", source.Type, err)
		}

		unmatchedRepos := s.getUnmatchedRepos(newSources, existentRepos)
		if len(unmatchedRepos) != 0 {
			err := s.mirrorRepos(unmatchedRepos)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
