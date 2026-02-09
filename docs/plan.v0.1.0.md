---
doc_type: plan
project: "NetSec StateKit (NS SK)"
plan_id: "PLAN-00001"
version: "0.1.0"
owners:
  - "cmarks"
last_updated: "2026-02-08"
change_control:
  amendment_required_for_substantive_change: true
  metadata_change_allowed_without_amendment: true
related:
  spec: "./spec.v0.5.0.md"
  amendments_dir: "./amendments/"
  changelog: "./changelog.md"
---

# NetSec StateKit — Implementation Plan v0.1.0

This plan is constrained to `./spec.v0.5.0.md` and `./AGENTS.md`.

## 0. Plan Guardrails (Mandatory)

- Plan introduces no scope outside `spec.v0.5.0.md`.
- Any substantive plan change requires amendment.
- Each task is atomic and uses default budgets unless explicitly allowed by spec.
- Tasks must not proceed if new ambiguities are found; affected tasks must be marked `Blocked` and planning must stop.
- A task is done only after its verification steps are executed and recorded in changelog.

### 0.1 Plan Review Round Log (Append-Only)

- Round ID: `PR-00001`
- Date: `2026-02-08`
- Reviewers:
  - Reviewer P1: Scope Enforcer
  - Reviewer P2: Atomicity/Budget Auditor
- Outcome: `PASS`
- Blockers count: `0`
- Summary of changes applied: validated task-to-spec mapping, added explicit dependency chain, tightened verification expectations.
- Amendment link: `none`

## 1. Enforced Default Budgets

Unless spec overrides:

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0 (unless explicitly specified in spec/task)

## 2. Phase Overview

| Phase | Title | Focus | Exit Criteria |
|------:|------|-------|--------------|
| I | CLI + Repo Foundation | command shell, repo safety, env model | safe init + env flows verified |
| II | Ingest Core | ordering, lock, extraction, identity, parsing | deterministic ingest outcomes verified |
| III | State + Exports | hash, topology, exports, commit ledger | deterministic outputs + atomic commit verified |
| IV | UX + Release | query commands, interactive shell, release artifacts | MVP acceptance checklist passes |

## 2.1 Worktree (Lanes for Execution Order)

Lane Index:

- `LANE1: cli-repo-core`
- `LANE2: ingest-parse`
- `LANE3: state-export-ux`

Lane view:

```text
LANE1          | LANE2         | LANE3
cli-repo-core  | ingest-parse  | state-export-ux
-----------------------------------------------
~~TASK-00001~~ | ~~TASK-00004~~ | ~~TASK-00007~~
~~TASK-00002~~ | ~~TASK-00005~~ | ~~TASK-00008~~
~~TASK-00003~~ | ~~TASK-00006~~ | TASK-00009
               |               | TASK-00010
               |               | TASK-00011
TASK-00012     |               |
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

### TASK-00001: Implement CLI root, global flags, and error/exit framework

- Objective: Create CLI entrypoint with deterministic global flag parsing and standardized error/exit behavior.
- Spec refs: SPEC §9.1, §9.2, §9.3
- Status: Done
- Blocked by: none
- Depends on: none
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Go module initialized.

#### Scope

- In:
  - Root command with `--repo` and `--env` defaults.
  - Centralized error type mapping to spec exit codes.
  - Standard `stderr` error output format.
- Out:
  - Command business logic.

#### Target Surface Area (Expected)

- `cmd/netsec-sk/main.go`
- `internal/cli/root.go`
- `internal/cli/errors.go`
- Public interfaces affected: CLI contract (SPEC §9.1-§9.3).

#### Acceptance Criteria (Task-Level)

- [ ] Global defaults are exactly `./default` repo and `default` env.
- [ ] All command errors emit `ERROR <error_code> <message>`.
- [ ] Exit codes match spec mapping.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestGlobalFlags`
  - `go test ./internal/cli -run TestExitCodeMapping`
- Expected results:
  - tests pass and assert exact stderr + exit code behavior.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0 (interface being implemented from spec)

### TASK-00002: Implement repo initialization and Git prerequisite checks

- Objective: Implement `init` repo bootstrap and hard failure when Git is unavailable.
- Spec refs: SPEC §3.1, §4, §9.4 (`init`)
- Status: Done
- Blocked by: none
- Depends on: TASK-00001
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- TASK-00001 complete.

#### Scope

- In:
  - Git binary presence check.
  - Repo directory creation and `git init` execution when needed.
  - Base directory scaffold creation.
- Out:
  - Environment folder creation.

#### Target Surface Area (Expected)

- `internal/repo/init.go`
- `internal/repo/git_check.go`
- `internal/repo/layout.go`
- Public interfaces affected: `init` command behavior (SPEC §9.4).

