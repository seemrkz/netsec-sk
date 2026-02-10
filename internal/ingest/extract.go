package ingest

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsafeArchivePath = errors.New("unsafe archive entry path")
var ErrUnsupportedArchiveType = errors.New("unsupported archive type")

func ExtractArchive(archivePath string, extractRoot string) error {
	if !isSupportedArchive(archivePath) {
		return ErrUnsupportedArchiveType
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if hdr == nil {
			continue
		}

		targetPath, err := safeExtractTarget(extractRoot, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink, tar.TypeLink:
			return ErrUnsafeArchivePath
		default:
			// Non-file entries are ignored for MVP extraction behavior.
			continue
		}
	}
}

func safeExtractTarget(extractRoot string, archiveEntryName string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(archiveEntryName)))
	// Many tarballs include "." / "./" as the first directory entry. Treat that as a no-op
	// that resolves to the extraction root.
	if clean == "." {
		return extractRoot, nil
	}
	if clean == "" {
		return "", ErrUnsafeArchivePath
	}
	if filepath.IsAbs(clean) {
		return "", ErrUnsafeArchivePath
	}
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", ErrUnsafeArchivePath
	}

	target := filepath.Join(extractRoot, clean)
	rel, err := filepath.Rel(extractRoot, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrUnsafeArchivePath
	}
	return target, nil
}
