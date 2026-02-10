package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	exportpkg "github.com/seemrkz/netsec-sk/internal/export"
	"github.com/seemrkz/netsec-sk/internal/ingest"
	"github.com/seemrkz/netsec-sk/internal/repo"
)

func TestGlobalFlags(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got, err := ParseGlobalFlags([]string{"init"})
		if err != nil {
			t.Fatalf("ParseGlobalFlags() unexpected error: %v", err)
		}

		if got.GlobalOptions.RepoPath != "./default" {
			t.Fatalf("repo default mismatch: got %q", got.GlobalOptions.RepoPath)
		}
		if got.GlobalOptions.EnvID != "default" {
			t.Fatalf("env default mismatch: got %q", got.GlobalOptions.EnvID)
		}
		if len(got.CommandArgs) != 1 || got.CommandArgs[0] != "init" {
			t.Fatalf("command args mismatch: got %#v", got.CommandArgs)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		got, err := ParseGlobalFlags([]string{"--repo", "/tmp/repo", "--env", "dev", "ingest"})
		if err != nil {
			t.Fatalf("ParseGlobalFlags() unexpected error: %v", err)
		}

		if got.GlobalOptions.RepoPath != "/tmp/repo" {
			t.Fatalf("repo override mismatch: got %q", got.GlobalOptions.RepoPath)
		}
		if got.GlobalOptions.EnvID != "dev" {
			t.Fatalf("env override mismatch: got %q", got.GlobalOptions.EnvID)
		}
		if len(got.CommandArgs) != 1 || got.CommandArgs[0] != "ingest" {
			t.Fatalf("command args mismatch: got %#v", got.CommandArgs)
		}
	})

	t.Run("command first with trailing globals", func(t *testing.T) {
		got, err := ParseGlobalFlags([]string{"ingest", "--repo", "/tmp/repo", "--env", "dev"})
		if err != nil {
			t.Fatalf("ParseGlobalFlags() unexpected error: %v", err)
		}

		if got.GlobalOptions.RepoPath != "/tmp/repo" {
			t.Fatalf("repo override mismatch: got %q", got.GlobalOptions.RepoPath)
		}
		if got.GlobalOptions.EnvID != "dev" {
			t.Fatalf("env override mismatch: got %q", got.GlobalOptions.EnvID)
		}
		if len(got.CommandArgs) != 1 || got.CommandArgs[0] != "ingest" {
			t.Fatalf("command args mismatch: got %#v", got.CommandArgs)
		}
	})
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantLine string
	}{
		{
			name:     "usage",
			err:      NewAppError(ErrUsage, "bad args"),
			wantCode: 2,
			wantLine: "ERROR E_USAGE bad args",
		},
		{
			name:     "git missing",
			err:      NewAppError(ErrGitMissing, "git missing"),
			wantCode: 3,
			wantLine: "ERROR E_GIT_MISSING git missing",
		},
		{
			name:     "repo unsafe",
			err:      NewAppError(ErrRepoUnsafe, "dirty"),
			wantCode: 4,
			wantLine: "ERROR E_REPO_UNSAFE dirty",
		},
		{
			name:     "lock held",
			err:      NewAppError(ErrLockHeld, "active lock"),
			wantCode: 5,
			wantLine: "ERROR E_LOCK_HELD active lock",
		},
		{
			name:     "parse fatal",
			err:      NewAppError(ErrParseFatal, "bad tsf"),
			wantCode: 6,
			wantLine: "ERROR E_PARSE_FATAL bad tsf",
		},
		{
			name:     "io",
			err:      NewAppError(ErrIO, "write failed"),
			wantCode: 6,
			wantLine: "ERROR E_IO write failed",
		},
		{
			name:     "parse partial",
			err:      NewAppError(ErrParsePart, "missing optional fields"),
			wantCode: 7,
			wantLine: "ERROR E_PARSE_PARTIAL missing optional fields",
		},
		{
			name:     "internal",
			err:      NewAppError(ErrInternal, "panic recovered"),
			wantCode: 9,
			wantLine: "ERROR E_INTERNAL panic recovered",
		},
		{
			name:     "non app error maps to internal",
			err:      errors.New("unexpected"),
			wantCode: 9,
			wantLine: "ERROR E_INTERNAL unexpected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCodeFor(tt.err); got != tt.wantCode {
				t.Fatalf("ExitCodeFor() = %d, want %d", got, tt.wantCode)
			}

			if got := FormatErrorLine(tt.err); got != tt.wantLine {
				t.Fatalf("FormatErrorLine() = %q, want %q", got, tt.wantLine)
			}
		})
	}
}

