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
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--repo", repo, "devices"}, "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP\n"},
		{[]string{"--repo", repo, "panorama"}, "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP\n"},
		{[]string{"--repo", repo, "topology"}, "Topology edges: 0\nOrphan zones: 0\n"},
		{[]string{"--repo", repo, "export"}, "Export complete: default\n"},
		{[]string{"--repo", repo, "ingest", "a.tgz"}, "Ingest complete: attempted=0 committed=0 skipped_duplicate_tsf=0 skipped_state_unchanged=0 parse_error_partial=0 parse_error_fatal=0\n"},
	}
	for _, tc := range cases {
		var out, err bytes.Buffer
		if code := Run(tc.args, &out, &err); code != 0 || err.Len() != 0 || out.String() != tc.want {
			t.Fatalf("args=%v code=%d out=%q err=%q", tc.args, code, out.String(), err.String())
		}
	}

	var out, err bytes.Buffer
	if code := Run([]string{"help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "open") {
		t.Fatalf("help contract failed code=%d out=%q err=%q", code, out.String(), err.String())
	}
}

func TestOpenShellCommandSet(t *testing.T) {
	repo := t.TempDir()
	devicePath := filepath.Join(repo, "envs", "default", "state", "devices", "id1")
	panoPath := filepath.Join(repo, "envs", "default", "state", "panorama", "id2")
	_ = os.MkdirAll(devicePath, 0o755)
	_ = os.MkdirAll(panoPath, 0o755)
	_ = os.WriteFile(filepath.Join(devicePath, "latest.json"), []byte(`{"kind":"device"}`), 0o644)
	_ = os.WriteFile(filepath.Join(panoPath, "latest.json"), []byte(`{"kind":"panorama"}`), 0o644)

	in := strings.NewReader("help\nenv list\nenv create dev\ndevices\npanorama\nshow device id1\nshow panorama id2\ntopology\nexport\ningest a.tgz\nquit\n")
	var out, err bytes.Buffer
	code := RunOpenSession(in, &out, &err, []string{"--repo", repo, "--env", "default"})
	if code != 0 {
		t.Fatalf("RunOpenSession code=%d err=%q", code, err.String())
	}
	txt := out.String()
	required := []string{"netsec-sk(env:default)>", "init env ingest export devices panorama show topology help open", "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP", "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP", "Topology edges: 0", "Export complete: default", "Ingest complete:"}
	for _, r := range required {
		if !strings.Contains(txt, r) {
			t.Fatalf("open shell output missing %q in %q", r, txt)
		}
	}
}
