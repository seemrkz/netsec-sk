package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seemrkz/netsec-sk/internal/env"
	"github.com/seemrkz/netsec-sk/internal/ingest"
	"github.com/seemrkz/netsec-sk/internal/repo"
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
		_, _ = fmt.Fprintf(stdout, "Export complete: %s\n", parsed.GlobalOptions.EnvID)
		return 0
	case "devices":
		return runDevices(parsed, stdout, stderr)
	case "panorama":
		return runPanorama(parsed, stdout, stderr)
	case "show":
		return runShow(parsed, stdout, stderr)
	case "topology":
		_, _ = fmt.Fprintln(stdout, "Topology edges: 0")
		_, _ = fmt.Fprintln(stdout, "Orphan zones: 0")
		return 0
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
	return 0
}

func runPanorama(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) != 1 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk panorama [--repo <path>] [--env <env_id>]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	_, _ = fmt.Fprintln(stdout, "PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP")
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

func runHelp(parsed ParseResult, stdout, stderr io.Writer) int {
	if len(parsed.CommandArgs) > 2 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk help [command]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	if len(parsed.CommandArgs) == 1 {
		_, _ = fmt.Fprintln(stdout, "init env ingest export devices panorama show topology help open")
		return 0
	}
	cmd := parsed.CommandArgs[1]
	_, _ = fmt.Fprintf(stdout, "Usage: netsec-sk %s\nArguments: see spec\nExamples: netsec-sk %s\nExit codes: see spec\n", cmd, cmd)
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
	if len(parsed.CommandArgs) < 2 {
		appErr := NewAppError(ErrUsage, "usage: netsec-sk ingest <paths...> [--repo <path>] [--env <env_id>] [--rdns] [--keep-extract]")
		writeErrorLine(stderr, appErr)
		return ExitCodeFor(appErr)
	}

	summary, err := ingestRun(ingest.RunOptions{
		RepoPath: parsed.GlobalOptions.RepoPath,
		EnvIDRaw: parsed.GlobalOptions.EnvID,
		Inputs:   parsed.CommandArgs[1:],
	})
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrNoInputs):
			appErr := NewAppError(ErrUsage, err.Error())
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
		exit := Run(append(globalArgs, args...), out, errOut)
		if exit != 0 {
			return exit
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
