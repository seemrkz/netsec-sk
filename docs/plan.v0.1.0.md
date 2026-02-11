---
doc_type: plan
project: "NetSec StateKit"
plan_id: "PLAN-00001"
version: "0.1.0"
owners:
  - "Chris Marks (Product Owner)"
  - "Implementation Agent"
last_updated: "2026-02-11"
change_control:
  amendment_required_for_substantive_change: true
  metadata_change_allowed_without_amendment: true
related:
  spec: "./spec.v0.1.3.md"
  amendments_dir: "./amendments/"
  changelog: "./changelog.md"
---

# NetSec StateKit — Implementation Plan

> This plan is based strictly on `docs/spec.v0.1.3.md` and `docs/AGENTS.md`.

---

## 0. Plan Guardrails (Mandatory)

- Plan MUST NOT introduce scope not explicitly present in `spec.v0.1.3.md`.
- Any substantive plan change requires an amendment.
- Tasks MUST be atomic and follow enforced budgets unless overridden by spec.
- Plan creation is BLOCKED unless the spec is deterministic (Blocking Questions empty, Spec Ambiguity Audit has 0 ambiguities, required decisions Active).
- If planning discovers missing or ambiguous spec information, mark the relevant task(s) as `Blocked`, document the blocker, and STOP planning.
- Plan is deterministic only when every task has all required fields, no tasks are `Blocked`, and verification steps are fully specified.

### 0.1 Plan Review Round Log (Append‑Only; does not require amendment)
- Round ID: PR-00001
- Date: 2026-02-11
- Reviewers (LLMs + roles): Reviewer P1 (Scope Enforcer), Reviewer P2 (Atomicity/Budget Auditor)
- Outcome: PASS
- Blockers count: 0
- Summary of changes applied: Drafted atomic tasks that map directly to spec sections, added explicit dependencies, verification proofs, budget declarations, commit proof fields, and worktree lanes.
- Amendment link (if substantive changes required): (none)

---

## 1. Enforced Default Budgets (Inherited from AGENTS.md)

Unless the spec explicitly overrides:
- Max files changed per task: 10
- Max new files per task: 3
- Max net new LOC per task: 300
- Max public interface changes per task: 0 (unless explicitly stated)

If any budget will be exceeded, split into multiple tasks.

---

## 2. Phase Overview

| Phase | Title | Focus | Exit Criteria |
|------:|------|-------|--------------|
| I | Runtime + Environment | startup/runtime contract and environment lifecycle | health + env create/list/delete contracts verified |
| II | Ingest Core | ingest pipeline, extraction, state persistence, logs | ingest statuses + schemas + commit rules verified |
| III | Advanced Flows | batch, RMA, topology, flow trace | all F2/F5/F6 acceptance criteria verified |
| IV | Release Verification | full proof capture per spec | §7.1 proof artifacts recorded in changelog |

---

## 2.1 Worktree (Lanes for Execution Order) (Mandatory)

Lane Index (parseable; used for git worktree mapping):
- LANE1: runtime-api
- LANE2: ingest-core
- LANE3: analysis-flow

Lane view:

LANE1          | LANE2        | LANE3
runtime-api    | ingest-core  | analysis-flow
----------------------------------------------
~~TASK-00001~~ | TASK-00004   | TASK-00010
~~TASK-00002~~ | TASK-00005   | TASK-00011
TASK-00003     | TASK-00006   | TASK-00012
               | TASK-00007   |
               | TASK-00008   |
               | TASK-00009   |

---

## 3. Task Registry (Mandatory)

**Rule:** 1 task = 1 changelog entry.

### 3.1 Task Schema Contract (Mandatory Fields)
A task is invalid unless it includes:
- Objective (one sentence)
- Spec references (section IDs)
- Preconditions (what must exist first)
- Dependencies (task IDs)
- Scope In / Scope Out
- Acceptance criteria (task-level)
- Verification steps (commands/checks + expected results)
- Budget declaration (inherit defaults or override if explicitly permitted by spec)
- Status + Blockers (if blocked)
- Commit requirement (always Yes)
- Commit proof on completion (hash + message)
- Changelog requirement (always Yes)
- Plan update requirement (status + worktree strike-through + commit proof update)

