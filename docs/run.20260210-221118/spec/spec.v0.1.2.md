---
doc_type: spec
project: "NetSec StateKit"
spec_id: "SPEC-00001"
version: "0.1.2"
owners:
  - "Chris Marks (Product Owner)"
stakeholders:
  - "Network Security Operators (Users)"
  - "AI Agent Consumers (Users)"
review_rounds_completed: 3
last_updated: "2026-02-10"
change_control:
  amendment_required_for_substantive_change: true
  metadata_change_allowed_without_amendment: true
related:
  plan: ""
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
- Outcome: FAIL
- Blockers count: 11
- Summary of issues found: Missing deterministic storage layout, schemas (state/logs), API contract, and a normative TSF extraction field set; batch failure semantics and canonical diff rules were underspecified.
- Amendment link (if substantive changes required): (none)

- Round ID: SR-00002
- Date: 2026-02-10
- Reviewers (LLMs + roles):
  - Reviewer A: Ambiguity Hunter
  - Reviewer B: Edge Cases & Failure Modes
  - Reviewer C: Security/Privacy/Abuse Modes
  - Reviewer D: Testability & Verification
- Outcome: PASS
- Blockers count: 0
- Summary of changes applied: Incorporated answered design decisions (storage root, ephemeral port, batch error policy, CIDR overlap inference, RMA confirmation UX), and added deterministic on-disk schemas + a normative TSF extraction appendix.
- Amendment link (if substantive changes required): `./amendments/AMD-00001-clarifications.v0.1.1.md`


- Round ID: SR-00003
- Date: 2026-02-10
- Reviewers (LLMs + roles):
  - Reviewer A: Ambiguity Hunter
  - Reviewer B: Edge Cases & Failure Modes
  - Reviewer C: Security/Privacy/Abuse Modes
  - Reviewer D: Testability & Verification
- Outcome: PASS
- Blockers count: 0
- Summary of changes applied: Added deterministic `meta.json`, `state.json`, and `intro.md` schemas/formatting (for both humans and AI agents), and aligned canonicalization rules to those schemas.
- Amendment link (if substantive changes required): `./amendments/AMD-00002-state-schema.v0.1.2.md`

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
NetSec StateKit (`netsec-sk`) is a local macOS application that builds and maintains a JSON representation of the "state" of a network security deployment by ingesting Palo Alto Networks tech support bundles (TSF `.tgz`) from firewalls and/or Panorama. It is designed so humans can quickly understand an environment (prod/lab/home/customerA/...) and AI agents can quickly gain reliable context from `state.json` + `intro.md`.

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

### 4.2 Decision Index
| ID | Title | Status | Introduced (AMD or Initial Spec) | Last Updated (AMD or Initial Spec) |
|----|-------|--------|-----------------------------------|------------------------------------|
| D-00001 | Platform + Execution Model | Active | Initial Spec | AMD-00001 |
| D-00002 | Storage Model (JSON-only) | Active | Initial Spec | AMD-00001 |
| D-00003 | TSF Handling + Retention | Active | Initial Spec | Initial Spec |
| D-00004 | Ingest Audit + Commit Rules | Active | Initial Spec | AMD-00001 |
| D-00005 | Dedupe Fingerprint | Active | Initial Spec | Initial Spec |
| D-00006 | Parsing Strategy + Field Set (Normative Appendix) | Active | Initial Spec | AMD-00001 |
| D-00007 | Topology Inference Rules (CIDR overlap) | Active | Initial Spec | AMD-00001 |
| D-00008 | Flow Trace Scope | Active | Initial Spec | AMD-00001 |
| D-00009 | RMA / Serial Replacement Strategy (User confirmation) | Active | Initial Spec | AMD-00001 |
| D-00010 | API Surface (Localhost, ephemeral port) | Active | Initial Spec | AMD-00001 |
| D-00011 | State Schema + Canonicalization | Active | Initial Spec | AMD-00002 |
| D-00012 | Batch Ingest Ordering + Concurrency | Active | Initial Spec | AMD-00001 |

### 4.3 Decision Records

