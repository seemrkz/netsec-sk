---
doc_type: spec
project: "NetSec StateKit (NS SK)"
spec_id: "SPEC-00001"
version: "0.6.0"
owners:
  - "cmarks"
last_updated: "2026-02-10"
human_approver: "cmarks"
review_rounds_completed: 4
project_state: "spec_ready_plan_refresh_required"
related:
  constitution: "./AGENTS.md"
  plan_target: "./plan.v0.3.0.md"
---

# NetSec StateKit (`netsec-sk`) MVP Specification v0.6.0

## 0) Document Status

- Tool name: NetSec StateKit (NS SK)
- CLI binary: `netsec-sk`
- Implementation language: Go
- Distribution for MVP: manual GitHub Releases binaries + checksums
- This document is the canonical deterministic spec for planning and implementation.
- Active implementation plan target: `./plan.v0.3.0.md`

## 1) Purpose

`netsec-sk` ingests Palo Alto Networks TSFs (firewall and Panorama), extracts facts-only state, and persists that state in a Git-backed repository so operators can answer inventory/topology/routing questions and inspect change history with Git.

Primary outcomes:

- Inventory of firewalls and Panorama instances by environment.
- HA, interface/zone/VR, and routing facts (runtime best-effort where stated).
- Panorama group/template/template-stack membership mapping.
- Deterministic per-TSF state commits for diff/history.

## 2) MVP Scope

### 2.1 End-state MVP in scope

- TSF archive inputs: `.tgz`, `.tar.gz`.
- Mixed ingest batches (firewall + Panorama TSFs).
- Multi-environment state in one repo under `envs/<env_id>/...` with strict isolation.
- Git-required operation with one atomic commit per TSF that changes state.
- Per-environment exports: `environment.json`, `inventory.csv`, `nodes.csv`, `edges.csv`, `topology.mmd`, `agent_context.md`.
- Optional reverse DNS enrichment on newly discovered devices via `--rdns`.
- Interactive shell (`open`) and one-shot commands.
- Deterministic state-change history visibility with commit and TSF provenance.
- Deterministic Mermaid topology view for current state and prior commit state.

### 2.2 Out of scope

- Live polling via Panorama/device APIs.
- Homebrew/Scoop automation.
- Full policy/rulebase graphing.
- IPv6 topology inference (MVP is IPv4-only).
- Data masking/redaction beyond facts-only model design.

### 2.3 Prototype P1 build target (required now)

P1 is the required implementation slice for this spec version and is binding for execution planning.

Required one-shot commands:

- `init`
- `env list`
- `env create`
- `ingest`
- `devices`
- `panorama`
- `show device`
- `show panorama`
- `history state`
- `open`
- `help`

Required in-shell command set for `open`:

- `help`
- `help <command>`
- `env list`
- `env create`
- `ingest`
- `devices`
- `panorama`
- `show device`
- `show panorama`
- `exit`
- `quit`

Notes:

- In-shell `export` and `topology` are deferred (Section 17).
- One-shot `export` and `topology` command contracts remain defined for MVP and may be implemented outside P1.
- One-shot `history state` command contract is required in this spec version.

## 3) Deterministic Constraints

### 3.1 Git required

- `init` MUST fail if `git` executable is not available on `PATH`.
- State repo MUST be initialized as a Git repo.

### 3.2 Safe working tree policy

`ingest` MUST refuse to run when target repo is in any unsafe state:

- merge/rebase/cherry-pick in progress,
- staged changes present,
- tracked file modifications present.

Untracked files are allowed only if they are not staged.

### 3.3 Atomic commit rule

- Exactly one Git commit per TSF that changes state.
- No commit when TSF is duplicate (`skipped_duplicate_tsf`) or state is unchanged (`skipped_state_unchanged`).

### 3.4 Deterministic batch ordering

For `ingest <paths...>`:

1. Expand directories recursively to files.
2. Canonicalize to absolute normalized paths.
3. Process in lexical ascending order of canonical path.
4. For each file in order:
   - if extension is `.tgz` or `.tar.gz`, run archive ingest flow;
   - otherwise emit ingest result `parse_error_fatal` with note `unsupported_extension`.

## 4) State Repository Layout

At repo root:

- `envs/<env_id>/state/...` (git-tracked)
- `envs/<env_id>/exports/...` (git-tracked)
- `envs/<env_id>/overrides/...` (git-tracked if present)
- `.netsec-state/...` (tool metadata; gitignored)

### 4.1 Per-environment structure

