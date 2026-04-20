package internal

import (
	"fmt"
	"time"
)

func FormatIssuesMessage(results []RepoIssues) string {
	if len(results) == 0 {
		return "📊 *GitHub Issues Report*\n\nNo repositories tracked or no recent issues found."
	}

	message := "📊 *GitHub Issues Report*\n"
	message += fmt.Sprintf("_Generated at %s_\n\n", time.Now().Format("2006-01-02 15:04:05"))
	totalIssues := 0
	for _, result := range results {
		totalIssues += len(result.Issues)
	}

	message += fmt.Sprintf("*Total Issues: %d*\n", totalIssues)
	message += fmt.Sprintf("*Repositories: %d*\n\n", len(results))
	for _, result := range results {
		if len(result.Issues) == 0 {
			continue
		}

		message += fmt.Sprintf("🔹 *%s* (%d issues)\n", result.Repo, len(result.Issues))
		maxIssues := 5
		if len(result.Issues) < maxIssues {
			maxIssues = len(result.Issues)
		}

		for i := 0; i < maxIssues; i++ {
			issue := result.Issues[i]
			message += fmt.Sprintf("  • %s\n", issue.Title)
			message += fmt.Sprintf("    _%s_\n", issue.CreatedAt.Format("Jan 02, 15:04"))
		}
		if len(result.Issues) > maxIssues {
			message += fmt.Sprintf("  ... and %d more\n", len(result.Issues)-maxIssues)
		}
		message += "\n"
	}

	message += "---\n"
	return message
}