#### D-00001: Platform + Execution Model
- Status: Active
- Decision: Use a Go backend that serves a local web UI.
- Enforcement:
  - MUST: bind server to `127.0.0.1` only.
  - MUST: choose an ephemeral available TCP port on startup.
  - MUST: write the chosen URL to `~/.netsec-sk/runtime/server.json` (schema in §9.5).
  - MUST: print `NETSEC_SK_URL=<url>` to stdout once server is ready.
  - SHOULD: attempt to open the default browser to `<url>` (best-effort; failure does not block startup).
- Updated by: AMD-00001

#### D-00002: Storage Model (JSON-only)
- Status: Active
- Decision: Use JSON snapshots + NDJSON logs under a deterministic root folder.
- Enforcement:
  - Storage root MUST be `~/.netsec-sk/`.
  - Environments MUST be stored under `~/.netsec-sk/environments/<env_id>/` (see §9.1).
  - MUST: write `state.json` via `*.tmp` + fsync + atomic rename.
  - MUST: keep `state.json.bak` as last-known-good.
  - MUST: write NDJSON with one JSON object per line, newline terminated.
- Updated by: AMD-00001

#### D-00003: TSF Handling + Retention
- Status: Active
- Decision: Always discard TSF bytes after ingest completes.
- Enforcement:
  - MUST NOT: persist uploaded TSF bytes to disk.
  - MAY: persist extracted intermediate JSON (not TSF bytes) while an ingest is awaiting RMA confirmation (§5.2 Flow C / §9.4).
  - MUST: delete intermediate ingest files at completion or after TTL expiry (24h).
- Updated by: Initial Spec

#### D-00004: Ingest Audit + Commit Rules
- Status: Active
- Decision:
  - Always append an ingest log entry (success/failure/duplicate/no-change).
  - Append a commit entry only when canonical `state.json` changes.
- Enforcement:
  - MUST: include total wall-clock runtime (`duration_ms_total`) in ingest log.
  - MUST: include compute-only runtime excluding user wait (`duration_ms_compute`) in ingest log.
  - MUST: include stage timings (`duration_ms_by_stage`) in ingest log.
  - MUST: include `status` in ingest log.
- Updated by: AMD-00001

#### D-00005: Dedupe Fingerprint
- Status: Active
- Decision: Primary dedupe key is SHA-256 of the original uploaded archive bytes (per Appendix A).
- Enforcement:
  - MUST: compute sha256 while streaming upload.
  - MUST: store `fingerprint_sha256` in ingest log.
- Updated by: Initial Spec

#### D-00006: Parsing Strategy + Field Set (Normative Appendix)
- Status: Active
- Decision:
  - Discover sources using patterns (not fixed names).
  - For each field: runtime CLI section (preferred) → Panorama merged config (if present) → local saved configs → `not_found`.
  - **MVP field set and extraction rules are defined normatively in Appendix A**: `spec/appendices/PALOALTO_DATA_MAPPING.MVP.v1.md`.
- Enforcement:
  - MUST: implement Appendix A extraction map (Output fields + Extraction map sections).
  - MUST: record `not_found` explicitly for required fields; MUST NOT omit required keys.
- Updated by: AMD-00001

#### D-00007: Topology Inference Rules (CIDR overlap)
- Status: Active
- Decision: Inference is always on and uses CIDR overlap (not only exact-match).
- Enforcement:
  - MUST: ignore default routes (`0.0.0.0/0`) for adjacency inference.
  - MUST: treat any CIDR overlap as potential adjacency evidence:
    - overlap is true iff CIDR A intersects CIDR B (including containment).
  - MUST: for an inferred firewall↔firewall edge, store `evidence[]` listing the overlapping CIDR pairs and route reasoning on both sides.
  - MUST: choose the most specific overlapping CIDR pair(s) (highest prefix length) as primary evidence; include all ties.
  - MUST NOT: store numeric confidence.
- Updated by: AMD-00001

#### D-00008: Flow Trace Scope
- Status: Active
- Decision: Flow trace outputs firewall-only hops with ingress/egress zone reasoning.
- Enforcement:
  - MUST: determine source firewall deterministically:
    1) Prefer firewall where `src_ip` is within any connected/interface subnet.
    2) Else prefer firewall where `src_ip` matches a longest-prefix route.
    3) Tie-breaker: lexicographic by `logical_device_id`.
  - MUST NOT: evaluate security policy rules in MVP.