```text
envs/
  <env_id>/
    state/
      commits.ndjson
      devices/
        <device_id>/
          latest.json
          snapshots/
            <timestamp>_<stateHash>.json
      panorama/
        <panorama_id>/
          latest.json
          snapshots/
            <timestamp>_<stateHash>.json
    exports/
      environment.json
      inventory.csv
      nodes.csv
      edges.csv
      topology.mmd
      agent_context.md
    overrides/
      topology_links.json      (optional)
      vr_equivalence.json      (optional)
```

### 4.2 Tool metadata and extraction workspace

```text
.netsec-state/
  config.json
  extract/
  ingest.ndjson
  lock
```

Rules:

- `.netsec-state/` MUST be gitignored.
- TSF archives and extracted raw files MUST never be committed.

## 5) Environment Model

### 5.1 `env_id` validation and normalization

- Regex: `^[a-z0-9](?:[a-z0-9-]{0,30}[a-z0-9])?$`
- Length: 1..32 chars.
- Allowed chars: lowercase letters, digits, hyphen.
- Cannot start/end with hyphen.
- CLI normalization: trim surrounding whitespace then lowercase.
- Invalid IDs return usage error (exit code 2, `E_USAGE`).

### 5.2 Environment creation rules

- `init` creates base repo only; no environment directories.
- `env create <env_id>` creates environment explicitly.
- `ingest --env <env_id>` auto-creates environment if missing.
- Missing `--env` uses `default` (created on first ingest if absent).
- Common environment examples include: `prod`, `development`, `cloud`, `lab`, `home`, `customera`, `customerb`.
- The examples above are illustrative only; all environment names MUST still satisfy Section 5.1 validation and normalization.

### 5.3 Isolation guarantees

- No dedupe across environments.
- No cross-environment topology edges.
- Same serial MAY appear in multiple environments independently.

## 6) Ingest Lifecycle

### 6.1 Accepted ingest inputs

- File and directory arguments are allowed.
- Every expanded file is an ingest attempt and MUST produce exactly one `.netsec-state/ingest.ndjson` entry.
- Non-archive files are not extracted and MUST be logged with `parse_error_fatal` + note `unsupported_extension`.
- Ingest summary counters MUST include unsupported extensions in both `attempted` and `parse_error_fatal`.

### 6.2 Locking policy (`.netsec-state/lock`)

Lock content is JSON with fields:

- `pid` (int)
- `started_at_utc` (RFC3339)
- `started_at_unix` (int)
- `command` (string)

Lock handling:

- If lock file exists and process is active with matching PID start time, ingest MUST fail (`E_LOCK_HELD`, exit code 5).
- If lock file exists but PID is missing, PID start time mismatches, or lock age > 8 hours, lock is stale and MUST be removed with warning.

### 6.3 Extraction workspace cleanup

- Extract each TSF to `.netsec-state/extract/<run_id>/...`.
- On ingest start, remove stale extract dirs older than 24h (best-effort).
- After each TSF ingest (success or failure), remove that TSF extract dir unless `--keep-extract` is set.
- Cleanup failures are warnings only and do not change ingest result code.

### 6.4 TSF identity (`tsf_id`)

Identity source is TSF internal metadata under `tmp/cli/` only.

- Locate candidate text file using patterns in priority order:
  1. `(^|.*/)?tmp/cli/[^/]+\.txt$`
  2. `(^|.*/)?tmp/cli/.*\.txt$`
- Prefer candidate containing serial line; ties resolved by lexical path order.

Derive fields:

- `tsf_original_name` from selected metadata filename:
  - strip trailing `.txt` when suffix is `.tgz.txt` or `.tar.gz.txt`.
  - else extract shortest `[A-Za-z0-9._-]+_ts\.(?:tgz|tar\.gz)` when present.
  - else use filename as-is.
- `serial` by first match (case-insensitive):
  - `serial:`
  - `serial number:`
  - `device serial:`

Construction:

- `tsf_id = "<serial>|<tsf_original_name>"`.
- If metadata file is missing entirely: `tsf_id = "unknown"`.
- If serial missing: keep empty serial with delimiter, e.g. `|PA-440_ts.tgz`.

### 6.5 Dedupe behavior

- Dedupe key is `tsf_id`, scoped to environment.
- Seen-TSF source of truth is `.netsec-state/ingest.ndjson` entries for that `env_id`.
- If `tsf_id == "unknown"`, duplicate-TSF skip is disabled.

### 6.6 Parse result taxonomy

Result values in `.netsec-state/ingest.ndjson`:

- `committed`
- `skipped_duplicate_tsf`
- `skipped_state_unchanged`
- `parse_error_partial`
- `parse_error_fatal`

Boundary rules:

- `parse_error_fatal`:
  - archive unreadable/corrupt, or
  - cannot classify entity type (`firewall|panorama`), or
  - cannot derive `entity_id`.
- `parse_error_partial`:
  - entity type + entity ID determined,
  - snapshot written with required fields,
  - one or more optional fields unavailable.

### 6.7 State unchanged detection

- Compute `state_sha256` over normalized facts-only state (Section 7.4).
- Compare against existing `latest.json` hash for same entity + environment.
- If unchanged: skip commit with `skipped_state_unchanged`.

### 6.8 RDNS policy (`--rdns`)

- RDNS runs only for newly discovered firewall devices.
- Lookup target: `device.mgmt_ip` when present and IPv4.
- Resolver behavior:
  - timeout 1 second per attempt,
  - one retry (max 2 attempts total),
  - no caching across process runs in MVP.
- Populate `device.dns.reverse` as `{ip, ptr_name, status, looked_up_at_utc}` with status `ok|not_found|timeout|error`.

### 6.9 Archive extraction contract

- Accepted archive formats: `.tgz`, `.tar.gz`.
- Each supported archive attempt MUST extract under `.netsec-state/extract/<run_id>/<tsf_dir>/...`.
- Archive entry paths MUST be normalized and validated before extraction.
- Extraction MUST reject path traversal entries (`..`, absolute roots, symlink escapes) so writes cannot escape the assigned extract dir.
- Archive unreadable/corrupt failures MUST be classified as `parse_error_fatal`.

### 6.10 Prototype parse minimum required fields

For P1, parse minimums are:

- Firewall non-fatal minimum:
  - `entity_type = firewall`
  - `device.id`
  - `device.serial`
- Panorama non-fatal minimum:
  - `entity_type = panorama`
  - `panorama_instance.id`
  - `panorama_instance.serial`

Classification rules:

- Missing required identity fields above => `parse_error_fatal`.
- Missing non-identity fields (for example hostname/model/version/mgmt_ip/routing details) => `parse_error_partial`.
- `parse_error_partial` MUST still write a snapshot containing required envelope + minimum identity fields.

## 7) Snapshot Data Contracts

### 7.1 Shared envelope (all snapshots)

Required fields:

- `snapshot_version` (int, fixed `1`)
- `source` object:
  - `tsf_id` (string)
  - `tsf_original_name` (string)
  - `input_archive_name` (string)
  - `ingested_at_utc` (RFC3339)
- `state_sha256` (lowercase hex, 64 chars)

### 7.2 Firewall snapshot schema

Required object `device`:

- `id` (string; serial when present, else deterministic fallback)
- `hostname` (string, empty allowed)
- `serial` (string, empty allowed)
- `model` (string, empty allowed)
- `sw_version` (string, empty allowed)
- `mgmt_ip` (string, empty allowed)

`ha` object:

- `enabled` (bool)
- `mode` (string: `active-passive|active-active|unknown`)
- `local_state` (string)
- `peer_serial` (string)

`licenses[]` entries:

- `name`, `status`, `expires_on` (strings)

`network`:

- `interfaces[]`: `{name, ip_cidrs[], zone, virtual_router}`
- `zones[]`: `{name, type, interfaces[]}`

`routing`:

- `virtual_routers[]` entries:
  - `name`
  - `config.protocols_configured[]` subset of `static|ospf|bgp`
  - `runtime.protocols_active[]` subset of `static|ospf|bgp`
  - `counts.static_routes_configured_v4` (int)
  - `counts.static_routes_configured_v6` (int)
  - `counts.static_routes_installed_v4` (int)
  - `counts.static_routes_installed_v6` (int)
  - `runtime.health` (object, optional)
  - `runtime.vr_context` (`known|unknown`)
- `runtime_unknown` optional object for runtime signals not attributable to a specific VR.

### 7.3 Panorama snapshot schema

Required object `panorama_instance`:

- `id`, `hostname`, `serial`, `model`, `version`, `mgmt_ip`

Optional object `panorama_ha` best-effort.

Required object `panorama_config`:

- `device_groups[]`: `{name, parent, members_serials[]}`
- `templates[]`: `{name, members_serials[]}`
- `template_stacks[]`: `{name, templates[], members_serials[]}`
- `managed_devices[]`: `{serial, hostname, model, connected_status}`

### 7.4 `state_sha256` canonicalization

Hash input excludes volatile fields:

- all `source.*`
- `state_sha256`

Deterministic ordering before hash:

- firewall: `interfaces`, `zones`, `virtual_routers`, `licenses` by `name`
- panorama: `device_groups`, `templates`, `template_stacks` by `name`
- membership serial lists sorted ascending

