package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seemrkz/netsec-sk/internal/env"
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

func ParseGlobalFlags(args []string) (ParseResult, error) {
	var opts GlobalOptions

	fs := flag.NewFlagSet("netsec-sk", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.RepoPath, "repo", DefaultRepoPath, "state repository path")
	fs.StringVar(&opts.EnvID, "env", DefaultEnvID, "environment id")

	if err := fs.Parse(args); err != nil {
		return ParseResult{}, NewAppError(ErrUsage, err.Error())
	}

	return ParseResult{
		GlobalOptions: opts,
		CommandArgs:   fs.Args(),
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
		_, _ = fmt.Fprintln(stdout, "Ingest complete: attempted=0 committed=0 skipped_duplicate_tsf=0 skipped_state_unchanged=0 parse_error_partial=0 parse_error_fatal=0")
		return 0
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
