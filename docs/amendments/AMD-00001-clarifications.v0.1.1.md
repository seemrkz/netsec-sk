---
doc_type: amendment
amendment_id: "AMD-00001"
version: "0.1.1"
date: "2026-02-10"
author: "Chris Marks (Product Owner) + Spec-Writer Agent"
status: "Applied" # Draft | In Review | Applied
scope:
  modifies_spec: true
  modifies_plan: false
related:
  spec: "../spec/spec.v0.1.0.md"
  spec_version_before: "0.1.0"
  spec_version_after: "0.1.1"
  plan: ""
  plan_version_before: ""
  plan_version_after: ""
trigger:
  reason: "Apply answered blocking decisions (storage root, ephemeral port, batch error policy, CIDR overlap inference, RMA confirmation UX) and add deterministic schemas/appendix for implementability."
---

# Amendment AMD-00001 — Clarifications + Determinism Tightening

> Strict rule: **Any substantive spec/plan change requires an amendment.**
> This amendment is the authoritative explanation of **what changed** and **why**.

---

## 1) Summary (What changed?)
This amendment updates the spec to encode previously-unstated implementation-impacting decisions (macOS storage root `~/.netsec-sk`, ephemeral localhost port discovery, batch ingest continue-on-error semantics, CIDR overlap-based adjacency inference, and an explicit user confirmation flow for RMAs). It also adds normative on-disk schemas (state/log/runtime) and a normative appendix that defines the exact TSF extraction field set and fallback rules.

## 2) Change Type (Select all that apply)
- [x] Requirement change
- [x] Acceptance criteria change
- [x] Decision change
- [x] Constraint change (security/perf/ops)
- [x] Interface/schema change
- [x] Clarification only (still requires amendment)

## 3) Rationale (Why?)
Human answers provided concrete decisions required for determinism:
- storage root should be `~/.netsec-sk`
- backend uses ephemeral port and UI shows/copies URL
- batch ingest continues even if one file fails
- topology inference should treat overlapping CIDRs as adjacency evidence
- if hostname matches but serial differs, user should confirm RMA linkage

Additionally, the earlier spec lacked explicit schemas and a normative field set for TSF extraction (it referenced the concept but did not bind it to an authoritative artifact).

## 4) Options Considered (Required for decisions)
### Storage root
- Option A: `~/Library/Application Support/netsec-sk/` → macOS-native, hidden
- Option B: `~/.netsec-sk/` → simplest and CLI-friendly
- Chosen: B because the product is developer/operator oriented and benefits from a predictable dotfolder location.

### Server port
- Option A: fixed port (collision risk)
- Option B: ephemeral available port + publish URL
- Chosen: B to avoid collisions without requiring configuration.

### Adjacency inference matching
- Option A: exact CIDR equality only
- Option B: CIDR overlap/intersection counts
- Chosen: B to model real-world cases where routes differ in specificity but still represent shared reachability.

### RMA handling
- Option A: auto-link by heuristic only
- Option B: prompt user when hostname matches but serial differs
- Chosen: B to prevent silent mis-linking and preserve trust in state history.

## 5) Impact Analysis
### 5.1 Affected Spec Sections
- SPEC §4 Decisions: updated multiple decision records with concrete parameters.
- SPEC §5 Requirements: added RMA prompt feature and batch continue-on-error AC.
- SPEC §6 Algorithms: added deterministic definitions for overlap and flow trace tie-breaks.
- SPEC §9 Data Layout & Schemas: added normative schemas for logs/runtime storage.
- SPEC Appendix A: added normative mapping file reference.

### 5.3 Risk & Mitigation
- Risk: CIDR overlap inference increases false-positive adjacencies → Mitigation: store explicit evidence and use most-specific overlaps; ignore default route.
- Risk: RMA prompt pauses ingest and complicates UX → Mitigation: explicit `awaiting_user` state + persisted intermediate JSON (no TSF retention) with TTL.

## 6) Exact Deltas (Make it Executable)

### 6.1 Spec Deltas

#### Delta 1 — Storage root becomes explicit
- Replace (verbatim):
  - (no explicit storage root in v0.1.0)
- With (verbatim):
  - "Storage root MUST be `~/.netsec-sk/`. Environments MUST be stored under `~/.netsec-sk/environments/<env_id>/`." (SPEC §4.3 D-00002, and §9.1)

#### Delta 2 — Ephemeral port + discovery contract
- Replace (verbatim):
  - (no explicit port model in v0.1.0)
- With (verbatim):
  - "MUST: choose an ephemeral available TCP port on startup. MUST: write the chosen URL to `~/.netsec-sk/runtime/server.json`. MUST: print `NETSEC_SK_URL=<url>` to stdout once server is ready." (SPEC §4.3 D-00001, §9.5, §10.1)

#### Delta 3 — Batch ingest continues on error
- Replace (verbatim):
  - "Batch ingest is sequential; UI processes files sorted ascending by filename." (SPEC v0.1.0 D-00012)
- With (verbatim):
  - "Batch ingest is sequential; UI processes files sorted ascending by filename; failures do not stop subsequent files." (SPEC §4.3 D-00012 and AC-F2-5)

#### Delta 4 — Adjacency inference uses CIDR overlap
- Replace (verbatim):
  - "Inferred adjacency MUST be produced when two firewalls have non-default routes to the same destination CIDR (exact match after canonicalization)." (SPEC v0.1.0 AC-F5-2)
- With (verbatim):
  - "Inferred adjacency MUST be produced when two firewalls have non-default routed CIDRs that overlap (CIDR intersection is non-empty)." (SPEC §5.3 AC-F5-2)

#### Delta 5 — RMA prompt requirement
- Replace (verbatim):
  - "RMA heuristic: ... link it as replacement; else create a new logical device and record an `rma_candidates[]` entry for review." (SPEC v0.1.0 D-00009)
- With (verbatim):
  - "If an ingest yields a new serial and the hostname matches one or more existing logical devices, the ingest MUST prompt the user to confirm whether this is an RMA replacement." (SPEC §4.3 D-00009 and §5.2 Flow C2)

#### Delta 6 — Add normative schemas and appendix
- Replace (verbatim):
  - (no schemas section in v0.1.0)
- With (verbatim):
  - Added SPEC §9 (Data Layout & Schemas) and Appendix A reference to `spec/appendices/PALOALTO_DATA_MAPPING.MVP.v1.md`.

### 6.3 Decisions Deltas (if applicable)
- Updated decision records: D-00001, D-00002, D-00004, D-00006, D-00007, D-00008, D-00009, D-00010, D-00011, D-00012
- Updated Decision Index “Last Updated (AMD)” to `AMD-00001` for those decisions.

## 7) Verification Updates
- Added verification steps for:
  - batch continue-on-error
  - RMA prompt + decision
  - overlap-based adjacency inference

Evidence to capture:
- one `ingest.ndjson` line showing `duration_ms_total` and `duration_ms_compute`
- one ingest status transition showing `awaiting_user` → completion after RMA decision
- one inferred adjacency edge with overlap evidence

