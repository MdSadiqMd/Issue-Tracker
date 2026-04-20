package internal

import (
	"encoding/json"
	"fmt"

	"github.com/MdSadiqMd/issue-tracker/pkg"
)

type RepoObject struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

func FetchGistRepos(gistID, accessToken string) ([]RepoObject, error) {
	url := fmt.Sprintf("https://api.github.com/gists/%s", gistID)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"Accept":        "application/vnd.github.v3+json",
		"User-Agent":    "Issue-Tracker-Worker",
	}

	data, err := pkg.FetchJS(url, "GET", headers, "")
	if err != nil {
		return nil, fmt.Errorf("error fetching gist: %v", err)
	}

	var gistData map[string]interface{}
	if err := json.Unmarshal(data, &gistData); err != nil {
		return nil, fmt.Errorf("error decoding gist: %v", err)
	}

	files, ok := gistData["files"].(map[string]interface{})
	if !ok {
		return []RepoObject{}, nil
	}

	reposFile, ok := files["repos.json"].(map[string]interface{})
	if !ok {
		return []RepoObject{}, nil
	}

	content, ok := reposFile["content"].(string)
	if !ok || content == "" {
		return []RepoObject{}, nil
	}

	var repoObjects []RepoObject
	if err := json.Unmarshal([]byte(content), &repoObjects); err != nil {
		return nil, fmt.Errorf("error parsing repos.json: %v", err)
	}
	return repoObjects, nil
}
