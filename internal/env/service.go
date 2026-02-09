package env

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type Service struct {
	repoPath string
}

func NewService(repoPath string) Service {
	return Service{repoPath: repoPath}
}

func (s Service) Create(rawEnvID string) (string, bool, error) {
	envID := NormalizeEnvID(rawEnvID)
	if err := ValidateEnvID(envID); err != nil {
		return "", false, fmt.Errorf("%w: %s", ErrInvalidEnvID, envID)
	}

	envRoot := filepath.Join(s.repoPath, "envs", envID)
	if _, err := os.Stat(envRoot); err == nil {
		return envID, false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", false, err
	}

	dirs := []string{
		filepath.Join(envRoot, "state"),
		filepath.Join(envRoot, "exports"),
		filepath.Join(envRoot, "overrides"),
	}
	for _, path := range dirs {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return "", false, err
		}
	}

	return envID, true, nil
}

func (s Service) List() ([]string, error) {
	envsPath := filepath.Join(s.repoPath, "envs")
	entries, err := os.ReadDir(envsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		envID := entry.Name()
		if ValidateEnvID(envID) != nil {
			continue
		}
		out = append(out, envID)
	}

	sort.Strings(out)
	return out, nil
}