Canonical serialization:

- stable JSON key ordering,
- UTF-8,
- no non-deterministic whitespace.

## 8) Export Contracts

### 8.1 `exports/environment.json`

Schema (required keys):

```json
{
  "schema_version": 1,
  "environment": {
    "env_id": "string",
    "generated_at_utc": "RFC3339",
    "counts": {
      "firewalls": 0,
      "panorama": 0,
      "zones": 0,
      "topology_edges": 0
    }
  },
  "firewalls": [],
  "panorama": [],
  "topology": {
    "zone_edges": []
  }
}
```

### 8.2 `exports/inventory.csv`

Header order is fixed:

```text
entity_type,entity_id,hostname,serial,model,version,mgmt_ip,ha_enabled,ha_mode,ha_state,routing_protocols_configured,routing_protocols_active,source_tsf_id,state_sha256,last_ingested_at_utc
```

Rules:

- `entity_type` values: `firewall|panorama`
- protocol list fields use `;` separator and lexical order.
- rows sorted by `entity_type`, then `entity_id`.

### 8.3 `exports/nodes.csv`

Header order is fixed:

```text
node_id,node_type,env_id,device_id,panorama_id,zone,virtual_router,label
```

Rules:

- `node_type` values: `firewall|panorama|zone`.
- `node_id` is deterministic and unique within environment.
- rows sorted by `node_type`, then `node_id`.

### 8.4 `exports/edges.csv`

Header order is fixed:

```text
edge_id,edge_type,src_node_id,dst_node_id,src_device_id,src_zone,src_interface,src_vr,dst_device_id,dst_zone,dst_interface,dst_vr,evidence,source
```

Rules:

- `edge_type` values: `shared_subnet|manual_override`.
- `source` values: `inferred|override`.
- rows sorted by `edge_id`.

### 8.5 `exports/topology.mmd`

- Format is Mermaid `graph TD`.
- Node IDs are sanitized alphanumeric + underscore.
- Edge ordering follows sorted `edges.csv` order.

### 8.6 `exports/agent_context.md`

Required top-level headings (in order):

1. `# Environment Summary`
2. `## Inventory Counts`
3. `## Routing Usage`
4. `## Panorama Overview`
5. `## Topology Highlights`
6. `## Orphans and Unknowns`

## 9) CLI Public Interface Contract

### 9.1 Global flags

- `--repo <path>` default `./default`
- `--env <env_id>` default `default`

### 9.2 Exit codes

- `0`: success
- `2`: usage/validation error (`E_USAGE`)
- `3`: missing dependency (`E_GIT_MISSING`)
- `4`: repo unsafe state (`E_REPO_UNSAFE`)
- `5`: lock held (`E_LOCK_HELD`)
- `6`: ingest fatal parse/system error (`E_PARSE_FATAL|E_IO`)
- `7`: ingest completed with partial parse warnings (`E_PARSE_PARTIAL`)
- `9`: internal error (`E_INTERNAL`)

Exit-code precedence in ingest:

1. fatal/system error present -> `6`
2. else partial parse present -> `7`
3. else -> `0`

### 9.3 `stderr` format

All errors MUST emit one line:

```text
ERROR <error_code> <message>
```

`error_code` allowed values:

- `E_USAGE`
- `E_GIT_MISSING`
- `E_REPO_UNSAFE`
- `E_LOCK_HELD`
- `E_PARSE_FATAL`
- `E_PARSE_PARTIAL`
- `E_IO`
- `E_INTERNAL`

### 9.4 Command contracts

#### `init`

- Usage: `netsec-sk init [--repo <path>]`
- Behavior: initialize Git repo (if missing) and create base directories.
- Stdout success:
  - `Initialized repository: <absolute_repo_path>`
- Errors: `E_GIT_MISSING`, `E_IO`.

#### `env list`

- Usage: `netsec-sk env list [--repo <path>]`
- Stdout success:
  - one `env_id` per line, lexical order, no header.

#### `env create <env_id>`

- Usage: `netsec-sk env create <env_id> [--repo <path>]`
- Stdout success:
  - `Environment created: <env_id>` OR
  - `Environment already exists: <env_id>`
- Errors: `E_USAGE`, `E_IO`.

#### `ingest <paths...>`

- Usage: `netsec-sk ingest <paths...> [--repo <path>] [--env <env_id>] [--rdns] [--keep-extract]`
- Stdout success/partial/fatal summary:
  - `Ingest complete: attempted=<n> committed=<n> skipped_duplicate_tsf=<n> skipped_state_unchanged=<n> parse_error_partial=<n> parse_error_fatal=<n>`
