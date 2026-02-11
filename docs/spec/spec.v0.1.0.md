---
doc_type: spec
project: "NetSec StateKit"
spec_id: "SPEC-00001"
version: "0.1.0"
owners:
  - "Chris Marks (Product Owner)"
stakeholders:
  - "Network Security Operators (Users)"
  - "AI Agent Consumers (Users)"
review_rounds_completed: 1
last_updated: "2026-02-10"
change_control:
  amendment_required_for_substantive_change: true
  metadata_change_allowed_without_amendment: true
related:
  plan: "./plan.v0.1.0.md"
  amendments_dir: "./amendments/"
  changelog: "./changelog.md"
---

# NetSec StateKit — Specification (SSoT)

> **This document is the Single Source of Truth (SSoT).**
> If something is not explicitly stated here, it is not implementable.

---

## 0. Spec Determinism & Review History (Mandatory)

### 0.1 Review Round Log (Append‑Only; does not require amendment)
- Round ID: SR-00001
- Date: 2026-02-10
- Reviewers (LLMs + roles):
  - Reviewer A: Ambiguity Hunter
  - Reviewer B: Edge Cases & Failure Modes
  - Reviewer C: Security/Privacy/Abuse Modes
  - Reviewer D: Testability & Verification
- Outcome: PASS
- Blockers count: 0
- Summary of changes applied: Initial deterministic spec draft (v0.1.0).
- Amendment link (if substantive changes required): (none)

**Rule:** If any blockers exist, spec cannot be considered deterministic.  
**Determinism criteria:** Blocking Questions is empty, Spec Ambiguity Audit has 0 ambiguities, and all required decisions are Active.

### 0.2 Versioning Rules (Deterministic)
- Version format: `MAJOR.MINOR.PATCH` (e.g., `0.1.0`).
- Filename MUST match the front matter version: `spec.vX.Y.Z.md`.
- Baseline: blank templates start at `0.0.0`; first real draft is `0.1.0`.
- **MAJOR**: scope-breaking changes (Vision/Core Objectives/Non‑Goals, removal/renaming of top‑level features or interfaces).
- **MINOR**: backward‑compatible scope expansion (new top‑level feature, flow, or acceptance criteria).
- **PATCH (Fix)**: clarification or correction with no scope expansion or removal.

---

## 1. Prime Directive & Scope (Mandatory)

### 1.1 Vision (BLUF)
NetSec StateKit is a local macOS application that builds and maintains a JSON representation of the "state" of a network security deployment by ingesting Palo Alto Networks tech support bundles (TSF `.tgz`) from firewalls and/or Panorama. It is designed so humans can quickly understand an environment (prod/lab/home/customerA/...) and AI agents can quickly gain reliable context from `state.json` + `intro.md`.

### 1.2 Core Objectives
- O1: A user can create an Environment and ingest one or more TSFs to produce an updated `state.json` within that Environment.
- O2: Every ingest attempt is auditable (success/failure/duplicate/no-change), including total runtime and stage runtimes.
- O3: The app can infer firewall-to-firewall adjacency and trace firewall-only flow paths, with Mermaid diagrams.

### 1.3 Non‑Goals (Strict)
- NG1: No security policy or NAT rule evaluation for flow tracing in MVP.
- NG2: No background file watching; ingestion is user-initiated only.
- NG3: No retention of original TSF archives after ingestion completes.
- NG4: No export bundles or external integrations (cloud sync, APIs to third parties) in MVP.

### 1.4 Success Metrics
- Metric: Ingest completion time, baseline: unknown, target: ≤ 60s for a 1GB TSF on an M1-class Mac, by: 2026-04-01.
- Metric: Peak RSS during ingest, baseline: unknown, target: ≤ 1.5GB for a 1GB TSF, by: 2026-04-01.
- Metric: Deterministic state output, baseline: 0%, target: 100% identical `state.json` for identical input TSF bytes, by: 2026-04-01.

---

## 2. Determinism Contract (Mandatory)

