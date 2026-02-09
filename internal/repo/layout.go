package repo

import (
	"os"
	"path/filepath"
	"strings"
)

func createBaseLayout(repoPath string) error {
	paths := []string{
		filepath.Join(repoPath, "envs"),
		filepath.Join(repoPath, ".netsec-state"),
		filepath.Join(repoPath, ".netsec-state", "extract"),
	}

	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	return ensureGitIgnoreEntry(repoPath, ".netsec-state/")
}

func ensureGitIgnoreEntry(repoPath, entry string) error {
	gitIgnorePath := filepath.Join(repoPath, ".gitignore")
	content, err := os.ReadFile(gitIgnorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(content)
	if strings.Contains(text, entry) {
		return nil
	}

	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += entry + "\n"

	return os.WriteFile(gitIgnorePath, []byte(text), 0o644)
}