- Updated by: AMD-00001

#### D-00009: RMA / Serial Replacement Strategy (User confirmation)
- Status: Active
- Decision:
  - Each device in the environment has `logical_device_id` (stable UUID).
  - Hostname changes are tracked under the same logical device when serial matches.
  - If an ingest yields a **new serial** and the **hostname matches** one or more existing logical devices, the ingest MUST prompt the user to confirm whether this is an RMA replacement.
- Enforcement:
  - The backend MUST enter `awaiting_user` state and expose candidates to UI (§5.2 / §9.4 / §10.3).
  - User choices:
    - `link_replacement` (select target logical_device_id) → update serial history, set new serial current.
    - `treat_as_new_device` → create new logical device with the new serial.
  - If the user aborts (explicit cancel or timeout), ingest MUST finalize as `status="error"` with `error.code="ERR_USER_ABORTED"`, with no state change.
- Updated by: AMD-00001

#### D-00010: API Surface (Localhost, ephemeral port)
- Status: Active
- Decision: Use REST JSON over HTTP, served from the same localhost origin as the UI.
- Enforcement:
  - MUST: provide `/api/health` returning `{ version, started_at, url }`.
  - MUST: return structured error codes and messages (see §10.4).
- Updated by: AMD-00001

#### D-00011: State Schema + Canonicalization
- Status: Active
- Decision: State changes are detected by comparing canonicalized JSON (stable key ordering + stable array ordering rules).
- Enforcement:
  - MUST: write `state.json` with deterministic key ordering (lexicographic).
  - MUST: define array ordering rules (schema in §9.7).
- Updated by: AMD-00001

#### D-00012: Batch Ingest Ordering + Concurrency
- Status: Active
- Decision: Batch ingest is sequential; UI processes files sorted ascending by filename; failures do not stop subsequent files.
- Enforcement:
  - MUST NOT: run multiple TSF ingests concurrently within the same environment in MVP.
  - MUST: continue ingesting remaining files if one file fails; each file gets its own ingest log entry.
- Updated by: AMD-00001

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
  - F2.5: RMA confirmation prompt when hostname matches but serial differs
- F3: State persistence
  - F3.1: Write/update `state.json`
  - F3.2: Generate/update `intro.md`
- F4: Changelog/commit history
  - F4.1: Write `commits.ndjson` only on state change
  - F4.2: Commit detail view (diff summary)
- F5: Topology inference
  - F5.1: Derive zones/subnets/routes topology graph
  - F5.2: Infer firewall adjacency from shared routed subnets (CIDR overlap)
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
- On disk, an environment folder is created: `~/.netsec-sk/environments/<env_id>/` with empty logs.

#### Flow B: Ingest a TSF (single)
1. From Environment list, user clicks "Ingest".
2. User selects a `.tgz` file.
3. UI uploads file to `POST /api/environments/{env_id}/ingests`.
4. UI shows staged progress until completion (poll `GET /api/ingests/{ingest_id}`).
5. If RMA prompt is required, UI shows confirmation dialog and submits decision (Flow C2).
Expected outcome:
- `ingest.ndjson` gains one final entry.
- If state changed: `commits.ndjson` gains one entry and dashboard updates.

#### Flow C: Ingest a directory of TSFs (batch)
1. From Environment list, user clicks "Ingest".
2. User selects a folder of `.tgz` files.
3. UI sorts files ascending by `filename` (bytewise lexicographic) and uploads them sequentially as independent ingests.
4. UI shows per-file staged progress with overall rollup.
Expected outcome:
- Each file produces exactly one ingest log entry (success|duplicate|no_change|error).
- Failures do not stop the batch; remaining files continue.

#### Flow C2: RMA confirmation (prompt)
Trigger: an ingest identifies `hostname` matching existing logical device(s) but `serial` differs.