---

## 4. Tasks

### TASK-00001: Runtime Bootstrap + Health Contract
- Objective: Implement startup/runtime behavior for localhost ephemeral serving and health endpoint.
- Spec refs: SPEC §4.3 D-00001, SPEC §4.3 D-00010, SPEC §9.5, SPEC §10.1.
- Status: Done
- Blocked by: none
- Depends on: none
- Commit requirement: Yes
- Commit proof: 181a68b | TASK-00001: implement runtime bootstrap and health endpoint
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- `spec.v0.1.3.md` remains the active deterministic spec.

#### Scope
- In:
  - Bind backend to `127.0.0.1` on ephemeral port.
  - Write `~/.netsec-sk/runtime/server.json` with required schema.
  - Print `NETSEC_SK_URL=<url>` once server is ready.
  - Implement `GET /api/health` response contract.
- Out:
  - Environment CRUD.
  - Ingest, topology, and flow trace behavior.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Runtime server startup path.
  - Health handler path.
  - Runtime metadata writer for `~/.netsec-sk/runtime/server.json`.
- Public interfaces affected: `GET /api/health` (SPEC §10.1).

#### Acceptance Criteria (Task-Level)
- [ ] Server binds only to `127.0.0.1` and uses an ephemeral available port.
- [ ] `runtime/server.json` contains `url`, `port`, `pid`, `started_at`, `version`.
- [ ] `GET /api/health` returns `200` with `{ version, started_at, url }`.

#### Verification (Proof Required)
- Commands/checks:
  - `jq -e '.url and .port and .pid and .started_at and .version' "$HOME/.netsec-sk/runtime/server.json"`
  - `BASE_URL="$(jq -r '.url' "$HOME/.netsec-sk/runtime/server.json")" && curl -sS "$BASE_URL/api/health" | jq -e '.version and .started_at and .url'`
- Expected results:
  - `server.json` validates with all required keys.
  - Health endpoint returns required fields with HTTP 200.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes ≤ 1 (explicitly permitted by SPEC §10.1)

---

### TASK-00002: Environment Lifecycle (Create/List/Delete Soft Delete)
- Objective: Implement deterministic environment create/list/delete behavior and on-disk metadata lifecycle.
- Spec refs: SPEC §5.1 F1, SPEC §5.2 Flow A, SPEC §5.3 AC-F1-1..3, SPEC §9.1, SPEC §9.6, SPEC §10.2, SPEC §10.5.
- Status: Done
- Blocked by: none
- Depends on: TASK-00001
- Commit requirement: Yes
- Commit proof: 48475b1 | TASK-00002: implement environment create list delete APIs
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Runtime process and base API routing operational (TASK-00001).

#### Scope
- In:
  - `POST /api/environments` with required validation and `ERR_ENV_NAME_REQUIRED`.
  - `GET /api/environments` returns non-soft-deleted environments.
  - `DELETE /api/environments/{env_id}` soft-deletes and moves folder to `~/.netsec-sk/trash/<env_id>/`.
  - Persist `meta.json` per schema in §9.6.
- Out:
  - State retrieval, commits retrieval, ingest behavior.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Environment repository/store implementation.
  - Environment API handlers for create/list/delete.
  - `~/.netsec-sk/environments/<env_id>/meta.json` and `~/.netsec-sk/trash/<env_id>/...` behavior.
- Public interfaces affected: `POST /api/environments`, `GET /api/environments`, `DELETE /api/environments/{env_id}` (SPEC §10.2).

#### Acceptance Criteria (Task-Level)
- [ ] Environment creation persists metadata and returns the §10.2 success shape.
- [ ] Empty name returns HTTP 400 with `ERR_ENV_NAME_REQUIRED`.
- [ ] Deletion moves environment to trash and removed env is omitted from default list.

