package internal

import "fmt"

func FormatIssuesMessage(results []RepoIssues) string {
	if len(results) == 0 {
		return "No repositories tracked or no recent issues found."
	}

	message := ""
	for _, result := range results {
		if len(result.Issues) == 0 {
			continue
		}

		message += fmt.Sprintf("*%s*\n", result.Repo)
		for _, issue := range result.Issues {
			message += fmt.Sprintf("- %s - %s\n", issue.Title, issue.HTMLURL)
		}
		message += "\n"
	}

	return message
}
