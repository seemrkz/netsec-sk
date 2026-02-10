package ingest

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/seemrkz/netsec-sk/internal/repo"
	"github.com/seemrkz/netsec-sk/internal/state"
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
		filepath.Join(dirB, "skip.txt"),
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

func TestMultiTSFAttemptAndCommitOutcomes(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)

	inputDir := filepath.Join(repoPath, "inputs")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(inputDir, "01-first.tgz")
	duplicatePath := filepath.Join(inputDir, "02-duplicate.tgz")
	secondPath := filepath.Join(inputDir, "03-second.tgz")

	if err := writeTGZ(firstPath, []tarEntry{{Name: "tmp/cli/PA-440_ts.tgz.txt", Body: "firewall\nserial: B1\nhostname: fw-b1\nmgmt_ip: 10.6.6.1"}}); err != nil {
		t.Fatal(err)
	}
	if err := writeTGZ(duplicatePath, []tarEntry{{Name: "tmp/cli/PA-440_ts.tgz.txt", Body: "firewall\nserial: B1\nhostname: fw-b1\nmgmt_ip: 10.6.6.1"}}); err != nil {
		t.Fatal(err)
	}
	if err := writeTGZ(secondPath, []tarEntry{{Name: "tmp/cli/PA-445_ts.tgz.txt", Body: "firewall\nserial: B2\nhostname: fw-b2\nmgmt_ip: 10.6.6.2"}}); err != nil {
		t.Fatal(err)
	}

	summary, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{inputDir},
		Now:      time.Unix(1_700_005_000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run() err=%v", err)
	}
	if summary.Attempted != 3 || summary.Committed != 2 || summary.SkippedDuplicateTSF != 0 || summary.SkippedStateUnchanged != 1 || summary.ParseErrorPartial != 0 || summary.ParseErrorFatal != 0 {
		t.Fatalf("summary=%+v, want attempted=3 committed=2 skipped_state_unchanged=1", summary)
	}

	logPath := filepath.Join(repoPath, ".netsec-state", "ingest.ndjson")
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open ingest log err=%v", err)
	}
	defer f.Close()

	committed := 0
	unchanged := 0
	total := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		total++
		var row IngestLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatalf("invalid ingest row: %v", err)
		}
		switch row.Result {
		case "committed":
			committed++
			if row.GitCommit == "" {
				t.Fatalf("committed row missing git commit: %#v", row)
			}
		case "skipped_state_unchanged":
			unchanged++
			if row.GitCommit != "" {
				t.Fatalf("unchanged row must not include git commit: %#v", row)
			}
		default:
			t.Fatalf("unexpected ingest row result: %#v", row)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan ingest log err=%v", err)
	}
	if total != 3 || committed != 2 || unchanged != 1 {
		t.Fatalf("ingest rows total=%d committed=%d unchanged=%d, want 3/2/1", total, committed, unchanged)
	}

	countOut, err := exec.Command("git", "-C", repoPath, "rev-list", "--count", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list failed: %v, out=%s", err, string(countOut))
	}
	if strings.TrimSpace(string(countOut)) != "2" {
		t.Fatalf("commit count=%q, want 2", strings.TrimSpace(string(countOut)))
	}
}

