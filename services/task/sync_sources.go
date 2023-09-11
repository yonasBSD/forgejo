package task

import (
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/services/migrations"
	"context"
	"fmt"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/sources"
	"code.gitea.io/gitea/modules/util"
)

// Migrator define an interface that wraps the MigrateRepository call, this is needed
// because we want to test this file, it's unappropriated to make tests that would really
// clone lots of repos
type Migrator interface {
	Migrate(doer, u *user.User, opt migrations.MigrateOptions) error
}

type RealMigrator struct{}

func (m RealMigrator) Migrate(doer, u *user.User, opt migrations.MigrateOptions) error {
	return MigrateRepository(doer, u, opt)
}

type SourceSyncer struct {
	Context  context.Context
	Doer     *user.User
	User     *user.User
	Migrator Migrator
}

func NewSourceSyncer(ctx context.Context, doer, u *user.User) *SourceSyncer {
	return &SourceSyncer{
		Doer:     doer,
		User:     u,
		Context:  ctx,
		Migrator: RealMigrator{},
	}
}

// getUserRepoNames just retrieve all repos (names) the user owns in its local instance
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

func (s *SourceSyncer) mirrorRepos(repos sources.SourceRepos) error {
	for _, r := range repos {
		migrateOptions := migrations.MigrateOptions{
			CloneAddr:      r.URL,
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

func (s *SourceSyncer) SyncSources() error {
	modelSources, err := models.GetSourcesByUserID(s.Context, s.User.ID)
	if err != nil {
		return err
	}

	repoNames, err := s.getUserRepoNames()
	if err != nil {
		return err
	}

	for _, source := range modelSources {
		if source.Type == models.GithubStarred {
			newSources, err := sources.GithubStars(source.RemoteUsername, source.Token)
			if err != nil {
				return fmt.Errorf("failed to get GitHub starred repos: %w", err)
			}

			unmatchedRepos := util.LeftDiff(newSources.GetNames(), repoNames)
			if len(unmatchedRepos) != 0 {

				// Keep only new repositories from a remote source the user doesn't own in local inst.
				newSources.Filter(unmatchedRepos)
				err := s.mirrorRepos(newSources)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
