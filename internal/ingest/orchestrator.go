package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/seemrkz/netsec-sk/internal/env"
)

const extractStaleAfter = 24 * time.Hour

type PrepResult struct {
	EnvID          string
	RunID          string
	OrderedInputs  []string
	RunExtractRoot string
	Warnings       []string
}

type PrepareOptions struct {
	RepoPath string
	EnvIDRaw string
	Inputs   []string
	Now      time.Time
}

func Prepare(options PrepareOptions) (PrepResult, error) {
	svc := env.NewService(options.RepoPath)
	envID, _, err := svc.Create(options.EnvIDRaw)
	if err != nil {
		return PrepResult{}, err
	}

	ordered, err := ResolveInputs(options.Inputs)
	if err != nil {
		return PrepResult{}, err
	}

	extractRoot := filepath.Join(options.RepoPath, ".netsec-state", "extract")
	warnings, err := cleanupStaleExtractDirs(extractRoot, options.Now)
	if err != nil {
		return PrepResult{}, err
	}

	runID := fmt.Sprintf("run-%d", options.Now.UTC().UnixNano())
	runExtractRoot := filepath.Join(extractRoot, runID)
	if err := os.MkdirAll(runExtractRoot, 0o755); err != nil {
		return PrepResult{}, err
	}

	return PrepResult{
		EnvID:          envID,
		RunID:          runID,
		OrderedInputs:  ordered,
		RunExtractRoot: runExtractRoot,
		Warnings:       warnings,
	}, nil
}

func ResolveInputs(inputs []string) ([]string, error) {
	paths := make([]string, 0)
	for _, input := range inputs {
		abs, err := filepath.Abs(input)
		if err != nil {
			return nil, err
		}
		abs = filepath.Clean(abs)

		info, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			err = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.IsDir() {
					return nil
				}
				cleanPath := filepath.Clean(path)
				if isSupportedArchive(cleanPath) {
					paths = append(paths, cleanPath)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}

		if isSupportedArchive(abs) {
			paths = append(paths, abs)
		}
	}

	sort.Strings(paths)
	return paths, nil
}

func BeginTSFExtractDir(runExtractRoot string, archivePath string, index int) (string, error) {
	base := filepath.Base(archivePath)
	name := fmt.Sprintf("%03d_%s", index, sanitizePathPart(base))
	out := filepath.Join(runExtractRoot, name)
	if err := os.MkdirAll(out, 0o755); err != nil {
		return "", err
	}
	return out, nil
}

func FinishTSFExtractDir(extractDir string, keepExtract bool) error {
	if keepExtract {
		return nil
	}
	return os.RemoveAll(extractDir)
}

func cleanupStaleExtractDirs(extractRoot string, now time.Time) ([]string, error) {
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(extractRoot)
	if err != nil {
		return nil, err
	}

	warnings := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(extractRoot, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("extract_cleanup_stat_failed:%s", entry.Name()))
			continue
		}

		if now.Sub(info.ModTime()) <= extractStaleAfter {
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			warnings = append(warnings, fmt.Sprintf("extract_cleanup_remove_failed:%s", entry.Name()))
		}
	}

	return warnings, nil
}

func isSupportedArchive(path string) bool {
	return strings.HasSuffix(path, ".tgz") || strings.HasSuffix(path, ".tar.gz")
}

func sanitizePathPart(in string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		" ", "_",
		":", "_",
	)
	return replacer.Replace(in)
}
