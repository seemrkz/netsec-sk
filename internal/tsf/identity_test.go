package tsf

import (
	"fmt"
	"testing"
)

func TestIdentityDerivation(t *testing.T) {
	readFile := func(path string) ([]byte, error) {
		files := map[string]string{
			"run/tmp/cli/no-serial.txt":                     "hostname: fw-a",
			"run/tmp/cli/PA-440_ts.tgz.txt":                 "device serial: S12345",
			"run/tmp/cli/prefix_PA-440_ts.tgz_extra.txt":    "serial number: S777",
			"run/tmp/cli/another_PA-220_ts.tar.gz_meta.txt": "serial: S220",
		}
		v, ok := files[path]
		if !ok {
			return nil, fmt.Errorf("missing file: %s", path)
		}
		return []byte(v), nil
	}

	t.Run("unknown when metadata missing", func(t *testing.T) {
		got := DeriveIdentity([]string{"run/other/path/file.txt"}, readFile)
		if got.TSFID != "unknown" {
			t.Fatalf("TSFID = %q, want unknown", got.TSFID)
		}
	})

	t.Run("choose serial-bearing candidate and strip txt for tgz txt suffix", func(t *testing.T) {
		got := DeriveIdentity([]string{
			"run/tmp/cli/no-serial.txt",
			"run/tmp/cli/PA-440_ts.tgz.txt",
		}, readFile)
		if got.Serial != "S12345" {
			t.Fatalf("Serial = %q, want S12345", got.Serial)
		}
		if got.TSFOriginalName != "PA-440_ts.tgz" {
			t.Fatalf("TSFOriginalName = %q, want PA-440_ts.tgz", got.TSFOriginalName)
		}
		if got.TSFID != "S12345|PA-440_ts.tgz" {
			t.Fatalf("TSFID = %q", got.TSFID)
		}
	})

	t.Run("missing serial keeps empty prefix with delimiter", func(t *testing.T) {
		got := DeriveIdentity([]string{"run/tmp/cli/no-serial.txt"}, readFile)
		if got.TSFID != "|no-serial.txt" {
			t.Fatalf("TSFID = %q, want |no-serial.txt", got.TSFID)
		}
	})

	t.Run("extract shortest ts archive token from filename fallback", func(t *testing.T) {
		got := DeriveIdentity([]string{"run/tmp/cli/prefix_PA-440_ts.tgz_extra.txt"}, readFile)
		if got.TSFOriginalName != "PA-440_ts.tgz" {
			t.Fatalf("TSFOriginalName = %q, want PA-440_ts.tgz", got.TSFOriginalName)
		}
	})
}
