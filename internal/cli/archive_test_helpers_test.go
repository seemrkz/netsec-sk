package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
)

func writeTestTGZ(path string) error {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := []byte("fixture")
	hdr := &tar.Header{
		Name: "tmp/cli/sample.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}

	parseable := []byte("firewall\nserial: T1\nhostname: fw1\nmgmt_ip: 10.0.0.1\nmodel: PA-440\nsw_version: 11.0.0")
	parseHdr := &tar.Header{
		Name: "tmp/cli/PA-440_ts.tgz.txt",
		Mode: 0o644,
		Size: int64(len(parseable)),
	}
	if err := tw.WriteHeader(parseHdr); err != nil {
		return err
	}
	if _, err := tw.Write(parseable); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
