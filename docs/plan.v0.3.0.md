---
doc_type: plan
project: "NetSec StateKit (NS SK)"
plan_id: "PLAN-00001"
version: "0.3.0"
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

# NetSec StateKit — Implementation Plan v0.3.0

This plan is constrained to `./spec.v0.5.1.md` and `./AGENTS.md`.

## 0. Plan Guardrails (Mandatory)

- Plan introduces no scope outside `spec.v0.5.1.md`.
- Any substantive plan change requires amendment.
- Each task is atomic and uses default budgets unless explicitly allowed by spec.
- Tasks must not proceed if new ambiguities are found; affected tasks must be marked `Blocked` and planning must stop.
- A task is done only after its verification steps are executed and recorded in changelog.
- `1 task = 1 changelog entry` and `1 task = commit-proof capture` (hash + message).

### 0.1 Plan Review Round Log (Append-Only)

- Round ID: `PR-00003`
- Date: `2026-02-09`
- Reviewers:
  - Reviewer P1: Scope Enforcer
  - Reviewer P2: Atomicity/Budget Auditor
- Outcome: `PASS`
- Blockers count: `0`
- Summary of changes applied: expanded plan to fully implement all non-deferred requirements in `spec.v0.5.1.md`, including export/query/topology one-shot contracts, commit guarantees, and open-shell parity semantics.
- Amendment link: `./amendments/AMD-00004-full-build-plan.v0.3.0.md`

## 1. Enforced Default Budgets

Unless spec overrides:

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0 (unless explicitly specified in spec/task)

## 2. Phase Overview

| Phase | Title | Focus | Exit Criteria |
|------:|------|-------|--------------|
| I | Ingest Runtime | real ingest command path and deterministic attempt accounting | non-placeholder ingest with lock + ordering + extraction safety |
| II | Parse + State + Commit | snapshot correctness, dedupe/unchanged semantics, commit guarantees | one-commit-per-committed-result with valid ledgers |
| III | Exports + Queries | deterministic export files and query command contracts | export/topology/query outputs match spec contracts |
| IV | Shell + Acceptance | `open` shell behavior and full acceptance checklist | end-to-end fixture tests satisfy SPEC §11 |

## 2.1 Worktree (Lanes for Execution Order)

Lane Index:

- `LANE1: ingest-runtime`
- `LANE2: parse-state-commit`
- `LANE3: export-query`
- `LANE4: shell-acceptance`

Lane view:

| LANE1 | LANE2 | LANE3 | LANE4 |
|---|---|---|---|
| ~~TASK-00023~~ | TASK-00026 | TASK-00030 | TASK-00032 |
| ~~TASK-00024~~ | TASK-00027 | TASK-00031 | TASK-00033 |
| TASK-00025 | TASK-00028 |  |  |
|  | TASK-00029 |  |  |

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
- Commit requirement + proof capture fields
- Status + blockers
- Changelog requirement
- Plan update requirement

## 4. Tasks

### TASK-00023: Replace placeholder ingest command with real runtime entrypoint

- Objective: wire `netsec-sk ingest` to the runtime orchestration path and remove static placeholder output.
- Spec refs: SPEC §2.3, §6.1, §9.2, §9.4 (`ingest`)
- Status: Done
- Blocked by: none
- Depends on: none
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- CLI root command and global flags exist.

#### Scope

- In:
  - Route ingest command to orchestration service.
  - Return summary from real attempt results.
  - Preserve ingest exit-code precedence contract.
- Out:
  - archive extraction internals.

#### Target Surface Area (Expected)

- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `internal/ingest/orchestrator.go`
- Public interfaces affected: ingest runtime behavior.

#### Acceptance Criteria (Task-Level)

- [x] `ingest` no longer emits fixed zero-summary output.
- [x] summary values derive from real ingest attempt outcomes.
- [x] exit codes follow SPEC §9.2 ingest precedence.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestIngestCommandUsesRuntime`
  - `go test ./internal/cli -run TestIngestExitCodePrecedence`
- Expected results:
  - tests pass and assert real runtime usage.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00023: wire ingest command to runtime`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: TASK-00023: wire ingest command to runtime`

### TASK-00024: Implement safe archive extraction and mixed-input accounting

- Objective: extract supported archives safely and classify unsupported/corrupt inputs deterministically.
- Spec refs: SPEC §3.4, §6.1, §6.3, §6.9
- Status: Done
- Blocked by: none
- Depends on: TASK-00023
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Ingest orchestration entrypoint is active.

#### Scope

- In:
  - Extract `.tgz`/`.tar.gz` into run-scoped extract directory.
  - Enforce archive path normalization and traversal rejection.
  - Count unsupported extensions as attempts and classify as `parse_error_fatal` with `unsupported_extension` note.
  - Cleanup per SPEC §6.3 behavior.
- Out:
  - TSF identity and parse schema mapping.

#### Target Surface Area (Expected)

- `internal/ingest/extract.go`
- `internal/ingest/orchestrator.go`
- `internal/ingest/ingest_test.go`
- Public interfaces affected: ingest attempt accounting + extraction safety behavior.

#### Acceptance Criteria (Task-Level)

- [x] supported archives extract only within assigned extract root.
- [x] traversal/symlink escape attempts are rejected.
- [x] mixed inputs include unsupported files in `attempted` and `parse_error_fatal` counts.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/ingest -run TestArchiveExtractionSupportedFormats`
  - `go test ./internal/ingest -run TestArchivePathTraversalRejected`
  - `go test ./internal/ingest -run TestUnsupportedExtensionAccounting`
