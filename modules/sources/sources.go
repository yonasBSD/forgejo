package sources

import (
	"code.gitea.io/gitea/modules/httplib"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/structs"
	"fmt"
	"io"
	"net/http"
)

// SourceRepo represents a repository with its URL, Owner, and Name.
type SourceRepo struct {
	URL  string `json:"html_url"`
	Name string `json:"name"`
	Type structs.GitServiceType
}

type SourceRepos []SourceRepo

// GetNames extracts the names of the repositories from the SourceRepos slice.
func (s *SourceRepos) GetNames() []string {
	var names []string
	for _, repo := range *s {
		names = append(names, repo.Name)
	}
	return names
}

// Filter filters the SourceRepos based on the provided names.
// todo: decrease time complexity
func (s *SourceRepos) Filter(names []string) {
	var filteredRepos SourceRepos
	for _, repo := range *s {
		for _, name := range names {
			if repo.Name == name {
				filteredRepos = append(filteredRepos, repo)
				break
			}
		}
	}
	*s = filteredRepos
}

// GithubStars will return starred repos from a given user in GitHub.
func GithubStars(username, token string) (SourceRepos, error) {
	url := fmt.Sprintf(
		"https://api.github.com/users/%s/starred?per_page=20&page=1", username)
	req := httplib.NewRequest(
		url, "GET")

	if token != "" {
		req.Header("Authorization", "token "+token)
	}

	res, err := req.Response()
	if err != nil || res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data, status: %s, %s", res.Status, err)
	}

	var repos SourceRepos

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, err
	}

	for _, r := range repos {
		r.Type = structs.GithubService
	}

	return repos, nil
}
