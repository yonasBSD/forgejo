package migrations

import (
	"fmt"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/sources"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/services/task"
)

// SyncSources syncs sources based on user ID.
func SyncSources(ctx context.Context) error {
	modelSources := models.GetSourcesByUser(ctx.Doer.ID)
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
				// todo: maybe some logs of new repos being mirrored?

				newSources.Filter(unmatchedRepos)
				err := mirrorRepos(ctx.Doer, nil, newSources, structs.GithubService)
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
// todo: diff doer (user) vs u (user)
func mirrorRepos(doer, u *user.User, repos sources.SourceRepos, serviceType structs.GitServiceType) error {
	for _, r := range repos {
		opts := MigrateOptions{
			CloneAddr:      r.URL,
			RepoName:       r.Name,
			Mirror:         true,
			GitServiceType: serviceType,
		}

		err := task.MigrateRepository(doer, u, opts)
		if err != nil {
			return err
		}
	}
	return nil
}
