---
doc_type: amendment
amendment_id: "AMD-00002"
version: "0.1.2"
date: "2026-02-10"
author: "Chris Marks (Product Owner) + Spec-Writer Agent"
status: "Applied" # Draft | In Review | Applied
scope:
  modifies_spec: true
  modifies_plan: false
related:
  spec: "../spec/spec.v0.1.1.md"
  spec_version_before: "0.1.1"
  spec_version_after: "0.1.2"
  plan: ""
  plan_version_before: ""
  plan_version_after: ""
trigger:
  reason: "Add deterministic schemas for meta.json/state.json/intro.md and align canonicalization rules so the output is implementable for both humans and AI agents."
---

# Amendment AMD-00002 — State Schema + Intro Format

## 1) Summary (What changed?)
This amendment adds normative schemas for `meta.json`, `state.json`, and `intro.md`, including required keys, stable array ordering rules, and canonical hashing rules used by commits. It also fixes an internal reference in D-00011 to point to the correct schema section.

## 2) Change Type (Select all that apply)
- [x] Interface/schema change
- [x] Requirement change
- [x] Acceptance criteria change
- [x] Decision change
- [x] Clarification only (still requires amendment)

## 3) Rationale (Why?)
Without an explicit state schema and intro format, implementers would guess data shapes and paths, undermining:
- determinism (diffs and hashes),
- usability for AI agents consuming `state.json`,
- stable UI rendering and future migrations.

## 4) Options Considered (Required for decisions)
- Option A: Keep schema implicit; rely on code to define structure → fast now, ambiguous forever.
- Option B: Define a minimal but explicit MVP schema with ordering rules → deterministic, testable.
- Chosen: B because the product’s core output is the state representation.

## 5) Impact Analysis
### 5.1 Affected Spec Sections
- SPEC §9: Added §9.6–§9.8 defining `meta.json`, `state.json`, `intro.md`.
- SPEC §4.3 D-00011: updated reference to array ordering rules.
- SPEC §7: verification now can validate stable ordering/hashes.

### 5.3 Risk & Mitigation
- Risk: Schema may be incomplete vs future needs → Mitigation: MVP schema is intentionally minimal and tied to Appendix A; future expansions require amendments.
- Risk: Ordering rules might be missed in implementation → Mitigation: explicit canonicalization contract and hash definition.

## 6) Exact Deltas (Make it Executable)

### 6.1 Spec Deltas
- Replace (verbatim):
  - "MUST: define array ordering rules (schema in §9.3)."
- With (verbatim):
  - "MUST: define array ordering rules (schema in §9.7)."

- Add (verbatim):
  - SPEC §9.6 `meta.json` schema
  - SPEC §9.7 `state.json` schema (MVP) + array ordering + canonical hash rule
  - SPEC §9.8 `intro.md` required content/sections

### 6.3 Decisions Deltas
- Decision ID: D-00011
- Changes:
  - Clarified where canonicalization and ordering rules are specified.
- Decision Index updated: D-00011 “Last Updated” → `AMD-00002`.

## 7) Verification Updates
- New verification checks:
  - `state.json` contains all required top-level keys and uses required ordering rules.
  - `commits.ndjson` `state_hash_*` matches SHA-256 of canonical JSON representation.
  - `intro.md` contains required sections and pointers.

Evidence:
- excerpt of `state.json` showing schema_version and required paths
- hash recomputation script output matching `state_hash_after`
- `intro.md` excerpt with required bullet pointers