- Expected results:
  - tests pass for safety and deterministic accounting.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00024: add safe archive extraction and mixed-input accounting`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: TASK-00024: add safe archive extraction and mixed-input accounting`

### TASK-00025: Integrate repo unsafe-state gate and ingest locking lifecycle

- Objective: enforce working-tree safety checks and lock acquire/release semantics in live ingest command flow.
- Spec refs: SPEC §3.2, §6.2, §9.2, §9.4 (`ingest`)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00023
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Ingest command runtime path is active.

#### Scope

- In:
  - Detect unsafe git working tree states and fail with `E_REPO_UNSAFE`.
  - Acquire lock before ingest run and release lock on all terminal paths.
  - Apply stale lock removal policy and warning behavior.
- Out:
  - parse/persistence pipeline logic.

#### Target Surface Area (Expected)

- `internal/repo/git_check.go`
- `internal/ingest/lock.go`
- `internal/ingest/orchestrator.go`
- `internal/cli/root.go`
- `internal/ingest/ingest_test.go`
- Public interfaces affected: ingest error semantics for unsafe repo and lock handling.

#### Acceptance Criteria (Task-Level)

- [ ] unsafe tracked/staged states fail ingest deterministically.
- [ ] active lock returns `E_LOCK_HELD`.
- [ ] stale lock is removed and ingest proceeds.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/ingest -run TestRepoUnsafeStateBlocksIngest`
  - `go test ./internal/ingest -run TestLockStaleRules`
  - `go test ./internal/cli -run TestIngestErrorCodeMapping`
- Expected results:
  - tests pass for all gate branches.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00025: enforce repo safety and lock lifecycle in ingest`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00026: Implement TSF identity, classification, and minimum parse mapping in runtime

- Objective: derive identity from extracted TSF metadata and apply deterministic parse taxonomy for firewall/panorama snapshots.
- Spec refs: SPEC §6.4, §6.5, §6.6, §6.10, §7.1, §7.2, §7.3
- Status: Pending
- Blocked by: none
- Depends on: TASK-00024
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Extracted file corpus is available.

#### Scope

- In:
  - Select identity metadata files per priority and tie-break rules.
  - Determine entity type and minimum required identity fields.
  - Emit `parse_error_partial` vs `parse_error_fatal` per boundaries.
  - Ensure snapshots retain required envelope + required identity fields.
- Out:
  - commit creation and export generation.

#### Target Surface Area (Expected)

- `internal/tsf/identity.go`
- `internal/parse/classifier.go`
- `internal/parse/snapshots.go`
- `internal/parse/parse_test.go`
- Public interfaces affected: snapshot population and parse result taxonomy.

#### Acceptance Criteria (Task-Level)

- [ ] identity derivation follows tmp/cli candidate rules exactly.
- [ ] missing minimum identity fields are fatal.
- [ ] partial parse still writes required snapshot scaffolding.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/tsf -run TestIdentityDerivation`
  - `go test ./internal/parse -run TestPrototypeMinimumFields`
  - `go test ./internal/parse -run TestParseErrorClassification`
- Expected results:
  - tests pass for identity and taxonomy boundaries.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00026: integrate identity and parse taxonomy in runtime`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00027: Enforce canonical state hashing and snapshot persistence semantics

- Objective: apply hash canonicalization, unchanged-state detection, and deterministic snapshot/latest file writes.
- Spec refs: SPEC §4.1, §6.7, §7.4
- Status: Pending
- Blocked by: none
- Depends on: TASK-00026
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Parse output is available for both entity types.

#### Scope

- In:
  - Compute `state_sha256` excluding volatile fields.
  - Compare with existing `latest.json`.
  - Write `latest.json` and snapshot file only for changed states.
- Out:
  - commit and export generation.

#### Target Surface Area (Expected)