#### Verification (Proof Required)
- Commands/checks:
  - `BASE_URL="$(jq -r '.url' "$HOME/.netsec-sk/runtime/server.json")"`
  - `CREATE_OUT="$(curl -sS -X POST "$BASE_URL/api/environments" -H 'Content-Type: application/json' -d '{"name":"plan-e2e-env","description":"deterministic"}')" && echo "$CREATE_OUT" | jq -e '.env_id and .name=="plan-e2e-env"'`
  - `ENV_ID="$(echo "$CREATE_OUT" | jq -r '.env_id')" && curl -sS "$BASE_URL/api/environments" | jq -e --arg id "$ENV_ID" '.environments | any(.env_id == $id)'`
  - `curl -sS -X DELETE "$BASE_URL/api/environments/$ENV_ID" | jq -e '.soft_deleted == true and .soft_deleted_at'`
  - `curl -sS "$BASE_URL/api/environments" | jq -e --arg id "$ENV_ID" '.environments | all(.env_id != $id)'`
  - `test -f "$HOME/.netsec-sk/trash/$ENV_ID/meta.json"`
- Expected results:
  - API responses and on-disk movement match spec.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes ≤ 3 (explicitly permitted by SPEC §10.2)

---

### TASK-00003: Environment Read APIs (State + Commits)
- Objective: Expose deterministic environment state and commit history read endpoints.
- Spec refs: SPEC §5.2 Flow D/E, SPEC §5.3 AC-F4-3..4, SPEC §9.3, SPEC §9.7, SPEC §10.2, SPEC §10.5.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00002
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00003: implement environment state and commits read APIs>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Environment lifecycle APIs and storage paths are available (TASK-00002).

#### Scope
- In:
  - `GET /api/environments/{env_id}/state` returns current `state.json` object.
  - `GET /api/environments/{env_id}/commits` returns unpaginated commit list.
  - Commit list ordering by `timestamp` desc, tie-break `commit_id` asc.
  - Error mapping for missing/deleted/not-found conditions.
- Out:
  - Producing `state.json` and `commits.ndjson` content.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Environment read API handlers.
  - Commit record loader/ordering logic.
  - Error mapping to `ERR_ENV_NOT_FOUND`, `ERR_ENV_ALREADY_DELETED`, `ERR_ENV_STATE_NOT_FOUND`.
- Public interfaces affected: `GET /api/environments/{env_id}/state`, `GET /api/environments/{env_id}/commits` (SPEC §10.2).

#### Acceptance Criteria (Task-Level)
- [ ] State endpoint returns exactly the stored state object shape.
- [ ] Commits endpoint returns all commit entries unpaginated with deterministic ordering.
- [ ] Error responses conform to global error contract.

#### Verification (Proof Required)
- Commands/checks:
  - `BASE_URL="$(jq -r '.url' "$HOME/.netsec-sk/runtime/server.json")"`
  - `curl -sS "$BASE_URL/api/environments/$ENV_ID/state" | jq -e --arg id "$ENV_ID" '.state.schema_version == "1.0.0" and .state.env.env_id == $id'`
  - `API_COUNT="$(curl -sS "$BASE_URL/api/environments/$ENV_ID/commits" | jq '.commits | length')" && FILE_COUNT="$(wc -l < "$HOME/.netsec-sk/environments/$ENV_ID/commits.ndjson")" && test "$API_COUNT" -eq "$FILE_COUNT"`
  - `curl -sS "$BASE_URL/api/environments/$ENV_ID/commits" | jq -e '.commits as $c | ($c | length) < 2 or ([range(0; ($c|length)-1)] | all($c[.].timestamp >= $c[.+1].timestamp))'`
- Expected results:
  - API output count and ordering match disk-backed source.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes ≤ 2 (explicitly permitted by SPEC §10.2)

---

