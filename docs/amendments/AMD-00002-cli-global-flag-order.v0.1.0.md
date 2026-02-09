---
doc_type: amendment
amendment_id: "AMD-00002"
slug: "cli-global-flag-order"
version: "0.1.0"
status: "Applied"
created_at: "2026-02-09"
authors:
  - "cmarks"
plan_file: "../plan.v0.1.0.md"
plan_version_before: "0.1.0"
plan_version_after: "0.1.0"
---

# AMD-00002 — CLI Global Flag Order Compatibility (Plan v0.1.0)

## Reason

Post-MVP usage testing found a spec/implementation mismatch:

- Spec command examples in Section 9.4 support command-first invocation with trailing global flags (for example `netsec-sk init --repo <path>`).
- Current implementation required global flags before the command, causing `E_USAGE` for command-first forms.

## Applied Plan Deltas

- Added `TASK-00013` to implement global-flag placement compatibility across all commands.
- Added `TASK-00014` to add regression coverage for both invocation forms in CLI and e2e tests.
- Updated lane/worktree tracking to include and complete `TASK-00013` and `TASK-00014`.

## Scope and Impact

- Scope: plan + implementation alignment patch within existing spec boundaries.
- Public interface impact: compatibility fix only, no new commands or flags.
- Backward compatibility: existing global-first invocation remains supported.

## Verification

- `go test ./internal/cli -run TestGlobalFlags`
- `go test ./internal/cli -run TestGlobalFlagPlacementCompatibility`
- `go test ./internal/cli -run TestCommandOutputContracts`
- `go test ./internal/cli -run TestOpenShellCommandSet`
- `go test ./e2e -run TestMVPAcceptanceChecklist`