1. Backend enters `awaiting_user` status for that ingest and returns candidate logical devices (see §10.3).
2. UI prompts:
   - "This TSF appears to be a replacement device (RMA). Link this new serial to an existing device?"
   - Options:
     - Link replacement (select target logical device)
     - Treat as new device
     - Cancel ingest
3. UI submits `POST /api/ingests/{ingest_id}/rma-decision`.
Expected outcome:
- If linked: environment device serial history updates and ingest continues to completion.
- If new device: new logical device created and ingest continues.
- If canceled: ingest finalizes `error` with `ERR_USER_ABORTED`.

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
- AC-F1-3: If the user deletes an environment, then it is soft-deleted (not listed by default) and its on-disk folder is moved to `~/.netsec-sk/trash/`.

#### F2: TSF ingest
**Ordering rule (batch):** When ingesting a directory, `.tgz` files MUST be processed sequentially, sorted ascending by filename (bytewise lexicographic). If one file fails, the batch continues with the next file.

- AC-F2-1 (success): Given a valid `.tgz`, when ingested, then:
  - an ingest log entry is appended with `status="success"`,
  - `duration_ms_total > 0`,
  - `duration_ms_compute > 0`,
  - `duration_ms_by_stage` includes all stages observed,
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
  - `error.stage` is set to one of: `receive|scan|identify|extract|derive|diff|persist|awaiting_user`,
  - `error.code` is one of the defined codes below,
  - and `state.json` remains unchanged.
- AC-F2-5 (batch continue): Given a folder with 3 TSFs where the 2nd is invalid, when ingested, then:
  - ingest log contains 3 entries,
  - the 2nd entry has `status="error"` and the 3rd still completes (any of success/duplicate/no_change/error),
  - and commits are appended only for the subset that change state.
- AC-F2-6 (RMA prompt): Given a TSF whose extracted hostname matches an existing logical device hostname but serial differs, then the ingest enters `awaiting_user` and does not mutate state until a decision is posted; after decision, ingest completes and logs `rma.prompted=true` and `rma.decision` in the final ingest log entry.
- AC-F2-7 (Panorama TSF): Given a Panorama TSF, when ingested, then `state.json` includes Panorama inventory fields and no firewall-only assumptions are applied to topology unless the environment already contains firewall TSFs.

**Error codes (ingest):**
- `ERR_INVALID_ARCHIVE`: not a readable gzip/tar
- `ERR_ARCHIVE_SCAN_FAILED`: tar listing failed
- `ERR_REQUIRED_SOURCE_MISSING`: no CLI text and no config XML found
- `ERR_PARSE_CLI_FAILED`: CLI parsing failed
- `ERR_PARSE_XML_FAILED`: XML parsing failed
- `ERR_DERIVE_TOPOLOGY_FAILED`: topology inference failed
- `ERR_PERSIST_FAILED`: unable to write state/logs atomically
- `ERR_USER_ABORTED`: user canceled or timed out during confirmation

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
- AC-F5-2: Inferred adjacency MUST be produced when two firewalls have non-default routed CIDRs that overlap (CIDR intersection is non-empty).
- AC-F5-3: Default route (`0.0.0.0/0`) MUST NOT create inferred adjacency.
- AC-F5-4: Each inferred adjacency edge MUST include `evidence[]` with route type reasoning for both firewalls and the overlapping CIDR(s).

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

## 6. Algorithms (Deterministic)

### 6.1 TSF ingest stage model (timing + progress)
Stages (ordered):
1. `receive` (accept upload stream, compute sha256)
2. `scan` (enumerate tar members, identify candidate member paths)
3. `identify` (extract device identity: hostname/model/serial/version/mgmt IP + device type)
4. `extract` (extract required fields per Appendix A)
5. `derive` (derive topology graph + inferred adjacencies)
6. `diff` (canonicalize and compute diff vs prior state)
7. `persist` (atomic writes: logs, state.json, intro.md, commit if applicable)

If RMA prompt is required, insert stage:
- `awaiting_user` (pause) between `identify` and `extract` OR between `extract` and `diff` (implementation MUST pause before any state mutation).

Timing capture:
- `duration_ms_by_stage` MUST include each stage the ingest entered.
- `duration_ms_compute` is sum of non-`awaiting_user` stages.
- `duration_ms_total` includes all time from ingest start to final status, including any time in `awaiting_user`.

