package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	internal "github.com/MdSadiqMd/issue-tracker/internal"
	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare"
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

		gistID := cloudflare.Getenv("GIST_ID")
		accessToken := cloudflare.Getenv("GITHUB_ACCESS_TOKEN")
		if gistID == "" || accessToken == "" {
			http.Error(w, "Missing GIST_ID or GITHUB_ACCESS_TOKEN environment variables", http.StatusInternalServerError)
			return
		}
		fmt.Printf("Using Gist ID: %s\n", gistID)

		repos, err := internal.LoadReposFromGistDB(gistID, accessToken)
		if err != nil {
			fmt.Printf("Failed to load repos from gist: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to load repos: %v", err), http.StatusInternalServerError)
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
	http.HandleFunc("/add-repo", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		gistID := cloudflare.Getenv("GIST_ID")
		accessToken := cloudflare.Getenv("GITHUB_ACCESS_TOKEN")
		if gistID == "" || accessToken == "" {
			http.Error(w, "Missing GIST_ID or GITHUB_ACCESS_TOKEN environment variables", http.StatusInternalServerError)
			return
		}

		var addRepoReq internal.AddRepoRequest
		if err := json.NewDecoder(req.Body).Decode(&addRepoReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if addRepoReq.RepoURL == "" {
			http.Error(w, "repo_url is required", http.StatusBadRequest)
			return
		}

		repoName, err := internal.ExtractRepoName(addRepoReq.RepoURL)
		if err != nil {
			response := internal.AddRepoResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid repository URL: %v", err),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := internal.AddRepoToGistDB(gistID, accessToken, repoName); err != nil {
			fmt.Printf("Failed to add repo to gist: %v\n", err)
			response := internal.AddRepoResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to add repository: %v", err),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := internal.AddRepoResponse{
			Success: true,
			Message: "Repository added successfully",
			Repo:    repoName,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})
	workers.Serve(nil) // use http.DefaultServeMux
}
