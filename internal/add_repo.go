package internal

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/MdSadiqMd/issue-tracker/pkg"
)

type AddRepoRequest struct {
	RepoURL string `json:"repo_url"`
}

type AddRepoResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Repo    string `json:"repo,omitempty"`
}

func ExtractRepoName(repoURL string) (string, error) {
	repoURL = regexp.MustCompile(`\.git$`).ReplaceAllString(repoURL, "")
	patterns := []string{
		`github\.com/([^/]+/[^/]+)`,
		`^([^/]+/[^/]+)$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(repoURL)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("invalid GitHub repository URL format")
}

func AddRepoToGistDB(gistID, accessToken, repoName string) error {
	fmt.Printf("Fetching current gist to add repo: %s\n", repoName)

	currentRepos, err := FetchGistRepos(gistID, accessToken)
	if err != nil {
		return fmt.Errorf("error fetching current gist: %v", err)
	}

	for _, repo := range currentRepos {
		if repo.Name == repoName {
			return fmt.Errorf("repository %s already exists", repoName)
		}
	}

	newID := fmt.Sprintf("%d", time.Now().Unix())
	currentRepos = append(currentRepos, RepoObject{ID: newID, Name: repoName})

	updatedContent, err := json.Marshal(currentRepos)
	if err != nil {
		return fmt.Errorf("error marshaling repos: %v", err)
	}
	fmt.Printf("Updating gist with new repo: %s\n", repoName)

	updateBody := map[string]interface{}{
		"files": map[string]interface{}{
			"repos.json": map[string]string{
				"content": string(updatedContent),
			},
		},
	}

	updateBodyJSON, err := json.Marshal(updateBody)
	if err != nil {
		return fmt.Errorf("error marshaling update body: %v", err)
	}

	gistURL := fmt.Sprintf("https://api.github.com/gists/%s", gistID)
	patchHeaders := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"Content-Type":  "application/json",
		"Accept":        "application/vnd.github.v3+json",
		"User-Agent":    "Issue-Tracker-Worker",
	}

	_, err = pkg.FetchJS(gistURL, "PATCH", patchHeaders, string(updateBodyJSON))
	if err != nil {
		return fmt.Errorf("error updating gist: %v", err)
	}

	fmt.Printf("Successfully added repo %s\n", repoName)
	return nil
}
