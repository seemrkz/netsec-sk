package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

func PersistIfChanged(snapshot map[string]any, latestPath string, snapshotDir string, snapshotStamp string) (bool, string, string, error) {
	unchanged, hash, err := IsStateUnchanged(snapshot, latestPath)
	if err != nil {
		return false, "", "", err
	}
	if unchanged {
		return true, hash, "", nil
	}

	snapshot["state_sha256"] = hash

	if err := os.MkdirAll(filepath.Dir(latestPath), 0o755); err != nil {
		return false, "", "", err
	}
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return false, "", "", err
	}

	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return false, "", "", err
	}
	body = append(body, '\n')

	if err := os.WriteFile(latestPath, body, 0o644); err != nil {
		return false, "", "", err
	}

	snapshotName := snapshotStamp + "_" + hash + ".json"
	snapshotPath := filepath.Join(snapshotDir, snapshotName)
	if err := os.WriteFile(snapshotPath, body, 0o644); err != nil {
		return false, "", "", err
	}

	return false, hash, snapshotPath, nil
}
