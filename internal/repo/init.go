package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type InitDeps struct {
	LookPath LookPathFunc
	GitInit  func(repoPath string) error
}

func Init(repoPath string) (string, error) {
	return InitWithDeps(repoPath, InitDeps{})
}

func InitWithDeps(repoPath string, deps InitDeps) (string, error) {
	if deps.LookPath == nil {
		deps.LookPath = exec.LookPath
	}
	if deps.GitInit == nil {
		deps.GitInit = gitInit
	}

	if err := CheckGitAvailable(deps.LookPath); err != nil {
		return "", err
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repo path: %w", err)
	}

	if err := os.MkdirAll(absRepoPath, 0o755); err != nil {
		return "", fmt.Errorf("create repo directory: %w", err)
	}

	if _, err := os.Stat(filepath.Join(absRepoPath, ".git")); os.IsNotExist(err) {
		if err := deps.GitInit(absRepoPath); err != nil {
			return "", fmt.Errorf("git init failed: %w", err)
		}
	}

	if err := createBaseLayout(absRepoPath); err != nil {
		return "", fmt.Errorf("create base layout: %w", err)
	}

	return absRepoPath, nil
}

func gitInit(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "init")
	return cmd.Run()
}
