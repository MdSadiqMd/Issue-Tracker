package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	internal "github.com/MdSadiqMd/issue-tracker/internal"
	"github.com/MdSadiqMd/issue-tracker/pkg"
	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare/cron"
)

func main() {
	cron.ScheduleTaskNonBlock(pkg.CronTask)
	http.HandleFunc("/{$}", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Thalaivar</title><style>html,body{height:100%;margin:0;background:#000;display:grid;place-items:center}img{max-width:100%;max-height:100%;object-fit:contain}</style></head><body><img src="/Thalaivar%20GIF%20by%20RajiniGifs.gif" alt="Thalaivar GIF"></body></html>`))
	})
	http.HandleFunc("/fetch-issues", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
			return
		}

		results, err := internal.FetchIssuesLogic()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})
	http.HandleFunc("/send-report", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg, err := internal.GetGreenAPIConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		results, err := internal.FetchIssuesLogic()
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			errorMsg := fmt.Sprintf("Error fetching GitHub issues\n\n%v", err)
			if sendErr := pkg.SendWhatsAppMessage(cfg.APIURL, cfg.IdInstance, cfg.ApiTokenInstance, cfg.ChatID, errorMsg); sendErr != nil {
				http.Error(w, fmt.Sprintf("Failed to send error notification: %v", sendErr), http.StatusInternalServerError)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to fetch issues: %v", err), http.StatusInternalServerError)
			return
		}

		totalIssues := 0
		for _, r := range results {
			totalIssues += len(r.Issues)
		}

		if totalIssues == 0 {
			fmt.Println("No issues found, skipping WhatsApp notification")
			response := map[string]interface{}{
				"success":      true,
				"message":      "No issues found, notification skipped",
				"issues_count": 0,
				"repos_count":  len(results),
				"skipped":      true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		message := internal.FormatIssuesMessage(results)
		if err := pkg.SendWhatsAppMessage(cfg.APIURL, cfg.IdInstance, cfg.ApiTokenInstance, cfg.ChatID, message); err != nil {
			fmt.Printf("Error sending WhatsApp message: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to send WhatsApp message: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success":      true,
			"message":      "WhatsApp report sent successfully",
			"issues_count": totalIssues,
			"repos_count":  len(results),
			"skipped":      false,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	http.HandleFunc("/add-repo", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg, err := internal.GetGistConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		gistID := cfg.GistID
		accessToken := cfg.AccessToken

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