### 6.2 CIDR overlap definition
CIDR overlap between networks A and B is true iff `A ∩ B ≠ ∅`. This includes:
- A contains B
- B contains A
- partial overlap (rare for CIDRs; effectively containment due to CIDR alignment, but the definition remains intersection-based)

### 6.3 Adjacency inference algorithm (firewall-only)
Inputs: per firewall, a set of non-default routed CIDRs with route records and zone/interface mapping.

Algorithm:
1. Build each firewall’s `routed_cidrs` set from runtime routes if present else config routes; exclude `0.0.0.0/0`.
2. For each unordered firewall pair (FWi, FWj):
   - compute overlapping CIDR pairs between `routed_cidrs_i` and `routed_cidrs_j` (using §6.2).
   - if none, no edge.
   - if overlaps exist, create one inferred edge with `evidence[]`:
     - select the most specific overlapping pair(s) (highest prefix length; include ties).
     - for each evidence item include:
       - `cidr_i`, `cidr_j`
       - `fw_i` route record (dest, vr, interface, zone, source_type, source_reason)
       - `fw_j` route record (...)
3. Store all inferred edges under `state.topology.inferred_adjacencies[]`, sorted lexicographically by `(fw_a.logical_device_id, fw_b.logical_device_id)`.

### 6.4 Flow trace algorithm (firewall-only)
Given `src_ip`, `dst_ip`, determine hop list:
1. Determine candidate source firewalls:
   - `connected_candidates`: firewalls where `src_ip` is contained in any connected/interface subnet.
   - if non-empty, pick the one with smallest lexicographic `logical_device_id`.
   - else `route_candidates`: firewalls where `src_ip` matches any route destination by longest-prefix; pick longest-prefix, tie-break lexicographic `logical_device_id`.
   - if none, return `ERR_FLOW_SRC_NOT_FOUND`.
2. On current firewall, determine egress zone for `dst_ip`:
   - find longest-prefix route match to `dst_ip` among non-default routes; if none, optionally use default route (`0.0.0.0/0`) as fallback and mark `used_default=true`.
3. If `dst_ip` is connected on current firewall, stop (destination reached).
4. Else find next firewall:
   - use `state.topology.inferred_adjacencies` to find an edge whose `fw_a==current` OR `fw_b==current` and whose evidence CIDRs overlap the selected route CIDR for dst.
   - if multiple, choose lexicographically by next firewall `logical_device_id`.
   - if none, return `ERR_FLOW_PATH_NOT_FOUND`.
5. Repeat with loop detection: if a firewall repeats, return `ERR_FLOW_PATH_NOT_FOUND` with `details.loop=true`.

---

## 7. Verification Strategy (Mandatory)

### 7.1 Required Verification Proof
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
- Verify batch continue:
  - Ingest folder with one corrupted `.tgz`.
  - Confirm remaining files still process.
- Verify RMA prompt:
  - Ingest TSF A (hostname X, serial S1).
  - Ingest TSF B (hostname X, serial S2).
  - Confirm ingest pauses `awaiting_user`; confirm linking choice changes serial history accordingly.
- Verify flow trace:
  - Use a known environment where overlapping CIDR routes exist across two firewalls.
  - Confirm hop list includes only firewalls and Mermaid source renders.

Artifacts of proof to record in changelog (outside this run folder):
- `state.json` checksum for each verification run
- `ingest.ndjson` line excerpt showing `duration_ms_total`, `duration_ms_compute`
- `commits.ndjson` excerpt showing commits-only-on-change

### 7.2 Definition of Done (Global)
A change is “Done” only if:
- acceptance criteria satisfied,
- verification executed and recorded,
- changelog entry created,
- any required amendments created + incorporated.

---

## 8. Glossary
- TSF: Palo Alto Networks tech support file bundle (`.tgz`).
- Environment: A named container of devices + derived topology state.
- Ingest: A single attempt to process one TSF and update environment state.
- Commit: A changelog entry produced only when derived state changes.
- Logical device: Stable identity within an environment that may contain multiple physical serials over time (RMA).
- Physical serial: Hardware serial extracted from TSF.
- CLI aggregate file: Text file in TSF containing multiple `> show ...` sections.
- Default route: `0.0.0.0/0`.