### 2.1 No Defaults Policy
- This repo forbids implicit defaults.
- Any behavior that could vary between agents (ordering, time format, error codes) MUST be explicitly specified.

### 2.2 Implementability Rule
A feature is implementable only if it includes:
- acceptance criteria that are testable,
- explicit edge-case behavior,
- explicit error behavior (if applicable),
- explicit verification requirements.

If any of these are missing, implementation is BLOCKED.

### 2.3 Observable Behavior Requirements
For each externally observable interface (API, UI flow):
- Inputs (fields, constraints)
- Outputs (fields, constraints)
- Errors (codes/messages/payloads) if applicable
- Ordering rules if applicable
- Time rules (format + timezone) if applicable
- Verification proof requirements

**Global time rule:** all timestamps persisted to disk MUST be RFC3339 UTC (e.g., `2026-02-10T22:11:18Z`).

---

## 3. Blocking Questions (Mandatory)

(no blocking questions)

---

## 4. Decisions (Mandatory)

> Purpose: capture durable choices so future agents do not re-decide them.  
> Rule: if a choice affects behavior, interfaces, constraints, or the plan, it must be recorded here.

### 4.1 Decision Protocol (Non‑Negotiable)
- New decisions MUST be added as a new `D-0000X` record.
- Any substantive change to an existing decision MUST:
  1) create a new amendment (`/amendments/AMD-00001-<slug>.vX.Y.Z.md`)
  2) update the relevant decision record to reflect the new truth
  3) update the Decision Index “Last Updated (AMD or Initial Spec)”
- Decisions MUST NOT be implied.
- If a decision is `Proposed` or unresolved, implementation is BLOCKED.
- For the initial spec before any amendments exist, set Introduced/Updated to `Initial Spec`.

### 4.2 Decision Index
| ID | Title | Status | Introduced (AMD or Initial Spec) | Last Updated (AMD or Initial Spec) |
|----|-------|--------|-----------------------------------|------------------------------------|
| D-00001 | Platform + Execution Model | Active | Initial Spec | Initial Spec |
| D-00002 | Storage Model (JSON-only) | Active | Initial Spec | Initial Spec |
| D-00003 | TSF Handling + Retention | Active | Initial Spec | Initial Spec |
| D-00004 | Ingest Audit + Commit Rules | Active | Initial Spec | Initial Spec |
| D-00005 | Dedupe Fingerprint | Active | Initial Spec | Initial Spec |
| D-00006 | Parsing Strategy + Fallback Order | Active | Initial Spec | Initial Spec |
| D-00007 | Topology Inference Rules | Active | Initial Spec | Initial Spec |
| D-00008 | Flow Trace Scope | Active | Initial Spec | Initial Spec |
| D-00009 | RMA / Serial Replacement Strategy | Active | Initial Spec | Initial Spec |
| D-00010 | API Surface (Localhost) | Active | Initial Spec | Initial Spec |
| D-00011 | State Schema + Canonicalization | Active | Initial Spec | Initial Spec |
| D-00012 | Batch Ingest Ordering + Concurrency | Active | Initial Spec | Initial Spec |

### 4.3 Decision Records

#### D-00001: Platform + Execution Model
- Status: Active
- Context: The app must be local, resource-efficient, and parse large `.tgz` quickly.
- Options considered:
  - Option A: Go backend + browser UI
  - Option B: Node backend + browser UI
- Decision: Use a Go backend that serves a local web UI.
- Consequences:
  - Single local server process (plus OS browser process).
  - Streaming archive parsing and bounded memory use.
- Enforcement:
  - MUST: bind server to `127.0.0.1` only.
  - MUST NOT: expose the server on LAN interfaces by default.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00002: Storage Model (JSON-only)
- Status: Active
- Context: User prefers JSON instead of SQLite.
- Options considered:
  - Option A: SQLite
  - Option B: JSON snapshots + NDJSON logs
- Decision: Use `state.json` + `intro.md` + append-only `ingest.ndjson` and `commits.ndjson`.
- Consequences:
  - Atomic write requirements to prevent corruption.
