package cli

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/seemrkz/netsec-sk/internal/env"
	exportpkg "github.com/seemrkz/netsec-sk/internal/export"
	"github.com/seemrkz/netsec-sk/internal/ingest"
	"github.com/seemrkz/netsec-sk/internal/repo"
	"github.com/seemrkz/netsec-sk/internal/state"
)

const (
	DefaultRepoPath = "./default"
	DefaultEnvID    = "default"
)

type GlobalOptions struct {
	RepoPath string
	EnvID    string
}

type ParseResult struct {
	GlobalOptions GlobalOptions
	CommandArgs   []string
}

var ingestRun = ingest.Run
var exportRun = exportpkg.Run

func ParseGlobalFlags(args []string) (ParseResult, error) {
	opts := GlobalOptions{
		RepoPath: DefaultRepoPath,
		EnvID:    DefaultEnvID,
	}
	commandArgs := make([]string, 0, len(args))
	commandSeen := false
	stopParsingGlobals := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if stopParsingGlobals {
			commandArgs = append(commandArgs, arg)
			continue
		}
		if arg == "--" {
			stopParsingGlobals = true
			continue
		}

		if name, value, ok := inlineGlobalFlag(arg); ok {
			setGlobalFlag(&opts, name, value)
			continue
		}
		if arg == "--repo" || arg == "--env" {
			if i+1 >= len(args) {
				return ParseResult{}, NewAppError(ErrUsage, fmt.Sprintf("flag needs an argument: %s", arg))
			}
			i++
			setGlobalFlag(&opts, arg, args[i])
			continue
		}

		if strings.HasPrefix(arg, "--") && !commandSeen {
			return ParseResult{}, NewAppError(ErrUsage, fmt.Sprintf("flag provided but not defined: %s", strings.TrimPrefix(arg, "-")))
		}

		commandSeen = true
		commandArgs = append(commandArgs, arg)
	}

	return ParseResult{
		GlobalOptions: opts,
		CommandArgs:   commandArgs,
	}, nil
}

func Run(args []string, stdout, stderr io.Writer) int {
	parsed, err := ParseGlobalFlags(args)
	if err != nil {
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}

	if len(parsed.CommandArgs) == 0 {
		err = NewAppError(ErrUsage, "missing command")
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}

	switch parsed.CommandArgs[0] {
	case "init":
		return runInit(parsed, stdout, stderr)
	case "env":
		return runEnv(parsed, stdout, stderr)
	case "ingest":
		return runIngest(parsed, stdout, stderr)
	case "export":
		return runExport(parsed, stdout, stderr)
	case "devices":
		return runDevices(parsed, stdout, stderr)
	case "panorama":
		return runPanorama(parsed, stdout, stderr)
	case "show":
		return runShow(parsed, stdout, stderr)
	case "history":
		return runHistory(parsed, stdout, stderr)
	case "topology":
		return runTopology(parsed, stdout, stderr)
	case "help":
		return runHelp(parsed, stdout, stderr)
	case "open":
		return runOpen(parsed, stdout, stderr)
	default:
		err = NewAppError(ErrInternal, fmt.Sprintf("command not yet implemented: %s", parsed.CommandArgs[0]))
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}
}

func writeErrorLine(w io.Writer, err error) {
	if err == nil {
		return
	}

	_, _ = fmt.Fprintln(w, FormatErrorLine(err))
}

func runInit(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 1 {
		err := NewAppError(ErrUsage, "init does not accept positional arguments")
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}

	absRepoPath, err := repo.Init(parsed.GlobalOptions.RepoPath)
	if err != nil {
		appErr := mapInitError(err)
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	_, _ = fmt.Fprintf(stdout, "Initialized repository: %s\n", absRepoPath)
	return 0
}

func mapInitError(err error) error {
	if errors.Is(err, repo.ErrGitMissing) {
		return NewAppError(ErrGitMissing, "git executable is not available on PATH")
	}

	return NewAppError(ErrIO, err.Error())
}

func runEnv(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) < 2 {
		err := NewAppError(ErrUsage, "env requires a subcommand: list|create")
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}

	svc := env.NewService(parsed.GlobalOptions.RepoPath)
	switch parsed.CommandArgs[1] {
	case "list":
		return runEnvList(parsed.CommandArgs, svc, stdout, stderr)
	case "create":
		return runEnvCreate(parsed.CommandArgs, svc, stdout, stderr)
	default:
		err := NewAppError(ErrUsage, fmt.Sprintf("unknown env subcommand: %s", parsed.CommandArgs[1]))
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}
}

