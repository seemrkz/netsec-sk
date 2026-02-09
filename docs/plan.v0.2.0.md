---
doc_type: plan
project: "NetSec StateKit (NS SK)"
plan_id: "PLAN-00001"
version: "0.2.0"
owners:
  - "cmarks"
last_updated: "2026-02-09"
change_control:
  amendment_required_for_substantive_change: true
  metadata_change_allowed_without_amendment: true
related:
  spec: "./spec.v0.5.1.md"
  amendments_dir: "./amendments/"
  changelog: "./changelog.md"
---

# NetSec StateKit — Implementation Plan v0.2.0

This plan is constrained to `./spec.v0.5.1.md` and `./AGENTS.md`.

## 0. Plan Guardrails (Mandatory)

- Plan introduces no scope outside `spec.v0.5.1.md`.
- Any substantive plan change requires amendment.
- Each task is atomic and uses default budgets unless explicitly allowed by spec.
- Tasks must not proceed if new ambiguities are found; affected tasks must be marked `Blocked` and planning must stop.
- A task is done only after its verification steps are executed and recorded in changelog.

### 0.1 Plan Review Round Log (Append-Only)

- Round ID: `PR-00002`
- Date: `2026-02-09`
- Reviewers:
  - Reviewer P1: Scope Enforcer
  - Reviewer P2: Atomicity/Budget Auditor
- Outcome: `PASS`
- Blockers count: `0`
- Summary of changes applied: reset execution lane to prototype-focused open-shell ingest flow aligned to `spec.v0.5.1.md`.
- Amendment link: `./amendments/AMD-00003-open-useful-prototype.v0.5.1.md`

## 1. Enforced Default Budgets

Unless spec overrides:

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0 (unless explicitly specified in spec/task)

## 2. Phase Overview

| Phase | Title | Focus | Exit Criteria |
|------:|------|-------|--------------|
| I | Ingest Runtime | real ingest command execution and extraction safety | mixed input ingest is deterministic and non-placeholder |
| II | Parse + State | minimum parse mapping and state persistence semantics | snapshots + ledgers + dedupe/unchanged behavior proven |
| III | Commit + Query | commit guarantees and query output from persisted state | commit pipeline + device/panorama/show contracts proven |
| IV | Open Shell + E2E | interactive shell behavior and end-to-end acceptance | open-shell ingest roundtrip acceptance passes |

## 2.1 Worktree (Lanes for Execution Order)

Lane Index:

- `LANE1: ingest-runtime`
- `LANE2: parse-state`
- `LANE3: shell-e2e`

Lane view:

```text
LANE1            | LANE2        | LANE3
-----------------------------------------------
TASK-00015       | TASK-00017   | TASK-00021
TASK-00016       | TASK-00018   | TASK-00022
TASK-00019       | TASK-00020   |
```

## 3. Task Registry

Rule: `1 task = 1 changelog entry`.

### 3.1 Task Schema Contract

Every task section includes:

- Objective
- Spec refs
- Preconditions
- Dependencies
- Scope In / Scope Out
- Acceptance criteria
- Verification steps with expected results
- Budget declaration
- Status + blockers
- Changelog requirement
- Plan update requirement

## 4. Tasks

### TASK-00015: Wire `ingest` command to real runtime orchestration

- Objective: Replace placeholder `ingest` execution path with real orchestration entrypoint and deterministic summary accounting.
- Spec refs: SPEC §2.3, §3.4, §6.1, §9.4 (`ingest`)
- Status: Pending
- Blocked by: none
- Depends on: none
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Existing CLI root and global flags are stable.

#### Scope

- In:
  - Route `ingest` command to orchestration flow instead of fixed placeholder output.
  - Compute summary counters from actual attempt results.
  - Include unsupported extension files in `attempted` and `parse_error_fatal` totals.
- Out:
  - Deep archive extraction implementation details.

#### Target Surface Area (Expected)

- `internal/cli/root.go`
- `internal/ingest/orchestrator.go`
- `internal/cli/root_test.go`
- Public interfaces affected: `ingest` runtime behavior and summary semantics.

#### Acceptance Criteria (Task-Level)

