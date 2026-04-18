package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	internal "github.com/MdSadiqMd/issue-tracker/internal"
	"github.com/syumai/workers"
)

func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
		msg := "Hello!"
		w.Write([]byte(msg))
	})
	http.HandleFunc("/echo", func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}
		io.Copy(w, bytes.NewReader(b))
	})
	http.HandleFunc("/fetch-issues", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
			return
		}

		repos, err := internal.LoadReposFromFile()
		if err != nil {
			http.Error(w, "Failed to load repos", http.StatusInternalServerError)
			return
		}

		var results []internal.RepoIssues
		for _, repo := range repos {
			issues, err := internal.FetchIssues(repo)
			if err != nil {
				fmt.Printf("Failed to fetch issues for %s: %v\n", repo, err)
				continue
			}
			results = append(results, internal.RepoIssues{
				Repo:   repo,
				Issues: issues,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})
	workers.Serve(nil) // use http.DefaultServeMux
}