### TASK-00004: Ingest Orchestration + Status Endpoint
- Objective: Implement ingest lifecycle orchestration with deterministic stage/status reporting.
- Spec refs: SPEC §5.1 F2.1/F2.3/F2.4, SPEC §5.2 Flow B, SPEC §6.1, SPEC §9.2, SPEC §10.3.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00001, TASK-00002
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00004: implement ingest orchestration and status API>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Runtime service and environment existence checks are operational.

#### Scope
- In:
  - `POST /api/environments/{env_id}/ingests` accepts multipart file and returns `202 ingest_id`.
  - `GET /api/ingests/{ingest_id}` returns `{ status, stage, progress }` and `final_record` on completion.
  - Stage order and status transitions reflect §6.1.
  - Track `duration_ms_by_stage`, `duration_ms_compute`, `duration_ms_total`.
- Out:
  - Detailed extraction logic.
  - RMA decision handling.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Ingest coordinator/job state module.
  - Upload endpoint and ingest status endpoint handlers.
  - In-memory or persisted ingest status tracking.
- Public interfaces affected: ingest create/status APIs (SPEC §10.3).

#### Acceptance Criteria (Task-Level)
- [ ] Ingest creation returns HTTP 202 + UUID `ingest_id`.
- [ ] Status endpoint reports valid `status` + `stage` values and progress object.
- [ ] Completed ingests include `final_record` that matches §9.2 schema contract.

#### Verification (Proof Required)
- Commands/checks:
  - `BASE_URL="$(jq -r '.url' "$HOME/.netsec-sk/runtime/server.json")"`
  - `INGEST_ID="$(curl -sS -X POST -F "file=@$FIREWALL_TSF" "$BASE_URL/api/environments/$ENV_ID/ingests" | jq -r '.ingest_id')" && test -n "$INGEST_ID"`
  - `curl -sS "$BASE_URL/api/ingests/$INGEST_ID" | jq -e '.ingest_id and .env_id and .status and .stage and .progress.pct >= 0 and .progress.pct <= 100'`
  - `curl -sS "$BASE_URL/api/ingests/$INGEST_ID" | jq -e 'if .status == "completed" then (.final_record.ingest_id == .ingest_id) else true end'`
- Expected results:
  - Ingest lifecycle fields are present and valid.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes ≤ 2 (explicitly permitted by SPEC §10.3)

---

### TASK-00005: TSF Extraction + Classification (Normative Appendix)
- Objective: Implement deterministic TSF field extraction and device classification per Appendix A.
- Spec refs: SPEC §4.3 D-00006, SPEC §5.3 AC-F2-7, SPEC §9.7.1, SPEC §9.7.2, SPEC Appendix A.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00004
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00005: implement TSF extraction and device classification>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Ingest stage orchestration exists and can pass extracted payloads.

#### Scope
- In:
  - Source discovery by pattern (CLI/config/panorama merged config).
  - Fallback order: runtime CLI -> panorama merged config -> local config -> `not_found`.
  - Required output extraction for identity, management, HA, licenses, network inventory.
  - Device type classification `firewall|panorama|unknown`.
  - Panorama-only inventory extraction fields.
- Out:
  - Topology inference and flow path algorithms.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - TSF archive scanner/parser module.
  - Appendix A extraction mapping implementation module.
  - Device normalization/classification module.
- Public interfaces affected: None (internal extraction logic only).

#### Acceptance Criteria (Task-Level)
- [ ] Required extracted fields are present in payload/state target structures; required keys are never omitted.
- [ ] Missing fields are represented as `"not_found"` where specified.
- [ ] Panorama payloads populate panorama-only sections in state schema when applicable.

#### Verification (Proof Required)
- Commands/checks:
  - `tail -n 1 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -e '.device.device_type and .device.serial and .device.hostname'`
  - `jq -e '.devices.logical[] | .current.identity | has("hostname") and has("model") and has("serial") and has("panos_version") and has("mgmt_ip")' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"`
  - `jq -e '.devices.logical[] | select(.device_type=="panorama") | .current.panorama.managed_device_serials' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"`
