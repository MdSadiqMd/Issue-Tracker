package internal

import (
	"fmt"

	"github.com/syumai/workers/cloudflare"
)

type GistConfig struct {
	GistID      string
	AccessToken string
}

func GetGistConfig() (*GistConfig, error) {
	gistID := cloudflare.Getenv("GIST_ID")
	accessToken := cloudflare.Getenv("GITHUB_ACCESS_TOKEN")
	if gistID == "" || accessToken == "" {
		return nil, fmt.Errorf("Missing GIST_ID or GITHUB_ACCESS_TOKEN environment variables")
	}
	return &GistConfig{
		GistID:      gistID,
		AccessToken: accessToken,
	}, nil
}

type GreenAPIConfig struct {
	APIURL string
	ChatID string
}

func GetGreenAPIConfig() (*GreenAPIConfig, error) {
	apiURL := cloudflare.Getenv("GREEN_API_URL")
	chatID := cloudflare.Getenv("CHAT_ID")
	if apiURL == "" || chatID == "" {
		return nil, fmt.Errorf("Missing GREEN_API_URL or CHAT_ID environment variables")
	}
	return &GreenAPIConfig{
		APIURL: apiURL,
		ChatID: chatID,
	}, nil
}
