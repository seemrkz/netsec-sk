package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandOutputContracts(t *testing.T) {
	repo := t.TempDir()
	initGitRepoForTests(t, repo)
	ingestPath := filepath.Join(repo, "a.tgz")
	devicePath := filepath.Join(repo, "envs", "default", "state", "devices", "id1")
	panoPath := filepath.Join(repo, "envs", "default", "state", "panorama", "id2")
	_ = writeTestTGZ(ingestPath)
	_ = os.MkdirAll(devicePath, 0o755)
	_ = os.MkdirAll(panoPath, 0o755)
	_ = os.WriteFile(filepath.Join(devicePath, "latest.json"), []byte(`{"device":{"id":"id1","hostname":"fw1","model":"PA-440","sw_version":"11.0.0","mgmt_ip":"10.0.0.1"}}`), 0o644)
	_ = os.WriteFile(filepath.Join(panoPath, "latest.json"), []byte(`{"panorama_instance":{"id":"id2","hostname":"pano1","model":"M-200","version":"11.0.0","mgmt_ip":"10.0.0.2"}}`), 0o644)

	// init supports both global-first and command-first invocation.
	{
		var out, err bytes.Buffer
		if code := Run([]string{"--repo", repo, "init"}, &out, &err); code != 0 || err.Len() != 0 || !strings.Contains(out.String(), "Initialized repository: ") {
			t.Fatalf("init global-first failed code=%d out=%q err=%q", code, out.String(), err.String())
		}
	}
	{
		var out, err bytes.Buffer
		if code := Run([]string{"init", "--repo", repo}, &out, &err); code != 0 || err.Len() != 0 || !strings.Contains(out.String(), "Initialized repository: ") {
			t.Fatalf("init command-first failed code=%d out=%q err=%q", code, out.String(), err.String())
		}
	}

	cases := []struct {
		args []string
		want string
		mode string
	}{
		{[]string{"--repo", repo, "devices"}, "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP\nid1\tfw1\tPA-440\t11.0.0\t10.0.0.1\n", "exact"},
		{[]string{"devices", "--repo", repo}, "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP\nid1\tfw1\tPA-440\t11.0.0\t10.0.0.1\n", "exact"},
		{[]string{"--repo", repo, "panorama"}, "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP\nid2\tpano1\tM-200\t11.0.0\t10.0.0.2\n", "exact"},
		{[]string{"panorama", "--repo", repo}, "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP\nid2\tpano1\tM-200\t11.0.0\t10.0.0.2\n", "exact"},
		{[]string{"--repo", repo, "topology"}, "Topology edges: 0\nOrphan zones: 0\n", "exact"},
		{[]string{"topology", "--repo", repo}, "Topology edges: 0\nOrphan zones: 0\n", "exact"},
		{[]string{"--repo", repo, "--env", "default", "export"}, "Export complete: default\n", "exact"},
		{[]string{"export", "--repo", repo, "--env", "default"}, "Export complete: default\n", "exact"},
		{[]string{"--repo", repo, "ingest", ingestPath}, "Ingest complete: attempted=1 committed=1 skipped_duplicate_tsf=0 skipped_state_unchanged=0 parse_error_partial=0 parse_error_fatal=0\n", "exact"},
		{[]string{"ingest", ingestPath, "--repo", repo}, "Ingest complete: attempted=1 committed=0 skipped_duplicate_tsf=0 skipped_state_unchanged=1 parse_error_partial=0 parse_error_fatal=0\n", "exact"},
		{[]string{"--repo", repo, "show", "device", "id1"}, "\"hostname\": \"fw1\"", "contains"},
		{[]string{"show", "device", "id1", "--repo", repo}, "\"hostname\": \"fw1\"", "contains"},
		{[]string{"--repo", repo, "show", "panorama", "id2"}, "\"hostname\": \"pano1\"", "contains"},
		{[]string{"show", "panorama", "id2", "--repo", repo}, "\"hostname\": \"pano1\"", "contains"},
	}
	for _, tc := range cases {
		var out, err bytes.Buffer
		if code := Run(tc.args, &out, &err); code != 0 || err.Len() != 0 {
			t.Fatalf("args=%v code=%d out=%q err=%q", tc.args, code, out.String(), err.String())
		}
		if tc.mode == "exact" && out.String() != tc.want {
			t.Fatalf("args=%v unexpected out=%q want=%q", tc.args, out.String(), tc.want)
		}
		if tc.mode == "contains" && !strings.Contains(out.String(), tc.want) {
			t.Fatalf("args=%v output missing %q in %q", tc.args, tc.want, out.String())
		}
	}

	var out, err bytes.Buffer
	if code := Run([]string{"help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "open") {
		t.Fatalf("help contract failed code=%d out=%q err=%q", code, out.String(), err.String())
	}
}

func TestOpenShellCommandSet(t *testing.T) {
	repo := t.TempDir()
	initGitRepoForTests(t, repo)
	ingestPath := filepath.Join(repo, "a.tgz")
	devicePath := filepath.Join(repo, "envs", "default", "state", "devices", "id1")
	panoPath := filepath.Join(repo, "envs", "default", "state", "panorama", "id2")
	_ = writeTestTGZ(ingestPath)
	_ = os.MkdirAll(devicePath, 0o755)
	_ = os.MkdirAll(panoPath, 0o755)
	_ = os.WriteFile(filepath.Join(devicePath, "latest.json"), []byte(`{"device":{"id":"id1","hostname":"fw1","model":"PA-440","sw_version":"11.0.0","mgmt_ip":"10.0.0.1"}}`), 0o644)
	_ = os.WriteFile(filepath.Join(panoPath, "latest.json"), []byte(`{"panorama_instance":{"id":"id2","hostname":"pano1","model":"M-200","version":"11.0.0","mgmt_ip":"10.0.0.2"}}`), 0o644)

	var directOut, directErr bytes.Buffer
	if code := Run([]string{"open", "--repo", repo, "--env", "default"}, &directOut, &directErr); code != 0 || directErr.Len() != 0 {
		t.Fatalf("command-first open failed code=%d out=%q err=%q", code, directOut.String(), directErr.String())
	}
	if directOut.String() != "netsec-sk(env:default)>\n" {
		t.Fatalf("unexpected command-first open output: %q", directOut.String())
	}

	in := strings.NewReader("help\nenv list\nenv create dev\ndevices\npanorama\nshow device id1\nshow panorama id2\ningest " + ingestPath + "\nquit\n")
	var out, err bytes.Buffer
	code := RunOpenSession(in, &out, &err, []string{"--repo", repo, "--env", "default"})
	if code != 0 {
		t.Fatalf("RunOpenSession code=%d err=%q", code, err.String())
	}
	txt := out.String()
	required := []string{"netsec-sk(env:default)>", "init: initialize repository", "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP", "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP", "Ingest complete:"}
	for _, r := range required {
		if !strings.Contains(txt, r) {
			t.Fatalf("open shell output missing %q in %q", r, txt)
		}
	}
}