- [ ] `ingest` no longer emits static zero-summary output.
- [ ] summary counters reflect real attempt outcomes.
- [ ] unsupported extensions count as fatal attempts.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestIngestCommandUsesRuntime`
  - `go test ./internal/ingest -run TestMixedInputAttemptAccounting`
- Expected results:
  - tests pass and assert deterministic counter behavior.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00016: Implement safe `.tgz`/`.tar.gz` extraction pipeline

- Objective: Add archive extraction for ingest attempts with path traversal protections.
- Spec refs: SPEC §6.3, §6.9, §9.4 (`ingest`)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00015
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Ingest orchestration entrypoint exists.

#### Scope

- In:
  - Extract supported archives into `.netsec-state/extract/<run_id>/<tsf_dir>/...`.
  - Reject unsafe archive entries that would escape extraction root.
  - Corrupt/unreadable archives classified as `parse_error_fatal`.
- Out:
  - Parser field extraction logic.

#### Target Surface Area (Expected)

- `internal/ingest/extract.go`
- `internal/ingest/orchestrator.go`
- `internal/ingest/ingest_test.go`
- Public interfaces affected: ingest archive extraction behavior.

#### Acceptance Criteria (Task-Level)

- [ ] supported archives extract under run directory.
- [ ] path traversal attempts are blocked.
- [ ] corrupt archive path yields fatal classification.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/ingest -run TestArchiveExtractionSupportedFormats`
  - `go test ./internal/ingest -run TestArchivePathTraversalRejected`
  - `go test ./internal/ingest -run TestCorruptArchiveFatal`
- Expected results:
  - tests pass for safety and fatal-classification branches.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00017: Implement prototype minimum parse mapping from extracted content

- Objective: Parse extracted TSF text into minimum required firewall/panorama identity fields with deterministic partial/fatal classification.
- Spec refs: SPEC §6.6, §6.10, §7.1, §7.2, §7.3
- Status: Pending
- Blocked by: none
- Depends on: TASK-00016
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Extraction pipeline yields accessible extracted text files.

#### Scope

- In:
  - Build parse input from extracted content.
  - Enforce minimum identity requirements for firewall/panorama non-fatal results.
  - Emit `parse_error_partial` for missing non-identity fields with snapshot write.
- Out:
  - Full policy/routing deep extraction expansion.

#### Target Surface Area (Expected)

- `internal/parse/classifier.go`
- `internal/parse/snapshots.go`
- `internal/parse/parse_test.go`
- Public interfaces affected: parse taxonomy and snapshot population.

#### Acceptance Criteria (Task-Level)

- [ ] firewall/panorama identity minimums gate fatal vs partial behavior.
- [ ] partial parse still writes required envelope + identity fields.
- [ ] entity type and identity are deterministically derived from extracted content.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/parse -run TestPrototypeMinimumFields`
  - `go test ./internal/parse -run TestPartialStillWritesSnapshot`
- Expected results:
  - tests pass for fatal and partial boundaries.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00018: Persist snapshots and enforce dedupe/unchanged semantics in real ingest flow

- Objective: Integrate state hash comparison and snapshot persistence in command runtime.
- Spec refs: SPEC §6.4, §6.5, §6.7, §7.4, §10.4
- Status: Pending
- Blocked by: none
- Depends on: TASK-00015, TASK-00017
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Parsing output from TASK-00017 is available.

#### Scope

- In:
  - Read seen TSF IDs from ingest ledger by environment.
  - Apply duplicate and unchanged-state skip logic in live ingest pipeline.
  - Persist `latest.json`, snapshot files, and ingest ledger entries.
- Out:
  - Git commit creation details.

#### Target Surface Area (Expected)

- `internal/ingest/orchestrator.go`
- `internal/state/compare.go`
- `internal/ingest/ingest_test.go`
- Public interfaces affected: ingest result semantics.

#### Acceptance Criteria (Task-Level)

- [ ] duplicate TSFs are skipped per environment.
- [ ] unchanged state skips commit path.
- [ ] ingest ledger has one row per attempt.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/ingest -run TestDuplicateAndUnchangedSemantics`
  - `go test ./internal/state -run TestUnchangedStateSkip`
- Expected results:
  - tests pass and assert no-commit outcomes where required.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00019: Enforce commit pipeline guarantees in command runtime

- Objective: Ensure committed ingest results create exactly one git commit with strict allowlist/subject semantics.
- Spec refs: SPEC §3.3, §10.1, §10.2, §10.3, §10.5
- Status: Pending
- Blocked by: none
- Depends on: TASK-00018
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Snapshot persistence path is integrated.

#### Scope

- In:
  - Stage only allowlisted files for commit.
  - Create commit subject per deterministic format.
  - Append `state/commits.ndjson` only on committed outcomes.
- Out:
  - advanced git history optimizations.

#### Target Surface Area (Expected)

