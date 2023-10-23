package origin

import (
	"code.gitea.io/gitea/modules/httplib"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/structs"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Repositories per page. This number should be big enough to avoid too many
// requests but yet don't exceed limits
const PageSize = 100

// RemoteRepo represents a repository with its URL, Owner, and Name.
type RemoteRepo struct {
	CloneURL string `json:"clone_url"`
	Name     string `json:"name"`
	Type     structs.GitServiceType
}

type RemoteRepos []RemoteRepo

// GetNames extracts the names of the repositories from the RemoteRepos slice.
func (r *RemoteRepos) GetNames() []string {
	names := make([]string, len(*r))
	for i, repo := range *r {
		names[i] = repo.Name
	}
	return names
}

// FilterBy filters the RemoteRepos based on the provided names and returns a new RemoteRepos.
// FilterBy just keeps repos within names array
func (r *RemoteRepos) FilterBy(names []string) RemoteRepos {
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	var filteredRepos RemoteRepos
	for _, repo := range *r {
		if nameMap[repo.Name] {
			filteredRepos = append(filteredRepos, repo)
		}
	}
	return filteredRepos
}

// GithubStars will return starred repos from a given user in GitHub.
// Token should allow more requests and private repos
func GithubStars(username, token string) (RemoteRepos, error) {
	var allRepos RemoteRepos
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/users/%s/starred?per_page=%d&page=%d", username, PageSize, page)
		req := httplib.NewRequest(url, "GET")

		if token != "" {
			req.Header("Authorization", "token "+token)
		}

		res, err := req.Response()
		if err != nil || res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch data, status: %s, %s", res.Status, err)
		}

		var repos RemoteRepos
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &repos); err != nil {
			return nil, err
		}

		for i := range repos {
			repos[i].Type = structs.GithubService
		}

		allRepos = append(allRepos, repos...)

		// Check if there are more pages
		links := res.Header.Get("Link")
		if !containsRelNext(links) {
			break
		}

		page++
	}

	return allRepos, nil
}

func CodebergStars(username string) (RemoteRepos, error) {
	var allRepos RemoteRepos
	page := 1

	baseURL := "https://codeberg.org/api/v1"

	for {
		url := fmt.Sprintf("%s/users/%s/starred?page=%d&limit=%d", baseURL, username, page, PageSize)
		res, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch data, status: %s", res.Status)
		}

		var repos RemoteRepos
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &repos); err != nil {
			return nil, err
		}

		for i := range repos {
			repos[i].Type = structs.ForgejoService
		}

		allRepos = append(allRepos, repos...)

		// Assuming that if the number of repos returned is less than the PageSize, there are no more pages.
		if len(repos) < PageSize {
			break
		}

		page++
	}

	return allRepos, nil
}

// containsRelNext checks if the Link header contains rel="next", indicating more pages.
func containsRelNext(linkHeader string) bool {
	return strings.Contains(linkHeader, `rel="next"`)
}
