# Changelog

## TASK-00001

- Date: 2026-02-09
- Type: Added
- Summary: Implemented CLI root entrypoint, global flags, and centralized error/exit framework.
- Files:
  - `go.mod`
  - `cmd/netsec-sk/main.go`
  - `internal/cli/root.go`
  - `internal/cli/errors.go`
  - `internal/cli/root_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/cli -run TestGlobalFlags` -> `ok   github.com/seemrkz/netsec-sk/internal/cli (cached)`
  - `go test ./internal/cli -run TestExitCodeMapping` -> `ok   github.com/seemrkz/netsec-sk/internal/cli (cached)`

## TASK-00002

- Date: 2026-02-09
- Type: Added
- Summary: Implemented `init` repository bootstrap with Git prerequisite checks and deterministic base layout creation.
- Files:
  - `internal/repo/git_check.go`
  - `internal/repo/layout.go`
  - `internal/repo/init.go`
  - `internal/repo/init_test.go`
  - `internal/cli/root.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/repo -run TestInitCreatesBaseLayout` -> `ok   github.com/seemrkz/netsec-sk/internal/repo`
  - `go test ./internal/repo -run TestInitFailsWithoutGit` -> `ok   github.com/seemrkz/netsec-sk/internal/repo`
