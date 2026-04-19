package internal

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

type GistDBResponse struct {
	Status  int                    `json:"status"`
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
	Error   string                 `json:"error,omitempty"`
}

type RepoObject struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

func LoadReposFromGistDB(gistID, accessToken string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/gists/%s", gistID)
	fmt.Printf("Fetching repos from GitHub gist: %s\n", url)

	headers := js.Global().Get("Object").New()
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	headers.Set("Accept", "application/vnd.github.v3+json")
	headers.Set("User-Agent", "Issue-Tracker-Worker")

	options := js.Global().Get("Object").New()
	options.Set("method", "GET")
	options.Set("headers", headers)

	fetchFunc := js.Global().Get("fetch")
	promise := fetchFunc.Invoke(url, options)

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
		jsonThenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()
			resultChan <- []byte(jsonStr)
			return nil
		})
		jsonCatchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errorChan <- fmt.Errorf("failed to parse JSON: %v", args[0].String())
			return nil
		})

		jsonPromise.Call("then", jsonThenFunc)
		jsonPromise.Call("catch", jsonCatchFunc)

		return nil
	})

	catchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorChan <- fmt.Errorf("fetch failed: %v", args[0].String())
		return nil
	})

	promise.Call("then", thenFunc)
	promise.Call("catch", catchFunc)

	select {
	case data := <-resultChan:
		var gistData map[string]interface{}
		if err := json.Unmarshal(data, &gistData); err != nil {
			return nil, fmt.Errorf("error decoding gist: %v", err)
		}

		files, ok := gistData["files"].(map[string]interface{})
		if !ok {
			return []string{}, nil
		}

		reposFile, ok := files["repos.json"].(map[string]interface{})
		if !ok {
			fmt.Printf("repos.json not found, returning empty list\n")
			return []string{}, nil
		}

		content, ok := reposFile["content"].(string)
		if !ok || content == "" {
			fmt.Printf("No content in repos.json, returning empty list\n")
			return []string{}, nil
		}

		var repoObjects []RepoObject
		if err := json.Unmarshal([]byte(content), &repoObjects); err != nil {
			return nil, fmt.Errorf("error parsing repos.json: %v", err)
		}

		var repos []string
		for _, repo := range repoObjects {
			if repo.Name != "" {
				repos = append(repos, repo.Name)
			}
		}

		fmt.Printf("Loaded %d repos from gist\n", len(repos))
		return repos, nil

	case err := <-errorChan:
		return nil, fmt.Errorf("error fetching from GitHub: %v", err)

	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout fetching from GitHub")
	}
}
