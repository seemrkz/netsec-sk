package ingest

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/seemrkz/netsec-sk/internal/enrich"
	"github.com/seemrkz/netsec-sk/internal/env"
	"github.com/seemrkz/netsec-sk/internal/parse"
	"github.com/seemrkz/netsec-sk/internal/repo"
	"github.com/seemrkz/netsec-sk/internal/tsf"
)

const extractStaleAfter = 24 * time.Hour

var ErrNoInputs = errors.New("ingest requires at least one input path")
var checkRepoSafe = repo.CheckSafeWorkingTree
var acquireRunLock = AcquireLock
var releaseRunLock = ReleaseLock
var currentPID = os.Getpid
var currentProcessInspector LockInspector = ProcessInspector{}

type Summary struct {
	Attempted             int
	Committed             int
	SkippedDuplicateTSF   int
	SkippedStateUnchanged int
	ParseErrorPartial     int
	ParseErrorFatal       int
}

type RunOptions struct {
	RepoPath string
	EnvIDRaw string
	Inputs   []string
	Now      time.Time
}

func Run(options RunOptions) (Summary, error) {
	if len(options.Inputs) == 0 {
		return Summary{}, ErrNoInputs
	}

	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	if err := checkRepoSafe(options.RepoPath); err != nil {
		return Summary{}, err
	}

	if _, err := acquireRunLock(options.RepoPath, now, currentPID(), "ingest", currentProcessInspector); err != nil {
		return Summary{}, err
	}
	defer func() {
		_ = releaseRunLock(options.RepoPath)
	}()

	prep, err := Prepare(PrepareOptions{
		RepoPath: options.RepoPath,
		EnvIDRaw: options.EnvIDRaw,
		Inputs:   options.Inputs,
		Now:      now,
	})
	if err != nil {
		return Summary{}, err
	}

	summary := Summary{}
	for idx, input := range prep.OrderedInputs {
		summary.Attempted++

		if !isSupportedArchive(input) {
			summary.ParseErrorFatal++
			continue
		}

		extractDir, err := BeginTSFExtractDir(prep.RunExtractRoot, input, idx+1)
		if err != nil {
			summary.ParseErrorFatal++
			continue
		}

		extractErr := ExtractArchive(input, extractDir)
		if extractErr != nil {
			_ = FinishTSFExtractDir(extractDir, false)
			summary.ParseErrorFatal++
			continue
		}

		fileContents, extractedPaths, err := readExtractedFiles(extractDir)
		_ = FinishTSFExtractDir(extractDir, false)
		if err != nil {
			summary.ParseErrorFatal++
			continue
		}

		identity := tsf.DeriveIdentity(extractedPaths, os.ReadFile)
		out, err := parse.ParseSnapshot(parse.ParseContext{
			TSFID:            identity.TSFID,
			TSFOriginalName:  identity.TSFOriginalName,
			InputArchiveName: filepath.Base(input),
			IngestedAtUTC:    now.UTC().Format(time.RFC3339),
		}, fileContents)
		if err != nil || out.Result == "parse_error_fatal" {
			summary.ParseErrorFatal++
			continue
		}
		if out.Result == "parse_error_partial" {
			summary.ParseErrorPartial++
			continue
		}

		// State persistence and commit semantics are applied in follow-up tasks.
		summary.SkippedStateUnchanged++
	}
	return summary, nil
}

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
				paths = append(paths, cleanPath)
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}

		paths = append(paths, abs)
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

func readExtractedFiles(extractDir string) (map[string]string, []string, error) {
	files := make(map[string]string)
	paths := make([]string, 0)

	err := filepath.WalkDir(extractDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		clean := filepath.Clean(path)
		content, err := os.ReadFile(clean)
		if err != nil {
			return err
		}
		files[clean] = string(content)
		paths = append(paths, clean)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Strings(paths)
	return files, paths, nil
}

type IngestLogEntry struct {
	AttemptedAtUTC   string `json:"attempted_at_utc,omitempty"`
	RunID            string `json:"run_id,omitempty"`
	EnvID            string `json:"env_id"`
	InputArchivePath string `json:"input_archive_path,omitempty"`
	TSFID            string `json:"tsf_id,omitempty"`
	EntityType       string `json:"entity_type,omitempty"`
	EntityID         string `json:"entity_id,omitempty"`
	Result           string `json:"result,omitempty"`
	GitCommit        string `json:"git_commit,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

func ReadSeenTSFIDs(ingestLogPath string, envID string) (map[string]struct{}, error) {
	out := make(map[string]struct{})

	f, err := os.Open(ingestLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry IngestLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}

		if entry.EnvID != envID || entry.TSFID == "" {
			continue
		}
		out[entry.TSFID] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func IsDuplicateTSF(tsfID string, seen map[string]struct{}) bool {
	if tsfID == "unknown" || tsfID == "" {
		return false
	}
	_, ok := seen[tsfID]
	return ok
}

func AppendIngestAttempt(path string, entry IngestLogEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

func ApplyRDNS(enabled bool, isNewDevice bool, mgmtIP string, now time.Time, lookup enrich.LookupFunc) (enrich.ReverseDNS, bool) {
	return enrich.MaybeLookup(enrich.Options{
		Enabled:     enabled,
		IsNewDevice: isNewDevice,
		MgmtIP:      mgmtIP,
		Now:         now,
		Lookup:      lookup,
	})
}
