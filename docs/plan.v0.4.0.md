---
doc_type: plan
project: "NetSec StateKit (NS SK)"
plan_id: "PLAN-00001"
version: "0.4.0"
owners:
  - "cmarks"
last_updated: "2026-02-10"
change_control:
  amendment_required_for_substantive_change: true
  metadata_change_allowed_without_amendment: true
related:
  spec: "./spec.v0.6.0.md"
  amendments_dir: "./amendments/"
  changelog: "./changelog.md"
---

# NetSec StateKit — Follow-On Implementation Plan v0.4.0 (Single Lane)

This plan supersedes implementation scope of `./plan.v0.3.0.md` for the deltas introduced by `./spec.v0.6.0.md`.

## 0. Plan Guardrails (Mandatory)

- Constrained to `./spec.v0.6.0.md` and `./AGENTS.md`.
- Single sequential lane only; no parallel task execution.
- `1 task = 1 changelog entry` and `1 task = commit-proof capture`.
- Default budgets apply unless task explicitly states otherwise.
- Substantive plan changes require amendment (`AMD-00006`).

### 0.1 Plan Review Round Log (Append-Only)

- Round ID: `PR-00004`
- Date: `2026-02-10`
- Reviewers:
  - Reviewer P1: Scope Enforcer
  - Reviewer P2: Atomicity/Budget Auditor
- Outcome: `PASS`
- Blockers count: `0`
- Summary: one-lane sequential implementation plan for `spec.v0.6.0` history/topology/provenance deltas.
- Amendment link: `./amendments/AMD-00006-v0-6-0-follow-on-plan.v0.4.0.md`

## 1. Public Interface / Type Changes

1. Added `history state [--repo <path>] [--env <env_id>]` command contract.
2. Changed `topology` one-shot contract to Mermaid text output and added `--at-commit <hash>` historical mode.
3. Extended `commits.ndjson` rows with `changed_scope` and `changed_paths`.
4. Preserved export as one deterministic bundle command (`export`).

## 2. Worktree Lanes (Single Sequential Lane)

Lane Index:

- `LANE1: v0-6-0-follow-on`

Lane view:

| LANE1 |
|---|
| ~~TASK-00034~~ |
| ~~TASK-00035~~ |
| ~~TASK-00036~~ |
| ~~TASK-00037~~ |
| ~~TASK-00038~~ |

Execution rule: strict top-to-bottom sequencing.

## 3. Enforced Default Budgets

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0 unless explicitly allowed by task/spec

## 4. Task Registry

### TASK-00034 — Ledger Provenance Fields

- Objective: populate deterministic `changed_scope` and `changed_paths` in commit ledger rows.
- Spec refs: §9.4 (`history state`), §10.3, §11(25), §11(26), D-00013.
- Dependency: none.
- Status: Done.
- Scope delivered:
  - `commits.ndjson` schema extended at write path.
  - deterministic state-diff scope classifier (`device|feature|route|other`) added.
  - deterministic repo-relative lexical `changed_paths` added.
- Verification:
  - `go test ./internal/state -run TestChangedScopeClassifier` -> `ok   github.com/seemrkz/netsec-sk/internal/state`
  - `go test ./internal/ingest -run TestCommitLedgerIncludesChangedScopeAndPaths` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`
  - `go test ./internal/ingest -run TestChangedPathsLexicalRepoRelative` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`
- Commit proof:
  - `commit_hash: 1479084`
  - `commit_message: TASK-00034: add deterministic changed scope and paths to commit ledger`

### TASK-00035 — `history state` Command Contract

- Objective: expose deterministic state-change provenance rows via one-shot CLI.
- Spec refs: §2.3, §9.4 (`history state`), §10.3, §11(25), §11(26), D-00013.
- Dependency: `TASK-00034`.
- Status: Done.
- Scope delivered:
  - added `history` command with `state` subcommand.
  - fixed TSV header contract.
  - deterministic sort by `committed_at_utc`, then `git_commit`.
  - header-only success for missing/empty history.
  - error mapping to `E_IO` for parse/read failures.