#### Acceptance Criteria (Task-Level)

- [ ] `init` fails with `E_GIT_MISSING` if Git is unavailable.
- [ ] `init` creates required base folders and no env directory.
- [ ] success stdout matches spec exactly.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/repo -run TestInitCreatesBaseLayout`
  - `go test ./internal/repo -run TestInitFailsWithoutGit`
- Expected results:
  - tests confirm layout and failure mode contracts.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00003: Implement environment lifecycle and validation

- Objective: Implement env ID normalization/validation plus `env list` and `env create` contracts.
- Spec refs: SPEC §5.1, §5.2, §9.4 (`env list`, `env create`)
- Status: Done
- Blocked by: none
- Depends on: TASK-00001, TASK-00002
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Base repo layout creation available.

#### Scope

- In:
  - Env ID regex validation and normalization.
  - `env create` idempotent behavior.
  - `env list` lexical ordering.
- Out:
  - Auto-create on ingest.

#### Target Surface Area (Expected)

- `internal/env/validate.go`
- `internal/env/service.go`
- `internal/cli/cmd_env.go`
- Public interfaces affected: env commands (SPEC §9.4).

#### Acceptance Criteria (Task-Level)

- [ ] invalid `env_id` values return `E_USAGE` + exit code 2.
- [ ] `env create` prints created/already-exists messages exactly.
- [ ] `env list` prints one env per line, sorted.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/env -run TestEnvIDValidation`
  - `go test ./internal/cli -run TestEnvCommandOutputs`
- Expected results:
  - tests verify regex, normalization, message contracts.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00004: Implement ingest orchestrator ordering, lock handling, and extraction cleanup

- Objective: Build deterministic ingest runtime skeleton with ordering, lock semantics, and extraction lifecycle.
- Spec refs: SPEC §3.4, §6.1, §6.2, §6.3, §9.4 (`ingest`)
- Status: Done
- Blocked by: none
- Depends on: TASK-00001, TASK-00002, TASK-00003
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Command framework and env model complete.

#### Scope

- In:
  - Input path expansion and lexical canonical ordering.
  - Lock file creation, active/stale detection, and cleanup.
  - Extraction workspace creation and per-TSF cleanup.
  - Auto-create env on ingest path.
- Out:
  - TSF content parsing.

#### Target Surface Area (Expected)

- `internal/ingest/orchestrator.go`
- `internal/ingest/lock.go`
- `internal/ingest/extract.go`
- Public interfaces affected: ingest runtime behavior (SPEC §6).

#### Acceptance Criteria (Task-Level)

- [ ] ingest order is deterministic for mixed file/dir input.
- [ ] active lock blocks with `E_LOCK_HELD`; stale lock removed with warning.
- [ ] per-TSF extraction dirs are removed after processing (unless `--keep-extract`).

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/ingest -run TestInputOrdering`
  - `go test ./internal/ingest -run TestLockStaleRules`
  - `go test ./internal/ingest -run TestExtractCleanup`
- Expected results:
  - tests assert exact lock-age and cleanup behavior.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00005: Implement TSF identity derivation and duplicate detection

- Objective: Parse `/tmp/cli` metadata and enforce per-environment TSF dedupe.
- Spec refs: SPEC §6.4, §6.5, §10.4
- Status: Done
- Blocked by: none
- Depends on: TASK-00004
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Ingest orchestrator provides extract paths and env context.

#### Scope

- In:
  - Candidate metadata file selection and tie-break behavior.
  - `tsf_original_name` derivation and serial extraction patterns.
  - `tsf_id` construction rules including `unknown` fallback.
  - Duplicate check using `.netsec-state/ingest.ndjson`.
- Out:
  - Deep snapshot parsing.

#### Target Surface Area (Expected)

- `internal/tsf/identity.go`
- `internal/ingest/dedupe.go`
- `internal/ingest/ingestlog_reader.go`
- Public interfaces affected: ingest dedupe behavior (SPEC §6.4-§6.5).

#### Acceptance Criteria (Task-Level)

- [ ] renamed archives dedupe based on TSF internal identity.
- [ ] `unknown` identity bypasses duplicate-TSF skip.
- [ ] duplicate result is logged as `skipped_duplicate_tsf`.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/tsf -run TestIdentityDerivation`
  - `go test ./internal/ingest -run TestDuplicateDetection`
- Expected results:
  - tests validate all identity fallback branches.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00006: Implement firewall and Panorama parsers with partial/fatal taxonomy

- Objective: Build facts-only parsers for firewall and Panorama snapshots with deterministic error classification.
- Spec refs: SPEC §6.6, §7.1, §7.2, §7.3
- Status: Done
- Blocked by: none
- Depends on: TASK-00004
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Extracted TSF file access available.