- Expected results:
  - Required fields are present and normalized according to Appendix A + schema.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes = 0 (default)

---

### TASK-00006: Deterministic State Persistence + Intro Generation
- Objective: Build deterministic state representation and atomic persistence behavior.
- Spec refs: SPEC §4.3 D-00002, SPEC §4.3 D-00011, SPEC §5.3 AC-F3-1..3, SPEC §9.1, SPEC §9.7, SPEC §9.7.3, SPEC §9.8.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00005
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00006: implement canonical state persistence and intro generation>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Extraction payloads are available for state assembly.

#### Scope
- In:
  - Write `state.json` with lexicographic object keys and required array ordering rules.
  - Persist atomically via temp file + fsync + rename; maintain `state.json.bak`.
  - Generate and rewrite `intro.md` on every successful ingest including `no_change`.
  - Ensure canonical JSON hashing rules are implementable for commit records.
- Out:
  - Ingest/commit log append policies.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - State builder/canonicalizer module.
  - Atomic file writer utility.
  - Intro markdown renderer.
  - `~/.netsec-sk/environments/<env_id>/state.json`, `.bak`, `intro.md`.
- Public interfaces affected: None (on-disk schema behavior only).

#### Acceptance Criteria (Task-Level)
- [ ] Persisted `state.json` includes required top-level keys and schema_version `1.0.0`.
- [ ] `intro.md` includes all mandatory sections and pointer bullets.
- [ ] Persistence failures preserve prior valid state via backup semantics.

#### Verification (Proof Required)
- Commands/checks:
  - `jq -e '.schema_version == "1.0.0" and .generated_at and .env.env_id and .devices.logical and .topology.inferred_adjacencies' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"`
  - `rg -n '^# |Quick facts|Where to look in state.json|AI Agent notes|/devices/logical|/topology/inferred_adjacencies|/devices/logical\[i\]/current/network' "$HOME/.netsec-sk/environments/$ENV_ID/intro.md"`
  - `tail -c1 "$HOME/.netsec-sk/environments/$ENV_ID/state.json" | od -An -t x1 | rg -q '0a'`
- Expected results:
  - State and intro satisfy schema and formatting contracts.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes = 0 (default)

---

### TASK-00007: Ingest/Commit Logs + Dedupe + No-Change Semantics
- Objective: Enforce ingest log and commit log rules including duplicate/no-change behavior.
- Spec refs: SPEC §4.3 D-00004, SPEC §4.3 D-00005, SPEC §5.3 AC-F2-1..4, SPEC §5.3 AC-F4-1..2, SPEC §9.2, SPEC §9.3, SPEC §9.7.3.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00004, TASK-00006
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00007: implement ingest and commit log semantics>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Ingest lifecycle and state persistence are operational.

#### Scope
- In:
  - Append one final ingest log record per ingest attempt.
  - Compute/store `fingerprint_sha256` while streaming upload.
  - Implement `duplicate` detection by archive hash.
  - Implement `no_change` detection by canonical state comparison.
  - Append commit record only when canonical state changes.
  - Enforce append-only + fsync behavior for NDJSON logs.
- Out:
  - Batch sequencing and RMA-specific decision flows.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Ingest log writer for `ingest.ndjson`.
  - Commit log writer for `commits.ndjson`.
  - Fingerprint and canonical diff utilities.
- Public interfaces affected: None (API shapes unchanged).

#### Acceptance Criteria (Task-Level)
- [ ] Success, duplicate, no_change, and error ingest statuses are logged with required timing fields.
- [ ] Duplicate and no_change ingests do not append commit entries.
- [ ] Commit entries include required hashes, summary fields, and changed JSON pointer paths.

