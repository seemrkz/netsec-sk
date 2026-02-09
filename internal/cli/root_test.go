package cli

import (
	"bytes"
	"errors"
	"os"
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

	cases := [][]string{
		{"devices", "--repo", repoPath, "--env", "prod"},
		{"panorama", "--repo", repoPath, "--env", "prod"},
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
	if stdout.String() != "Topology edges: 1\nOrphan zones: 1\n" {
		t.Fatalf("topology output mismatch: %q", stdout.String())
	}
}
