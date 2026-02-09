package commit

import (
	"path/filepath"
	"testing"
)

func TestAtomicCommitAllowlist(t *testing.T) {
	repo := "/tmp/repo"
	got := BuildAllowlist(repo, "prod", "firewall", "SER123", "20260209_abcd1234.json")
	if len(got) != 9 {
		t.Fatalf("allowlist len=%d, want 9", len(got))
	}

	wantSnapshot := filepath.Join(repo, "envs", "prod", "state", "devices", "SER123", "snapshots", "20260209_abcd1234.json")
	foundSnapshot := false
	for _, p := range got {
		if p == wantSnapshot {
			foundSnapshot = true
		}
		if filepath.Base(p) == "ingest.ndjson" || filepath.Base(filepath.Dir(p)) == ".netsec-state" {
			t.Fatalf("allowlist includes forbidden metadata path: %s", p)
		}
	}
	if !foundSnapshot {
		t.Fatalf("allowlist missing snapshot path: %s", wantSnapshot)
	}
}

func TestCommitMessageFormat(t *testing.T) {
	subject := BuildCommitSubject(Meta{
		EnvID:      "prod",
		EntityType: "firewall",
		EntityID:   "SER123",
		StateSHA:   "abcdef0123456789",
		TSFID:      "SER123|my tsf.tgz",
	})
	want := "ingest(prod): firewall/SER123 abcdef012345 SER123|my_tsf.tgz"
	if subject != want {
		t.Fatalf("subject=%q\nwant=%q", subject, want)
	}
}
