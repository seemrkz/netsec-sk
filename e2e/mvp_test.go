package e2e

import (
	"bytes"
	"testing"

	"github.com/seemrkz/netsec-sk/internal/cli"
)

func TestMVPAcceptanceChecklist(t *testing.T) {
	repo := t.TempDir()
	commands := [][]string{
		{"--repo", repo, "init"},
		{"--repo", repo, "env", "create", "prod"},
		{"--repo", repo, "--env", "prod", "env", "list"},
		{"--repo", repo, "--env", "prod", "help"},
		{"--repo", repo, "--env", "prod", "open"},
	}
	for _, args := range commands {
		var out, err bytes.Buffer
		if code := cli.Run(args, &out, &err); code != 0 {
			t.Fatalf("command failed code=%d args=%v stderr=%q", code, args, err.String())
		}
	}
}