- Prototype ingest execution requirement:
  - `ingest` MUST execute the real ingest pipeline (no placeholder summary implementation).
  - Pipeline stages MUST include: ordering, lock handling, extraction, identity, parse classification, state comparison, persistence, commit creation, and ledger writes.
- Errors: `E_USAGE`, `E_REPO_UNSAFE`, `E_LOCK_HELD`, `E_PARSE_FATAL`, `E_IO`.

#### `export`

- Usage: `netsec-sk export [--repo <path>] [--env <env_id>]`
- Stdout success:
  - `Export complete: <env_id>`
- Errors: `E_USAGE`, `E_IO`.

#### `devices`

- Usage: `netsec-sk devices [--repo <path>] [--env <env_id>]`
- Stdout header + TSV rows sorted by `device_id`:
  - `DEVICE_ID\tHOSTNAME\tMODEL\tSW_VERSION\tMGMT_IP`

#### `panorama`

- Usage: `netsec-sk panorama [--repo <path>] [--env <env_id>]`
- Stdout header + TSV rows sorted by `panorama_id`:
  - `PANORAMA_ID\tHOSTNAME\tMODEL\tVERSION\tMGMT_IP`

#### `show device <device_id>`

- Usage: `netsec-sk show device <device_id> [--repo <path>] [--env <env_id>]`
- Stdout: pretty JSON of `state/devices/<device_id>/latest.json`.

#### `show panorama <panorama_id>`

- Usage: `netsec-sk show panorama <panorama_id> [--repo <path>] [--env <env_id>]`
- Stdout: pretty JSON of `state/panorama/<panorama_id>/latest.json`.

#### `history state`

- Usage: `netsec-sk history state [--repo <path>] [--env <env_id>]`
- Stdout header + TSV rows sorted by `committed_at_utc` ascending, then `git_commit` ascending:
  - `COMMITTED_AT_UTC\tGIT_COMMIT\tTSF_ID\tTSF_ORIGINAL_NAME\tCHANGED_SCOPE`
- Row semantics:
  - one row per state-changing commit in the selected environment,
  - each row MUST include commit hash and source TSF provenance,
  - `CHANGED_SCOPE` MUST summarize changed state paths and include device/feature/route scope where applicable.
- Empty history behavior:
  - print header only and return success (`0`).

#### `topology`

- Usage: `netsec-sk topology [--repo <path>] [--env <env_id>] [--at-commit <hash>]`
- Current mode (no `--at-commit`):
  - read topology from current working tree export `envs/<env_id>/exports/topology.mmd`.
- Historical mode (`--at-commit <hash>`):
  - read topology from the specified Git commit at `envs/<env_id>/exports/topology.mmd` without mutating working tree state.
- Stdout:
  - Mermaid graph text (`graph TD ...`) for the selected state.
- Errors:
  - `E_USAGE` for invalid commit hash format or incompatible arguments.
  - `E_IO` for missing topology artifact or unresolved commit content.

#### `help` and `help <command>`

- `help` prints command list + one-line summaries.
- `help <command>` prints usage, arguments, examples, and exit-code notes.

#### `open`

- Usage: `netsec-sk open [--repo <path>] [--env <env_id>]`
- Prompt: `netsec-sk(env:<env_id>)>`
- Interactive shell supports: `help`, `help <command>`, `env list`, `env create`, `ingest`, `devices`, `panorama`, `show device`, `show panorama`, `exit`, `quit`.
- Shell behavior:
  - Prompt MUST be emitted before each command read.
  - Empty input line is a no-op.
  - `exit`, `quit`, or EOF MUST terminate session with exit code `0`.
  - Command errors MUST emit standard one-line `ERROR ...` output and session MUST continue.
  - For supported commands, one-shot and in-shell outputs MUST be identical for the same effective arguments.

## 10) Commit and Ledger Contracts

### 10.1 Commit content allowlist

When TSF results in commit, stage and commit only:

- `envs/<env_id>/state/commits.ndjson`
- `envs/<env_id>/state/devices/<device_id>/latest.json` or panorama equivalent
- `envs/<env_id>/state/devices/<device_id>/snapshots/<timestamp>_<stateHash>.json` or panorama equivalent
- `envs/<env_id>/exports/environment.json`
- `envs/<env_id>/exports/inventory.csv`
- `envs/<env_id>/exports/nodes.csv`
- `envs/<env_id>/exports/edges.csv`
- `envs/<env_id>/exports/topology.mmd`
- `envs/<env_id>/exports/agent_context.md`

### 10.2 Commit message format

Commit subject MUST be:

```text
ingest(<env_id>): <entity_type>/<entity_id> <state_sha256_12> <tsf_id>
```