#### Verification (Proof Required)
- Commands/checks:
  - `INGEST_LINES_BEFORE="$(wc -l < "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson")" && COMMIT_LINES_BEFORE="$(wc -l < "$HOME/.netsec-sk/environments/$ENV_ID/commits.ndjson")"`
  - `# ingest same TSF twice; second should be duplicate`
  - `tail -n 1 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -e '.status == "duplicate" and .fingerprint_sha256 and .duration_ms_total >= 0 and .duration_ms_compute >= 0'`
  - `COMMIT_LINES_AFTER="$(wc -l < "$HOME/.netsec-sk/environments/$ENV_ID/commits.ndjson")" && test "$COMMIT_LINES_BEFORE" -eq "$COMMIT_LINES_AFTER"`
- Expected results:
  - Duplicate/no_change behavior and commit-on-change-only rule are enforced.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes = 0 (default)

---

### TASK-00008: Batch Ingest Sequencing + Continue-on-Error
- Objective: Implement deterministic batch ordering and failure-continuation behavior.
- Spec refs: SPEC §4.3 D-00012, SPEC §5.2 Flow C, SPEC §5.3 AC-F2-5.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00004, TASK-00007
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00008: implement batch ingest order and continue semantics>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Single-ingest processing and logging are complete.

#### Scope
- In:
  - Ensure per-file ingests for batch execute sequentially.
  - Enforce bytewise lexicographic filename ordering.
  - Continue processing remaining files after individual ingest failures.
  - Preserve one final ingest log entry per file.
- Out:
  - RMA decision logic and topology/flow semantics.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Batch ingest coordinator.
  - Filename sorter and sequential execution control.
- Public interfaces affected: None (existing ingest contract reused).

#### Acceptance Criteria (Task-Level)
- [ ] Batch with mixed valid/invalid files yields one ingest log entry per input file.
- [ ] Failure in one file does not block subsequent files.
- [ ] Processing order is deterministic by filename ascending.

#### Verification (Proof Required)
- Commands/checks:
  - `for f in $(find "$BATCH_DIR" -name '*.tgz' -maxdepth 1 | LC_ALL=C sort); do curl -sS -X POST -F "file=@$f" "$BASE_URL/api/environments/$ENV_ID/ingests" >/dev/null; done`
  - `tail -n 3 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -s 'length == 3 and any(.[]; .status=="error")'`
  - `tail -n 3 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -r '.source.filenames[0]'`
- Expected results:
  - Exactly three entries recorded for three input files and ordering is ascending by filename.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes = 0 (default)

---

### TASK-00009: RMA Awaiting-User Decision Flow
- Objective: Implement RMA confirmation pause/decision semantics with deterministic state mutation rules.
- Spec refs: SPEC §4.3 D-00009, SPEC §5.2 Flow C2, SPEC §5.3 AC-F2-6, SPEC §6.1, SPEC §9.2, SPEC §9.4, SPEC §10.3, SPEC §10.5.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00004, TASK-00005, TASK-00006, TASK-00007
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00009: implement RMA awaiting user workflow>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Ingest extraction and state persistence foundations are complete.

#### Scope
- In:
  - Detect hostname match + serial mismatch scenario.
  - Transition ingest to `awaiting_user` before state mutation.
  - Expose RMA candidates via `GET /api/ingests/{ingest_id}`.
  - Process `POST /api/ingests/{ingest_id}/rma-decision` for `link_replacement|treat_as_new_device|canceled`.
  - Persist optional runtime ingest payload while awaiting decision and enforce deletion/TTL rules.
- Out:
  - Batch ordering logic.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - RMA matcher and decision coordinator.
  - Ingest status API response model.
  - Runtime temporary ingest payload manager for `~/.netsec-sk/runtime/ingests/<ingest_id>.json`.
- Public interfaces affected: `GET /api/ingests/{ingest_id}` RMA prompt fields, `POST /api/ingests/{ingest_id}/rma-decision` (SPEC §10.3).

#### Acceptance Criteria (Task-Level)
- [ ] Ingest pauses in `awaiting_user` for qualifying RMA scenarios and does not mutate state before decision.
- [ ] Link/new/cancel decisions execute specified outcomes and final status/error behavior.
- [ ] Final ingest record captures `rma.prompted` and `rma.decision`.

