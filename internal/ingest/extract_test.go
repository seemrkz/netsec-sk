package ingest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractArchive_AllowsDotRootEntry(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "x.tgz")
	extractRoot := filepath.Join(root, "out")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Common tarball root entry.
	if err := tw.WriteHeader(&tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "./a.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("ok"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ExtractArchive(archivePath, extractRoot); err != nil {
		t.Fatalf("ExtractArchive() err=%v", err)
	}
	got, err := os.ReadFile(filepath.Join(extractRoot, "a.txt"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(got) != "ok" {
		t.Fatalf("extracted contents=%q, want %q", string(got), "ok")
	}
}
