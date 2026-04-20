package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MdSadiqMd/issue-tracker/pkg"
)

type Issue struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type RepoIssues struct {
	Repo   string  `json:"repo"`
	Issues []Issue `json:"issues"`
}

func fetchIssues(repo string) ([]Issue, error) {
	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues?per_page=10&state=all&sort=created&direction=desc", repo)

	fmt.Printf("Fetching recent issues for %s (filtering for created after: %s UTC)\n", repo, oneHourAgo.Format(time.RFC3339))

	headers := map[string]string{
		"Accept":     "application/vnd.github.v3+json",
		"User-Agent": "Go-GitHub-Issues-Fetcher",
	}

	data, err := pkg.FetchJS(url, "GET", headers, "")
	if err != nil {
		fmt.Printf("Error fetching issues for %s: %v\n", repo, err)
		return []Issue{}, nil
	}

	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		fmt.Printf("Error decoding JSON for %s: %v\n", repo, err)
		return []Issue{}, nil
	}

	fmt.Printf("Received %d total issues for %s\n", len(issues), repo)

	var recentIssues []Issue
	for _, issue := range issues {
		issueCreatedUTC := issue.CreatedAt.UTC()
		if issueCreatedUTC.After(oneHourAgo) {
			recentIssues = append(recentIssues, issue)
			fmt.Printf("  ✓ Issue: '%s', Created: %s UTC (within last hour)\n", issue.Title, issueCreatedUTC.Format("2006-01-02 15:04:05"))
		}
	}

	fmt.Printf("✅ Found %d issues created in last hour for %s\n", len(recentIssues), repo)
	return recentIssues, nil
}

func FetchIssuesLogic() ([]RepoIssues, error) {
	cfg, err := GetGistConfig()
	if err != nil {
		return nil, err
	}

	repos, err := LoadReposFromGistDB(cfg.GistID, cfg.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("Failed to load repos from gist: %v", err)
	}

	fmt.Printf("Fetching issues from %d repositories concurrently\n", len(repos))

	type repoResult struct {
		repo   string
		issues []Issue
		err    error
	}

	resultChan := make(chan repoResult, len(repos))
	for _, repo := range repos {
		go func(r string) {
			issues, err := fetchIssues(r)
			resultChan <- repoResult{repo: r, issues: issues, err: err}
		}(repo)
	}

	var results []RepoIssues
	for i := 0; i < len(repos); i++ {
		result := <-resultChan
		if result.err != nil {
			fmt.Printf("Failed to fetch issues for %s: %v\n", result.repo, result.err)
			continue
		}
		results = append(results, RepoIssues{
			Repo:   result.repo,
			Issues: result.issues,
		})
	}
	return results, nil
}
