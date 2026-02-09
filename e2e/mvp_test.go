package e2e

import (
	"bytes"
	"testing"

	"github.com/seemrkz/netsec-sk/internal/cli"
)

func TestMVPAcceptanceChecklist(t *testing.T) {
	repo := t.TempDir()
	commands := [][]string{
		{"init", "--repo", repo},
		{"--repo", repo, "init"},
		{"env", "create", "prod", "--repo", repo},
		{"--repo", repo, "env", "create", "prod"},
		{"env", "list", "--repo", repo, "--env", "prod"},
		{"help", "--repo", repo, "--env", "prod"},
		{"open", "--repo", repo, "--env", "prod"},
	}
	for _, args := range commands {
		var out, err bytes.Buffer
		if code := cli.Run(args, &out, &err); code != 0 {
			t.Fatalf("command failed code=%d args=%v stderr=%q", code, args, err.String())
		}
	}
}
