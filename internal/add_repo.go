package internal

import (
	"encoding/json"
	"fmt"
	"regexp"
	"syscall/js"
	"time"
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
	gistURL := fmt.Sprintf("https://api.github.com/gists/%s", gistID)
	fmt.Printf("Fetching current gist: %s\n", gistURL)

	headers := js.Global().Get("Object").New()
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	headers.Set("Accept", "application/vnd.github.v3+json")
	headers.Set("User-Agent", "Issue-Tracker-Worker")

	options := js.Global().Get("Object").New()
	options.Set("method", "GET")
	options.Set("headers", headers)

	fetchFunc := js.Global().Get("fetch")
	promise := fetchFunc.Invoke(gistURL, options)

	resultChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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

		return nil
	})

	promise.Call("then", thenFunc)
	promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorChan <- fmt.Errorf("fetch failed: %v", args[0].String())
		return nil
	}))

	var currentRepos []RepoObject
	select {
	case data := <-resultChan:
		var gistData map[string]interface{}
		if err := json.Unmarshal(data, &gistData); err != nil {
			return fmt.Errorf("error decoding gist: %v", err)
		}

		files, ok := gistData["files"].(map[string]interface{})
		if ok {
			if reposFile, ok := files["repos.json"].(map[string]interface{}); ok {
				if content, ok := reposFile["content"].(string); ok && content != "" {
					json.Unmarshal([]byte(content), &currentRepos)
				}
			}
		}

	case err := <-errorChan:
		return fmt.Errorf("error fetching gist: %v", err)

	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout fetching gist")
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

	patchHeaders := js.Global().Get("Object").New()
	patchHeaders.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	patchHeaders.Set("Content-Type", "application/json")
	patchHeaders.Set("Accept", "application/vnd.github.v3+json")
	patchHeaders.Set("User-Agent", "Issue-Tracker-Worker")

	patchOptions := js.Global().Get("Object").New()
	patchOptions.Set("method", "PATCH")
	patchOptions.Set("headers", patchHeaders)
	patchOptions.Set("body", string(updateBodyJSON))

	patchPromise := fetchFunc.Invoke(gistURL, patchOptions)

	resultChan = make(chan []byte, 1)
	errorChan = make(chan error, 1)

	patchThenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		status := response.Get("status").Int()

		if status != 200 {
			textPromise := response.Call("text")
			textPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				errorText := args[0].String()
				errorChan <- fmt.Errorf("GitHub API returned status %d: %s", status, errorText)
				return nil
			}))
			return nil
		}

		resultChan <- []byte("success")
		return nil
	})

	patchPromise.Call("then", patchThenFunc)
	patchPromise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorChan <- fmt.Errorf("patch failed: %v", args[0].String())
		return nil
	}))

	select {
	case <-resultChan:
		fmt.Printf("Successfully added repo %s\n", repoName)
		return nil

	case err := <-errorChan:
		return fmt.Errorf("error updating gist: %v", err)

	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout updating gist")
	}
}
