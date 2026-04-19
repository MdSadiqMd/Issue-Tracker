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
	url := fmt.Sprintf("https://gist-db.mohammadsadiq4950.workers.dev/api/%s?collection_name=repos", gistID)
	fmt.Printf("Fetching repos from GistDB: %s\n", url)

	headers := js.Global().Get("Object").New()
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	headers.Set("Content-Type", "application/json")

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
			errorChan <- fmt.Errorf("GistDB API returned status %d", status)
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
		fmt.Printf("Received GistDB response: %s\n", string(data))

		var gistResponse GistDBResponse
		if err := json.Unmarshal(data, &gistResponse); err != nil {
			return nil, fmt.Errorf("error decoding GistDB response: %v", err)
		}
		if gistResponse.Status != 200 {
			return nil, fmt.Errorf("GistDB error: %s", gistResponse.Error)
		}

		reposCollection, ok := gistResponse.Data["repos"]
		if !ok {
			return nil, fmt.Errorf("repos collection not found in response")
		}

		reposJSON, err := json.Marshal(reposCollection)
		if err != nil {
			return nil, fmt.Errorf("error marshaling repos: %v", err)
		}

		var repoObjects []RepoObject
		if err := json.Unmarshal(reposJSON, &repoObjects); err != nil {
			return nil, fmt.Errorf("error parsing repos: %v", err)
		}

		var repos []string
		for _, repo := range repoObjects {
			if repo.Name != "" {
				repos = append(repos, repo.Name)
			}
		}

		fmt.Printf("Loaded %d repos from GistDB\n", len(repos))
		return repos, nil

	case err := <-errorChan:
		return nil, fmt.Errorf("error fetching from GistDB: %v", err)

	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout fetching from GistDB")
	}
}
