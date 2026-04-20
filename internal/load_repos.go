package internal

import "fmt"

func LoadReposFromGistDB(gistID, accessToken string) ([]string, error) {
	fmt.Printf("Fetching repos from GitHub gist: https://api.github.com/gists/%s\n", gistID)

	repoObjects, err := FetchGistRepos(gistID, accessToken)
	if err != nil {
		return nil, err
	}

	var repos []string
	for _, repo := range repoObjects {
		if repo.Name != "" {
			repos = append(repos, repo.Name)
		}
	}

	fmt.Printf("Loaded %d repos from gist\n", len(repos))
	return repos, nil
}
