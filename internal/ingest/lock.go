package ingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const lockStaleAfter = 8 * time.Hour

var ErrLockHeld = errors.New("ingest lock is currently held by an active process")

type LockFile struct {
	PID           int    `json:"pid"`
	StartedAtUTC  string `json:"started_at_utc"`
	StartedAtUnix int64  `json:"started_at_unix"`
	Command       string `json:"command"`
}

type LockInspector interface {
	ProcessStartUnix(pid int) (int64, bool)
}

func AcquireLock(repoPath string, now time.Time, pid int, command string, inspector LockInspector) ([]string, error) {
	lockPath := filepath.Join(repoPath, ".netsec-state", "lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}

	warnings := make([]string, 0)
	data, err := os.ReadFile(lockPath)
	if err == nil {
		var current LockFile
		if json.Unmarshal(data, &current) == nil {
			if !isStaleLock(current, now, inspector) {
				return nil, ErrLockHeld
			}
		}

		if rmErr := os.Remove(lockPath); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return nil, rmErr
		}
		warnings = append(warnings, "stale_lock_removed")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	lock := LockFile{
		PID:           pid,
		StartedAtUTC:  now.UTC().Format(time.RFC3339),
		StartedAtUnix: now.UTC().Unix(),
		Command:       command,
	}
	encoded, err := json.Marshal(lock)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(lockPath, encoded, 0o644); err != nil {
		return nil, err
	}

	return warnings, nil
}

func ReleaseLock(repoPath string) error {
	lockPath := filepath.Join(repoPath, ".netsec-state", "lock")
	if err := os.Remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func isStaleLock(lock LockFile, now time.Time, inspector LockInspector) bool {
	if lock.PID <= 0 || lock.StartedAtUnix <= 0 {
		return true
	}

	if now.UTC().Unix()-lock.StartedAtUnix > int64(lockStaleAfter.Seconds()) {
		return true
	}

	if inspector == nil {
		return true
	}

	start, ok := inspector.ProcessStartUnix(lock.PID)
	if !ok {
		return true
	}

	return start != lock.StartedAtUnix
}

func lockPath(repoPath string) string {
	return filepath.Join(repoPath, ".netsec-state", "lock")
}

func ReadLock(repoPath string) (LockFile, error) {
	data, err := os.ReadFile(lockPath(repoPath))
	if err != nil {
		return LockFile{}, err
	}

	var out LockFile
	if err := json.Unmarshal(data, &out); err != nil {
		return LockFile{}, fmt.Errorf("decode lock: %w", err)
	}
	return out, nil
}