func TestArchiveExtractionSupportedFormats(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)
	dir := filepath.Join(repoPath, "inputs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(dir, "a.tgz")
	tarGzPath := filepath.Join(dir, "b.tar.gz")
	if err := writeTGZ(tgzPath, []tarEntry{{Name: "tmp/cli/a.txt", Body: "firewall\nserial: A1\nhostname: fw-a\nmgmt_ip: 10.0.0.1"}}); err != nil {
		t.Fatal(err)
	}
	if err := writeTGZ(tarGzPath, []tarEntry{{Name: "tmp/cli/b.txt", Body: "panorama\nserial: P1\nhostname: p1\nmgmt_ip: 10.0.0.2"}}); err != nil {
		t.Fatal(err)
	}

	summary, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{dir},
		Now:      time.Unix(1_700_000_100, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if summary.Attempted != 2 || summary.ParseErrorFatal != 0 || summary.SkippedStateUnchanged != 0 {
		t.Fatalf("summary=%+v, want attempted=2 parse_error_fatal=0 skipped_state_unchanged=0", summary)
	}

	extractRoot := filepath.Join(repoPath, ".netsec-state", "extract")
	runDirs, err := os.ReadDir(extractRoot)
	if err != nil {
		t.Fatalf("read extract root err=%v", err)
	}
	if len(runDirs) != 1 {
		t.Fatalf("run dir count=%d, want 1", len(runDirs))
	}
	leftovers, err := os.ReadDir(filepath.Join(extractRoot, runDirs[0].Name()))
	if err != nil {
		t.Fatalf("read run dir err=%v", err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("expected per-archive extract cleanup, leftovers=%d", len(leftovers))
	}
}

func TestArchivePathTraversalRejected(t *testing.T) {
	repoPath := t.TempDir()
	archivePath := filepath.Join(repoPath, "bad.tgz")
	if err := writeTGZ(archivePath, []tarEntry{{Name: "link-out", Typeflag: tar.TypeSymlink, Linkname: "../../escape.txt"}}); err != nil {
		t.Fatal(err)
	}

	summary, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archivePath},
		Now:      time.Unix(1_700_000_200, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if summary.Attempted != 1 || summary.ParseErrorFatal != 1 || summary.SkippedStateUnchanged != 0 {
		t.Fatalf("summary=%+v, want attempted=1 parse_error_fatal=1 skipped_state_unchanged=0", summary)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("escape file should not be created, stat err=%v", err)
	}
}

func TestUnsupportedExtensionAccounting(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)
	dir := filepath.Join(repoPath, "inputs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := writeTGZ(filepath.Join(dir, "good.tgz"), []tarEntry{{Name: "tmp/cli/good.txt", Body: "firewall\nserial: G1\nhostname: fw-g\nmgmt_ip: 10.0.0.3"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("txt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{dir},
		Now:      time.Unix(1_700_000_300, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if summary.Attempted != 3 {
		t.Fatalf("attempted=%d, want 3", summary.Attempted)
	}
	if summary.ParseErrorFatal != 2 {
		t.Fatalf("parse_error_fatal=%d, want 2", summary.ParseErrorFatal)
	}
	if summary.SkippedStateUnchanged != 0 {
		t.Fatalf("skipped_state_unchanged=%d, want 0", summary.SkippedStateUnchanged)
	}
}

func TestRepoUnsafeStateBlocksIngest(t *testing.T) {
	repoPath := t.TempDir()
	if _, err := repo.Init(repoPath); err != nil {
		t.Skipf("git unavailable for repo unsafe test: %v", err)
	}

	tracked := filepath.Join(repoPath, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", repoPath, "add", "tracked.txt").Run(); err != nil {
		t.Fatalf("git add tracked failed: %v", err)
	}
	if err := os.WriteFile(tracked, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(repoPath, "a.tgz")
	if err := writeTGZ(archivePath, []tarEntry{{Name: "tmp/cli/a.txt", Body: "ok"}}); err != nil {
		t.Fatal(err)
	}

	_, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archivePath},
		Now:      time.Unix(1_700_000_400, 0).UTC(),
	})
	if err == nil {
		t.Fatal("expected unsafe repo error, got nil")
	}
	if err != repo.ErrRepoUnsafe {
		t.Fatalf("Run() err=%v, want %v", err, repo.ErrRepoUnsafe)
	}
}

func TestSnapshotPersistenceOnChange(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)

	dir := filepath.Join(repoPath, "inputs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "fw.tgz")

	if err := writeTGZ(archive, []tarEntry{{Name: "tmp/cli/fw.txt", Body: "firewall\nserial: S1\nhostname: fw1\nmgmt_ip: 10.0.0.1"}}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_700_000_500, 0).UTC()
	first, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archive},
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Run(first) err=%v", err)
	}
	if first.Attempted != 1 || first.SkippedStateUnchanged != 0 || first.ParseErrorFatal != 0 {
		t.Fatalf("first summary=%+v", first)
	}

	latest := filepath.Join(repoPath, "envs", "prod", "state", "devices", "S1", "latest.json")
	if _, err := os.Stat(latest); err != nil {
		t.Fatalf("missing latest.json: %v", err)
	}
	snapshotDir := filepath.Join(repoPath, "envs", "prod", "state", "devices", "S1", "snapshots")
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		t.Fatalf("read snapshots dir err=%v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("snapshot count=%d, want 1", len(entries))
	}

	second, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archive},
		Now:      now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Run(second same) err=%v", err)
	}
	if second.SkippedStateUnchanged != 1 {
		t.Fatalf("second skipped_state_unchanged=%d, want 1", second.SkippedStateUnchanged)
	}
	entries, err = os.ReadDir(snapshotDir)
	if err != nil {
		t.Fatalf("read snapshots dir err=%v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("snapshot count after unchanged=%d, want 1", len(entries))
	}

	if err := writeTGZ(archive, []tarEntry{{Name: "tmp/cli/fw.txt", Body: "firewall\nserial: S1\nhostname: fw1-changed\nmgmt_ip: 10.0.0.1"}}); err != nil {
		t.Fatal(err)
	}
	third, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archive},
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Run(third changed) err=%v", err)
	}
	if third.SkippedStateUnchanged != 0 || third.ParseErrorFatal != 0 {
		t.Fatalf("third summary=%+v", third)
	}
	entries, err = os.ReadDir(snapshotDir)
	if err != nil {
		t.Fatalf("read snapshots dir err=%v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("snapshot count after changed=%d, want 2", len(entries))
	}
}

