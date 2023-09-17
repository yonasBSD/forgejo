package sources

import (
	"code.gitea.io/gitea/modules/httplib"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/structs"
	"fmt"
	"io"
	"net/http"
	"strings"
)

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

// FilterBy filters the RemoteRepos based on the provided names.
// FilterBy will overwrite the list with just repos containing names
func (r *RemoteRepos) FilterBy(names []string) {
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
	*r = filteredRepos
}

// GithubStars will return starred repos from a given user in GitHub.
// Token should allow more requests and private repos
func GithubStars(username, token string) (RemoteRepos, error) {
	var allRepos RemoteRepos
	page := 1
	perPage := 100 // Set to maximum allowed by GitHub to minimize requests

	for {
		url := fmt.Sprintf("https://api.github.com/users/%s/starred?per_page=%d&page=%d", username, perPage, page)
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

// containsRelNext checks if the Link header contains rel="next", indicating more pages.
func containsRelNext(linkHeader string) bool {
	return strings.Contains(linkHeader, `rel="next"`)
}
