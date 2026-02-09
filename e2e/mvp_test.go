package e2e

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seemrkz/netsec-sk/internal/cli"
)

func TestMVPAcceptanceChecklist(t *testing.T) {
	repo := t.TempDir()
	envID := "prod"

	runOK(t, []string{"init", "--repo", repo})
	configGitIdentity(t, repo)
	runOK(t, []string{"env", "create", envID, "--repo", repo})

	archive := filepath.Join(repo, "fw.tgz")
	unsupported := filepath.Join(repo, "skip.txt")
	writeTGZ(t, archive, "tmp/cli/PA-440_ts.tgz.txt", "firewall\nserial: SER-E2E-001\nhostname: fw-e2e\nmodel: PA-440\nsw_version: 11.0.0\nmgmt_ip: 10.10.10.1")
	if err := os.WriteFile(unsupported, []byte("not-archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _ := runCode(t, []string{"--repo", repo, "--env", envID, "ingest", archive, unsupported}, 6)
	if !strings.Contains(out, "attempted=2") || !strings.Contains(out, "committed=1") || !strings.Contains(out, "parse_error_fatal=1") {
		t.Fatalf("unexpected ingest summary: %q", out)
	}

	// Duplicate unchanged ingest should not add commits.
	out, _ = runOK(t, []string{"--repo", repo, "--env", envID, "ingest", archive})
	if !strings.Contains(out, "committed=0") || !strings.Contains(out, "skipped_state_unchanged=1") {
		t.Fatalf("unexpected unchanged summary: %q", out)
	}

	commitCount := strings.TrimSpace(string(mustExec(t, "git", "-C", repo, "rev-list", "--count", "HEAD")))
	if commitCount != "1" {
		t.Fatalf("commit count=%q, want 1", commitCount)
	}

	ingestLog := filepath.Join(repo, ".netsec-state", "ingest.ndjson")
	rows := readNDJSONRows(t, ingestLog)
	if len(rows) != 3 {
		t.Fatalf("ingest row count=%d, want 3", len(rows))
	}
	foundUnsupported := false
	for _, row := range rows {
		if row["result"] == "parse_error_fatal" && row["notes"] == "unsupported_extension" {
			foundUnsupported = true
		}
	}
	if !foundUnsupported {
		t.Fatalf("missing unsupported_extension ingest row: %#v", rows)
	}

	out, _ = runOK(t, []string{"--repo", repo, "--env", envID, "show", "device", "SER-E2E-001"})
	if !strings.Contains(out, "\"hostname\": \"fw-e2e\"") {
		t.Fatalf("show output missing expected hostname: %q", out)
	}

	_, _ = runOK(t, []string{"--repo", repo, "--env", envID, "export"})
	requiredExports := []string{
		"environment.json",
		"inventory.csv",
		"nodes.csv",
		"edges.csv",
		"topology.mmd",
		"agent_context.md",
	}
	for _, name := range requiredExports {
		p := filepath.Join(repo, "envs", envID, "exports", name)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing export %s: %v", name, err)
		}
	}
	inventory := string(mustRead(t, filepath.Join(repo, "envs", envID, "exports", "inventory.csv")))
	if !strings.HasPrefix(inventory, "entity_type,entity_id,hostname,serial,model,version,mgmt_ip,ha_enabled,ha_mode,ha_state,routing_protocols_configured,routing_protocols_active,source_tsf_id,state_sha256,last_ingested_at_utc\n") {
		t.Fatalf("inventory header mismatch: %q", inventory)
	}

	out, _ = runOK(t, []string{"--repo", repo, "--env", envID, "devices"})
	if !strings.Contains(out, "SER-E2E-001\tfw-e2e\tPA-440\t11.0.0\t10.10.10.1") {
		t.Fatalf("devices output missing row: %q", out)
	}
	out, _ = runOK(t, []string{"--repo", repo, "--env", envID, "topology"})
	if !strings.HasPrefix(out, "Topology edges: ") {
		t.Fatalf("unexpected topology output: %q", out)
	}

	helpOut, _ := runOK(t, []string{"help", "ingest"})
	for _, s := range []string{"Usage:", "Arguments:", "Examples:", "Exit codes:"} {
		if !strings.Contains(helpOut, s) {
			t.Fatalf("help ingest missing %q: %q", s, helpOut)
		}
	}

	// Open shell should continue after non-fatal errors.
	in := strings.NewReader("show device missing\ndevices\nquit\n")
	var shellOut, shellErr bytes.Buffer
	if code := cli.RunOpenSession(in, &shellOut, &shellErr, []string{"--repo", repo, "--env", envID}); code != 0 {
		t.Fatalf("RunOpenSession code=%d err=%q", code, shellErr.String())
	}
	if !strings.Contains(shellErr.String(), "ERROR E_IO ") || !strings.Contains(shellOut.String(), "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP") {
		t.Fatalf("open shell resilience failed out=%q err=%q", shellOut.String(), shellErr.String())
	}

	// .netsec-state must never be committed.
	tracked := string(mustExec(t, "git", "-C", repo, "ls-files"))
	if strings.Contains(tracked, ".netsec-state/") {
		t.Fatalf("forbidden tracked path found:\n%s", tracked)
	}
}

func runOK(t *testing.T, args []string) (string, string) {
	t.Helper()
	return runCode(t, args, 0)
}

func runCode(t *testing.T, args []string, wantCode int) (string, string) {
	t.Helper()
	var out, err bytes.Buffer
	if code := cli.Run(args, &out, &err); code != wantCode {
		t.Fatalf("command failed code=%d want=%d args=%v stdout=%q stderr=%q", code, wantCode, args, out.String(), err.String())
	}
	return out.String(), err.String()
}

func configGitIdentity(t *testing.T, repo string) {
	t.Helper()
	mustExec(t, "git", "-C", repo, "config", "user.email", "e2e@example.com")
	mustExec(t, "git", "-C", repo, "config", "user.name", "E2E")
}

func mustExec(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(out))
	}
	return out
}

func writeTGZ(t *testing.T, path string, name string, body string) {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readNDJSONRows(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	out := make([]map[string]any, 0)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		row := map[string]any{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		out = append(out, row)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