---

## 9. Data Layout & Schemas (Normative)

### 9.1 On-disk directory layout (deterministic)
Storage root: `~/.netsec-sk/`

```
~/.netsec-sk/
  environments/<env_id>/
    meta.json
    state.json
    state.json.bak
    intro.md
    ingest.ndjson
    commits.ndjson
  runtime/
    server.json
    ingests/<ingest_id>.json   # optional extracted intermediate payload while awaiting user
  trash/<env_id>/...           # soft-deleted env folders
```

### 9.2 `ingest.ndjson` schema (one line per ingest attempt; final only)
Each line MUST be a JSON object with:

Required:
- `ingest_id` (uuid string)
- `env_id` (uuid string)
- `started_at` (RFC3339 UTC)
- `finished_at` (RFC3339 UTC)
- `status` enum: `"success" | "duplicate" | "no_change" | "error"`
- `source`:
  - `mode` enum: `"file" | "batch"`
  - `filenames[]` (non-empty)
  - `user_label` (string, optional)
- `fingerprint_sha256` (hex string)
- `device`:
  - `device_type` enum: `"firewall" | "panorama" | "unknown"`
  - `serial` (string or `"not_found"`)
  - `hostname` (string or `"not_found"`)
- `duration_ms_total` (int >= 0)
- `duration_ms_compute` (int >= 0)
- `duration_ms_by_stage` (object mapping stage name to int >= 0)

Optional:
- `rma` (only when prompted or decided):
  - `prompted` (bool)
  - `candidates[]` (array of `{ logical_device_id, current_serial, current_hostname }`)
  - `decision` enum: `"link_replacement" | "treat_as_new_device" | "canceled"`
  - `linked_logical_device_id` (uuid, only if link)
- `result` (small summary):
  - `commit_id` (uuid, only if commit written)
  - `state_hash_after` (hex sha256 of canonical state.json, optional)
- `error` (only if status==error):
  - `stage` enum: `receive|scan|identify|extract|derive|diff|persist|awaiting_user`
  - `code` enum: from §5.3
  - `message` (string, max 512 chars)

### 9.3 `commits.ndjson` schema (one line per state change)
Required:
- `commit_id` (uuid)
- `env_id` (uuid)
- `ingest_id` (uuid)
- `timestamp` (RFC3339 UTC)
- `source_summary` (string)
- `change_summary[]` (array of human-readable bullet strings)
- `change_paths[]` (array of JSON Pointer strings, e.g. `/devices/logical/0/current/identity/hostname`)
- `state_hash_before` (hex sha256)
- `state_hash_after` (hex sha256)

### 9.4 `runtime/ingests/<ingest_id>.json` schema (optional)
If an ingest is awaiting user confirmation, backend MAY persist extracted intermediate payload:

Required:
- `ingest_id`, `env_id`, `started_at`
- `device_identity` (as parsed)
- `extracted_payload` (the would-be device record updates, without applying to state)
- `rma_candidates[]`

TTL:
- MUST be deleted when ingest completes.
- MUST be deleted if older than 24 hours on backend startup.

### 9.5 `runtime/server.json` schema
Written on backend startup; overwritten on each start.
Required:
- `url` (string, e.g. `http://127.0.0.1:51342`)
- `port` (int)
- `pid` (int)
- `started_at` (RFC3339 UTC)
- `version` (string, matches app version)

### 9.6 `environments/<env_id>/meta.json` schema
Required (single JSON object):
- `env_id` (uuid)
- `name` (string)
- `description` (string or empty string)
- `created_at` (RFC3339 UTC)
- `updated_at` (RFC3339 UTC)
- `soft_deleted` (bool)
- `soft_deleted_at` (RFC3339 UTC or empty string)

### 9.7 `environments/<env_id>/state.json` schema (MVP)
Top-level required keys:
- `schema_version` (string literal `"1.0.0"`)
- `generated_at` (RFC3339 UTC)
- `env`:
  - `env_id` (uuid)
  - `name` (string)