- Verification:
  - `go test ./internal/cli -run TestHistoryStateCommandContract` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`
  - `go test ./internal/cli -run TestHistoryStateSortOrder` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`
  - `go test ./internal/cli -run TestHelpCommandContracts` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`
- Commit proof:
  - `commit_hash: 1a88646`
  - `commit_message: TASK-00035: implement history state command contract`

### TASK-00036 — `topology --at-commit` Mermaid Contract

- Objective: return Mermaid topology for current state and historical commit state without mutating worktree.
- Spec refs: §9.4 (`topology`), §11(27), §11(28), D-00014.
- Dependency: `TASK-00035`.
- Status: Done.
- Scope delivered:
  - switched `topology` stdout to Mermaid graph text.
  - added `--at-commit <hash>` argument handling.
  - historical reads via `git show <hash>:envs/<env>/exports/topology.mmd`.
  - usage/hash validation mapped to `E_USAGE`; unresolved/missing content mapped to `E_IO`.
- Verification:
  - `go test ./internal/cli -run TestTopologyCurrentMermaidOutput` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`
  - `go test ./internal/cli -run TestTopologyAtCommitOutput` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`
  - `go test ./internal/cli -run TestTopologyAtCommitValidationAndErrors` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`
- Commit proof:
  - `commit_hash: 9121759`
  - `commit_message: TASK-00036: implement topology mermaid output with at-commit`

### TASK-00037 — v0.6.0 Acceptance and Regression Coverage

- Objective: encode deterministic automated checks for v0.6.0 acceptance deltas.
- Spec refs: §11(23)-§11(29), §5.1-§5.2, §9.4.
- Dependency: `TASK-00036`.
- Status: Done.
- Scope delivered:
  - representative environment ID test coverage.
  - multi-TSF deterministic attempt/commit/no-commit coverage.
  - history + topology contract/e2e assertions (including historical non-mutation check).
  - ingest runtime updated to regenerate exports before commit so history commit hashes are topology-addressable.
- Verification:
  - `go test ./internal/cli -run TestEnvRepresentativeIDs` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`
  - `go test ./internal/ingest -run TestMultiTSFAttemptAndCommitOutcomes` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`
  - `go test ./internal/state -run TestChangedScopeIncludesRouteWhenRoutingChanges` -> `ok   github.com/seemrkz/netsec-sk/internal/state`
  - `go test ./internal/export -run TestCSVHeadersAndOrdering` -> `ok   github.com/seemrkz/netsec-sk/internal/export`
  - `go test ./...` -> `ok` across all packages
- Commit proof:
  - `commit_hash: 4191d3f`
  - `commit_message: TASK-00037: add v0.6.0 acceptance and regression coverage`

### TASK-00038 — Final Acceptance Closure and UX Test Packet

- Objective: close the lane with final acceptance proof and operator-facing manual test guidance.
- Spec refs: §0, §9.4, §11(23)-§11(29).
- Dependency: `TASK-00037`.
- Status: Done.
- Scope delivered:
  - final acceptance matrix executed.
  - operator checklist added: `./user-journey-test-v0.6.0.md`.
  - changelog evidence captured with command outcomes and manual run notes.
- Verification:
  - `go test ./e2e -run TestMVPAcceptanceChecklist` -> `ok   github.com/seemrkz/netsec-sk/e2e`
  - `go test ./...` -> `ok` across all packages
  - manual checklist executed once in temp repo -> `PASS`
- Commit proof:
  - `commit_hash: 8eea9b6`
  - `commit_message: TASK-00038: finalize v0.6.0 acceptance and user journey test script`

## 5. User Feel / Experience Validation

Manual operator test script is captured in:

- `./user-journey-test-v0.6.0.md`

The checklist validates:

1. environment naming and deterministic list ordering,
2. ingest + provenance readability (`history state`),
3. export bundle discoverability,
4. Mermaid current/historical topology workflow,
5. deterministic and actionable error UX.

## 6. Explicit Assumptions

1. Baseline source of truth is `./spec.v0.6.0.md`.
2. In-shell `history` and in-shell `topology` remain out of scope.
3. `topology` historical mode is commit-hash based only.
4. Export remains bundle-only (`export` has no format-selection flags).
