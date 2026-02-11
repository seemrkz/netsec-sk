# AGENTS.md — Constitution for Spec-Driven Development (SSoT)

This repository uses a strict, deterministic, **spec-first** workflow designed for AI coding agents and human oversight.

---

## 0) Prime Directive (Non‑Negotiable)

- **`spec.vX.Y.Z.md` is the Single Source of Truth (SSoT) for intent.**
- **If a behavior, requirement, constraint, interface, decision, or verification requirement is not explicitly written in `spec.vX.Y.Z.md`, implementation is BLOCKED.**
- **No implicit defaults. No assumptions. No “best practices” additions unless explicitly specified.**
- If information is missing: add it to **Blocking Questions**, **ask the human**, create an amendment, re-run **Spec Review**, and continue only after the human has responded and the spec is deterministic.

---

## 1) Artifact Authority & Precedence

When content conflicts, resolve by precedence (highest wins):

1. `AGENTS.md` (this constitution)
2. `spec.vX.Y.Z.md` (latest deterministic version)
3. `/amendments/AMD-*.vX.Y.Z.md` (applied; newer overrides older)
4. `plan.vX.Y.Z.md` (execution only; MUST remain within spec)
5. `changelog.md` (historical record of completed tasks; not intent)
6. Codebase (implementation reality; MUST NOT override spec)

If a conflict exists between spec and code:
- Create an amendment describing the conflict and the proposed resolution.
- Do NOT implement until the spec is updated and deterministic again.

---

## 2) Workflow Phases (Hard Gates)

Hard gate rules:

- You MAY draft the initial spec without an amendment. Any spec change after the initial version MUST be documented via an amendment before editing `spec.vX.Y.Z.md`.
- You MAY NOT draft a plan until the spec is deterministic: Blocking Questions is empty, Spec Ambiguity Audit has 0 ambiguities, and all required decisions are Active.
- If planning discovers missing or ambiguous spec information, the relevant task(s) MUST be marked `Blocked`, planning MUST stop, and context MUST be shared with the human to restart spec/amendment.
- You MAY NOT write/modify production code until:
  - spec is deterministic (Blocking Questions empty, Spec Ambiguity Audit has 0 ambiguities, required decisions Active), AND
  - plan is deterministic, AND
  - the task is explicitly defined in the plan using the required task schema.

---

## 3) Spec Review is Recursive (Multi‑LLM)

Spec review MUST be performed in iterative rounds.

Each review round MUST produce:
- **BLOCKERS**: ambiguities, missing requirements, contradictions, missing constraints that prevent deterministic implementation.
- **NON‑BLOCKERS**: improvements that do not affect determinism.
- **SPEC DELTAS**: exact proposed text changes (edit instructions).

Stop conditions:
- If any BLOCKERS remain unresolved: implementation is prohibited.
- Spec is considered deterministic only when Blocking Questions is empty, Spec Ambiguity Audit has 0 ambiguities, and all required decisions are Active.

Suggested reviewer roles (each round):
- Reviewer A: Ambiguity Hunter (completeness + contradictions)
- Reviewer B: Edge Cases & Failure Modes (error behaviors + boundary conditions)
- Reviewer C: Security/Privacy/Abuse Modes (generic; policy + logging redaction)
- Reviewer D: Testability & Verification (AC completeness + verifiable commands)

---

## 4) No Defaults Policy (Strict)

This repo does not permit defaulting unless the spec explicitly allows it.

If required information is missing:
1. Add it to `spec.vX.Y.Z.md` under **Blocking Questions** (with options and consequences).
2. Ask the human.
3. Create an amendment documenting the change.
4. Re-run spec review.
5. Continue only after the human has responded and the spec is deterministic.

---

## 5) Amendment‑Only Change Control (Strict)

### 5.1 What requires an amendment
Any substantive change to:
- requirements / non-goals
- acceptance criteria
- decisions
- interfaces or schemas
- constraints (security/perf/ops)
- verification requirements
- plan ordering / tasks
- naming/structure rules

…requires an amendment file: `/amendments/AMD-00001-<slug>.vX.Y.Z.md`

### 5.2 Metadata carve-out (allowed without amendment)
The following updates do **not** require an amendment:
- `version`, `last_updated`
- appending a new entry to the Spec/Plan review round logs
- updating counters like “review rounds completed”
- adding new changelog entries for completed tasks
- updating task execution tracking in `plan.vX.Y.Z.md` (task `Status`, `Blocked by`, and Worktree strike-through for completed tasks), as long as scope/order/requirements are unchanged

**Rule:** if you change meaning/intent/execution, you need an amendment.

---

## 6) Decisions Live in `spec.vX.Y.Z.md` (No Separate ADR Files)