- `state_sha256_12` is first 12 chars of hash.
- If `tsf_id` contains spaces, replace with `_`.

### 10.3 `commits.ndjson` schema (git-tracked)

Required fields per line:

- `committed_at_utc`
- `tsf_id`
- `tsf_original_name`
- `entity_type` (`firewall|panorama`)
- `entity_id`
- `state_sha256`
- `git_commit`
- `changed_scope` (string summary used by `history state`)
- `changed_paths` (string array of repo-relative changed state paths, lexical order)
- `summary` (optional)

### 10.4 `.netsec-state/ingest.ndjson` schema (gitignored)

Required fields per line:

- `attempted_at_utc`
- `run_id`
- `env_id`
- `input_archive_path`
- `tsf_id` (if derivable)
- `entity_type` (if derivable)
- `entity_id` (if derivable)
- `result`
- `git_commit` (only when committed)
- `notes` (optional)

### 10.5 Commit operation guarantee

- For every ingest result row with `result = committed`, exactly one Git commit MUST be created in that same ingest run.
- A single committed TSF result MUST NOT map to multiple commits.

## 11) Acceptance Criteria

1. `init` initializes Git repo and base folder structure in selected repo path.
2. `init` does not create environment directories.
3. `env_id` validation/normalization follows Section 5.1 exactly.
4. `ingest --env X` auto-creates environment `X` when missing.
5. Ingest refuses on unsafe Git states (Section 3.2).
6. Lock behavior and stale lock cleanup follow Section 6.2.
7. Extraction cleanup and extraction-safety behavior follow Sections 6.3 and 6.9.
8. Every expanded file input is counted as an attempt (including unsupported extensions) per Sections 3.4 and 6.1.
9. Unsupported extension attempts are logged as `parse_error_fatal` with note `unsupported_extension`.
10. TSF identity and dedupe follow Sections 6.4 and 6.5.
11. Parse result classification follows Sections 6.6 and 6.10 boundaries.
12. State unchanged detection uses Section 7.4 hash rules.
13. Each state-changing TSF creates exactly one commit with Section 10.1 allowlist and Section 10.5 guarantee.
14. Commit message format matches Section 10.2.
15. `open` session executes supported in-shell commands from Section 9.4 and preserves session after non-fatal command errors.
16. At least one real `.tgz` ingest creates `latest.json` at the expected environment/entity path and can be read by `show`.
17. Duplicate TSF and unchanged-state cases produce `skipped_duplicate_tsf` / `skipped_state_unchanged` with no commit.
18. Per-environment exports are generated with exact contracts in Section 8 when `export` is invoked.
19. Topology inference remains IPv4-only and VR-aware with no cross-env edges.
20. Help output exists for all commands and `help <command>` usage/examples.
21. RDNS behavior follows Section 6.8 and writes status fields deterministically.
22. No files under `.netsec-state/` are committed.
23. Environment creation flow is validated for representative IDs: `prod`, `development`, `cloud`, `lab`, `home`, `customera`, `customerb`.
24. Multi-TSF ingest in a single environment records one ingest attempt row per expanded input and deterministic commit/no-commit outcomes.
25. `history state` returns deterministic rows with `git_commit`, `tsf_id`, `tsf_original_name`, and `changed_scope` for each state-changing commit.
26. `history state` covers route/feature-oriented state changes through `changed_scope` summary, not only device identity changes.
27. `topology` returns Mermaid graph output for current state.
28. `topology --at-commit <hash>` returns Mermaid graph output for the specified commit state without mutating working tree state.
29. `export` command writes the full deterministic artifact bundle defined in Section 8 (JSON + CSV + Mermaid + agent context).

## 12) Decisions (All Required Decisions Active)

### D-00001: Canonical spec artifact and version

- Status: Active
- Context: prior spec filename/version were inconsistent.
- Options considered:
  - Keep legacy filename/version.
  - Canonicalize to AGENTS naming/version rules.
- Decision: canonical spec is `docs/spec.v0.6.0.md` with front matter version `0.6.0`.
- Consequences: legacy file remains as superseded reference only.
- Enforcement: CI/spec checks MUST compare filename version to front matter version.

### D-00002: CLI error and exit-code contract

- Status: Active
- Context: prior spec lacked deterministic CLI failure signaling.
- Options considered:
  - Free-form stderr.
  - Structured deterministic line + fixed exit code map.
- Decision: one-line `ERROR <error_code> <message>` format and Section 9.2 exit map.
- Consequences: all commands must map failures into fixed code set.
- Enforcement: command tests MUST validate stderr prefix and exit code.

