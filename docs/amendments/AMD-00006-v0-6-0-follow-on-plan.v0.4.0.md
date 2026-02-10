---
doc_type: amendment
amendment_id: "AMD-00006"
slug: "v0-6-0-follow-on-plan"
version: "0.4.0"
status: "Applied"
created_at: "2026-02-10"
authors:
  - "cmarks"
plan_file: "../plan.v0.4.0.md"
plan_version_before: "0.3.0"
plan_version_after: "0.4.0"
spec_file: "../spec.v0.6.0.md"
spec_version_before: "0.6.0"
spec_version_after: "0.6.0"
---

# AMD-00006 — Follow-On Plan Update for Spec v0.6.0

## Reason

`plan.v0.3.0.md` completed the prior implementation lane but did not include the v0.6.0 user-journey deltas for deterministic history provenance, historical Mermaid topology retrieval, and related acceptance closure requirements.

A new follow-on plan version is required to sequence the remaining work in one strict sequential lane.

## Applied Plan Deltas

1. Added `docs/plan.v0.4.0.md` as the active follow-on implementation artifact for `spec.v0.6.0.md`.
2. Defined one sequential lane (`LANE1`) with strict top-to-bottom execution.
3. Added task set `TASK-00034` through `TASK-00038` covering:
   - commit-ledger provenance fields (`changed_scope`, `changed_paths`),
   - `history state` contract,
   - `topology --at-commit` Mermaid contract,
   - acceptance/regression expansion,
   - final acceptance closure and user journey packet.
4. Added plan review round `PR-00004` with PASS outcome and blocker count `0`.
5. Added explicit user feel/experience validation deliverable (`docs/user-journey-test-v0.6.0.md`).

## Scope and Impact

- Scope: plan-only update; spec remains unchanged.
- Public interface impact: none directly from the amendment artifact itself.
- Expected implementation impact: complete v0.6.0 deltas and close acceptance criteria §11(23)-§11(29).

## Verification Requirements

Each task requires:

- spec-mapped acceptance and verification steps,
- changelog evidence,
- commit hash + commit message proof capture,
- lane status/strike-through updates on completion.