All non-trivial decisions MUST be recorded in `spec.vX.Y.Z.md` under **Decisions** using the required structure (context → options → decision → consequences → enforcement).

Rule:
- If a decision is `Proposed` or unresolved, implementation is BLOCKED.

---

## 7) Atomic Tasks + Enforced Budgets (Non‑Negotiable)

### 7.1 Atomic Task Definition
A task is atomic if:
- it implements **one** focused unit of value (typically one acceptance criterion or a minimal cluster that cannot be split),
- it is independently verifiable,
- it has a small blast radius and can be safely reverted.

### 7.2 Enforced Change Budgets (Default)
Unless the spec explicitly overrides, each task MUST stay within:

- **Max files changed:** 10
- **Max new files:** 3
- **Max net new LOC:** 300
- **Max public interface changes:** 0 (unless explicitly specified in the task/spec)

### 7.2.1 Public Interface Definition (Required)
For this repo, a "public interface" is any externally observable contract that consumers can depend on, including:
- API endpoints, request/response schemas, status codes, and error payloads.
- CLI commands, flags, exit codes, and stdout/stderr formats.
- Configuration file formats, env var names, and validation rules.
- Data formats exchanged with external systems (files, events, webhooks, messages).
- Database schemas or migrations that are directly consumed by other services or users.
- UI flows, routes, and user-visible strings that are explicitly defined as part of a user contract in the spec.

Non-exhaustive exclusions (do not count as public interface changes unless the spec explicitly says otherwise):
- Copy-only changes that do not alter documented user-visible strings in the spec.

A "public interface change" is any addition, removal, or modification to the above contracts.

If a task exceeds any budget:
- Split into multiple tasks.
- Update the plan (requires amendment if plan is already deterministic).
- Do NOT proceed with an oversized change.

### 7.3 1 Task = 1 Changelog Entry (Required)
- Every completed task MUST create exactly one changelog entry.
- Each changelog entry MUST include `Type`: Added | Changed | Fixed.

---

## 8) Commit Discipline (Atomic Commits)

Minimum requirement:
- Work MUST be organized so it is revertable by task.
- A task MUST NOT be marked Done until at least one commit has been created for that task.

Recommended default:
- **1 task = 1 commit** (preferred) OR 1 task = small commit series that is still revertable as a unit.
- If a task uses multiple commits, each commit message MUST start with the same task prefix (for example, `TASK-00001:`).

Commit message format (deterministic):
- `TASK-00001: <imperative summary>`

---

## 9) Verification & Logging Requirements (Hard Gate)

A task MAY be marked Done only if:
- Verification steps defined in the task have been executed, AND
- Results are recorded in the changelog entry, AND
- commit proof is recorded in the changelog entry (commit hash + commit message), AND
- Any required amendment(s) have been created and incorporated.
- If commit creation fails, the task MUST remain `Blocked` or not `Done`.

No “tests passed” without proof.

---

## 10) Prohibited Agent Behaviors (Examples)

Agents MUST NOT:
- implement scope not explicitly in spec,
- refactor “for cleanliness” unless the spec/task explicitly asks,
- introduce new abstractions “for maintainability” unless required,
- invent interface schemas/status codes/error messages/UX behavior,
- change tooling, formatting, or repo structure unless specified.

When in doubt: BLOCK and ask the human.

---

## 11) Identifier Policy (Deterministic)

- IDs are five digits and zero-padded (e.g., 00001).
- The first ID for each type MUST be 00001.
- IDs increment by 1 and are never reused.

---

## 12) Versioning & Filenames (Deterministic)

### 12.1 Format
- Versions MUST be `MAJOR.MINOR.PATCH` (e.g., `0.1.0`).
- Filenames MUST include the version and MUST match the document’s front matter:
  - `spec.vX.Y.Z.md`
  - `plan.vX.Y.Z.md`
  - `amendments/AMD-00001-<slug>.vX.Y.Z.md`

### 12.2 Baseline
- Blank templates start at `0.0.0`.
- The first real draft becomes `0.1.0` and the filename MUST be renamed to match.

### 12.3 Increment Rules (Major/Minor/Fix)
- **MAJOR**: Scope-breaking change (e.g., changes to Vision, Core Objectives, Non‑Goals, or removal/renaming of a top‑level feature or externally observable interface).
- **MINOR**: Backward‑compatible scope expansion (e.g., adding a top‑level feature, new user flow, or new acceptance criteria without breaking prior scope).
- **PATCH (Fix)**: Clarification or correction that does NOT expand or contract scope (no new top‑level features; no removals).

### 12.4 Amendment Version Source
- If an amendment modifies the spec, its version MUST match `spec_version_after`.
- If it modifies only the plan, its version MUST match `plan_version_after`.