### D-00003: `env_id` grammar

- Status: Active
- Context: environment naming was unspecified.
- Options considered:
  - unrestricted string.
  - constrained slug.
- Decision: Section 5.1 regex and normalization.
- Consequences: invalid names rejected early.
- Enforcement: input validation unit tests and CLI integration tests.

### D-00004: Batch ingest ordering

- Status: Active
- Context: order-dependent behavior could produce nondeterministic outcomes.
- Options considered:
  - filter unsupported files before ordering.
  - include all expanded files in canonical ordering and classify unsupported files as fatal attempts.
- Decision: Section 3.4 canonical ordering over all expanded files with unsupported-extension fatal classification.
- Consequences: deterministic batch processing and deterministic accounting for mixed input sets.
- Enforcement: integration test with unsorted input arguments.

### D-00005: Parse error taxonomy boundary

- Status: Active
- Context: partial vs fatal parse outcome was ambiguous.
- Options considered:
  - best-effort only.
  - explicit fatal/partial boundaries.
- Decision: Sections 6.6 and 6.10 boundary rules including prototype minimum required identity fields.
- Consequences: consistent ingest result accounting.
- Enforcement: parser tests for each boundary condition.

### D-00006: Lock staleness threshold

- Status: Active
- Context: stale lock behavior needed deterministic criteria.
- Options considered:
  - PID-only check.
  - PID + start time + max age threshold.
- Decision: Section 6.2 criteria with 8-hour max age.
- Consequences: lower chance of false positives from PID reuse.
- Enforcement: lock unit tests for active/stale branches.

### D-00007: Export schema fixed columns and ordering

- Status: Active
- Context: downstream agents/tools need stable tabular contracts.
- Options considered:
  - flexible columns.
  - fixed headers/order.
- Decision: Section 8 fixed CSV headers/order and sorting rules.
- Consequences: strict compatibility expectation for downstream parsers.
- Enforcement: golden-file tests for export outputs.

### D-00008: RDNS timeout/retry policy

- Status: Active
- Context: RDNS behavior could vary by resolver/network.
- Options considered:
  - unlimited retries.
  - deterministic short timeout and bounded retry.
- Decision: 1s timeout + 1 retry in Section 6.8.
- Consequences: predictable ingest time overhead.
- Enforcement: resolver adapter tests with timeout simulation.

### D-00009: Per-TSF commit message format

- Status: Active
- Context: commit messages were not specified for ingest commits.
- Options considered:
  - free-form messages.
  - deterministic template including entity/hash/tsf identity.
- Decision: Section 10.2 template.
- Consequences: predictable history parsing for automation.
- Enforcement: commit integration tests assert exact subject format.

### D-00010: Prototype P1 required command surface

- Status: Active
- Context: full MVP command surface is broader than the immediate prototype build target.
- Options considered:
  - require full in-shell command parity now.
  - require a prototype-focused in-shell set while keeping end-state contracts documented.
- Decision: Section 2.3 defines P1 required one-shot and in-shell command sets.
- Consequences: implementation can prioritize an executable operator loop (`open -> ingest -> query`) without losing end-state MVP intent.
- Enforcement: e2e test matrix MUST validate P1 shell command set and behavior.

### D-00011: `open` shell execution semantics

- Status: Active
- Context: interactive shell behavior was under-specified for prompt cadence and error handling.
- Options considered:
  - shell exits on first command error.
  - shell continues after command errors with deterministic prompt behavior.
- Decision: Section 9.4 `open` shell behavior rules require prompt-per-read, empty-line no-op, and continue-on-error.
- Consequences: resilient interactive operator flow with deterministic session semantics.
- Enforcement: CLI shell tests MUST assert session continuation after invalid command input.

### D-00012: Unsupported extension accounting

- Status: Active
- Context: previous text conflicted on whether unsupported files were filtered out or counted as ingest attempts.
- Options considered:
  - pre-filter unsupported files from attempts.
  - count unsupported files as attempts with explicit fatal classification.
- Decision: Sections 3.4 and 6.1 count unsupported files as attempts and classify as `parse_error_fatal` with `unsupported_extension`.
- Consequences: deterministic ingest summary accounting for mixed file sets.
- Enforcement: ingest tests MUST assert attempted/fatal counters and ingest.ndjson notes for unsupported files.

### D-00013: State history command for user journey provenance

- Status: Active
- Context: operators need deterministic visibility into when state was added/changed and which TSF produced the change.
- Options considered:
  - rely on raw Git commands only.
  - add a deterministic CLI history projection over environment state commits.