func TestRDNSOnlyForNewDevicesInRuntime(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)

	archive := filepath.Join(repoPath, "fw.tgz")
	if err := writeTGZ(archive, []tarEntry{{Name: "tmp/cli/fw.txt", Body: "firewall\nserial: S-RDNS\nhostname: fw-rdns\nmgmt_ip: 10.1.1.1"}}); err != nil {
		t.Fatal(err)
	}

	originalLookup := rdnsLookup
	defer func() { rdnsLookup = originalLookup }()
	calls := 0
	rdnsLookup = func(ctx context.Context, ip string) (string, error) {
		calls++
		return "fw-rdns.example.net", nil
	}

	now := time.Unix(1_700_001_000, 0).UTC()
	first, err := Run(RunOptions{
		RepoPath:   repoPath,
		EnvIDRaw:   "prod",
		Inputs:     []string{archive},
		EnableRDNS: true,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("Run(first) err=%v", err)
	}
	if calls != 1 {
		t.Fatalf("rdns calls after first run=%d, want 1", calls)
	}
	if first.ParseErrorFatal != 0 {
		t.Fatalf("first summary=%+v", first)
	}

	latestPath := filepath.Join(repoPath, "envs", "prod", "state", "devices", "S-RDNS", "latest.json")
	data, err := os.ReadFile(latestPath)
	if err != nil {
		t.Fatalf("read latest err=%v", err)
	}
	var latest map[string]any
	if err := json.Unmarshal(data, &latest); err != nil {
		t.Fatalf("unmarshal latest err=%v", err)
	}
	device := latest["device"].(map[string]any)
	dns := device["dns"].(map[string]any)
	reverse := dns["reverse"].(map[string]any)
	if reverse["status"] != "ok" || reverse["ptr_name"] != "fw-rdns.example.net" {
		t.Fatalf("unexpected reverse dns fields: %#v", reverse)
	}

	second, err := Run(RunOptions{
		RepoPath:   repoPath,
		EnvIDRaw:   "prod",
		Inputs:     []string{archive},
		EnableRDNS: true,
		Now:        now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Run(second) err=%v", err)
	}
	if calls != 1 {
		t.Fatalf("rdns calls after second run=%d, want still 1", calls)
	}
	if second.SkippedStateUnchanged != 1 {
		t.Fatalf("second summary=%+v", second)
	}
}

func TestCommittedResultCreatesOneCommit(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)

	archive := filepath.Join(repoPath, "fw.tgz")
	if err := writeTGZ(archive, []tarEntry{{Name: "tmp/cli/fw.txt", Body: "firewall\nserial: C1\nhostname: fw-commit\nmgmt_ip: 10.2.2.2"}}); err != nil {
		t.Fatal(err)
	}

	summary, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archive},
		Now:      time.Unix(1_700_002_000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run() err=%v", err)
	}
	if summary.Committed != 1 {
		t.Fatalf("summary.committed=%d, want 1 (summary=%+v)", summary.Committed, summary)
	}

	countOut, err := exec.Command("git", "-C", repoPath, "rev-list", "--count", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list failed: %v, out=%s", err, string(countOut))
	}
	if strings.TrimSpace(string(countOut)) != "1" {
		t.Fatalf("commit count=%q, want 1", strings.TrimSpace(string(countOut)))
	}

	logPath := filepath.Join(repoPath, ".netsec-state", "ingest.ndjson")
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open ingest log err=%v", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	rows := 0
	for scanner.Scan() {
		rows++
		var row IngestLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatalf("unmarshal ingest row: %v", err)
		}
		if row.Result != "committed" || row.GitCommit == "" {
			t.Fatalf("unexpected ingest row: %#v", row)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan ingest log err=%v", err)
	}
	if rows != 1 {
		t.Fatalf("ingest row count=%d, want 1", rows)
	}
}

