---
doc_type: amendment
amendment_id: "AMD-00003"
slug: "open-useful-prototype"
version: "0.5.1"
status: "Applied"
created_at: "2026-02-09"
authors:
  - "cmarks"
spec_file: "../spec.v0.5.1.md"
spec_version_before: "0.5.0"
spec_version_after: "0.5.1"
plan_file: "../plan.v0.2.0.md"
plan_version_before: "0.1.0"
plan_version_after: "0.2.0"
---

# AMD-00003 — Open + Useful Prototype Build Target

## Reason

The prior spec established deterministic MVP contracts but did not isolate a prototype execution slice for a working interactive operator loop. Current implementation state also exposed ambiguity in ingest accounting for mixed file sets and under-specified `open` shell behavior. This amendment adds deterministic, build-ready constraints for a prototype target while preserving end-state MVP intent.

## Applied Spec Deltas

1. Versioned spec to `docs/spec.v0.5.1.md` and updated front matter metadata (`last_updated`, `review_rounds_completed`, `project_state`, `plan_target`).
2. Added Section `2.3 Prototype P1 build target (required now)` with explicit required one-shot and in-shell command sets.
3. Deferred in-shell `export` and `topology` requirement to Section 17.
4. Resolved ingest ordering/input ambiguity:
   - ordering now applies to all expanded files,
   - unsupported extensions are classified as `parse_error_fatal` attempts with `unsupported_extension` note,
   - summary counters must include unsupported files in `attempted` and `parse_error_fatal`.
5. Added Section `6.9 Archive extraction contract` with extraction location and path traversal safety requirements.
6. Added Section `6.10 Prototype parse minimum required fields` with deterministic fatal vs partial boundaries for firewall/panorama minimum identity extraction.
7. Strengthened Section `9.4`:
   - `ingest` must run full pipeline (no placeholder summary implementation),
   - `open` shell prompt cadence, empty-line, EOF/quit, continue-on-error, and one-shot parity requirements are explicit.
8. Added Section `10.5 Commit operation guarantee` to require exactly one commit per `committed` ingest result row.
9. Rewrote Section `11 Acceptance Criteria` for prototype-evaluable end-to-end behavior, including real `.tgz` ingest, shell roundtrip, and unsupported extension accounting.
10. Added Decisions `D-00010` to `D-00012` and updated decision enforcement text for D-00001, D-00004, and D-00005.
11. Updated ambiguity audit checklist and appended `SR-00003` review log with PASS outcome.

## Scope and Impact

- Scope: spec + plan targets updated for prototype execution readiness.
- Public interface impact:
  - clarified `open` shell required command subset and behavior,
  - clarified ingest summary semantics for unsupported extensions,
  - clarified ingest operational requirement to execute full persistence/commit pipeline.
- Backward compatibility: no CLI command/flag names changed.

## Verification Requirements Added

- Validate shell roundtrip: `open` -> `ingest <fixture.tgz>` -> `show <entity>` returns persisted data.
- Validate mixed file ingest accounting includes unsupported extensions as fatal attempts.
- Validate first changed TSF creates one commit and duplicate/unchanged reruns do not commit.
- Validate `open` continues session after command error until `quit`/`exit`/EOF.

## Follow-on Planning

- New execution plan target is `docs/plan.v0.2.0.md`.
- Plan v0.2.0 defines approximately eight prototype-focused tasks aligned to spec v0.5.1.