func runEnvList(args []string, svc env.Service, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		err := NewAppError(ErrUsage, "usage: netsec-sk env list [--repo <path>]")
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}

	envs, err := svc.List()
	if err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	if len(envs) > 0 {
		_, _ = fmt.Fprintln(stdout, strings.Join(envs, "\n"))
	}
	return 0
}

func runEnvCreate(args []string, svc env.Service, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		err := NewAppError(ErrUsage, "usage: netsec-sk env create <env_id> [--repo <path>]")
		writeErrorLine(stderr, err)
		return ExitCodeFor(err)
	}

	envID, created, err := svc.Create(args[2])
	if err != nil {
		if errors.Is(err, env.ErrInvalidEnvID) {
			appErr := NewAppError(ErrUsage, fmt.Sprintf("invalid env_id: %s", env.NormalizeEnvID(args[2])))
			writeErrorLine(stderr, appErr)
			return ExitCodeFor(appErr)
		}

		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	if created {
		_, _ = fmt.Fprintf(stdout, "Environment created: %s\n", envID)
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "Environment already exists: %s\n", envID)
	return 0
}

func runDevices(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 1 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk devices [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	_, _ = fmt.Fprintln(stdout, "DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP")
	rows, err := listDeviceRows(parsed.GlobalOptions.RepoPath, parsed.GlobalOptions.EnvID)
	if err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	for _, r := range rows {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Hostname, r.Model, r.Version, r.MgmtIP)
	}
	return 0
}

func runPanorama(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 1 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk panorama [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	_, _ = fmt.Fprintln(stdout, "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP")
	rows, err := listPanoramaRows(parsed.GlobalOptions.RepoPath, parsed.GlobalOptions.EnvID)
	if err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	for _, r := range rows {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Hostname, r.Model, r.Version, r.MgmtIP)
	}
	return 0
}

