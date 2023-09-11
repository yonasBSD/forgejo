package task

import (
	"code.gitea.io/gitea/models/user"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/services/migrations"
	"fmt"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/sources"
	"code.gitea.io/gitea/modules/util"
)

// SyncSources syncs sources based on user ID.
func SyncSources(ctx *ctx.Context, doer, u *user.User) error {
	modelSources, err := models.GetSourcesByUserID(u.ID)
	if err != nil {
		return err
	}

	repoNames, err := getUserRepoNames()

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

				newSources.Filter(unmatchedRepos)

				err := mirrorRepos(doer, u, newSources)

				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// getUserRepoNames returns a list of repository names for a user.
func getUserRepoNames() ([]string, error) {
	userRepos, _, err := repo.GetUserRepositories(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user repositories: %w", err)
	}

	var repoNames []string
	for _, r := range userRepos {
		repoNames = append(repoNames, r.Name)
	}
	return repoNames, nil
}

// mirrorRepos mirrors repositories for a user.
func mirrorRepos(doer, u *user.User, repos sources.SourceRepos) error {

	for _, r := range repos {
		migrateOptions := migrations.MigrateOptions{
			CloneAddr:      r.URL,
			RepoName:       r.Name,
			Mirror:         true,
			GitServiceType: r.Type,
		}
		err := MigrateRepository(doer, u, migrateOptions)
		if err != nil {
			return err
		}
	}
	return nil
}
