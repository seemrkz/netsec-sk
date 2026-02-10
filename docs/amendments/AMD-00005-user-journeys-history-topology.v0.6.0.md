---
doc_type: amendment
amendment_id: "AMD-00005"
slug: "user-journeys-history-topology"
version: "0.6.0"
status: "Applied"
created_at: "2026-02-10"
authors:
  - "cmarks"
spec_file: "../spec.v0.6.0.md"
spec_version_before: "0.5.1"
spec_version_after: "0.6.0"
---

# AMD-00005 — User Journey Coverage for Environment, History, Topology, and Export

## Reason

The prior spec captured core ingest and export contracts but did not make key operator journeys explicit and acceptance-testable for:

- representative environment creation patterns,
- state-change provenance visibility across commits/TSFs,
- Mermaid topology retrieval for current and historical commit state,
- deterministic export-bundle expectation framing.

This amendment introduces deterministic contracts for those journeys.

## Applied Spec Deltas

1. Versioned spec to `docs/spec.v0.6.0.md` and updated status metadata.
2. Extended scope language to explicitly include provenance and historical topology journeys.
3. Updated P1 command surface to include one-shot `history state`.
4. Expanded Section 5.2 with representative environment examples and explicit normalization/validation coupling.
5. Added Section 9.4 command contract for `history state` with fixed TSV columns and deterministic sorting.
6. Expanded Section 9.4 `topology` contract with optional `--at-commit <hash>` for non-mutating historical Mermaid retrieval.
7. Updated Section 10.3 ledger schema to require `changed_scope` and `changed_paths` for deterministic history rendering.
8. Expanded Section 11 acceptance criteria with explicit user-journey test requirements for environment examples, ingest outcomes, history provenance, route/feature scope visibility, current/historical Mermaid output, and full export bundle output.
9. Added decisions `D-00013` and `D-00014` for history and historical topology command behavior.
10. Updated ambiguity audit and appended review round `SR-00004` with PASS outcome.

## Scope and Impact

- Scope: specification only (no implementation changes in this amendment).
- Public interface impact: yes.
  - Added `history state` command contract.
  - Expanded `topology` command contract with `--at-commit <hash>`.
  - Strengthened ledger contract required fields to support deterministic history output.
- Backward compatibility:
  - additive command-surface expansion plus topology contract expansion.
  - no existing command/flag removals in this amendment.

## Verification Requirements Added

- Validate representative environment creation examples with normalization/validation.
- Validate multi-TSF ingest outcome accounting and provenance capture.
- Validate `history state` deterministic provenance rows and ordering.
- Validate route/feature change visibility through `changed_scope`.
- Validate `topology` Mermaid output for current state and `--at-commit` historical state without working-tree mutation.
- Validate export command still writes full deterministic bundle.
