---
doc_type: amendment
amendment_id: "AMD-00001"
slug: "determinism-closure"
version: "0.5.0"
status: "Applied"
created_at: "2026-02-08"
authors:
  - "cmarks"
spec_file: "../spec.v0.5.0.md"
spec_version_before: "0.5.0"
spec_version_after: "0.5.0"
---

# AMD-00001 — Determinism Closure for Spec v0.5.0

## Reason

Spec review round `SR-00001` failed with blockers that prevented deterministic planning under `AGENTS.md` hard gates.

## Blockers Addressed

1. Missing CLI exit-code contract.
2. Missing stderr error payload format.
3. Missing fixed export CSV schemas.
4. Missing `env_id` validation grammar.
5. Missing stale lock definition.
6. Missing parse error taxonomy boundary (`parse_error_partial` vs `parse_error_fatal`).
7. Missing deterministic batch ingest ordering.
8. Missing RDNS timeout/retry rules.
9. Missing per-TSF commit message format.

## Applied Spec Deltas

- Added deterministic command contracts in `spec.v0.5.0.md` Section 9.
- Added fixed export schema contracts in Section 8.
- Added `env_id` grammar and normalization in Section 5.1.
- Added lock policy and stale lock thresholds in Section 6.2.
- Added parse taxonomy boundaries in Section 6.6.
- Added deterministic ingest ordering in Section 3.4.
- Added RDNS policy in Section 6.8.
- Added commit message format in Section 10.2.
- Added complete Decisions section (`D-00001` through `D-00009`) with `Active` status.
- Added `Blocking Questions` and `Spec Ambiguity Audit` sections showing zero unresolved items.

## Scope and Impact

- Scope: specification only (no code changes).
- Public interface impact: yes, clarified/locked contracts for CLI outputs, exit codes, and export files.
- Backward compatibility: n/a (pre-implementation specification hardening).

## Verification

- `Blocking Questions` section exists and is empty.
- `Spec Ambiguity Audit` unresolved count is `0`.
- All required decisions are present and `Active`.
- `SR-00002` logged with blocker count `0`.
