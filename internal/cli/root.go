package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
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