- Enforcement:
  - MUST: write `state.json` via `*.tmp` + fsync + atomic rename.
  - MUST: keep `state.json.bak` as last-known-good.
  - MUST: write NDJSON with one JSON object per line, newline terminated.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00003: TSF Handling + Retention
- Status: Active
- Context: TSF must never be retained.
- Options considered:
  - Option A: retain TSFs for reprocessing
  - Option B: discard TSF bytes after parsing
- Decision: Always discard TSF bytes after ingest completes.
- Consequences:
  - Re-ingest requires the user to select the TSF again.
- Enforcement:
  - MUST NOT: persist uploaded TSF bytes to disk.
  - MUST: if an OS/temp file is unavoidable for streaming, it MUST be deleted before ingest returns `success|duplicate|no_change|error`.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00004: Ingest Audit + Commit Rules
- Status: Active
- Context: Git-like history is desired, but only when state changes.
- Options considered:
  - Option A: commit for every ingest
  - Option B: commit only when state changes
- Decision:
  - Always write an ingest log entry (success/failure/duplicate/no-change).
  - Write a commit entry only when canonical `state.json` changes.
- Consequences:
  - Auditability without noise in changelog.
- Enforcement:
  - MUST: include total runtime (`duration_ms_total`) in ingest log.
  - MUST: include stage timings (`duration_ms_by_stage`) in ingest log.
  - MUST: include `status` in ingest log.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00005: Dedupe Fingerprint
- Status: Active
- Context: TSFs may be renamed; need deterministic dedupe.
- Options considered:
  - Option A: Use uploaded archive SHA-256
  - Option B: Use tuple (serial, filename, capture time)
- Decision: Primary dedupe key is SHA-256 of the original uploaded archive bytes.
- Consequences:
  - Identical bytes => duplicate.
- Enforcement:
  - MUST: compute sha256 while streaming upload.
  - MUST: store `fingerprint_sha256` in ingest log.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00006: Parsing Strategy + Fallback Order
- Status: Active
- Context: Must support all PAN-OS versions and inconsistent TSF contents.
- Options considered:
  - Option A: hardcode fixed file names
  - Option B: pattern-based discovery + robust fallback
- Decision:
  - Discover sources using patterns (not fixed names).
  - For each field: runtime CLI section (preferred) → Panorama merged config (if present) → local running config → `not_found`.
- Consequences:
  - Parsing is resilient to missing sources.
- Enforcement:
  - MUST: attempt to locate CLI aggregate output at `tmp/cli/techsupport_*.txt`; fallback to any large file containing section headers like `> show ...`.
  - MUST: attempt to locate config XML at `**/saved-configs/running-config.xml` and `**/saved-configs/techsupport-saved-currcfg.xml`.
  - MUST: attempt to locate Panorama merged config at `**/panorama_pushed/mergesp.xml` (or other `*push*.xml` fallback).
  - MUST: record `not_found` explicitly for required fields; MUST NOT omit required keys.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00007: Topology Inference Rules
- Status: Active
- Context: Inference must always be on. Do not store confidence, but store reasoning.
- Options considered:
  - Option A: inference opt-in
  - Option B: inference always on
- Decision: Inference is always on.
- Consequences:
  - Ambiguous edges are still produced with explicit evidence.
- Enforcement:
  - MUST: ignore default routes (`0.0.0.0/0`) for adjacency inference.
  - MUST: store route reasoning for each route and for inferred links.
  - MUST NOT: store numeric confidence.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00008: Flow Trace Scope
- Status: Active
- Context: MVP flow trace ignores middle hops and focuses on firewalls.
- Options considered:
  - Option A: include intermediate L3/L2 hops
  - Option B: firewall-only hop list
- Decision: Flow trace outputs firewall-only hops with ingress/egress zone reasoning.
- Consequences:
  - Simpler algorithm and output.
- Enforcement:
  - MUST NOT: evaluate security policy rules in MVP.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00009: RMA / Serial Replacement Strategy