- `internal/commit/committer.go`
- `internal/state/commits_ledger.go`
- `internal/ingest/orchestrator.go`
- `internal/commit/committer_test.go`
- Public interfaces affected: git history and commit ledger contracts.

#### Acceptance Criteria (Task-Level)

- [ ] each `committed` result maps to exactly one commit.
- [ ] commit subject and file allowlist match spec.
- [ ] non-committed results create no commit.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/commit -run TestAtomicCommitAllowlist`
  - `go test ./internal/commit -run TestCommitMessageFormat`
  - `go test ./internal/ingest -run TestCommittedResultCreatesOneCommit`
- Expected results:
  - tests pass for one-commit-per-result guarantee.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00020: Back query commands by persisted state data

- Objective: Make `devices`, `panorama`, and `show` commands read real persisted state and output deterministic views.
- Spec refs: SPEC §2.3, §9.4 (`devices`, `panorama`, `show`)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00018
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- State files are persisted by ingest runtime.

#### Scope

- In:
  - Enumerate state directories and render sorted TSV rows.
  - Keep `show` reading and pretty printing latest snapshots.
  - Ensure output consistency between one-shot and shell invocation.
- Out:
  - query-time derived topology/export generation.

#### Target Surface Area (Expected)

- `internal/cli/root.go`
- `internal/cli/task12_test.go`
- Public interfaces affected: query command output contents.

#### Acceptance Criteria (Task-Level)

- [ ] `devices` and `panorama` output real rows sorted by entity ID.
- [ ] `show` outputs pretty JSON from persisted `latest.json`.
- [ ] outputs are identical across one-shot vs in-shell invocation.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestCommandOutputContracts`
  - `go test ./internal/cli -run TestQueryCommandsFromPersistedState`
- Expected results:
  - tests pass for deterministic query outputs.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00021: Implement deterministic `open` shell behavior and command parity

- Objective: Enforce `open` shell behavior contract (prompt cadence, no-op empty line, continue-on-error, exit semantics).
- Spec refs: SPEC §2.3, §9.4 (`open`)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00015, TASK-00020
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Core command handlers are wired to real runtime behavior.

#### Scope

- In:
  - Prompt before each read.
  - Continue session after command errors.
  - Exit on `exit`/`quit`/EOF with code 0.
  - Enforce supported in-shell command set from spec.
- Out:
  - shell completion/history enhancements.

#### Target Surface Area (Expected)

- `internal/cli/root.go`
- `internal/cli/task12_test.go`
- Public interfaces affected: interactive shell behavior.

#### Acceptance Criteria (Task-Level)

- [ ] invalid in-shell command emits standard error line and shell continues.
- [ ] empty line does not change state and re-prompts.
- [ ] supported in-shell commands match spec command set.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestOpenShellCommandSet`
  - `go test ./internal/cli -run TestOpenShellContinuesAfterError`
- Expected results:
  - tests pass for prompt/continuation/exit behavior.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00022: Add prototype e2e fixtures and acceptance checklist coverage

- Objective: Validate end-to-end prototype behavior using real archive fixtures and shell roundtrip scenarios.
- Spec refs: SPEC §11, §2.3, §3.4, §6.1, §9.4, §10.5
- Status: Pending
- Blocked by: none
- Depends on: TASK-00019, TASK-00021
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Runtime pipeline and shell behavior are integrated.

#### Scope

- In:
  - Add fixture archives for firewall/panorama minimum parse cases.
  - Add mixed input test (`valid.tgz` + `invalid.txt`) for unsupported-extension accounting.
  - Add shell ingest roundtrip test (`open` -> `ingest` -> `show`).
  - Assert commit/no-commit behavior across changed/duplicate/unchanged scenarios.
- Out:
  - performance benchmarking.

#### Target Surface Area (Expected)

- `e2e/mvp_test.go`
- `e2e/fixtures/` (new fixture files)
- `internal/cli/task12_test.go` (if needed for shell parity assertions)
- Public interfaces affected: none (verification only).

#### Acceptance Criteria (Task-Level)

- [ ] e2e tests cover all spec v0.5.1 prototype acceptance scenarios.
- [ ] fixture-driven ingest creates persisted state for query commands.
- [ ] unsupported extension fatal attempt accounting is asserted in e2e.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./e2e -run TestPrototypeOpenIngestRoundtrip`
  - `go test ./e2e -run TestMixedInputUnsupportedExtensionAccounting`
  - `go test ./e2e -run TestCommitDuplicateUnchangedBehavior`
- Expected results:
  - all e2e acceptance tests pass and map directly to SPEC §11 criteria.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0