#### Verification (Proof Required)
- Commands/checks:
  - `# ingest TSF A (hostname X, serial S1), then TSF B (hostname X, serial S2)`
  - `curl -sS "$BASE_URL/api/ingests/$INGEST_B" | jq -e '.status=="awaiting_user" and .rma_prompt.required==true and (.rma_prompt.candidates|length)>0'`
  - `curl -sS -X POST "$BASE_URL/api/ingests/$INGEST_B/rma-decision" -H 'Content-Type: application/json' -d '{"decision":"link_replacement","target_logical_device_id":"'$TARGET_LOGICAL_ID'"}' | jq -e '.'`
  - `curl -sS "$BASE_URL/api/ingests/$INGEST_B" | jq -e '.status=="completed" and .final_record.rma.prompted==true and .final_record.rma.decision=="link_replacement"'`
  - `test ! -f "$HOME/.netsec-sk/runtime/ingests/$INGEST_B.json"`
- Expected results:
  - Awaiting-user pause, decision processing, and cleanup behavior match spec.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes ≤ 1 (explicitly permitted by SPEC §10.3)

---

### TASK-00010: Topology Inference (CIDR Overlap + Evidence)
- Objective: Implement deterministic firewall adjacency inference from routed CIDR overlap.
- Spec refs: SPEC §4.3 D-00007, SPEC §5.1 F5, SPEC §5.3 AC-F5-1..4, SPEC §6.2, SPEC §6.3, SPEC §9.7, SPEC §9.7.3.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00005, TASK-00006
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00010: implement topology inference with overlap evidence>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Device network inventory and canonical state persistence are available.

#### Scope
- In:
  - Build routed CIDR sets excluding default routes.
  - Compute overlap pairs by CIDR intersection.
  - Build inferred adjacency edges with evidence fields from both sides.
  - Select most-specific overlap evidence (ties included).
  - Sort inferred adjacency array deterministically.
- Out:
  - Flow trace endpoint output.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Topology derivation/inference module.
  - CIDR overlap helper.
  - State topology writer for `/topology/inferred_adjacencies`.
- Public interfaces affected: None (internal algorithm and persisted state content).

#### Acceptance Criteria (Task-Level)
- [ ] Adjacency is inferred only when non-default routed CIDRs overlap.
- [ ] Default route `0.0.0.0/0` is excluded from adjacency inference.
- [ ] Every inferred edge includes evidence from both firewalls and overlapping CIDRs.

#### Verification (Proof Required)
- Commands/checks:
  - `jq -e '.topology.inferred_adjacencies | type=="array"' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"`
  - `jq -e '.topology.inferred_adjacencies[] | (.evidence | length) > 0' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"`
  - `jq -e '.topology.inferred_adjacencies[] | .evidence[] | (.cidr_i != "0.0.0.0/0" and .cidr_j != "0.0.0.0/0")' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"`
- Expected results:
  - Inference output matches algorithm constraints and evidence requirements.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes = 0 (default)

---

### TASK-00011: Flow Trace API + Mermaid Output
- Objective: Implement deterministic flow-trace endpoint behavior and Mermaid output contract.
- Spec refs: SPEC §4.3 D-00008, SPEC §5.1 F6, SPEC §5.3 AC-F6-1..4, SPEC §6.4, SPEC §10.4, SPEC §10.5.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00003, TASK-00010
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00011: implement flow trace endpoint and mermaid output>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- Environment state/commits endpoints and topology data are available.

#### Scope
- In:
  - Implement `POST /api/environments/{env_id}/flow-trace` request validation for IP literals.
  - Implement deterministic hop resolution rules and loop handling.
  - Return raw Mermaid source and stable hop list.
  - Map errors to `ERR_INVALID_IP`, `ERR_FLOW_SRC_NOT_FOUND`, `ERR_FLOW_PATH_NOT_FOUND`.