#### Scope

- In:
  - Entity classification (`firewall|panorama`).
  - Identity derivation (`id`, serial fallback rules).
  - Best-effort parsing for HA/network/routing and Panorama config groups.
  - Partial vs fatal parse boundary enforcement.
- Out:
  - RDNS enrichment.

#### Target Surface Area (Expected)

- `internal/parse/firewall.go`
- `internal/parse/panorama.go`
- `internal/parse/classifier.go`
- Public interfaces affected: snapshot JSON contracts (SPEC §7).

#### Acceptance Criteria (Task-Level)

- [ ] snapshots contain required envelope + identity fields.
- [ ] missing optional fields produce `parse_error_partial` not fatal.
- [ ] missing entity type or entity ID produces `parse_error_fatal`.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/parse -run TestFirewallSnapshotRequiredFields`
  - `go test ./internal/parse -run TestPanoramaSnapshotRequiredFields`
  - `go test ./internal/parse -run TestParseErrorClassification`
- Expected results:
  - tests verify schema and taxonomy boundaries.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00007: Implement RDNS enrichment for newly discovered firewall devices

- Objective: Apply optional RDNS with deterministic timeout/retry policy for new devices.
- Spec refs: SPEC §6.8, §7.2
- Status: Done
- Blocked by: none
- Depends on: TASK-00006
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Firewall parser returns device identity and `mgmt_ip`.

#### Scope

- In:
  - RDNS execution only when `--rdns` and device newly discovered.
  - 1-second timeout + one retry.
  - status mapping `ok|not_found|timeout|error`.
- Out:
  - DNS caching across runs.

#### Target Surface Area (Expected)

- `internal/enrich/rdns.go`
- `internal/ingest/orchestrator.go`
- Public interfaces affected: `device.dns.reverse` field behavior (SPEC §6.8).

#### Acceptance Criteria (Task-Level)

- [ ] existing devices do not trigger RDNS.
- [ ] timeout/retry behavior matches policy exactly.
- [ ] reverse DNS output fields are populated deterministically.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/enrich -run TestRDNSOnlyForNewDevices`
  - `go test ./internal/enrich -run TestRDNSTimeoutRetry`
- Expected results:
  - tests verify call count and status mapping.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00008: Implement canonical state hashing and unchanged-state skip

- Objective: Compute canonical `state_sha256` and skip commits when state is unchanged.
- Spec refs: SPEC §6.7, §7.4
- Status: Done
- Blocked by: none
- Depends on: TASK-00005, TASK-00006
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Parsed snapshots available.

#### Scope

- In:
  - Exclude volatile fields before hashing.
  - Deterministic sorting of arrays/lists.
  - Stable JSON serialization.
  - unchanged-state comparison against current `latest.json`.
- Out:
  - dedupe by full TSF payload hash.

#### Target Surface Area (Expected)

- `internal/state/hash.go`
- `internal/state/normalize.go`
- `internal/state/compare.go`
- Public interfaces affected: `state_sha256` semantics (SPEC §7.4).

#### Acceptance Criteria (Task-Level)

- [ ] identical logical state produces identical hash across runs.
- [ ] source metadata changes alone do not alter hash.
- [ ] unchanged state yields `skipped_state_unchanged` and no commit.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/state -run TestHashCanonicalization`
  - `go test ./internal/state -run TestUnchangedStateSkip`
- Expected results:
  - tests pass with stable golden hashes.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00009: Implement topology inference and manual override integration

- Objective: Build VR-aware IPv4 topology edges and merge optional override files.
- Spec refs: SPEC §2.2 (IPv4-only), §8.4, §8.5
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00006, TASK-00008
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Parsed interfaces/zones/VR data available.

#### Scope

- In:
  - shared-subnet edge inference (IPv4 only).
  - VR-context matching for inferred edges.
  - optional `vr_equivalence.json` and `topology_links.json` ingestion.
  - deterministic edge IDs and ordering.
- Out:
  - IPv6 inference.

#### Target Surface Area (Expected)

- `internal/topology/infer.go`
- `internal/topology/overrides.go`
- Public interfaces affected: `edges.csv` + `topology.mmd` content (SPEC §8.4-§8.5).

#### Acceptance Criteria (Task-Level)

- [ ] inferred edges are IPv4-only and VR-aware.
- [ ] override edges are included as `manual_override`.
- [ ] no cross-environment edges are produced.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/topology -run TestInferSharedSubnetEdges`
  - `go test ./internal/topology -run TestOverrideMerge`
- Expected results:
  - tests verify source/type flags and sorting.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00010: Implement export writers for JSON/CSV/Mermaid/agent context