- Status: Active
- Context: Must track hostname changes by serial and handle RMAs (serial replacement).
- Options considered:
  - Option A: treat new serial as new device always
  - Option B: maintain logical device identity with serial history
- Decision:
  - Each device in the environment has `logical_device_id` (stable UUID).
  - Each ingest has `physical_serial` extracted from TSF.
  - Hostname changes are tracked under the same logical device when serial matches.
  - RMA heuristic: if a new serial appears and it matches exactly one existing logical device by (`hostname` OR `mgmt_ip`), link it as replacement; else create a new logical device and record an `rma_candidates[]` entry for review.
- Consequences:
  - Automatic linkage when obvious; safe fallback when ambiguous.
- Enforcement:
  - MUST: preserve per-serial history in state.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00010: API Surface (Localhost)
- Status: Active
- Context: Browser UI needs deterministic interfaces.
- Options considered:
  - Option A: REST
  - Option B: JSON-RPC
- Decision: Use REST JSON over HTTP.
- Consequences:
  - Simple integration.
- Enforcement:
  - MUST: return structured error codes and messages (see §5.3).
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00011: State Schema + Canonicalization
- Status: Active
- Context: Commits depend on stable diffing.
- Options considered:
  - Option A: compare raw JSON text
  - Option B: compare canonicalized JSON
- Decision: State changes are detected by comparing canonicalized JSON (stable key ordering + stable array ordering rules).
- Consequences:
  - Deterministic diffs across runs.
- Enforcement:
  - MUST: sort object keys lexicographically at write time.
  - MUST: arrays MUST have explicit sort keys (defined in schema) or preserve input order if the array is inherently ordered.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

#### D-00012: Batch Ingest Ordering + Concurrency
- Status: Active
- Context: Resource efficiency and deterministic ordering.
- Options considered:
  - Option A: parallel ingest
  - Option B: sequential ingest
- Decision: Batch ingest is sequential; UI processes files sorted ascending by filename.
- Consequences:
  - Lower peak resource use; deterministic ordering.
- Enforcement:
  - MUST NOT: run multiple TSF ingests concurrently within the same environment in MVP.
- Introduced by: Initial Spec
- Updated by: Initial Spec
- Superseded by: INIT

---

## 5. Requirements (Mandatory)

### 5.1 Feature Inventory (Hierarchical)
- F1: Environment management
  - F1.1: Create environment
  - F1.2: List environments
  - F1.3: Delete environment (soft delete)
- F2: TSF ingest
  - F2.1: Ingest single `.tgz`
  - F2.2: Ingest directory of `.tgz` (batch)
  - F2.3: Staged progress reporting
  - F2.4: Ingest audit log (`ingest.ndjson`) with status + timings
- F3: State persistence
  - F3.1: Write/update `state.json`
  - F3.2: Generate/update `intro.md`
- F4: Changelog/commit history
  - F4.1: Write `commits.ndjson` only on state change
  - F4.2: Commit detail view (diff summary)
- F5: Topology inference
  - F5.1: Derive zones/subnets/routes topology graph
  - F5.2: Infer firewall adjacency from shared routed subnets
- F6: Flow trace + Mermaid
  - F6.1: Trace firewall-only hop path for src/dst IP
  - F6.2: Render Mermaid diagrams and provide raw Mermaid source

### 5.2 User Flows (Step-by-Step)

#### Flow A: Create environment
1. User clicks "Create Environment".
2. User inputs `name` (required) and `description` (optional).
3. UI calls `POST /api/environments`.
Expected outcome:
- Environment appears in list.

#### Flow B: Ingest a TSF (single)
1. From Environment list, user clicks "Ingest".
2. User selects a `.tgz` file.
3. UI uploads file to `POST /api/environments/{env_id}/ingests`.
4. UI shows staged progress until completion.
Expected outcome:
- `ingest.ndjson` gains one entry.
- If state changed: `commits.ndjson` gains one entry and dashboard updates.

