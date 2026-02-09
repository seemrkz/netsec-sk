package cli

import (
	"errors"
	"testing"
)

func TestGlobalFlags(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got, err := ParseGlobalFlags([]string{"init"})
		if err != nil {
			t.Fatalf("ParseGlobalFlags() unexpected error: %v", err)
		}

		if got.GlobalOptions.RepoPath != "./default" {
			t.Fatalf("repo default mismatch: got %q", got.GlobalOptions.RepoPath)
		}
		if got.GlobalOptions.EnvID != "default" {
			t.Fatalf("env default mismatch: got %q", got.GlobalOptions.EnvID)
		}
		if len(got.CommandArgs) != 1 || got.CommandArgs[0] != "init" {
			t.Fatalf("command args mismatch: got %#v", got.CommandArgs)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		got, err := ParseGlobalFlags([]string{"--repo", "/tmp/repo", "--env", "dev", "ingest"})
		if err != nil {
			t.Fatalf("ParseGlobalFlags() unexpected error: %v", err)
		}

		if got.GlobalOptions.RepoPath != "/tmp/repo" {
			t.Fatalf("repo override mismatch: got %q", got.GlobalOptions.RepoPath)
		}
		if got.GlobalOptions.EnvID != "dev" {
			t.Fatalf("env override mismatch: got %q", got.GlobalOptions.EnvID)
		}
		if len(got.CommandArgs) != 1 || got.CommandArgs[0] != "ingest" {
			t.Fatalf("command args mismatch: got %#v", got.CommandArgs)
		}
	})
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantLine string
	}{
		{
			name:     "usage",
			err:      NewAppError(ErrUsage, "bad args"),
			wantCode: 2,
			wantLine: "ERROR E_USAGE bad args",
		},
		{
			name:     "git missing",
			err:      NewAppError(ErrGitMissing, "git missing"),
			wantCode: 3,
			wantLine: "ERROR E_GIT_MISSING git missing",
		},
		{
			name:     "repo unsafe",
			err:      NewAppError(ErrRepoUnsafe, "dirty"),
			wantCode: 4,
			wantLine: "ERROR E_REPO_UNSAFE dirty",
		},
		{
			name:     "lock held",
			err:      NewAppError(ErrLockHeld, "active lock"),
			wantCode: 5,
			wantLine: "ERROR E_LOCK_HELD active lock",
		},
		{
			name:     "parse fatal",
			err:      NewAppError(ErrParseFatal, "bad tsf"),
			wantCode: 6,
			wantLine: "ERROR E_PARSE_FATAL bad tsf",
		},
		{
			name:     "io",
			err:      NewAppError(ErrIO, "write failed"),
			wantCode: 6,
			wantLine: "ERROR E_IO write failed",
		},
		{
			name:     "parse partial",
			err:      NewAppError(ErrParsePart, "missing optional fields"),
			wantCode: 7,
			wantLine: "ERROR E_PARSE_PARTIAL missing optional fields",
		},
		{
			name:     "internal",
			err:      NewAppError(ErrInternal, "panic recovered"),
			wantCode: 9,
			wantLine: "ERROR E_INTERNAL panic recovered",
		},
		{
			name:     "non app error maps to internal",
			err:      errors.New("unexpected"),
			wantCode: 9,
			wantLine: "ERROR E_INTERNAL unexpected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCodeFor(tt.err); got != tt.wantCode {
				t.Fatalf("ExitCodeFor() = %d, want %d", got, tt.wantCode)
			}

			if got := FormatErrorLine(tt.err); got != tt.wantLine {
				t.Fatalf("FormatErrorLine() = %q, want %q", got, tt.wantLine)
			}
		})
	}
}