func runShow(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 3 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk show <device|panorama> <id> [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	kind, id := parsed.CommandArgs[1], parsed.CommandArgs[2]
	var path string
	switch kind {
	case "device":
		path = filepath.Join(parsed.GlobalOptions.RepoPath, "envs", parsed.GlobalOptions.EnvID, "state", "devices", id, "latest.json")
	case "panorama":
		path = filepath.Join(parsed.GlobalOptions.RepoPath, "envs", parsed.GlobalOptions.EnvID, "state", "panorama", id, "latest.json")
	default:
		appErr := NewAppError(ErrUsage, "show expects device|panorama")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	_, _ = fmt.Fprintln(stdout, string(pretty))
	return 0
}

func runHistory(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) < 2 {
		appErr := NewAppError(ErrUsage, "history requires a subcommand: state")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	switch parsed.CommandArgs[1] {
	case "state":
		return runHistoryState(parsed, stdout, stderr)
	default:
		appErr := NewAppError(ErrUsage, fmt.Sprintf("unknown history subcommand: %s", parsed.CommandArgs[1]))
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
}

func runHistoryState(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 2 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk history state [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	rows, err := readHistoryStateRows(parsed.GlobalOptions.RepoPath, parsed.GlobalOptions.EnvID)
	if err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	_, _ = fmt.Fprintln(stdout, "COMMITTED_AT_UTC\tGIT_COMMIT\tTSF_ID\tTSF_ORIGINAL_NAME\tCHANGED_SCOPE")
	for _, row := range rows {
		_, _ = fmt.Fprintf(
			stdout,
			"%s\t%s\t%s\t%s\t%s\n",
			row.CommittedAtUTC,
			row.GitCommit,
			row.TSFID,
			row.TSFOriginal,
			row.ChangedScope,
		)
	}
	return 0
}

type historyStateRow struct {
	CommittedAtUTC string
	GitCommit      string
	TSFID          string
	TSFOriginal    string
	ChangedScope   string
}

func readHistoryStateRows(repoPath, envID string) ([]historyStateRow, error) {
	path := filepath.Join(repoPath, "envs", envID, "state", "commits.ndjson")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	rows := make([]historyStateRow, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry state.CommitLedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		rows = append(rows, historyStateRow{
			CommittedAtUTC: entry.CommittedAtUTC,
			GitCommit:      entry.GitCommit,
			TSFID:          entry.TSFID,
			TSFOriginal:    entry.TSFOriginal,
			ChangedScope:   entry.ChangedScope,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CommittedAtUTC == rows[j].CommittedAtUTC {
			return rows[i].GitCommit < rows[j].GitCommit
		}
		return rows[i].CommittedAtUTC < rows[j].CommittedAtUTC
	})
	return rows, nil
}

func runHelp(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) > 2 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk help [command]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	if len(parsed.CommandArgs) == 1 {
		_, _ = fmt.Fprintln(stdout, "init: initialize repository")
		_, _ = fmt.Fprintln(stdout, "env: list/create environments")
		_, _ = fmt.Fprintln(stdout, "ingest: ingest TSF archives into state")
		_, _ = fmt.Fprintln(stdout, "export: generate deterministic export artifacts")
		_, _ = fmt.Fprintln(stdout, "devices: list persisted firewall inventory")
		_, _ = fmt.Fprintln(stdout, "panorama: list persisted panorama inventory")
		_, _ = fmt.Fprintln(stdout, "show: pretty-print latest snapshot by id")
		_, _ = fmt.Fprintln(stdout, "history: print deterministic state-change history")
		_, _ = fmt.Fprintln(stdout, "topology: print topology edge/orphan counts")
		_, _ = fmt.Fprintln(stdout, "help: show command usage details")
		_, _ = fmt.Fprintln(stdout, "open: interactive shell")
		return 0
	}
	cmd := parsed.CommandArgs[1]
	usage, argsLine, example := helpDetails(cmd)
	_, _ = fmt.Fprintf(stdout, "Usage: %s\nArguments: %s\nExamples: %s\nExit codes: see spec\n", usage, argsLine, example)
	return 0
}

func runOpen(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 1 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk open [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	_, _ = fmt.Fprintf(stdout, "netsec-sk(env:%s)>\n", parsed.GlobalOptions.EnvID)
	return 0
}

func runIngest(parsed ParseResult, stdout, stderr io.Writer) int {
	inputs, enableRDNS, keepExtract, parseErr := parseIngestArgs(parsed.CommandArgs[1:])
	if parseErr != nil {
		appErr := NewAppError(ErrUsage, parseErr.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	if len(inputs) == 0 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk ingest <paths...> [--repo <path>] [--env <env_id>] [--rdns] [--keep-extract]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	summary, err := ingestRun(ingest.RunOptions{
		RepoPath:    parsed.GlobalOptions.RepoPath,
		EnvIDRaw:    parsed.GlobalOptions.EnvID,
		Inputs:      inputs,
		EnableRDNS:  enableRDNS,
		KeepExtract: keepExtract,
	})
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrNoInputs):
			appErr := NewAppError(ErrUsage, err.Error())
			writeErrorLine(stderr, appErr)
			return ExitCodeFor(appErr)
		case errors.Is(err, repo.ErrRepoUnsafe):
			appErr := NewAppError(ErrRepoUnsafe, err.Error())
			writeErrorLine(stderr, appErr)
			return ExitCodeFor(appErr)
		case errors.Is(err, ingest.ErrLockHeld):
			appErr := NewAppError(ErrLockHeld, err.Error())
			writeErrorLine(stderr, appErr)
			return ExitCodeFor(appErr)
		default:
			appErr := NewAppError(ErrIO, err.Error())
			writeErrorLine(stderr, appErr)
			return ExitCodeFor(appErr)
		}
	}

	_, _ = fmt.Fprintf(
		stdout,
		"Ingest complete: attempted=%d committed=%d skipped_duplicate_tsf=%d skipped_state_unchanged=%d parse_error_partial=%d parse_error_fatal=%d\n",
		summary.Attempted,
		summary.Committed,
		summary.SkippedDuplicateTSF,
		summary.SkippedStateUnchanged,
		summary.ParseErrorPartial,
		summary.ParseErrorFatal,
	)

	if summary.ParseErrorFatal > 0 {
		return ExitCodeFor(NewAppError(ErrParseFatal, "ingest completed with fatal parse errors"))
	}
	if summary.ParseErrorPartial > 0 {
		return ExitCodeFor(NewAppError(ErrParsePart, "ingest completed with partial parse warnings"))
	}
	return 0
}

func runExport(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 1 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk export [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	if err := exportRun(exportpkg.RunOptions{
		RepoPath: parsed.GlobalOptions.RepoPath,
		EnvID:    parsed.GlobalOptions.EnvID,
		Now:      time.Now().UTC(),
	}); err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	_, _ = fmt.Fprintf(stdout, "Export complete: %s\n", parsed.GlobalOptions.EnvID)
	return 0
}

func runTopology(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 1 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk topology [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	edges, orphans, err := topologyCounts(parsed.GlobalOptions.RepoPath, parsed.GlobalOptions.EnvID)
	if err != nil {
		appErr := NewAppError(ErrIO, err.Error())
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}
	_, _ = fmt.Fprintf(stdout, "Topology edges: %d\n", edges)
	_, _ = fmt.Fprintf(stdout, "Orphan zones: %d\n", orphans)
	return 0
}

func parseIngestArgs(args []string) ([]string, bool, bool, error) {
	paths := make([]string, 0, len(args))
	enableRDNS := false
	keepExtract := false

	for _, arg := range args {
		switch arg {
		case "--rdns":
			enableRDNS = true
		case "--keep-extract":
			keepExtract = true
		default:
			if strings.HasPrefix(arg, "--") {
				return nil, false, false, fmt.Errorf("unknown ingest option: %s", arg)
			}
			paths = append(paths, arg)
		}
	}
	return paths, enableRDNS, keepExtract, nil
}

func RunOpenSession(in io.Reader, out io.Writer, errOut io.Writer, globalArgs []string) int {
	parsed, err := ParseGlobalFlags(append(globalArgs, "open"))
	if err != nil {
		writeErrorLine(errOut, err)
		return ExitCodeFor(err)
	}

	scanner := bufio.NewScanner(in)
	for {
		_, _ = fmt.Fprintf(out, "netsec-sk(env:%s)>", parsed.GlobalOptions.EnvID)
		if !scanner.Scan() {
			return 0
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return 0
		}
		args := strings.Fields(line)
		if !isOpenSupported(args) {
			writeErrorLine(errOut, NewAppError(ErrUsage, "open supports: help, env list|create, ingest, devices, panorama, show device|panorama, exit, quit"))
			continue
		}
		exit := Run(append(globalArgs, args...), out, errOut)
		if exit != 0 {
			// Open shell must continue after non-fatal command errors.
			continue
		}
	}
}

func SortedLines(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}

func inlineGlobalFlag(arg string) (name string, value string, ok bool) {
	if strings.HasPrefix(arg, "--repo=") {
		return "--repo", strings.TrimPrefix(arg, "--repo="), true
	}
	if strings.HasPrefix(arg, "--env=") {
		return "--env", strings.TrimPrefix(arg, "--env="), true
	}
	return "", "", false
}

func setGlobalFlag(opts *GlobalOptions, name string, value string) {
	switch name {
	case "--repo":
		opts.RepoPath = value
	case "--env":
		opts.EnvID = value
	}
}

func helpDetails(cmd string) (usage string, arguments string, example string) {
	switch cmd {
	case "init":
		return "netsec-sk init [--repo <path>]", "none", "netsec-sk init --repo ./default"
	case "env":
		return "netsec-sk env <list|create> [args]", "list | create <env_id>", "netsec-sk env create prod"
	case "ingest":
		return "netsec-sk ingest <paths...> [--repo <path>] [--env <env_id>] [--rdns] [--keep-extract]", "<paths...> plus optional flags", "netsec-sk ingest ./fixtures --env prod --rdns"
	case "export":
		return "netsec-sk export [--repo <path>] [--env <env_id>]", "no positional args", "netsec-sk export --env prod"
	case "devices":
		return "netsec-sk devices [--repo <path>] [--env <env_id>]", "no positional args", "netsec-sk devices --env prod"
	case "panorama":
		return "netsec-sk panorama [--repo <path>] [--env <env_id>]", "no positional args", "netsec-sk panorama --env prod"
	case "show":
		return "netsec-sk show <device|panorama> <id> [--repo <path>] [--env <env_id>]", "<device|panorama> <id>", "netsec-sk show device SER123 --env prod"
	case "history":
		return "netsec-sk history state [--repo <path>] [--env <env_id>]", "state", "netsec-sk history state --env prod"
	case "topology":
		return "netsec-sk topology [--repo <path>] [--env <env_id>]", "no positional args", "netsec-sk topology --env prod"
	case "open":
		return "netsec-sk open [--repo <path>] [--env <env_id>]", "no positional args", "netsec-sk open --env prod"
	case "help":
		return "netsec-sk help [command]", "[command] optional", "netsec-sk help ingest"
	default:
		return "netsec-sk help [command]", "[command] optional", "netsec-sk help"
	}
}

func isOpenSupported(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "help", "devices", "panorama", "ingest", "exit", "quit":
		return true
	case "env":
		return len(args) >= 2 && (args[1] == "list" || args[1] == "create")
	case "show":
		return len(args) >= 3 && (args[1] == "device" || args[1] == "panorama")
	default:
		return false
	}
}

type deviceRow struct {
	ID       string
	Hostname string
	Model    string
	Version  string
	MgmtIP   string
}

func listDeviceRows(repoPath, envID string) ([]deviceRow, error) {
	root := filepath.Join(repoPath, "envs", envID, "state", "devices")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	rows := make([]deviceRow, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		doc, err := readLatestDoc(filepath.Join(root, id, "latest.json"))
		if err != nil {
			return nil, err
		}
		device, _ := doc["device"].(map[string]any)
		rowID := strField(device["id"])
		if rowID == "" {
			rowID = id
		}
		rows = append(rows, deviceRow{
			ID:       rowID,
			Hostname: strField(device["hostname"]),
			Model:    strField(device["model"]),
			Version:  strField(device["sw_version"]),
			MgmtIP:   strField(device["mgmt_ip"]),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

func listPanoramaRows(repoPath, envID string) ([]deviceRow, error) {
	root := filepath.Join(repoPath, "envs", envID, "state", "panorama")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	rows := make([]deviceRow, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		doc, err := readLatestDoc(filepath.Join(root, id, "latest.json"))
		if err != nil {
			return nil, err
		}
		inst, _ := doc["panorama_instance"].(map[string]any)
		rowID := strField(inst["id"])
		if rowID == "" {
			rowID = id
		}
		rows = append(rows, deviceRow{
			ID:       rowID,
			Hostname: strField(inst["hostname"]),
			Model:    strField(inst["model"]),
			Version:  strField(inst["version"]),
			MgmtIP:   strField(inst["mgmt_ip"]),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

func readLatestDoc(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func topologyCounts(repoPath, envID string) (int, int, error) {
	edgesPath := filepath.Join(repoPath, "envs", envID, "exports", "edges.csv")
	nodesPath := filepath.Join(repoPath, "envs", envID, "exports", "nodes.csv")

	edgeCount, connectedZones, err := readEdges(edgesPath)
	if err != nil {
		return 0, 0, err
	}
	allZones, err := readZoneNodes(nodesPath)
	if err != nil {
		return 0, 0, err
	}

	orphanCount := 0
	for zone := range allZones {
		if _, ok := connectedZones[zone]; !ok {
			orphanCount++
		}
	}
	return edgeCount, orphanCount, nil
}

func readEdges(path string) (int, map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, map[string]struct{}{}, nil
		}
		return 0, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return 0, nil, err
	}
	if len(rows) == 0 {
		return 0, map[string]struct{}{}, nil
	}
	connected := map[string]struct{}{}
	for _, row := range rows[1:] {
		if len(row) < 4 {
			continue
		}
		if strings.HasPrefix(row[2], "zone_") {
			connected[row[2]] = struct{}{}
		}
		if strings.HasPrefix(row[3], "zone_") {
			connected[row[3]] = struct{}{}
		}
	}
	return len(rows) - 1, connected, nil
}

func readZoneNodes(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	out := map[string]struct{}{}
	for _, row := range rows[1:] {
		if len(row) < 2 {
			continue
		}
		if row[1] == "zone" {
			out[row[0]] = struct{}{}
		}
	}
	return out, nil
}

func strField(v any) string {
	s, _ := v.(string)
	return s
}