func TestEnvCommandOutputs(t *testing.T) {
	repoPath := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exit := Run([]string{"--repo", repoPath, "env", "create", "Dev"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("Run(env create) exit = %d, want 0", exit)
	}
	if stderr.String() != "" {
		t.Fatalf("Run(env create) stderr = %q, want empty", stderr.String())
	}
	if stdout.String() != "Environment created: dev\n" {
		t.Fatalf("Run(env create) stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"--repo", repoPath, "env", "create", "dev"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("Run(env create idempotent) exit = %d, want 0", exit)
	}
	if stdout.String() != "Environment already exists: dev\n" {
		t.Fatalf("Run(env create idempotent) stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"--repo", repoPath, "env", "create", "BAD_NAME"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("Run(env create invalid) exit = %d, want 2", exit)
	}
	if stderr.String() != "ERROR E_USAGE invalid env_id: bad_name\n" {
		t.Fatalf("Run(env create invalid) stderr = %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"--repo", repoPath, "env", "create", "prod"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("Run(env create prod) exit = %d, want 0", exit)
	}

	stdout.Reset()
	stderr.Reset()
	exit = Run([]string{"--repo", repoPath, "env", "list"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("Run(env list) exit = %d, want 0", exit)
	}
	if stderr.String() != "" {
		t.Fatalf("Run(env list) stderr = %q, want empty", stderr.String())
	}
	if stdout.String() != "dev\nprod\n" {
		t.Fatalf("Run(env list) stdout = %q", stdout.String())
	}

	expectedDirs := []string{
		filepath.Join(repoPath, "envs", "dev", "state"),
		filepath.Join(repoPath, "envs", "dev", "exports"),
		filepath.Join(repoPath, "envs", "dev", "overrides"),
	}
	for _, dir := range expectedDirs {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("expected directory %s to exist", dir)
		}
	}
}

func TestEnvRepresentativeIDs(t *testing.T) {
	repoPath := t.TempDir()
	representative := []string{"Prod", "Development", "Cloud", "Lab", "Home", "CustomerA", "CustomerB"}
	for _, input := range representative {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"env", "create", input, "--repo", repoPath}, &stdout, &stderr); code != 0 {
			t.Fatalf("env create input=%q code=%d stderr=%q", input, code, stderr.String())
		}
		if stderr.String() != "" {
			t.Fatalf("env create input=%q unexpected stderr=%q", input, stderr.String())
		}
		if !strings.HasPrefix(stdout.String(), "Environment created: ") {
			t.Fatalf("env create input=%q unexpected stdout=%q", input, stdout.String())
		}
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--repo", repoPath, "env", "list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("env list code=%d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("env list stderr=%q, want empty", stderr.String())
	}
	want := "cloud\ncustomera\ncustomerb\ndevelopment\nhome\nlab\nprod\n"
	if stdout.String() != want {
		t.Fatalf("env list output mismatch:\n%s", stdout.String())
	}
}

func TestGlobalFlagPlacementCompatibility(t *testing.T) {
	repoPath := t.TempDir()
	var stdout, stderr bytes.Buffer
	ingestPath := filepath.Join(repoPath, "a.tgz")
	if err := writeTestTGZ(ingestPath); err != nil {
		t.Fatal(err)
	}
	initGitRepoForTests(t, repoPath)

	if code := Run([]string{"--repo", repoPath, "init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("global-first init failed code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Initialized repository: ") {
		t.Fatalf("unexpected init stdout: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--repo", repoPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("command-first init failed code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Initialized repository: ") {
		t.Fatalf("unexpected init stdout: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"env", "create", "Prod", "--repo", repoPath}, &stdout, &stderr); code != 0 || stdout.String() != "Environment created: prod\n" {
		t.Fatalf("command-first env create failed code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}

	devicePath := filepath.Join(repoPath, "envs", "prod", "state", "devices", "id1")
	panoPath := filepath.Join(repoPath, "envs", "prod", "state", "panorama", "id2")
	if err := os.MkdirAll(devicePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(panoPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devicePath, "latest.json"), []byte(`{"kind":"device"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(panoPath, "latest.json"), []byte(`{"kind":"panorama"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	exportsPath := filepath.Join(repoPath, "envs", "prod", "exports")
	if err := os.MkdirAll(exportsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exportsPath, "topology.mmd"), []byte("graph TD\nA-->B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := [][]string{
		{"devices", "--repo", repoPath, "--env", "prod"},
		{"panorama", "--repo", repoPath, "--env", "prod"},
		{"history", "state", "--repo", repoPath, "--env", "prod"},
		{"topology", "--repo", repoPath, "--env", "prod"},
		{"ingest", ingestPath, "--repo", repoPath, "--env", "prod"},
		{"export", "--repo", repoPath, "--env", "prod"},
		{"show", "device", "id1", "--repo", repoPath, "--env", "prod"},
		{"show", "panorama", "id2", "--repo", repoPath, "--env", "prod"},
		{"help", "devices", "--repo", repoPath, "--env", "prod"},
		{"open", "--repo", repoPath, "--env", "prod"},
	}
	for _, args := range cases {
		stdout.Reset()
		stderr.Reset()
		if code := Run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("command-first args=%v failed code=%d err=%q", args, code, stderr.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--bad-flag", "init"}, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid global flag code=%d want 2, stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE ") {
		t.Fatalf("invalid global flag stderr mismatch: %q", stderr.String())
	}
}

func TestIngestCommandUsesRuntime(t *testing.T) {
	original := ingestRun
	defer func() { ingestRun = original }()

	var gotOpts ingest.RunOptions
	ingestRun = func(opts ingest.RunOptions) (ingest.Summary, error) {
		gotOpts = opts
		return ingest.Summary{
			Attempted:             3,
			Committed:             1,
			SkippedDuplicateTSF:   1,
			SkippedStateUnchanged: 1,
		}, nil
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--repo", "/tmp/repo", "--env", "prod", "ingest", "a.tgz", "b.tar.gz", "c.tgz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(ingest) code=%d, want 0 stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("Run(ingest) stderr=%q, want empty", stderr.String())
	}
	if gotOpts.RepoPath != "/tmp/repo" || gotOpts.EnvIDRaw != "prod" {
		t.Fatalf("runtime options mismatch: %#v", gotOpts)
	}
	if strings.Join(gotOpts.Inputs, ",") != "a.tgz,b.tar.gz,c.tgz" {
		t.Fatalf("runtime inputs mismatch: %#v", gotOpts.Inputs)
	}

	want := "Ingest complete: attempted=3 committed=1 skipped_duplicate_tsf=1 skipped_state_unchanged=1 parse_error_partial=0 parse_error_fatal=0\n"
	if stdout.String() != want {
		t.Fatalf("Run(ingest) stdout=%q, want %q", stdout.String(), want)
	}
}

func TestIngestExitCodePrecedence(t *testing.T) {
	tests := []struct {
		name string
		out  ingest.Summary
		want int
	}{
		{
			name: "fatal takes precedence",
			out: ingest.Summary{
				Attempted:         1,
				ParseErrorPartial: 1,
				ParseErrorFatal:   1,
			},
			want: 6,
		},
		{
			name: "partial when no fatal",
			out: ingest.Summary{
				Attempted:         1,
				ParseErrorPartial: 1,
			},
			want: 7,
		},
		{
			name: "success when no errors",
			out: ingest.Summary{
				Attempted: 1,
				Committed: 1,
			},
			want: 0,
		},
	}

	original := ingestRun
	defer func() { ingestRun = original }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingestRun = func(opts ingest.RunOptions) (ingest.Summary, error) {
				return tt.out, nil
			}
			var stdout, stderr bytes.Buffer
			got := Run([]string{"ingest", "a.tgz"}, &stdout, &stderr)
			if got != tt.want {
				t.Fatalf("Run(ingest) code=%d, want %d stdout=%q stderr=%q", got, tt.want, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "Ingest complete: attempted=1") {
				t.Fatalf("summary line missing expected prefix: %q", stdout.String())
			}
		})
	}
}

func TestIngestReportsContextOnSummaryFailures(t *testing.T) {
	original := ingestRun
	defer func() { ingestRun = original }()

	ingestRun = func(opts ingest.RunOptions) (ingest.Summary, error) {
		return ingest.Summary{
			Attempted:       1,
			ParseErrorFatal: 1,
			Issues: []ingest.Issue{
				{InputArchivePath: "/x/a.tgz", Result: "parse_error_fatal", Notes: "extract_failed", Error: "unsafe archive entry path"},
			},
		}, nil
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--repo", "/tmp/repo", "--env", "prod", "ingest", "/x/a.tgz"}, &stdout, &stderr)
	if code != 6 {
		t.Fatalf("Run(ingest) code=%d, want 6 stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Ingest issues (fatal):") || !strings.Contains(stdout.String(), "- /x/a.tgz: extract_failed") {
		t.Fatalf("stdout missing issues context: %q", stdout.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_PARSE_FATAL ") || !strings.Contains(stderr.String(), "log=/tmp/repo/.netsec-state/ingest.ndjson") {
		t.Fatalf("stderr missing parse fatal context: %q", stderr.String())
	}
}

func TestIngestErrorCodeMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantPrefix string
	}{
		{
			name:       "repo unsafe",
			err:        repo.ErrRepoUnsafe,
			wantCode:   4,
			wantPrefix: "ERROR E_REPO_UNSAFE ",
		},
		{
			name:       "lock held",
			err:        ingest.ErrLockHeld,
			wantCode:   5,
			wantPrefix: "ERROR E_LOCK_HELD ",
		},
	}

	original := ingestRun
	defer func() { ingestRun = original }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingestRun = func(opts ingest.RunOptions) (ingest.Summary, error) {
				return ingest.Summary{}, tt.err
			}

			var stdout, stderr bytes.Buffer
			code := Run([]string{"ingest", "a.tgz"}, &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("Run(ingest) code=%d, want %d stderr=%q", code, tt.wantCode, stderr.String())
			}
			if stdout.String() != "" {
				t.Fatalf("Run(ingest) stdout=%q, want empty", stdout.String())
			}
			if !strings.HasPrefix(stderr.String(), tt.wantPrefix) {
				t.Fatalf("Run(ingest) stderr=%q, want prefix %q", stderr.String(), tt.wantPrefix)
			}
		})
	}
}

func TestIngestFlagParsingAndPassThrough(t *testing.T) {
	original := ingestRun
	defer func() { ingestRun = original }()

	var got ingest.RunOptions
	ingestRun = func(opts ingest.RunOptions) (ingest.Summary, error) {
		got = opts
		return ingest.Summary{Attempted: 1}, nil
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--repo", "/tmp/repo", "--env", "prod", "ingest", "--rdns", "--keep-extract", "a.tgz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(ingest flags) code=%d stderr=%q", code, stderr.String())
	}
	if !got.EnableRDNS || !got.KeepExtract {
		t.Fatalf("flags not passed through: %+v", got)
	}
	if len(got.Inputs) != 1 || got.Inputs[0] != "a.tgz" {
		t.Fatalf("unexpected ingest inputs: %#v", got.Inputs)
	}
	if !strings.Contains(stdout.String(), "Ingest complete: attempted=1") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"ingest", "--bad-opt", "a.tgz"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run(ingest bad flag) code=%d want 2", code)
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE unknown ingest option: --bad-opt") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestExportCommandContract(t *testing.T) {
	original := exportRun
	defer func() { exportRun = original }()

	var got exportpkg.RunOptions
	exportRun = func(opts exportpkg.RunOptions) error {
		got = opts
		return nil
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--repo", "/tmp/repo", "--env", "prod", "export"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(export) code=%d stderr=%q", code, stderr.String())
	}
	if stdout.String() != "Export complete: prod\n" {
		t.Fatalf("Run(export) stdout=%q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("Run(export) stderr=%q", stderr.String())
	}
	if got.RepoPath != "/tmp/repo" || got.EnvID != "prod" {
		t.Fatalf("unexpected export options: %+v", got)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"export", "extra"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run(export extra) code=%d want 2", code)
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE usage: netsec-sk export") {
		t.Fatalf("unexpected usage stderr: %q", stderr.String())
	}
}

func TestOpenShellContinuesAfterError(t *testing.T) {
	repo := t.TempDir()
	initGitRepoForTests(t, repo)
	_ = os.MkdirAll(filepath.Join(repo, "envs", "default", "state", "devices", "id1"), 0o755)
	_ = os.WriteFile(filepath.Join(repo, "envs", "default", "state", "devices", "id1", "latest.json"), []byte(`{"device":{"id":"id1","hostname":"fw1","model":"PA-440","sw_version":"11.0.0","mgmt_ip":"10.0.0.1"}}`), 0o644)

	in := strings.NewReader("show device missing-id\ndevices\nquit\n")
	var out, errOut bytes.Buffer
	code := RunOpenSession(in, &out, &errOut, []string{"--repo", repo, "--env", "default"})
	if code != 0 {
		t.Fatalf("RunOpenSession code=%d, want 0", code)
	}
	if !strings.Contains(errOut.String(), "ERROR E_IO ") {
		t.Fatalf("expected command error in shell stderr, got %q", errOut.String())
	}
	if !strings.Contains(out.String(), "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP") {
		t.Fatalf("shell did not continue to next command after error, out=%q", out.String())
	}
}

func TestHelpCommandContracts(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help code=%d stderr=%q", code, stderr.String())
	}
	required := []string{
		"init: initialize repository",
		"env: list/create environments",
		"ingest: ingest TSF archives into state",
		"history: print deterministic state-change history",
		"open: interactive shell",
	}
	for _, r := range required {
		if !strings.Contains(stdout.String(), r) {
			t.Fatalf("help missing %q in %q", r, stdout.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"help", "ingest"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help ingest code=%d stderr=%q", code, stderr.String())
	}
	for _, label := range []string{"Usage: netsec-sk ingest", "Arguments:", "Examples:", "Exit codes:"} {
		if !strings.Contains(stdout.String(), label) {
			t.Fatalf("help ingest missing %q in %q", label, stdout.String())
		}
	}
}

func TestHistoryStateCommandContract(t *testing.T) {
	repoPath := t.TempDir()
	envID := "prod"

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--repo", repoPath, "--env", envID, "history", "state"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history state empty code=%d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("history state empty stderr=%q, want empty", stderr.String())
	}
	if stdout.String() != "COMMITTED_AT_UTC\tGIT_COMMIT\tTSF_ID\tTSF_ORIGINAL_NAME\tCHANGED_SCOPE\n" {
		t.Fatalf("history state empty output mismatch: %q", stdout.String())
	}

	ledgerPath := filepath.Join(repoPath, "envs", envID, "state", "commits.ndjson")
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{
		{
			"committed_at_utc":  "2026-02-10T10:00:00Z",
			"git_commit":        "aaa111",
			"tsf_id":            "S1|a.tgz",
			"tsf_original_name": "a.tgz",
			"entity_type":       "firewall",
			"entity_id":         "S1",
			"state_sha256":      "abc",
			"changed_scope":     "device",
			"changed_paths":     []string{"envs/prod/state/devices/S1/latest.json"},
		},
	}
	var b strings.Builder
	for _, row := range rows {
		line, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(ledgerPath, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"history", "state", "--repo", repoPath, "--env", envID}, &stdout, &stderr); code != 0 {
		t.Fatalf("history state command-first code=%d stderr=%q", code, stderr.String())
	}
	want := "COMMITTED_AT_UTC\tGIT_COMMIT\tTSF_ID\tTSF_ORIGINAL_NAME\tCHANGED_SCOPE\n" +
		"2026-02-10T10:00:00Z\taaa111\tS1|a.tgz\ta.tgz\tdevice\n"
	if stdout.String() != want {
		t.Fatalf("history state output mismatch:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"history", "state", "extra", "--repo", repoPath, "--env", envID}, &stdout, &stderr); code != 2 {
		t.Fatalf("history state usage code=%d want 2 stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE usage: netsec-sk history state") {
		t.Fatalf("history state usage stderr mismatch: %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"history", "--repo", repoPath, "--env", envID}, &stdout, &stderr); code != 2 {
		t.Fatalf("history missing subcommand code=%d want 2 stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE history requires a subcommand: state") {
		t.Fatalf("history missing subcommand stderr mismatch: %q", stderr.String())
	}

	if err := os.WriteFile(ledgerPath, []byte("{bad-json}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--repo", repoPath, "--env", envID, "history", "state"}, &stdout, &stderr); code != 6 {
		t.Fatalf("history parse error code=%d want 6 stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_IO ") {
		t.Fatalf("history parse error stderr mismatch: %q", stderr.String())
	}
}

func TestHistoryStateSortOrder(t *testing.T) {
	repoPath := t.TempDir()
	envID := "prod"
	ledgerPath := filepath.Join(repoPath, "envs", envID, "state", "commits.ndjson")
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		t.Fatal(err)
	}

	rows := []map[string]any{
		{
			"committed_at_utc":  "2026-02-10T10:00:00Z",
			"git_commit":        "bbb222",
			"tsf_id":            "S2|b.tgz",
			"tsf_original_name": "b.tgz",
			"entity_type":       "firewall",
			"entity_id":         "S2",
			"state_sha256":      "hash2",
			"changed_scope":     "feature",
			"changed_paths":     []string{"envs/prod/state/devices/S2/latest.json"},
		},
		{
			"committed_at_utc":  "2026-02-09T10:00:00Z",
			"git_commit":        "zzz999",
			"tsf_id":            "S1|a.tgz",
			"tsf_original_name": "a.tgz",
			"entity_type":       "firewall",
			"entity_id":         "S1",
			"state_sha256":      "hash1",
			"changed_scope":     "device",
			"changed_paths":     []string{"envs/prod/state/devices/S1/latest.json"},
		},
		{
			"committed_at_utc":  "2026-02-10T10:00:00Z",
			"git_commit":        "aaa111",
			"tsf_id":            "S3|c.tgz",
			"tsf_original_name": "c.tgz",
			"entity_type":       "firewall",
			"entity_id":         "S3",
			"state_sha256":      "hash3",
			"changed_scope":     "route",
			"changed_paths":     []string{"envs/prod/state/devices/S3/latest.json"},
		},
	}
	var b strings.Builder
	for _, row := range rows {
		line, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(ledgerPath, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--repo", repoPath, "--env", envID, "history", "state"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history state code=%d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("history state stderr=%q, want empty", stderr.String())
	}
	want := "COMMITTED_AT_UTC\tGIT_COMMIT\tTSF_ID\tTSF_ORIGINAL_NAME\tCHANGED_SCOPE\n" +
		"2026-02-09T10:00:00Z\tzzz999\tS1|a.tgz\ta.tgz\tdevice\n" +
		"2026-02-10T10:00:00Z\taaa111\tS3|c.tgz\tc.tgz\troute\n" +
		"2026-02-10T10:00:00Z\tbbb222\tS2|b.tgz\tb.tgz\tfeature\n"
	if stdout.String() != want {
		t.Fatalf("history state sort mismatch:\n%s", stdout.String())
	}
}

func TestTopologyCurrentMermaidOutput(t *testing.T) {
	repoPath := t.TempDir()
	envID := "prod"
	exports := filepath.Join(repoPath, "envs", envID, "exports")
	if err := os.MkdirAll(exports, 0o755); err != nil {
		t.Fatal(err)
	}
	want := "graph TD\nA[A-ID] --> B[B-ID]\n"
	if err := os.WriteFile(filepath.Join(exports, "topology.mmd"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--repo", repoPath, "--env", envID, "topology"}, &stdout, &stderr); code != 0 {
		t.Fatalf("topology current code=%d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("topology current stderr=%q, want empty", stderr.String())
	}
	if stdout.String() != want {
		t.Fatalf("topology current output=%q want=%q", stdout.String(), want)
	}
}

func TestTopologyAtCommitOutput(t *testing.T) {
	repoPath := t.TempDir()
	initGitRepoForTests(t, repoPath)
	envID := "prod"
	exports := filepath.Join(repoPath, "envs", envID, "exports")
	if err := os.MkdirAll(exports, 0o755); err != nil {
		t.Fatal(err)
	}

	first := "graph TD\nA-->B\n"
	second := "graph TD\nA-->C\n"
	topologyPath := filepath.Join(exports, "topology.mmd")

	if err := os.WriteFile(topologyPath, []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repoPath, "add", "envs").CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v out=%s", err, string(out))
	}
	if out, err := exec.Command("git", "-C", repoPath, "commit", "-m", "topology first").CombinedOutput(); err != nil {
		t.Fatalf("git commit first failed: %v out=%s", err, string(out))
	}
	hashOut, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse failed: %v out=%s", err, string(hashOut))
	}
	firstHash := strings.TrimSpace(string(hashOut))

	if err := os.WriteFile(topologyPath, []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repoPath, "add", "envs").CombinedOutput(); err != nil {
		t.Fatalf("git add second failed: %v out=%s", err, string(out))
	}
	if out, err := exec.Command("git", "-C", repoPath, "commit", "-m", "topology second").CombinedOutput(); err != nil {
		t.Fatalf("git commit second failed: %v out=%s", err, string(out))
	}

	statusBefore, err := exec.Command("git", "-C", repoPath, "status", "--short").CombinedOutput()
	if err != nil {
		t.Fatalf("git status before failed: %v out=%s", err, string(statusBefore))
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--repo", repoPath, "--env", envID, "topology", "--at-commit", firstHash}, &stdout, &stderr); code != 0 {
		t.Fatalf("topology at-commit code=%d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("topology at-commit stderr=%q, want empty", stderr.String())
	}
	if stdout.String() != first {
		t.Fatalf("topology at-commit output=%q want=%q", stdout.String(), first)
	}

	statusAfter, err := exec.Command("git", "-C", repoPath, "status", "--short").CombinedOutput()
	if err != nil {
		t.Fatalf("git status after failed: %v out=%s", err, string(statusAfter))
	}
	if string(statusAfter) != string(statusBefore) {
		t.Fatalf("topology at-commit must not mutate repo status before=%q after=%q", string(statusBefore), string(statusAfter))
	}
}

func TestTopologyAtCommitValidationAndErrors(t *testing.T) {
	repoPath := t.TempDir()
	initGitRepoForTests(t, repoPath)
	envID := "prod"

	exports := filepath.Join(repoPath, "envs", envID, "exports")
	if err := os.MkdirAll(exports, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exports, "topology.mmd"), []byte("graph TD\nA-->B\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repoPath, "add", "envs").CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v out=%s", err, string(out))
	}
	if out, err := exec.Command("git", "-C", repoPath, "commit", "-m", "topology baseline").CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v out=%s", err, string(out))
	}
	hashOut, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse failed: %v out=%s", err, string(hashOut))
	}
	hash := strings.TrimSpace(string(hashOut))

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--repo", repoPath, "--env", envID, "topology", "--at-commit", "bad-hash"}, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid hash code=%d want 2 stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE invalid --at-commit hash: bad-hash") {
		t.Fatalf("invalid hash stderr=%q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"topology", "--at-commit", "--repo", repoPath, "--env", envID}, &stdout, &stderr); code != 2 {
		t.Fatalf("missing hash value code=%d want 2 stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE usage: netsec-sk topology") {
		t.Fatalf("missing hash value stderr=%q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"topology", "--repo", repoPath, "--env", envID, "--at-commit", hash[:6]}, &stdout, &stderr); code != 2 {
		t.Fatalf("short hash code=%d want 2 stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_USAGE invalid --at-commit hash: ") {
		t.Fatalf("short hash stderr=%q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--repo", repoPath, "--env", "missing", "topology", "--at-commit", hash}, &stdout, &stderr); code != 6 {
		t.Fatalf("missing historical artifact code=%d want 6 stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "ERROR E_IO ") {
		t.Fatalf("missing historical artifact stderr=%q", stderr.String())
	}
}

func TestQueryCommandsFromPersistedState(t *testing.T) {
	repoPath := t.TempDir()
	envID := "prod"

	deviceA := filepath.Join(repoPath, "envs", envID, "state", "devices", "b-id")
	deviceB := filepath.Join(repoPath, "envs", envID, "state", "devices", "a-id")
	panoA := filepath.Join(repoPath, "envs", envID, "state", "panorama", "p2")
	panoB := filepath.Join(repoPath, "envs", envID, "state", "panorama", "p1")
	for _, p := range []string{deviceA, deviceB, panoA, panoB} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(deviceA, "latest.json"), []byte(`{"device":{"id":"b-id","hostname":"fw-b","model":"PA-440","sw_version":"11.0.1","mgmt_ip":"10.0.0.2"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deviceB, "latest.json"), []byte(`{"device":{"id":"a-id","hostname":"fw-a","model":"PA-220","sw_version":"11.0.0","mgmt_ip":"10.0.0.1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(panoA, "latest.json"), []byte(`{"panorama_instance":{"id":"p2","hostname":"pano-b","model":"M-200","version":"11.0.1","mgmt_ip":"10.1.0.2"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(panoB, "latest.json"), []byte(`{"panorama_instance":{"id":"p1","hostname":"pano-a","model":"M-100","version":"11.0.0","mgmt_ip":"10.1.0.1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	exports := filepath.Join(repoPath, "envs", envID, "exports")
	if err := os.MkdirAll(exports, 0o755); err != nil {
		t.Fatal(err)
	}
	edgesCSV := "edge_id,edge_type,src_node_id,dst_node_id,src_device_id,src_zone,src_interface,src_vr,dst_device_id,dst_zone,dst_interface,dst_vr,evidence,source\n" +
		"e1,shared_subnet,zone_a_id_trust,zone_b_id_inside,a-id,trust,eth1,vr1,b-id,inside,eth2,vr1,,inferred\n"
	nodesCSV := "node_id,node_type,env_id,device_id,panorama_id,zone,virtual_router,label\n" +
		"zone_a_id_trust,zone,prod,a-id,,trust,vr1,a-id:trust\n" +
		"zone_b_id_inside,zone,prod,b-id,,inside,vr1,b-id:inside\n" +
		"zone_orphan,zone,prod,a-id,,dmz,vr1,a-id:dmz\n"
	if err := os.WriteFile(filepath.Join(exports, "edges.csv"), []byte(edgesCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exports, "nodes.csv"), []byte(nodesCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	topologyMMD := "graph TD\nzone_a_id_trust-->zone_b_id_inside\n"
	if err := os.WriteFile(filepath.Join(exports, "topology.mmd"), []byte(topologyMMD), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--repo", repoPath, "--env", envID, "devices"}, &stdout, &stderr); code != 0 {
		t.Fatalf("devices code=%d stderr=%q", code, stderr.String())
	}
	wantDevices := "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP\n" +
		"a-id\tfw-a\tPA-220\t11.0.0\t10.0.0.1\n" +
		"b-id\tfw-b\tPA-440\t11.0.1\t10.0.0.2\n"
	if stdout.String() != wantDevices {
		t.Fatalf("devices output mismatch:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--repo", repoPath, "--env", envID, "panorama"}, &stdout, &stderr); code != 0 {
		t.Fatalf("panorama code=%d stderr=%q", code, stderr.String())
	}
	wantPanorama := "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP\n" +
		"p1\tpano-a\tM-100\t11.0.0\t10.1.0.1\n" +
		"p2\tpano-b\tM-200\t11.0.1\t10.1.0.2\n"
	if stdout.String() != wantPanorama {
		t.Fatalf("panorama output mismatch:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--repo", repoPath, "--env", envID, "topology"}, &stdout, &stderr); code != 0 {
		t.Fatalf("topology code=%d stderr=%q", code, stderr.String())
	}
	if stdout.String() != topologyMMD {
		t.Fatalf("topology output mismatch: %q", stdout.String())
	}
}