- Objective: Generate all per-environment exports with exact schema/order contracts.
- Spec refs: SPEC §8.1, §8.2, §8.3, §8.4, §8.5, §8.6
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00008, TASK-00009
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Latest snapshot state + topology edges available.

#### Scope

- In:
  - `environment.json` writer with required keys.
  - fixed-header CSV writers.
  - deterministic Mermaid graph writer.
  - `agent_context.md` with required heading order.
- Out:
  - additional export formats.

#### Target Surface Area (Expected)

- `internal/export/environment_json.go`
- `internal/export/csv.go`
- `internal/export/mermaid.go`
- `internal/export/agent_context.go`
- Public interfaces affected: export file contracts (SPEC §8).

#### Acceptance Criteria (Task-Level)

- [ ] all six export files are produced for target env.
- [ ] CSV headers and row ordering match spec exactly.
- [ ] `agent_context.md` contains required heading sequence.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/export -run TestEnvironmentJSONSchema`
  - `go test ./internal/export -run TestCSVHeadersAndOrdering`
  - `go test ./internal/export -run TestAgentContextTemplate`
- Expected results:
  - tests validate exact schema and deterministic ordering.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00011: Implement commit pipeline and ingest/commit ledgers

- Objective: Persist snapshots + exports and create one atomic commit per changed TSF with strict file allowlist.
- Spec refs: SPEC §3.3, §10.1, §10.2, §10.3, §10.4
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00005, TASK-00008, TASK-00010
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Snapshot and export generation complete.

#### Scope

- In:
  - append to `.netsec-state/ingest.ndjson` for every attempt.
  - append to `state/commits.ndjson` only on commit.
  - stage only allowlisted files.
  - commit subject format per spec.
- Out:
  - multi-TSF squashed commits.

#### Target Surface Area (Expected)

- `internal/commit/committer.go`
- `internal/ingest/ingestlog_writer.go`
- `internal/state/commits_ledger.go`
- Public interfaces affected: git history + ledger contract (SPEC §10).

#### Acceptance Criteria (Task-Level)

- [ ] every state-changing TSF creates exactly one Git commit.
- [ ] non-changing TSFs never create commits.
- [ ] commit subject and staged file list match spec exactly.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/commit -run TestAtomicCommitAllowlist`
  - `go test ./internal/commit -run TestCommitMessageFormat`
  - `go test ./internal/ingest -run TestIngestLedgerAllAttempts`
- Expected results:
  - tests validate per-TSF commit count and ledger semantics.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0

### TASK-00012: Implement query commands, interactive shell, help system, and release verification

- Objective: Complete user-facing command set (`devices`, `panorama`, `show`, `topology`, `help`, `open`) and release outputs (binaries + checksums) with end-to-end acceptance tests.
- Spec refs: SPEC §0, §9.4, §11
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00002, TASK-00003, TASK-00010, TASK-00011
- Changelog requirement: Yes
- Plan update: On completion, set status to Done and strike through in lane view.

#### Preconditions

- Core ingest/export/commit path complete.

#### Scope

- In:
  - implement remaining one-shot commands and exact stdout contracts.
  - implement interactive `open` shell prompt and supported commands.
  - implement `help` and `help <command>` content contract.
  - define release build/checksum script for macOS arm64/amd64 and Windows amd64.
  - execute acceptance checklist from spec.
- Out:
  - package manager automation.

#### Target Surface Area (Expected)

- `internal/cli/cmd_devices.go`
- `internal/cli/cmd_panorama.go`
- `internal/cli/cmd_show.go`
- `internal/cli/cmd_topology.go`
- `internal/cli/cmd_help.go`
- `internal/cli/cmd_open.go`
- `scripts/release/build_and_checksum.sh`
- Public interfaces affected: CLI command outputs and release artifacts (SPEC §9.4, §11).

#### Acceptance Criteria (Task-Level)

- [ ] command outputs match exact line/header formats in spec.
- [ ] interactive shell prompt and supported commands match spec.
- [ ] release script produces required binaries and checksums.
- [ ] full acceptance checklist in SPEC §11 passes.

#### Verification (Proof Required)

- Commands/checks:
  - `go test ./internal/cli -run TestCommandOutputContracts`
  - `go test ./internal/cli -run TestOpenShellCommandSet`
  - `go test ./e2e -run TestMVPAcceptanceChecklist`
  - `./scripts/release/build_and_checksum.sh`
- Expected results:
  - tests pass and release script emits artifact + checksum files for specified targets.

#### Budget (Enforced)

- Files changed <= 10
- New files <= 3
- Net new LOC <= 300
- Public interface changes = 0
