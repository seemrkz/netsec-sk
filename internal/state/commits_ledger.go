package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type CommitLedgerEntry struct {
	CommittedAtUTC string `json:"committed_at_utc"`
	TSFID          string `json:"tsf_id"`
	TSFOriginal    string `json:"tsf_original_name"`
	EntityType     string `json:"entity_type"`
	EntityID       string `json:"entity_id"`
	StateSHA256    string `json:"state_sha256"`
	GitCommit      string `json:"git_commit"`
	Summary        string `json:"summary,omitempty"`
}

func AppendCommitLedger(path string, entry CommitLedgerEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