- `internal/state/hash.go`
- `internal/state/compare.go`
- `internal/ingest/orchestrator.go`
- `internal/state/state_test.go`
- Public interfaces affected: state dedupe and snapshot write behavior.

#### Acceptance Criteria (Task-Level)

- [ ] unchanged logical state is skipped with `skipped_state_unchanged`.
- [ ] changed state writes deterministic snapshot/latest files.
- [ ] hash behavior is stable across ordering differences.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/state -run TestHashCanonicalization`
  - `go test ./internal/state -run TestUnchangedStateSkip`
  - `go test ./internal/ingest -run TestSnapshotPersistenceOnChange`
- Expected results:
  - tests pass for canonical hash and write semantics.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00027: enforce state hashing and snapshot persistence`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00028: Integrate RDNS enrichment for newly discovered firewalls

- Objective: run RDNS lookups only for newly discovered firewall devices with deterministic timeout/retry policy.
- Spec refs: SPEC §6.8, §7.2, §11(21)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00027
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- New-vs-existing device detection exists in ingest runtime.

#### Scope

- In:
  - Trigger RDNS only when `--rdns` and device is newly discovered.
  - Apply 1-second timeout and one retry.
  - Persist reverse-DNS status fields deterministically.
- Out:
  - resolver caching across runs.

#### Target Surface Area (Expected)

- `internal/enrich/rdns.go`
- `internal/ingest/orchestrator.go`
- `internal/enrich/rdns_test.go`
- Public interfaces affected: `device.dns.reverse` field behavior.

#### Acceptance Criteria (Task-Level)

- [ ] existing devices do not trigger RDNS.
- [ ] lookup status mapping matches spec contract.
- [ ] timeout/retry policy is deterministic.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/enrich -run TestRDNSOnlyForNewDevices`
  - `go test ./internal/enrich -run TestRDNSTimeoutRetry`
- Expected results:
  - tests pass for new-device gating and status mapping.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00028: integrate deterministic rdns enrichment`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00029: Implement commit pipeline, allowlist staging, and ledgers

- Objective: enforce one atomic commit per committed TSF result and write both ingest and commit ledgers.
- Spec refs: SPEC §3.3, §10.1, §10.2, §10.3, §10.4, §10.5
- Status: Pending
- Blocked by: none
- Depends on: TASK-00025, TASK-00027, TASK-00028
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Runtime persistence path emits change/no-change outcomes.

#### Scope

- In:
  - Stage only allowlisted files for commit.
  - Commit with deterministic subject format.
  - Append ingest ledger for every attempt and commit ledger for committed results.
  - Guarantee exactly one commit per `committed` ingest row.
- Out:
  - history rewrite/squash behaviors.

#### Target Surface Area (Expected)

- `internal/commit/committer.go`
- `internal/state/commits_ledger.go`
- `internal/ingest/orchestrator.go`
- `internal/ingest/ingest_test.go`
- `internal/commit/committer_test.go`
- Public interfaces affected: git history and ledger contracts.

#### Acceptance Criteria (Task-Level)

- [ ] committed results each map to exactly one commit.
- [ ] duplicate and unchanged results create no commit.
- [ ] commit subject and staged files match spec.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/commit -run TestAtomicCommitAllowlist`
  - `go test ./internal/commit -run TestCommitMessageFormat`
  - `go test ./internal/ingest -run TestCommittedResultCreatesOneCommit`
  - `go test ./internal/ingest -run TestIngestLedgerAllAttempts`
- Expected results:
  - tests pass for commit guarantee and ledger semantics.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00029: enforce atomic commit pipeline and ledgers`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00030: Implement full export pipeline and `export` command contract

- Objective: generate all required export files deterministically from persisted state/topology and wire `export` command output.
- Spec refs: SPEC §8.1, §8.2, §8.3, §8.4, §8.5, §8.6, §9.4 (`export`)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00027, TASK-00029
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- State snapshots and commit semantics are stable.

#### Scope

- In:
  - Generate `environment.json`, `inventory.csv`, `nodes.csv`, `edges.csv`, `topology.mmd`, `agent_context.md`.
  - Enforce exact headers/sorting/schema constraints.
  - Ensure `export` command prints success contract line.
- Out:
  - additional export formats.

#### Target Surface Area (Expected)

- `internal/export/writers.go`
- `internal/topology/infer.go`
- `internal/cli/root.go`
- `internal/export/writers_test.go`
- `internal/cli/task12_test.go`
- Public interfaces affected: export and topology artifact contracts.

#### Acceptance Criteria (Task-Level)

