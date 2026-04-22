package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	internal "github.com/MdSadiqMd/issue-tracker/internal"
	"github.com/MdSadiqMd/issue-tracker/pkg"
	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare/cron"
)

func main() {
	cron.ScheduleTaskNonBlock(pkg.CronTask)
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