#### Flow C: Ingest a directory of TSFs (batch)
1. From Environment list, user clicks "Ingest".
2. User selects a folder of `.tgz` files.
3. UI uploads the files in deterministic order (sorted ascending by filename).
4. UI shows per-file staged progress with overall rollup.
Expected outcome:
- Each file produces an ingest entry.
- Commits produced only when the environment state changes.

#### Flow D: View environment state dashboard
1. User opens an Environment.
2. UI calls `GET /api/environments/{env_id}/state`.
Expected outcome:
- UI renders current environment summary and topology overview.

#### Flow E: View environment changelog
1. User opens "Changelog" within an Environment.
2. UI calls `GET /api/environments/{env_id}/commits`.
Expected outcome:
- UI renders a git-like commit list.

#### Flow F: Trace a flow
1. User opens "Flow Trace" within an Environment.
2. User inputs `src_ip` and `dst_ip`.
3. UI calls `POST /api/environments/{env_id}/flow-trace`.
Expected outcome:
- UI shows firewall hop list and Mermaid diagram.

### 5.3 Acceptance Criteria (Testable, Non‑Ambiguous)

#### F1: Environment management
- AC-F1-1: Given no environments exist, when the user creates an environment with a unique name, then the environment is persisted and listed by `GET /api/environments`.
- AC-F1-2: If the user attempts to create an environment with an empty name, then the API returns `400` with `{ code: "ERR_ENV_NAME_REQUIRED" }`.
- AC-F1-3: If the user deletes an environment, then it is soft-deleted (not listed by default) and its on-disk folder is moved to `trash/`.

#### F2: TSF ingest
**Ordering rule (batch):** When ingesting a directory, `.tgz` files MUST be processed sequentially, sorted ascending by filename (bytewise lexicographic).

- AC-F2-1 (success): Given a valid `.tgz`, when ingested, then:
  - an ingest log entry is appended with `status="success"`,
  - `duration_ms_total > 0`,
  - `duration_ms_by_stage` includes all stages,
  - and `state.json` is updated.
- AC-F2-2 (duplicate): Given a `.tgz` with a sha256 already seen for that environment, when ingested, then:
  - an ingest log entry is appended with `status="duplicate"`,
  - no commit is appended,
  - and `state.json` is unchanged.
- AC-F2-3 (no-change): Given a `.tgz` that parses successfully but produces identical canonical state to the prior `state.json`, when ingested, then:
  - an ingest log entry is appended with `status="no_change"`,
  - no commit is appended,
  - and `state.json` is unchanged.
- AC-F2-4 (error): If ingest fails at any stage, then:
  - an ingest log entry is appended with `status="error"`,
  - `error.stage` is set to one of: `receive|scan|identify|extract|derive|diff|persist`,
  - `error.code` is one of the defined codes below,
  - and `state.json` remains unchanged.
- AC-F2-5 (Panorama TSF): Given a Panorama TSF, when ingested, then `state.json` includes Panorama inventory fields and no firewall-only assumptions are applied.

**Error codes (ingest):**
- `ERR_INVALID_ARCHIVE`: not a readable gzip/tar
- `ERR_ARCHIVE_SCAN_FAILED`: tar listing failed
- `ERR_REQUIRED_SOURCE_MISSING`: no CLI text and no config XML found
- `ERR_PARSE_CLI_FAILED`: CLI parsing failed
- `ERR_PARSE_XML_FAILED`: XML parsing failed
- `ERR_DERIVE_TOPOLOGY_FAILED`: topology inference failed
- `ERR_PERSIST_FAILED`: unable to write state/logs atomically

#### F3: State persistence (`state.json` + `intro.md`)
- AC-F3-1: When `state.json` is written, it MUST be valid JSON and include `schema_version` and `generated_at`.
- AC-F3-2: `intro.md` MUST be regenerated on every successful ingest (including `no_change`), and MUST include a short summary plus pointers to where key data lives in `state.json`.
- AC-F3-3: If persistence fails, `state.json` MUST remain valid and revert to prior content via `state.json.bak`.

