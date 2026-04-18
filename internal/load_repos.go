package internal

import (
	"encoding/json"
	"fmt"
)

func LoadReposFromFile() ([]string, error) {
	reposJSON := `[
	  "ethereum/go-ethereum"
	]`

	var repos []string
	if err := json.Unmarshal([]byte(reposJSON), &repos); err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
		return nil, err
	}

	return repos, nil
}
