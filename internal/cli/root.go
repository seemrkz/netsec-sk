package cli

import (
	"flag"
	"fmt"
	"io"
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

	err = NewAppError(ErrInternal, fmt.Sprintf("command not yet implemented: %s", parsed.CommandArgs[0]))
	writeErrorLine(stderr, err)
	return ExitCodeFor(err)
}

func writeErrorLine(w io.Writer, err error) {
	if err == nil {
		return
	}

	_, _ = fmt.Fprintln(w, FormatErrorLine(err))
}
