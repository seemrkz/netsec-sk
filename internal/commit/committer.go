package commit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Meta struct {
	EnvID      string
	EntityType string
	EntityID   string
	StateSHA   string
	TSFID      string
}

var ErrNothingToCommit = errors.New("nothing to commit")

func BuildCommitSubject(m Meta) string {
	short := m.StateSHA
	if len(short) > 12 {
		short = short[:12]
	}
	tsf := strings.ReplaceAll(m.TSFID, " ", "_")
	return "ingest(" + m.EnvID + "): " + m.EntityType + "/" + m.EntityID + " " + short + " " + tsf
}

func BuildAllowlist(repoPath, envID, entityType, entityID, snapshotFile string) []string {
	base := filepath.Join(repoPath, "envs", envID)
	entityDir := "devices"
	if entityType == "panorama" {
		entityDir = "panorama"
	}

	out := []string{
		filepath.Join(base, "state", "commits.ndjson"),
		filepath.Join(base, "state", entityDir, entityID, "latest.json"),
		filepath.Join(base, "state", entityDir, entityID, "snapshots", snapshotFile),
		filepath.Join(base, "exports", "environment.json"),
		filepath.Join(base, "exports", "inventory.csv"),
		filepath.Join(base, "exports", "nodes.csv"),
		filepath.Join(base, "exports", "edges.csv"),
		filepath.Join(base, "exports", "topology.mmd"),
		filepath.Join(base, "exports", "agent_context.md"),
	}
	sort.Strings(out)
	return out
}

func CommitAllowlisted(repoPath string, allowlist []string, subject string) (string, error) {
	stage := make([]string, 0, len(allowlist))
	for _, p := range allowlist {
		if _, err := os.Stat(p); err == nil {
			stage = append(stage, p)
		}
	}
	if len(stage) == 0 {
		return "", ErrNothingToCommit
	}

	args := append([]string{"-C", repoPath, "add", "--"}, stage...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add: %w: %s", err, bytes.TrimSpace(out))
	}

	diff := exec.Command("git", "-C", repoPath, "diff", "--cached", "--quiet")
	if err := diff.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// there are staged changes
		} else {
			return "", fmt.Errorf("git diff --cached --quiet: %w", err)
		}
	} else {
		return "", ErrNothingToCommit
	}

	if out, err := exec.Command("git", "-C", repoPath, "commit", "-m", subject).CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit: %w: %s", err, bytes.TrimSpace(out))
	}
	hashOut, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(hashOut)), nil
}
