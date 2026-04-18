package internal

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

type Issue struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type RepoIssues struct {
	Repo   string  `json:"repo"`
	Issues []Issue `json:"issues"`
}

func FetchIssues(repo string) ([]Issue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues?per_page=100&state=open", repo)

	// Use syscall/js to call fetch API directly to avoid the "illegal invocation" error
	// This is necessary because Go's http.Client doesn't work in Cloudflare Workers

	// Create headers object
	headers := js.Global().Get("Object").New()
	headers.Set("Accept", "application/vnd.github.v3+json")
	headers.Set("User-Agent", "Go-GitHub-Issues-Fetcher")

	// Create fetch options
	options := js.Global().Get("Object").New()
	options.Set("method", "GET")
	options.Set("headers", headers)

	// Call fetch
	fetchFunc := js.Global().Get("fetch")
	promise := fetchFunc.Invoke(url, options)

	// Create channels for async handling
	resultChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	// Handle the promise
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]

		// Check status
		status := response.Get("status").Int()
		if status != 200 {
			errorChan <- fmt.Errorf("GitHub API returned status %d", status)
			return nil
		}

		// Get JSON
		jsonPromise := response.Call("json")
		jsonPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// Convert JS object to JSON string
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

	// Wait for result or error
	select {
	case data := <-resultChan:
		fmt.Printf("Received data for %s: %s\n", repo, string(data))

		var issues []Issue
		if err := json.Unmarshal(data, &issues); err != nil {
			fmt.Printf("Error decoding JSON for %s: %v\nData: %s\n", repo, err, string(data))
			return []Issue{}, nil
		}

		fmt.Printf("Successfully decoded %d issues for %s\n", len(issues), repo)

		var recentIssues []Issue
		oneMonthAgo := time.Now().AddDate(0, -1, 0)
		for _, issue := range issues {
			if issue.CreatedAt.After(oneMonthAgo) {
				recentIssues = append(recentIssues, issue)
			}
		}

		fmt.Printf("Found %d recent issues (last month) for %s\n", len(recentIssues), repo)
		return recentIssues, nil

	case err := <-errorChan:
		fmt.Printf("Error fetching issues for %s: %v\n", repo, err)
		return []Issue{}, nil

	case <-time.After(10 * time.Second):
		fmt.Printf("Timeout fetching issues for %s\n", repo)
		return []Issue{}, nil
	}
}