- [ ] all six export files are generated deterministically.
- [ ] CSV headers/order match spec exactly.
- [ ] export command output matches contract.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/export -run TestEnvironmentJSONSchema`
  - `go test ./internal/export -run TestCSVHeadersAndOrdering`
  - `go test ./internal/export -run TestAgentContextTemplate`
  - `go test ./internal/cli -run TestExportCommandContract`
- Expected results:
  - tests pass for artifact contract fidelity.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00030: implement deterministic export pipeline and command`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00031: Implement query command backends for devices/panorama/show/topology

- Objective: make query commands read persisted state/exports and satisfy output contracts.
- Spec refs: SPEC §8.4, §8.5, §9.4 (`devices`, `panorama`, `show`, `topology`)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00030
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Export and state artifacts are produced deterministically.

#### Scope

- In:
  - `devices` and `panorama` read entity latest snapshots and print sorted TSV rows.
  - `show` reads and pretty-prints latest JSON snapshots.
  - `topology` derives counts from deterministic edge/orphan outputs.
- Out:
  - in-shell `export` and `topology` command requirements (deferred by spec).

#### Target Surface Area (Expected)

- `internal/cli/root.go`
- `internal/cli/task12_test.go`
- `internal/cli/root_test.go`
- Public interfaces affected: query command output behavior.

#### Acceptance Criteria (Task-Level)

- [ ] devices/panorama tables include real rows sorted by entity ID.
- [ ] show returns pretty JSON from state latest files.
- [ ] topology count lines reflect generated topology data.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestCommandOutputContracts`
  - `go test ./internal/cli -run TestQueryCommandsFromPersistedState`
  - `go test ./internal/topology -run TestInferSharedSubnetEdges`
- Expected results:
  - tests pass for deterministic query outputs.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00031: back query commands with persisted state`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00032: Complete `open` shell semantics and help contract parity

- Objective: enforce shell session rules and help output contracts, including one-shot/in-shell parity.
- Spec refs: SPEC §2.3, §9.3, §9.4 (`help`, `open`)
- Status: Pending
- Blocked by: none
- Depends on: TASK-00023, TASK-00031
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- One-shot command behaviors are stable.

#### Scope

- In:
  - Ensure prompt-per-read, empty-line no-op, and continue-on-error session behavior.
  - Ensure `exit`/`quit`/EOF exits with code 0.
  - Ensure `help` and `help <command>` include required usage/args/examples/exit-code notes.
  - Ensure supported in-shell commands exactly match spec-required set.
- Out:
  - shell completion/history UI improvements.

#### Target Surface Area (Expected)

- `internal/cli/root.go`
- `internal/cli/task12_test.go`
- `internal/cli/root_test.go`
- Public interfaces affected: shell and help output behavior.

#### Acceptance Criteria (Task-Level)

- [ ] shell remains active after non-fatal command error.
- [ ] prompt and exit behavior match spec.
- [ ] help outputs satisfy command contract requirements.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestOpenShellCommandSet`
  - `go test ./internal/cli -run TestOpenShellContinuesAfterError`
  - `go test ./internal/cli -run TestHelpCommandContracts`
- Expected results:
  - tests pass for shell/help parity and behavior.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00032: enforce open shell and help contract parity`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`

### TASK-00033: Build end-to-end acceptance matrix and release verification coverage

- Objective: validate full spec acceptance criteria via e2e fixture tests and release artifact verification.
- Spec refs: SPEC §0, §11
- Status: Pending
- Blocked by: none
- Depends on: TASK-00029, TASK-00030, TASK-00031, TASK-00032
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Runtime, commit, export, query, and shell features are integrated.

#### Scope

- In:
  - Add fixture-driven e2e tests for ingest roundtrip, mixed input accounting, duplicate/unchanged behavior, commit guarantees, and shell resilience.
  - Validate acceptance criteria coverage matrix maps directly to SPEC §11 items.
  - Re-run release build/checksum script and capture results.
- Out:
  - benchmark/perf tuning.

#### Target Surface Area (Expected)

- `e2e/mvp_test.go`
- `e2e/fixtures/`
- `scripts/release/build_and_checksum.sh`
- `docs/changelog.md`
- Public interfaces affected: none (verification hardening only).

#### Acceptance Criteria (Task-Level)

- [ ] e2e coverage demonstrates full non-deferred spec conformance.
- [ ] acceptance checklist evidence is captured in changelog.
- [ ] release script outputs required artifacts and checksums.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./e2e -run TestMVPAcceptanceChecklist`
  - `go test ./...`
  - `./scripts/release/build_and_checksum.sh`
- Expected results:
  - full suite passes and release artifacts are generated.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

#### Commit Requirement (Required)

- Required commit message format: `TASK-00033: add e2e acceptance and release verification`
- Commit proof capture (record in changelog on completion):
  - `commit_hash: <TBD>`
  - `commit_message: <TBD>`
