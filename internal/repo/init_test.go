package repo

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesBaseLayout(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "state-repo")

	absRepoPath, err := InitWithDeps(repoPath, InitDeps{
		LookPath: func(file string) (string, error) {
			if file != "git" {
				t.Fatalf("unexpected LookPath input: %s", file)
			}
			return "/usr/bin/git", nil
		},
		GitInit: func(p string) error {
			if p != filepath.Clean(repoPath) {
				t.Fatalf("unexpected repo path: got %s want %s", p, filepath.Clean(repoPath))
			}
			return os.MkdirAll(filepath.Join(p, ".git"), 0o755)
		},
	})
	if err != nil {
		t.Fatalf("InitWithDeps() unexpected error: %v", err)
	}

	if absRepoPath != filepath.Clean(repoPath) {
		t.Fatalf("absolute path mismatch: got %s want %s", absRepoPath, filepath.Clean(repoPath))
	}

	requiredPaths := []string{
		filepath.Join(repoPath, ".git"),
		filepath.Join(repoPath, "envs"),
		filepath.Join(repoPath, ".netsec-state"),
		filepath.Join(repoPath, ".netsec-state", "extract"),
	}
	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path to exist %s: %v", path, err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(repoPath, "envs"))
	if err != nil {
		t.Fatalf("ReadDir(envs) unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no environments at init, got %d", len(entries))
	}

	gitIgnore, err := os.ReadFile(filepath.Join(repoPath, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) unexpected error: %v", err)
	}
	if string(gitIgnore) != ".netsec-state/\n" {
		t.Fatalf("unexpected .gitignore content: %q", string(gitIgnore))
	}
}

func TestInitFailsWithoutGit(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "state-repo")

	_, err := InitWithDeps(repoPath, InitDeps{
		LookPath: func(file string) (string, error) {
			return "", errors.New("not found")
		},
		GitInit: func(string) error {
			t.Fatal("GitInit should not be called when git is missing")
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrGitMissing) {
		t.Fatalf("expected ErrGitMissing, got %v", err)
	}
}
