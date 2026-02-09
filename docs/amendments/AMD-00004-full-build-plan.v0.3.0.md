---
doc_type: amendment
amendment_id: "AMD-00004"
slug: "full-build-plan"
version: "0.3.0"
status: "Applied"
created_at: "2026-02-09"
authors:
  - "cmarks"
plan_file: "../plan.v0.3.0.md"
plan_version_before: "0.2.0"
plan_version_after: "0.3.0"
spec_file: "../spec.v0.5.1.md"
spec_version_before: "0.5.1"
spec_version_after: "0.5.1"
---

# AMD-00004 — Full-Build Plan Update for Spec v0.5.1

## Reason

`plan.v0.2.0.md` focused on the prototype P1 slice but did not fully sequence all non-deferred contracts in `spec.v0.5.1.md` (notably full export/query/topology one-shot completion and acceptance-level verification closure). A new plan version is required to define an end-to-end execution path to full spec conformance.

## Applied Plan Deltas

1. Created `docs/plan.v0.3.0.md` targeting `spec.v0.5.1.md`.
2. Replaced task set with full-build sequence `TASK-00023` through `TASK-00033`.
3. Added explicit task-level commit requirement and proof-capture fields (commit hash + commit message) for each task.
4. Expanded phase coverage to include:
   - ingest runtime gates and extraction safety,
   - parse/state/commit guarantees,
   - export/query/topology command completion,
   - open-shell/help parity and final acceptance/release verification.
5. Added updated worktree lane model and dependency graph for full-build execution.
6. Added plan review round `PR-00003` with blockers count `0`.

## Scope and Impact

- Scope: plan-only update; spec unchanged.
- Public interface impact: none directly (execution planning artifact).
- Expected implementation impact: complete non-deferred spec implementation coverage.

## Verification Requirements

- Each task requires:
  - spec-mapped acceptance criteria,
  - command-level verification steps,
  - changelog evidence,
  - commit hash + message proof capture.

## Follow-on

- Execute tasks in `plan.v0.3.0.md` lane/dependency order.
- Keep `plan.v0.2.0.md` as historical reference.