- `devices`:
  - `logical[]` (array; sorted by `logical_device_id` ascending)
- `topology`:
  - `inferred_adjacencies[]` (array; sorted by `(fw_a_logical_device_id, fw_b_logical_device_id)`)

#### 9.7.1 `devices.logical[]` object schema
Required:
- `logical_device_id` (uuid)
- `device_type` enum: `"firewall" | "panorama"`
- `serial_history[]` (array; sorted by `serial` ascending)
- `current` (object; last-observed snapshot)

`serial_history[]` entry:
- `serial` (string)
- `first_seen_ingest_id` (uuid)
- `last_seen_ingest_id` (uuid)
- `first_seen_at` (RFC3339 UTC)
- `last_seen_at` (RFC3339 UTC)

`current` required:
- `observed_at` (RFC3339 UTC) — timestamp of the ingest that produced `current`
- `source`:
  - `ingest_id` (uuid)
  - `fingerprint_sha256` (hex)
- `identity`:
  - `hostname` (string or `"not_found"`)
  - `model` (string or `"not_found"`)
  - `serial` (string or `"not_found"`)
  - `panos_version` (string or `"not_found"`)
  - `mgmt_ip` (string or `"not_found"`)
- `management`:
  - `management_type` enum: `"panorama-managed" | "cloud-managed" | "standalone" | "undetermined"`
  - `panorama_servers[]` (array of strings; sorted ascending; may be empty)
  - `cloud_mode` (string or `"not_found"`)
- `ha`:
  - `enabled` enum: `"enabled" | "disabled" | "unknown"`
  - `mode` (string or `"not_found"`)
  - `peer` (string or `"not_found"`)
- `licenses[]` (array; sorted by `feature` ascending)
- `cloud_logging_service_forwarding`:
  - `enabled` enum: `"enabled" | "disabled" | "unknown"`
  - `region` (string or `"not_found"`)
  - `enhanced_application_logging_enabled` enum: `"enabled" | "disabled" | "unknown"`
  - `source_path` (string or `"not_found"`)
- `network` (for `device_type="firewall"`; MUST be present for firewalls; MUST be present but MAY be empty for panoramas):
  - `interfaces[]` (array; sorted by `name` ascending)
  - `zones[]` (array; sorted by `name` ascending)
  - `routes_config[]` (array; sorted by `(vr, destination, nexthop, interface)` ascending)
  - `routes_runtime[]` (array; sorted by `(vr, destination, nexthop, interface)` ascending)

`licenses[]` entry required:
- `feature` (string)
- `status` enum: `"active" | "expired" | "unknown"`
- `expires` (string or `"not_found"`)  # keep raw TSF value
- `description` (string or `"not_found"`)

`network.interfaces[]` entry (MVP required keys):
- `name` (string)
- `type` (string or `"not_found"`)  # e.g., ethernet, ae, tunnel, loopback, vlan
- `layer3_units[]` (array; sorted by `name` ascending; may be empty)
  - each unit: `{ name, ip_cidrs[] }` where `ip_cidrs[]` sorted ascending

`network.zones[]` entry (MVP required keys):
- `name` (string)
- `type` (string or `"not_found"`)
- `members[]` (array of interface names; sorted ascending)

`routes_config[]` and `routes_runtime[]` route entry required:
- `vr` (string or `"not_found"`)
- `destination` (CIDR string)
- `nexthop` (string or `"not_found"`)
- `interface` (string or `"not_found"`)
- `metric` (string or `"not_found"`)
- `reason` enum: `"connected" | "static" | "bgp" | "ospf" | "rip" | "configured" | "unknown"`
- `source_type` enum: `"runtime" | "config"`
- `source_path` (string or `"not_found"`)

#### 9.7.2 Panorama-only fields
If `device_type="panorama"`, `current.panorama` MUST exist with:
- `managed_device_serials[]` (array; sorted)
- `device_groups[]` (array; sorted by `device_group_name`)
- `template_stacks[]` (array; sorted by `template_stack_name`)
- `templates[]` (array; sorted)

`device_groups[]` entry:
- `device_group_name` (string)
- `firewall_serials[]` (array; sorted)
- `reference_templates[]` (array; sorted)