func TestCommitLedgerIncludesChangedScopeAndPaths(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)

	archive := filepath.Join(repoPath, "fw.tgz")
	if err := writeTGZ(archive, []tarEntry{{Name: "tmp/cli/fw.txt", Body: "firewall\nserial: L1\nhostname: fw-ledger\nmgmt_ip: 10.3.3.3"}}); err != nil {
		t.Fatal(err)
	}

	summary, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archive},
		Now:      time.Unix(1_700_003_000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run() err=%v", err)
	}
	if summary.Committed != 1 {
		t.Fatalf("summary.committed=%d, want 1 (summary=%+v)", summary.Committed, summary)
	}

	entries := readCommitLedgerEntries(t, filepath.Join(repoPath, "envs", "prod", "state", "commits.ndjson"))
	if len(entries) != 1 {
		t.Fatalf("commit ledger rows=%d, want 1", len(entries))
	}

	row := entries[0]
	if row.ChangedScope == "" {
		t.Fatalf("changed_scope must be non-empty: %#v", row)
	}
	if row.ChangedScope != "device,feature,route,other" {
		t.Fatalf("changed_scope=%q, want %q", row.ChangedScope, "device,feature,route,other")
	}
	if len(row.ChangedPaths) != 3 {
		t.Fatalf("changed_paths len=%d, want 3 (%#v)", len(row.ChangedPaths), row.ChangedPaths)
	}
	if row.ChangedPaths[0] != "envs/prod/state/commits.ndjson" {
		t.Fatalf("changed_paths[0]=%q, want commits ledger path", row.ChangedPaths[0])
	}
	if row.ChangedPaths[1] != "envs/prod/state/devices/L1/latest.json" {
		t.Fatalf("changed_paths[1]=%q, want latest path", row.ChangedPaths[1])
	}
	if !strings.HasPrefix(row.ChangedPaths[2], "envs/prod/state/devices/L1/snapshots/") {
		t.Fatalf("changed_paths[2]=%q, want snapshot path prefix", row.ChangedPaths[2])
	}
}

func TestChangedPathsLexicalRepoRelative(t *testing.T) {
	repoPath := t.TempDir()
	initRepoWithIdentity(t, repoPath)

	archive := filepath.Join(repoPath, "fw.tgz")
	if err := writeTGZ(archive, []tarEntry{{Name: "tmp/cli/fw.txt", Body: "firewall\nserial: L2\nhostname: fw-ledger\nmgmt_ip: 10.4.4.4"}}); err != nil {
		t.Fatal(err)
	}

	summary, err := Run(RunOptions{
		RepoPath: repoPath,
		EnvIDRaw: "prod",
		Inputs:   []string{archive},
		Now:      time.Unix(1_700_004_000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Run() err=%v", err)
	}
	if summary.Committed != 1 {
		t.Fatalf("summary.committed=%d, want 1 (summary=%+v)", summary.Committed, summary)
	}

	entries := readCommitLedgerEntries(t, filepath.Join(repoPath, "envs", "prod", "state", "commits.ndjson"))
	if len(entries) != 1 {
		t.Fatalf("commit ledger rows=%d, want 1", len(entries))
	}
	paths := entries[0].ChangedPaths
	if len(paths) == 0 {
		t.Fatal("changed_paths must be non-empty")
	}
	for _, p := range paths {
		if filepath.IsAbs(p) {
			t.Fatalf("changed path must be repo-relative, got %q", p)
		}
		if filepath.ToSlash(p) != p {
			t.Fatalf("changed path must use forward slashes, got %q", p)
		}
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(paths, sorted) {
		t.Fatalf("changed_paths must be lexical, got=%#v want=%#v", paths, sorted)
	}
}

type tarEntry struct {
	Name     string
	Body     string
	Typeflag byte
	Linkname string
}

func writeTGZ(path string, entries []tarEntry) error {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for _, entry := range entries {
		typeflag := entry.Typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		hdr := &tar.Header{
			Name:     entry.Name,
			Mode:     0o644,
			Size:     int64(len(entry.Body)),
			Typeflag: typeflag,
			Linkname: entry.Linkname,
		}
		if typeflag != tar.TypeReg && typeflag != tar.TypeRegA {
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if typeflag == tar.TypeReg || typeflag == tar.TypeRegA {
			if _, err := tw.Write([]byte(entry.Body)); err != nil {
				return err
			}
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func initRepoWithIdentity(t *testing.T, repoPath string) {
	t.Helper()
	if _, err := repo.Init(repoPath); err != nil {
		t.Skipf("git unavailable for test repo init: %v", err)
	}
	if err := exec.Command("git", "-C", repoPath, "config", "user.email", "tests@example.com").Run(); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}
	if err := exec.Command("git", "-C", repoPath, "config", "user.name", "Tests").Run(); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}
}

func readCommitLedgerEntries(t *testing.T, path string) []state.CommitLedgerEntry {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open commit ledger err=%v", err)
	}
	defer f.Close()

	out := make([]state.CommitLedgerEntry, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row state.CommitLedgerEntry
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("invalid commit ledger row: %v", err)
		}
		out = append(out, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan commit ledger err=%v", err)
	}
	return out
}
