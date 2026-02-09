package repo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrGitMissing = errors.New("git executable is not available on PATH")
var ErrRepoUnsafe = errors.New("repository is in an unsafe state for ingest")

type LookPathFunc func(file string) (string, error)

func CheckGitAvailable(lookPath LookPathFunc) error {
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	if _, err := lookPath("git"); err != nil {
		return fmt.Errorf("%w: %v", ErrGitMissing, err)
	}

	return nil
}

func CheckSafeWorkingTree(repoPath string) error {
	if hasUnsafeGitOperation(repoPath) {
		return ErrRepoUnsafe
	}

	// For non-git directories, this check is a no-op for now.
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		if x == '?' && y == '?' {
			continue
		}
		if x != ' ' || y != ' ' {
			return ErrRepoUnsafe
		}
	}

	return nil
}

func hasUnsafeGitOperation(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	markers := []string{
		"MERGE_HEAD",
		"CHERRY_PICK_HEAD",
		"rebase-apply",
		"rebase-merge",
	}

	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(gitDir, marker)); err == nil {
			return true
		}
	}
	return false
}