- Out:
  - Any policy/NAT/security rule evaluation.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - Flow trace algorithm implementation.
  - Flow trace API handler + response model.
  - Mermaid renderer utility.
- Public interfaces affected: `POST /api/environments/{env_id}/flow-trace` (SPEC §10.4).

#### Acceptance Criteria (Task-Level)
- [ ] Valid flow requests return firewall-only hop list and Mermaid source text.
- [ ] Invalid IP inputs return 400 with `ERR_INVALID_IP`.
- [ ] Source-not-found and path-not-found return specified 404 errors.

#### Verification (Proof Required)
- Commands/checks:
  - `curl -sS -X POST "$BASE_URL/api/environments/$ENV_ID/flow-trace" -H 'Content-Type: application/json' -d '{"src_ip":"10.0.0.10","dst_ip":"10.0.1.20"}' | jq -e '.hops|length>0 and (.mermaid|type=="string")'`
  - `curl -sS -X POST "$BASE_URL/api/environments/$ENV_ID/flow-trace" -H 'Content-Type: application/json' -d '{"src_ip":"not-an-ip","dst_ip":"10.0.1.20"}' | jq -e '.code=="ERR_INVALID_IP"'`
  - `curl -sS -X POST "$BASE_URL/api/environments/$ENV_ID/flow-trace" -H 'Content-Type: application/json' -d '{"src_ip":"203.0.113.10","dst_ip":"10.0.1.20"}' | jq -e '.code=="ERR_FLOW_SRC_NOT_FOUND"'`
- Expected results:
  - Success/error outputs conform exactly to §10.4 and §10.5.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes ≤ 1 (explicitly permitted by SPEC §10.4)

---

### TASK-00012: Release Verification + Proof Recording
- Objective: Execute the mandatory verification matrix and record release proof artifacts in changelog.
- Spec refs: SPEC §7.1, SPEC §7.2, SPEC §2.3, SPEC §10.5.
- Status: Not Started
- Blocked by: none
- Depends on: TASK-00001, TASK-00002, TASK-00003, TASK-00004, TASK-00005, TASK-00006, TASK-00007, TASK-00008, TASK-00009, TASK-00010, TASK-00011
- Commit requirement: Yes
- Commit proof: Pending | <hash - TASK-00012: execute verification matrix and record proof>
- Changelog requirement: Yes (record in `docs/changelog.md`)
- Plan update: On completion, update this task’s Status, strike it through in Worktree lanes, and fill commit proof.

#### Preconditions
- All prior tasks are complete and deployable in local environment.

#### Scope
- In:
  - Run all required verification scenarios from spec §7.1.
  - Capture and record required evidence in `docs/changelog.md`.
  - Confirm Definition of Done gates are met.
- Out:
  - New feature development.

#### Target Surface Area (Expected)
- Files/modules likely touched:
  - `docs/changelog.md` (proof log and task entries).
  - Test fixture inventory used for verification execution.
- Public interfaces affected: None.

#### Acceptance Criteria (Task-Level)
- [ ] Success, duplicate, no_change, batch-continue, RMA, and flow-trace scenarios are all executed.
- [ ] Required proof artifacts are recorded: state checksums, ingest timing excerpts, commit-change excerpts.
- [ ] Every completed task has matching commit proof and exactly one changelog entry.

#### Verification (Proof Required)
- Commands/checks:
  - `shasum -a 256 "$HOME/.netsec-sk/environments/$ENV_ID/state.json"`
  - `tail -n 5 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson"`
  - `tail -n 5 "$HOME/.netsec-sk/environments/$ENV_ID/commits.ndjson"`
  - `rg -n 'TASK-0000[1-9]|TASK-0001[0-2]' docs/changelog.md`
- Expected results:
  - Required proof artifacts exist and are logged for all mandatory scenarios.
- Record in changelog:
  - exact commands + observed results

#### Budget (Enforced)
- Files changed ≤ 10 (default)
- New files ≤ 3 (default)
- Net new LOC ≤ 300 (default)
- Public interface changes = 0 (default)
