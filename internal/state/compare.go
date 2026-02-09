package state

import (
	"encoding/json"
	"errors"
	"os"
)

func IsStateUnchanged(snapshot map[string]any, latestPath string) (bool, string, error) {
	currentHash, err := ComputeStateSHA256(snapshot)
	if err != nil {
		return false, "", err
	}

	latestData, err := os.ReadFile(latestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, currentHash, nil
		}
		return false, "", err
	}

	var latest map[string]any
	if err := json.Unmarshal(latestData, &latest); err != nil {
		return false, "", err
	}

	latestHash, err := ComputeStateSHA256(latest)
	if err != nil {
		return false, "", err
	}

	return latestHash == currentHash, currentHash, nil
}