#### F4: Changelog/commit history
- AC-F4-1: Given a successful ingest that changes state, then exactly one commit entry is appended to `commits.ndjson` with:
  - `commit_id`, `ingest_id`, timestamp, source summary
  - `change_paths[]` containing JSON Pointer paths changed
  - `change_summary` human-readable bullets.
- AC-F4-2: Given `duplicate` or `no_change` status, then no commit entry is appended.

#### F5: Topology inference
- AC-F5-1: For each firewall, the app MUST extract interfaces, zones, virtual routers, and routes (runtime and/or config) and store them in `state.json`.
- AC-F5-2: Inferred adjacency MUST be produced when two firewalls have non-default routes to the same destination CIDR (exact match after canonicalization).
- AC-F5-3: Default route (`0.0.0.0/0`) MUST NOT create inferred adjacency.
- AC-F5-4: Each inferred adjacency edge MUST include `evidence[]` with route type reasoning for both firewalls.

#### F6: Flow trace + Mermaid
- AC-F6-1: Given `src_ip` and `dst_ip`, the app MUST output a firewall-only hop list.
- AC-F6-2: If no source firewall can be determined, the API MUST return `404` with `{ code: "ERR_FLOW_SRC_NOT_FOUND" }`.
- AC-F6-3: If a path cannot be inferred, the API MUST return `404` with `{ code: "ERR_FLOW_PATH_NOT_FOUND" }`.
- AC-F6-4: Flow trace response MUST include Mermaid source text and a stable hop list.

### 5.4 Constraints (Security/Privacy/Performance/Operations)
- MUST: operate fully offline; MUST NOT transmit TSF data to any network destination.
- MUST: bind server to `127.0.0.1` only.
- MUST: sanitize and cap error messages to avoid leaking large TSF content to UI.
- MUST: stream `.tgz` parsing; MUST NOT extract entire archive to disk.
- MUST: never retain TSFs after ingest.
- MUST: write all logs as append-only and fsync.
- MUST: cap per-ingest memory usage by using streaming readers and bounded buffers.

---

## 6. Verification Strategy (Mandatory)

### 6.1 Required Verification Proof
For each release candidate:
- Verify ingest success:
  - Ingest a known-good firewall TSF and Panorama TSF.
  - Confirm `ingest.ndjson` appended with `status=success` and timing fields.
  - Confirm `state.json` updated and valid JSON.
- Verify duplicate:
  - Ingest the same TSF twice.
  - Confirm second ingest is `duplicate`, no commit appended.
- Verify no-change:
  - Ingest a TSF expected to yield identical state.
  - Confirm `no_change`, no commit appended.
- Verify flow trace:
  - Use a known environment where a shared subnet route exists.
  - Confirm hop list includes only firewalls and Mermaid source renders.

Artifacts of proof to record in changelog (outside this run folder):
- `state.json` checksum for each verification run
- `ingest.ndjson` line excerpt showing timing fields
- `commits.ndjson` excerpt showing commits-only-on-change

### 6.2 Definition of Done (Global)
A change is “Done” only if:
- acceptance criteria satisfied,
- verification executed and recorded,
- changelog entry created,
- any required amendments created + incorporated.

---

## 7. Glossary
- TSF: Palo Alto Networks tech support file bundle (`.tgz`).
- Environment: A named container of devices + derived topology state.
- Ingest: A single attempt to process one TSF and update environment state.
- Commit: A changelog entry produced only when derived state changes.
- Logical device: Stable identity within an environment that may contain multiple physical serials over time (RMA).
- Physical serial: Hardware serial extracted from TSF.
- CLI aggregate file: Text file in TSF containing multiple `> show ...` sections.
- Default route: `0.0.0.0/0`.

---

## 8. Spec Ambiguity Audit

- [x] Every feature has acceptance criteria.
- [x] Every acceptance criterion is verifiable.
- [x] Error behaviors are defined where applicable.
- [x] Edge cases are defined where applicable.
- [x] Any missing information is in Blocking Questions.
- [x] Decisions that affect behavior are recorded in Decisions.
- [x] No contradictions exist across sections.

List remaining ambiguities (must be empty for determinism):
- (none)