- Decision: add `history state` command contract with fixed columns and deterministic sort from Section 10 ledger fields.
- Consequences: commit provenance becomes directly operator-visible without ad-hoc git invocation.
- Enforcement: CLI tests MUST assert deterministic history row shape and ordering.

### D-00014: Historical Mermaid topology by commit hash

- Status: Active
- Context: operators need to view topology for both current and prior states.
- Options considered:
  - current-state topology only.
  - add commit-hash-based historical topology retrieval.
- Decision: `topology` supports optional `--at-commit <hash>` and returns Mermaid graph text for current or specified commit state.
- Consequences: topology investigation workflows become commit-addressable and reproducible.
- Enforcement: tests MUST validate current vs historical topology output and non-mutation guarantee.

## 13) Blocking Questions

None.

As of 2026-02-10, unresolved blocker count is 0.

## 14) Spec Ambiguity Audit

Checklist:

- CLI command arguments, stdout, stderr, and exit codes are fully specified.
- Export JSON/CSV/Mermaid/Markdown contracts are fully specified.
- `env_id` grammar and normalization are specified.
- Lock stale criteria and extract cleanup rules are specified.
- Archive extraction and path-safety rules are specified.
- Parse error taxonomy boundary is specified.
- Prototype parse minimum required identity fields are specified.
- Batch ingest ordering is specified.
- Unsupported extension attempt accounting is specified.
- RDNS timeout/retry behavior is specified.
- Commit message format and commit file allowlist are specified.
- `open` shell prompt cadence and continue-on-error behavior are specified.
- State history journey contract (`history state`) is specified.
- Historical Mermaid topology retrieval (`topology --at-commit`) is specified.

Result:

- Remaining ambiguities: `0`

## 15) Spec Review Round Log (Append-Only)

### SR-00001

- Date: 2026-02-08
- Reviewers:
  - Reviewer A (Ambiguity Hunter)
  - Reviewer B (Edge Cases and Failure Modes)
  - Reviewer C (Security/Abuse)
  - Reviewer D (Testability)
- Outcome: FAIL
- Blockers count: 8
- Blockers found:
  - missing CLI exit-code contract
  - missing stderr error format
  - missing CSV column schemas
  - missing `env_id` grammar
  - missing lock stale criteria
  - missing parse partial/fatal boundary
  - missing deterministic batch ingest order
  - missing RDNS timeout/retry constraints
- Spec deltas applied: documented in `./amendments/AMD-00001-determinism-closure.v0.5.0.md`

### SR-00002

- Date: 2026-02-08
- Reviewers:
  - Reviewer A (Ambiguity Hunter)
  - Reviewer B (Edge Cases and Failure Modes)
  - Reviewer C (Security/Abuse)
  - Reviewer D (Testability)
- Outcome: PASS
- Blockers count: 0
- Non-blockers count: 3
- Summary of changes applied: no additional substantive changes after AMD-00001; wording-only clarity fixes.

### SR-00003

- Date: 2026-02-09
- Reviewers:
  - Reviewer A (Ambiguity Hunter)
  - Reviewer B (Edge Cases and Failure Modes)
  - Reviewer C (Security/Abuse)
  - Reviewer D (Testability)
- Outcome: PASS
- Blockers count: 0
- Non-blockers count: 2
- Summary of changes applied: added prototype P1 command scope, explicit archive extraction safety rules, unsupported-extension attempt accounting, and deterministic `open` shell behavior requirements.

### SR-00004

- Date: 2026-02-10
- Reviewers:
  - Reviewer A (Ambiguity Hunter)
  - Reviewer B (Edge Cases and Failure Modes)
  - Reviewer C (Security/Abuse)
  - Reviewer D (Testability)
- Outcome: PASS
- Blockers count: 0
- Non-blockers count: 2
- Summary of changes applied: added explicit user journeys for environment creation, state history provenance, historical Mermaid topology retrieval by commit hash, and deterministic export bundle expectations.

## 16) Determinism Gate Closure

Gate checklist (per `AGENTS.md`):

- Blocking Questions empty: PASS
- Spec Ambiguity Audit remaining ambiguities = 0: PASS
- Required decisions status Active: PASS
- Human approver recorded: PASS (`cmarks`)

Determinism status: `READY_FOR_BUILD` (effective 2026-02-10).

Further substantive changes to this spec require an amendment before editing this file.

## 17) Deferred (Post-MVP)

- In-shell `export` command requirement for `open`.
- In-shell `topology` command requirement for `open`.
- Packaging automation (Homebrew/Scoop).
- Linux binaries.
- Rich policy/rulebase extraction and visualization.
- Deeper routing telemetry (neighbors/prefix-level runtime details).
