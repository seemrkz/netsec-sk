package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/seemrkz/netsec-sk/internal/tsf"
)

type fakeInspector struct {
	start map[int]int64
}

func (f fakeInspector) ProcessStartUnix(pid int) (int64, bool) {
	v, ok := f.start[pid]
	return v, ok
}

func TestInputOrdering(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	if err := os.MkdirAll(filepath.Join(dirA, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirB, 0o755); err != nil {
		t.Fatal(err)
	}

	files := []string{
		filepath.Join(dirA, "z.tgz"),
		filepath.Join(dirA, "nested", "m.tar.gz"),
		filepath.Join(dirB, "a.tgz"),
		filepath.Join(dirB, "skip.txt"),
	}
	for _, path := range files {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ResolveInputs([]string{dirA, filepath.Join(dirB, "a.tgz"), filepath.Join(dirB, "skip.txt")})
	if err != nil {
		t.Fatalf("ResolveInputs() unexpected error: %v", err)
	}

	want := []string{
		filepath.Join(dirA, "nested", "m.tar.gz"),
		filepath.Join(dirA, "z.tgz"),
		filepath.Join(dirB, "a.tgz"),
	}
	for i := range want {
		want[i] = filepath.Clean(want[i])
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveInputs() = %#v, want %#v", got, want)
	}
}

func TestLockStaleRules(t *testing.T) {
	root := t.TempDir()
	now := time.Unix(1_700_000_000, 0).UTC()

	_, err := AcquireLock(root, now, 101, "ingest", fakeInspector{
		start: map[int]int64{101: now.Unix()},
	})
	if err != nil {
		t.Fatalf("AcquireLock() unexpected error: %v", err)
	}

	_, err = AcquireLock(root, now.Add(30*time.Minute), 202, "ingest", fakeInspector{
		start: map[int]int64{101: now.Unix()},
	})
	if err != ErrLockHeld {
		t.Fatalf("AcquireLock() active lock error = %v, want %v", err, ErrLockHeld)
	}

	// stale due to PID start mismatch
	warnings, err := AcquireLock(root, now.Add(31*time.Minute), 303, "ingest", fakeInspector{
		start: map[int]int64{101: now.Unix() + 1, 303: now.Add(31 * time.Minute).Unix()},
	})
	if err != nil {
		t.Fatalf("AcquireLock() stale mismatch unexpected error: %v", err)
	}
	if len(warnings) != 1 || warnings[0] != "stale_lock_removed" {
		t.Fatalf("warnings = %#v, want stale_lock_removed", warnings)
	}

	// stale due to age > 8h
	warnings, err = AcquireLock(root, now.Add(9*time.Hour), 404, "ingest", fakeInspector{
		start: map[int]int64{303: now.Add(31 * time.Minute).Unix(), 404: now.Add(9 * time.Hour).Unix()},
	})
	if err != nil {
		t.Fatalf("AcquireLock() stale age unexpected error: %v", err)
	}
	if len(warnings) != 1 || warnings[0] != "stale_lock_removed" {
		t.Fatalf("warnings = %#v, want stale_lock_removed", warnings)
	}
}

func TestExtractCleanup(t *testing.T) {
	repoPath := t.TempDir()
	now := time.Unix(1_700_000_000, 0).UTC()

	oldRun := filepath.Join(repoPath, ".netsec-state", "extract", "old-run")
	if err := os.MkdirAll(oldRun, 0o755); err != nil {
		t.Fatal(err)
	}
	oldTime := now.Add(-25 * time.Hour)
	if err := os.Chtimes(oldRun, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	prep, err := Prepare(PrepareOptions{
		RepoPath: repoPath,
		EnvIDRaw: "Prod",
		Inputs:   []string{},
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Prepare() unexpected error: %v", err)
	}
	if prep.EnvID != "prod" {
		t.Fatalf("Prepare() env = %q, want prod", prep.EnvID)
	}
	if _, err := os.Stat(oldRun); !os.IsNotExist(err) {
		t.Fatalf("expected stale run dir removed, stat err = %v", err)
	}

	extractDir, err := BeginTSFExtractDir(prep.RunExtractRoot, "archive-1.tgz", 1)
	if err != nil {
		t.Fatalf("BeginTSFExtractDir() unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(extractDir, "tmp.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := FinishTSFExtractDir(extractDir, false); err != nil {
		t.Fatalf("FinishTSFExtractDir(remove) unexpected error: %v", err)
	}
	if _, err := os.Stat(extractDir); !os.IsNotExist(err) {
		t.Fatalf("expected extract dir removed, stat err = %v", err)
	}

	keepDir, err := BeginTSFExtractDir(prep.RunExtractRoot, "archive-2.tgz", 2)
	if err != nil {
		t.Fatalf("BeginTSFExtractDir(keep) unexpected error: %v", err)
	}
	if err := FinishTSFExtractDir(keepDir, true); err != nil {
		t.Fatalf("FinishTSFExtractDir(keep) unexpected error: %v", err)
	}
	if _, err := os.Stat(keepDir); err != nil {
		t.Fatalf("expected kept extract dir to exist: %v", err)
	}
}

func TestDuplicateDetection(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "ingest.ndjson")

	lines := []string{
		`{"env_id":"prod","tsf_id":"SER001|PA-440_ts.tgz","result":"committed"}`,
		`{"env_id":"dev","tsf_id":"SER001|PA-440_ts.tgz","result":"committed"}`,
		`{"env_id":"prod","tsf_id":"unknown","result":"parse_error_fatal"}`,
	}
	if err := os.WriteFile(logPath, []byte(lines[0]+"\n"+lines[1]+"\n"+lines[2]+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	seen, err := ReadSeenTSFIDs(logPath, "prod")
	if err != nil {
		t.Fatalf("ReadSeenTSFIDs() unexpected error: %v", err)
	}
	if !IsDuplicateTSF("SER001|PA-440_ts.tgz", seen) {
		t.Fatalf("expected duplicate for seen TSF ID")
	}
	if IsDuplicateTSF("unknown", seen) {
		t.Fatalf("unknown TSF must not be treated as duplicate")
	}
	if IsDuplicateTSF("SER999|other_ts.tgz", seen) {
		t.Fatalf("unexpected duplicate for unseen TSF ID")
	}

	readFile := func(path string) ([]byte, error) {
		content := map[string]string{
			"a/tmp/cli/PA-440_ts.tgz.txt": "serial: SER001",
			"b/tmp/cli/PA-440_ts.tgz.txt": "serial: SER001",
		}
		v, ok := content[path]
		if !ok {
			return nil, fmt.Errorf("missing %s", path)
		}
		return []byte(v), nil
	}

	id1 := tsf.DeriveIdentity([]string{"a/tmp/cli/PA-440_ts.tgz.txt"}, readFile)
	id2 := tsf.DeriveIdentity([]string{"b/tmp/cli/PA-440_ts.tgz.txt"}, readFile)
	if id1.TSFID != id2.TSFID {
		t.Fatalf("expected identity match for renamed archives: %q != %q", id1.TSFID, id2.TSFID)
	}

	seen[id1.TSFID] = struct{}{}
	if !IsDuplicateTSF(id2.TSFID, seen) {
		t.Fatalf("expected renamed archive to dedupe by internal identity")
	}
}

func TestIngestLedgerAllAttempts(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, ".netsec-state", "ingest.ndjson")

	entries := []IngestLogEntry{
		{AttemptedAtUTC: "2026-02-09T00:00:00Z", RunID: "run-1", EnvID: "prod", InputArchivePath: "/x/a.tgz", TSFID: "A|a.tgz", Result: "committed", GitCommit: "abc"},
		{AttemptedAtUTC: "2026-02-09T00:01:00Z", RunID: "run-1", EnvID: "prod", InputArchivePath: "/x/b.tgz", TSFID: "A|a.tgz", Result: "skipped_duplicate_tsf"},
		{AttemptedAtUTC: "2026-02-09T00:02:00Z", RunID: "run-1", EnvID: "prod", InputArchivePath: "/x/c.tgz", TSFID: "unknown", Result: "parse_error_fatal", Notes: "unsupported_extension"},
	}
	for _, e := range entries {
		if err := AppendIngestAttempt(logPath, e); err != nil {
			t.Fatalf("AppendIngestAttempt() err=%v", err)
		}
	}

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open ingest log err=%v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
		var got IngestLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &got); err != nil {
			t.Fatalf("invalid ndjson row: %v", err)
		}
		if got.EnvID != "prod" || got.Result == "" || got.RunID != "run-1" {
			t.Fatalf("missing required fields in row %#v", got)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan err=%v", err)
	}
	if count != len(entries) {
		t.Fatalf("row count=%d, want %d", count, len(entries))
	}
}
