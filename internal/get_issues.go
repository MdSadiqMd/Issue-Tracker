package internal

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/syumai/workers/cloudflare"
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

	headers := js.Global().Get("Object").New()
	headers.Set("Accept", "application/vnd.github.v3+json")
	headers.Set("User-Agent", "Go-GitHub-Issues-Fetcher")

	options := js.Global().Get("Object").New()
	options.Set("method", "GET")
	options.Set("headers", headers)

	fetchFunc := js.Global().Get("fetch")
	promise := fetchFunc.Invoke(url, options)

	resultChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		status := response.Get("status").Int()
		if status != 200 {
			errorChan <- fmt.Errorf("GitHub API returned status %d", status)
			return nil
		}

		jsonPromise := response.Call("json")
		jsonPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()
			resultChan <- []byte(jsonStr)
			return nil
		}))

		jsonPromise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errorChan <- fmt.Errorf("failed to parse JSON: %v", args[0].String())
			return nil
		}))

		return nil
	}))

	promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorChan <- fmt.Errorf("fetch failed: %v", args[0].String())
		return nil
	}))

	select {
	case data := <-resultChan:
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

	case err := <-errorChan:
		fmt.Printf("Error fetching issues for %s: %v\n", repo, err)
		return []Issue{}, nil

	case <-time.After(10 * time.Second):
		fmt.Printf("Timeout fetching issues for %s\n", repo)
		return []Issue{}, nil
	}
}

func FetchIssuesLogic() ([]RepoIssues, error) {
	gistID := cloudflare.Getenv("GIST_ID")
	accessToken := cloudflare.Getenv("GITHUB_ACCESS_TOKEN")
	if gistID == "" || accessToken == "" {
		return nil, fmt.Errorf("Missing GIST_ID or GITHUB_ACCESS_TOKEN environment variables")
	}

	repos, err := LoadReposFromGistDB(gistID, accessToken)
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
