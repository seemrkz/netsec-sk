---
doc_type: amendment
amendment_id: "AMD-00003"
version: "0.1.3"
date: "2026-02-11"
author: "Chris Marks (Product Owner) + Spec-Writer Agent"
status: "Applied" # Draft | In Review | Applied
scope:
  modifies_spec: true
  modifies_plan: false
related:
  spec: "../spec/spec.v0.1.2.md"
  spec_version_before: "0.1.2"
  spec_version_after: "0.1.3"
  plan: ""
  plan_version_before: ""
  plan_version_after: ""
trigger:
  reason: "Resolve API contract gaps for soft-delete, state fetch, commits list ordering, and flow trace endpoint so planning can proceed deterministically."
---

# Amendment AMD-00003 - API Contract Completion

## 1) Summary (What changed?)
This amendment adds deterministic API contracts that were implied by user flows and acceptance criteria but not fully specified in the API section:
- `DELETE /api/environments/{env_id}` soft delete behavior and response
- `GET /api/environments/{env_id}/state` response contract
- `GET /api/environments/{env_id}/commits` response contract, ordering, and unpaginated MVP stance
- `POST /api/environments/{env_id}/flow-trace` request/response contract and validation error behavior

It also updates acceptance criteria and the API decision record so interface behavior is fully implementable without defaults.

## 2) Change Type (Select all that apply)
- [x] Interface/schema change
- [x] Requirement change
- [x] Acceptance criteria change
- [x] Decision change
- [x] Clarification only (still requires amendment)

## 3) Rationale (Why?)
The spec already required soft delete and flow trace behavior (Flows + AC), but did not define all observable HTTP contracts under `§10 API`. This left implementation-level choices unresolved for statuses, payload shape, ordering, and validation errors. Under the no-defaults rule, these gaps blocked deterministic planning.

## 4) Options Considered (Required for decisions)
- Option A: Treat flow text as sufficient and allow implementer-defined API payloads.
- Option B: Add explicit endpoint contracts in `§10` with deterministic ordering/error behavior.
- Chosen: Option B.

## 5) Impact Analysis
### 5.1 Affected Spec Sections
- `§0.1`: appended review round `SR-00004`
- `§4.2` and `§4.3 D-00010`: API decision enforcement updated
- `§5.3 F4`: added commit-list ordering/pagination acceptance criteria
- `§10`: expanded endpoint contracts (`10.2`, new `10.4`, and global errors now in `10.5`)

### 5.2 Public Interface Impact
- Added normative contracts for four existing-scope API operations.
- No new top-level feature scope added.

### 5.3 Risk & Mitigation
- Risk: Over-constraining API before implementation details exist.
- Mitigation: Constraints were limited to already-required flows/AC and deterministic rules (ordering, schema shape, error codes).

## 6) Exact Deltas (Make it Executable)
### 6.1 Spec Deltas
- Added deterministic responses and rules for:
  - `DELETE /api/environments/{env_id}`
  - `GET /api/environments/{env_id}/state`
  - `GET /api/environments/{env_id}/commits`
  - `POST /api/environments/{env_id}/flow-trace`
- Added validation rule for flow-trace IP input with `ERR_INVALID_IP`.
- Added defined error code list in global errors section.
- Added `AC-F4-3` and `AC-F4-4` for commits endpoint determinism.

### 6.2 Decisions Deltas
- Decision ID: `D-00010`
- Change: Enforcement now explicitly requires deterministic contracts for environment lifecycle, state/commit retrieval, ingest, and flow trace endpoints in `§10`.
- Decision Index updated: D-00010 last updated is now `AMD-00003`.

## 7) Verification Updates
- Confirm new endpoint contracts are present in `spec.v0.1.3.md`.
- Confirm ambiguity list remains empty.
- Confirm review `SR-00004` is `PASS` with `blockers_count: 0`.