`template_stacks[]` entry:
- `template_stack_name` (string)
- `firewall_serials[]` (array; sorted)
- `templates[]` (array; sorted)

#### 9.7.3 Canonicalization rules for commit/diff
- Objects: keys lexicographically sorted.
- Arrays: MUST be sorted per the rules above before writing `state.json`.
- Canonical hash: `state_hash_*` values in commits are SHA-256 of UTF-8 encoded canonical JSON text of `state.json` (no trailing whitespace, newline optional but consistent; MUST include trailing newline).

### 9.8 `environments/<env_id>/intro.md` format (MVP)
`intro.md` MUST be rewritten on every successful ingest (including `no_change`). It MUST include:

1. Header line: `# <environment name>`
2. A short paragraph: purpose + last generated timestamp.
3. "Quick facts" bullet list:
   - number of logical devices
   - number of firewalls vs panoramas
   - last ingest status + finished_at
4. "Where to look in state.json" section with bullet pointers:
   - devices list path: `/devices/logical`
   - inferred adjacencies path: `/topology/inferred_adjacencies`
   - per-device network inventory: `/devices/logical[i]/current/network`
5. "AI Agent notes" section (plain English) stating:
   - The file is a derived snapshot; consult `commits.ndjson` for history.
   - The ingest log is `ingest.ndjson` (attempts include dupes/errors).
   - No TSFs are retained; provenance is via fingerprints and ingest IDs.



---

## 10. API (Normative)

Base URL is the `url` from `runtime/server.json` (ephemeral port).

### 10.1 Health
- `GET /api/health` → `200`:
```json
{ "version": "0.1.x", "started_at": "RFC3339", "url": "http://127.0.0.1:PORT" }
```

### 10.2 Environments
- `GET /api/environments` → list
- `POST /api/environments` body:
```json
{ "name": "string", "description": "string?" }
```
- errors: `ERR_ENV_NAME_REQUIRED`

### 10.3 Ingest
- `POST /api/environments/{env_id}/ingests` (multipart form-data, file field name `file`)
  - returns `202`:
```json
{ "ingest_id": "uuid" }
```

- `GET /api/ingests/{ingest_id}` → `200`:
```json
{
  "ingest_id": "uuid",
  "env_id": "uuid",
  "status": "running|awaiting_user|completed",
  "stage": "receive|scan|identify|extract|derive|diff|persist|awaiting_user",
  "progress": { "pct": 0-100, "message": "string" },
  "rma_prompt": {
    "required": true,
    "candidates": [ { "logical_device_id": "uuid", "current_serial": "S1", "current_hostname": "fw1" } ]
  }
}
```
If `status=completed`, response MUST include the final ingest log line object (same schema as §9.2) under `final_record`.

- `POST /api/ingests/{ingest_id}/rma-decision` body:
```json
{
  "decision": "link_replacement|treat_as_new_device|canceled",
  "target_logical_device_id": "uuid?" 
}
```
Rules:
- If `decision=link_replacement`, `target_logical_device_id` is required.
- If `decision=canceled`, ingest finalizes error `ERR_USER_ABORTED`.

### 10.4 Errors (global)
Error shape (any endpoint that errors):
```json
{ "code": "ERR_...", "message": "string", "details": {} }
```
- `message` max 512 chars.

---

## 11. Spec Ambiguity Audit

- [x] Every feature has acceptance criteria.
- [x] Every acceptance criterion is verifiable.
- [x] Error behaviors are defined where applicable.
- [x] Edge cases are defined where applicable.
- [x] Any missing information is in Blocking Questions.
- [x] Decisions that affect behavior are recorded in Decisions.
- [x] No contradictions exist across sections.
- [x] MVP TSF extraction field set is normative and included as an appendix.

List remaining ambiguities (must be empty for determinism):
- (none)

---

## Appendix A — Palo Alto TSF Data Mapping (Normative)
See `spec/appendices/PALOALTO_DATA_MAPPING.MVP.v1.md`. This appendix is normative for:
- required output fields,
- source discovery patterns,
- extraction map (regex/xpath),
- fallback order and normalization.
