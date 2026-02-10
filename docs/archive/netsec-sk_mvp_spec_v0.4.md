# NetSec StateKit (NS SK) — MVP Spec (v0.2)

> Superseded on 2026-02-08 by `docs/spec.v0.5.0.md`.
> This file is retained as historical draft input only.

**Tool name:** NetSec StateKit (**NS SK**)  
**Base/tool name (CLI binary):** `netsec-sk`  
**Embedded state repo template:** `netsec-sk` (bundled in CLI for `init`)  
**Default state repo folder name (if not specified):** `default`  
**Default environment name:** `default`  
**Implementation language:** Go  
**MVP distribution:** GitHub Releases (manual download of binaries)

---

## 1) Purpose

`netsec-sk` is a local CLI tool that ingests Palo Alto Networks **Tech Support Files (TSFs)** from **firewalls and Panorama**, extracts a **facts-only** representation of the deployment, and writes it into a **Git-backed state repo**.

It is designed to answer quickly:

- What devices exist (firewalls + Panorama), their models/versions/serials/mgmt IPs?
- How are firewalls deployed (HA, subscriptions/licensing best-effort)?
- What zones/interfaces/virtual routers exist?
- What routing protocols are in use (Static/OSPF/BGP), and are they up (runtime best-effort)?
- What Panorama device groups/templates/template stacks exist, and which devices are in them?
- What changed since last ingest? (via Git history/diffs)

---

## 2) MVP Scope

### In scope
- Ingest **`.tgz`** and **`.tar.gz`** TSFs only
- Ingest mixed batches containing **firewall TSFs and Panorama TSFs**
- Multi-environment support inside a single repo (`envs/<env_id>/...`) with **complete isolation**
- **Git required**; history/diff via Git
- **Atomic commits**: **one git commit per TSF** (when it changes state)
- Facts-only snapshots + exports per environment:
  - JSON
  - generic CSV
  - Mermaid diagram (one per env)
  - agent-friendly markdown summary
- Optional reverse DNS enrichment on *new devices* (`--rdns`)
- Extraction workspace is inside repo, git-ignored, auto-cleaned
- **No concurrency** (sequential ingest)

### Out of scope (MVP)
- Live API polling (no Panorama API / device API)
- Homebrew + Scoop packaging automation (post-MVP)
- Full policy/rulebase graphing
- IP masking/redaction beyond “facts-only” output design (no masking in MVP)
- IPv6 topology inference (IPv4 only for MVP)

---

## 3) User Experience

### 3.1 One-shot mode
Examples:
- `netsec-sk init` (creates `./default` repo)
- `netsec-sk init --repo ./my-state-repo`
- `netsec-sk env create prod --repo ./my-state-repo`
- `netsec-sk ingest ./tsfs --env prod --repo ./my-state-repo`
- `netsec-sk ingest ./tsfs --env prod --rdns --repo ./my-state-repo`
- `netsec-sk export --env prod --repo ./my-state-repo`

### 3.2 Interactive shell
- `netsec-sk open --repo ./my-state-repo --env prod`
- Prompt displays active env: `netsec-sk(env:prod)>`
- Must support `help` and `help <command>`

---

## 4) State Repo Model

### 4.1 Git is required (STRICT)
- `init` must verify `git` exists and is usable.
- Repo must be a git repo (`git init` if needed).
- If git isn’t available → fail.

#### Git working tree policy (STRICT)
Ingest MUST refuse to run if:
- repository is in the middle of merge/rebase/cherry-pick, OR
- there are **any** uncommitted changes (tracked modifications), OR
- there are **any** staged changes

Rationale: `netsec-sk` produces deterministic, atomic commits per TSF. A dirty repo risks mixing human edits with snapshot commits.

(Temporary extraction is stored under `.netsec-state/` which is gitignored and does not count as “dirty”.)

### 4.2 Facts-only rule
- **TSFs are never stored** in the repo.
- The tool extracts needed files temporarily and writes only facts-only JSON/CSV/Mermaid outputs.

---

## 5) Environments (Fully Isolated)

### 5.1 Isolation guarantees
- No dedupe across environments
- No cross-env topology inference
- Same serial can exist in two envs without collision (namespaced by folder)

### 5.2 Environment creation (clarified)
- `init` sets up the base repo only. **No environments exist immediately after `init`.**
- Environments are created explicitly:
  - `env create <env_id>`
  - **or** automatically on ingest: `ingest --env X` creates `X` if missing.

Default env is `default` if no `--env` is provided (and will be created on first ingest).

---

## 6) Repo Layout

At repo root:

- `envs/<env_id>/state/...` — authoritative facts history (git-tracked)
- `envs/<env_id>/exports/...` — derived exports (git-tracked)
- `envs/<env_id>/overrides/...` — optional manual edges / VR equivalence (git-tracked if present)
- `.netsec-state/` — tool metadata + extraction workspace (gitignored)

### 6.1 Per-environment structure
```
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
      topology_links.json        (optional)
      vr_equivalence.json        (optional)
```

### 6.2 Tool metadata + extraction workspace (gitignored)
```
.netsec-state/
  config.json
  extract/
  ingest.ndjson                 (ALL ingest attempts; gitignored)
  lock                          (process lock; gitignored)
```

#### Extraction policy (MVP)
- Extract into `.netsec-state/extract/<run_id>/...`
- Must be git-ignored
- Must be auto-cleaned after each TSF ingest (success or failure)
- On ingest start, remove stale extract dirs (best-effort)
- Optional debug flag `--keep-extract` (still ignored)

#### Locking (MVP)
- `ingest` creates `.netsec-state/lock` containing pid + start time + command.
- If lock exists and owning process is active → refuse to run.
- If lock exists but stale → remove it and proceed (log warning).

---

## 7) Logs & History (Separated, consistent)

### 7.1 `commits.ndjson` (git-tracked; commits only)
Location: `envs/<env_id>/state/commits.ndjson`

- Append-only.
- Contains **only entries that resulted in a git commit** (i.e., state changed).
- Intended as a human/agent-friendly “commit ledger” for that environment.

Each entry includes:
- `committed_at_utc`
- `tsf_id` (see TSF identity below)
- `tsf_original_name`
- `entity_type` (`firewall|panorama`)
- `entity_id`
- `state_sha256` (hash of normalized state; see §9.1)
- `git_commit` (the git commit SHA)
- `summary` (optional short string; no diff details)

### 7.2 `ingest.ndjson` (gitignored; all attempts)
Location: `.netsec-state/ingest.ndjson`

- Append-only.
- Records **every ingest attempt** including skips and parse errors.
- May include `git_commit` when a commit occurred.
- Does **not** record “what changed” (that is represented by git diffs and the `commits.ndjson` ledger).

Each entry includes:
- `attempted_at_utc`
- `env_id`
- `input_archive_path` (path user provided)
- `tsf_id` (if derivable)
- `entity_type` / `entity_id` (if derivable)
- `result` (`committed|skipped_duplicate_tsf|skipped_state_unchanged|parse_error_partial|parse_error_fatal`)
- `git_commit` (present only if committed)
- `notes` (optional)

---

## 8) TSF Identity & Dedupe (Per Environment)

### 8.1 TSF identity (`tsf_id`) — `/tmp/cli` filename + serial only

MVP duplicate detection uses a TSF identity derived from **TSF-internal metadata** (not archive filename and not file-content hashing). This is explicitly designed to handle cases where a user renames the `.tgz/.tar.gz` archive.

For MVP, `tsf_id` MUST be computed using only:
- `tsf_original_name`: derived from the **filename** of a CLI metadata text file found under `tmp/cli/` inside the TSF (see §8.1.1)
- `serial`: parsed from the **contents** of that same file (see §8.1.2)

No other signals (e.g., `show jobs processed`) are used for TSF identity in MVP.

#### 8.1.1 Locating the `/tmp/cli` metadata file (MUST)
The TSF contains CLI outputs under a path rooted at `tmp/cli/` (often nested under a top-level directory prefix inside the tar). `netsec-sk` MUST locate a **single** “metadata text file” by scanning archive member paths.

**Candidate path patterns (in priority order):**
1) Any text file directly under `tmp/cli/`:
   - `(^|.*/)?tmp/cli/[^/]+\.txt$`
2) If none found, allow any text file under `tmp/cli/` (one level or deeper):
   - `(^|.*/)?tmp/cli/.*\.txt$`

**Selection rule if multiple candidates exist:**
- Prefer the candidate that contains a `serial:` line (per §8.1.2).
- If multiple contain `serial:`, choose the first one in lexical order by full member path (deterministic).

The selected archive member path is referred to as `cli_metadata_path`.

#### 8.1.2 Parsing rules (MUST be best-effort and robust)
All parsing is line-oriented and MUST tolerate variable whitespace.

**A) Derive `tsf_original_name` from the filename**
Let `cli_metadata_filename` be the basename of `cli_metadata_path` (e.g., `PA-440_ts.tgz.txt`).

Derivation rules:
- If `cli_metadata_filename` ends with `.tgz.txt`, then `tsf_original_name = cli_metadata_filename` with the trailing `.txt` removed.
- If `cli_metadata_filename` ends with `.tar.gz.txt`, then `tsf_original_name = cli_metadata_filename` with the trailing `.txt` removed.
- Else, if `cli_metadata_filename` contains `_ts.tgz` or `_ts.tar.gz` anywhere, extract the shortest substring that matches:
  - `[A-Za-z0-9._-]+_ts\.(?:tgz|tar\.gz)`
- Else, set `tsf_original_name = cli_metadata_filename` (as-is).

**B) Extract `serial` from the file contents**
- Primary pattern (case-insensitive):
  - `\bserial\s*:\s*(?P<serial>[A-Za-z0-9]+)\b`
- Secondary patterns (case-insensitive; accept first match):
  - `\bserial\s*number\s*:\s*(?P<serial>[A-Za-z0-9]+)\b`
  - `\bdevice\s*serial\s*:\s*(?P<serial>[A-Za-z0-9]+)\b`

#### 8.1.3 TSF identity construction
`tsf_id` is a stable string:
```
tsf_id = "<serial>|<tsf_original_name>"
```
If either field is missing, it is left blank, but the delimiter remains.

**Fallback behavior:**
- If no `/tmp/cli` metadata text file can be located, set `tsf_id = "unknown"` and do not apply duplicate-TSF skip (state-unchanged skip may still apply).
- If `serial` cannot be parsed, still construct `tsf_id` with empty serial (e.g., `"|<tsf_original_name>"`).


### 8.2 Duplicate TSF detection (per env)
Before deep parsing, `ingest` checks whether the computed `tsf_id` has already been seen **in that environment**.

- Seen TSFs are tracked in `.netsec-state/ingest.ndjson` (gitignored).
- If `tsf_id` already seen in that env → skip (`skipped_duplicate_tsf`), no commit.

### 8.3 State unchanged detection (per env)
Even for a new TSF, the resulting state may be identical.

- Compute `state_sha256` (§9.1).
- If `state_sha256` equals the current `latest.json` state hash for that entity in that env → skip (`skipped_state_unchanged`), no commit.
- Record the attempt in `.netsec-state/ingest.ndjson`.

---

## 9) Data Model (Schema v1)

### 9.1 Hashes (explicit)
Two distinct concepts:

- `tsf_id`: identity of a TSF for duplicate detection, derived from `/tmp/cli` metadata filename + serial. (§8)
- `state_sha256`: hash of the **normalized facts-only state**, used to detect “no state change”.

#### `state_sha256` canonicalization (MVP)
- Exclude volatile fields:
  - all `source.*` fields (including `ingested_at_utc` and `tsf_id`)
  - the hash field itself
- Deterministic ordering:
  - interfaces sorted by `name`
  - zones sorted by `name`
  - VRs sorted by `name`
  - licenses sorted by `name`
  - Panorama group/template/stacks sorted by `name`
  - member serial lists sorted
- JSON canonicalization:
  - stable key ordering
  - stable whitespace/encoding

### 9.2 Identity fields (clarified)
Yes, both are required:

- `device.id` / `panorama_instance.id`: internal stable ID used for filesystem paths and cross-file references.
- `device.serial` / `panorama_instance.serial`: the vendor serial number, which may be missing in partial parses.

**Rule:** if serial is present, set `id = serial`. If serial is not present, derive `id` from hostname or other stable attributes and keep `serial` empty.

### 9.3 Shared snapshot fields
- `snapshot_version: 1`
- `source`:
  - `tsf_id`
  - `tsf_original_name`
  - `input_archive_name`
  - `ingested_at_utc`
- `state_sha256`

### 9.4 Firewall snapshot
- `device`: `id`, `hostname`, `serial`, `model`, `sw_version`, `mgmt_ip`
- `ha`: `enabled`, `mode`, `local_state`, `peer_serial` (best-effort)
- `licenses[]`: `{name, status, expires_on}` (best-effort)
- `network`:
  - `interfaces[]`: `{name, ip_cidrs[], zone, virtual_router}`
  - `zones[]`: `{name, type, interfaces[]}`
- `routing`:
  - `virtual_routers[]`:
    - `name`
    - `config.protocols_configured[]` (subset: `static`, `ospf`, `bgp`)
    - `runtime.protocols_active[]` (best-effort)
    - `counts.static_routes_configured_v4/v6`
    - `counts.static_routes_installed_v4/v6` (best-effort)
    - `runtime.health` (optional, best-effort counts)
    - `runtime.vr_context`: `known|unknown`
      - If runtime output cannot be attributed to a VR confidently, mark as `unknown` and store any protocol “up” signals in `routing.runtime_unknown` or a VR entry with `name:"unknown"`.

**Optional RDNS (if enabled, and device is newly discovered):**
- `device.dns.reverse`: `{ip, ptr_name, status, looked_up_at_utc}`

### 9.5 Panorama snapshot
- `panorama_instance`: identity fields (id/hostname/serial/model/version/mgmt_ip)
- `panorama_ha`: best-effort
- `panorama_config`:
  - `device_groups[]`: `{name, parent?, members_serials[]}`
  - `templates[]`: `{name, members_serials[]}`
  - `template_stacks[]`: `{name, templates[], members_serials[]}`
  - `managed_devices[]`: `{serial, hostname?, model?, connected_status?}` (best-effort)

### 9.6 Derived environment view (`exports/environment.json`)
Contains latest view only:
- `environment`: `{env_id, generated_at_utc, counts...}`
- `firewalls[]` (latest snapshots)
- `panorama[]` (latest snapshots)
- `topology.zone_edges[]` (VR-aware; IPv4 only for MVP)

---

## 10) Connectivity Inference (VR-aware, IPv4 only)

### 10.1 MVP rule (IPv4 only)
Create a zone-to-zone edge only if:
1) Interfaces share the same **IPv4** subnet, **and**
2) Interfaces are in the same VR context

### 10.2 VR equivalence (optional override)
If `envs/<env>/overrides/vr_equivalence.json` exists, it may declare VR name equivalences.

### 10.3 Manual topology links (optional override)
If `envs/<env>/overrides/topology_links.json` exists, it may add edges not inferable from subnets.

Edge records include:
- endpoint A: device_id, zone, interface, vr
- endpoint B: device_id, zone, interface, vr
- evidence: subnet + interface names
- type: `shared_subnet` or `manual_override`

---

## 11) Exports (Per Environment)

### JSON
- `exports/environment.json` — rolled-up “world view” for tools/agents

### CSV (generic)
- `exports/inventory.csv` — flat inventory + routing protocols + HA
- `exports/nodes.csv` and `exports/edges.csv` — generic graph import

### Mermaid
- `exports/topology.mmd` — one diagram per env

### Agent summary
- `exports/agent_context.md` — concise summary:
  - inventory counts
  - routing usage summary (configured vs active; unknown VR runtime called out)
  - Panorama DG/templates/stacks overview
  - topology highlights + orphans

---

## 12) CLI Commands (MVP)

Global flags:
- `--repo <path>` (default `./default`)
- `--env <env_id>` (default `default`)

Commands:
- `init`
- `open`
- `help` (shell and one-shot)
- `env list`
- `env create <env_id>`
- `ingest <paths...> [--env X] [--rdns] [--keep-extract]`
- `export [--env X]`
- `devices` (list firewalls in env)
- `panorama` (list panorama instances in env)
- `show device <device_id>`
- `show panorama <panorama_id>`
- `topology` (summary)

Help requirements:
- `help` shows command list + summaries
- `help <command>` shows usage + examples

---

## 13) Atomic Commit Contents (per TSF) — bullet list (explicit)

When a TSF results in a commit, the commit MUST include **only**:
- `envs/<env_id>/state/commits.ndjson` (new commit ledger entry appended)
- `envs/<env_id>/state/devices/<device_id>/snapshots/<timestamp>_<stateHash>.json` (or panorama equivalent)
- `envs/<env_id>/state/devices/<device_id>/latest.json` (or panorama equivalent)
- `envs/<env_id>/exports/environment.json`
- `envs/<env_id>/exports/inventory.csv`
- `envs/<env_id>/exports/nodes.csv`
- `envs/<env_id>/exports/edges.csv`
- `envs/<env_id>/exports/topology.mmd`
- `envs/<env_id>/exports/agent_context.md`

Never stage/commit:
- anything under `.netsec-state/`
- TSF archives
- extracted raw files

---

## 14) Distribution (MVP)

### GitHub Releases only
- Provide binaries for:
  - macOS (arm64 + amd64)
  - Windows (amd64)
- Provide checksums

Installation is “download from GitHub release page and place on PATH.”

---

## 15) MVP Acceptance Criteria (updated)

1) `init` initializes a git repo and base folder structure in `./default` (or `--repo` path).
2) `init` does not create an environment folder; envs are created via `env create` or first `ingest --env`.
3) Git is required; ingest refuses to run on dirty repos or unsafe git states.
4) No TSF files or raw extracted contents are committed to git.
5) Extraction workspace is inside repo, gitignored, lock-protected, and auto-cleaned every ingest.
6) `ingest` supports directories and mixed batches of firewall + panorama TSFs.
7) `ingest --env X` auto-creates env X if it does not exist.
8) Environments are isolated: no shared dedupe, no shared exports, no cross-env topology edges.
9) TSF duplicate detection uses `tsf_id` derived from the `/tmp/cli` metadata filename + serial found within that file; renamed archives still dedupe.
10) State unchanged detection uses `state_sha256` canonicalization (source metadata excluded).
11) Each TSF that changes state produces exactly **one atomic git commit** containing only the files in §13.
12) Routing:
   - config-derived protocols always attempted
   - runtime routing fields populated best-effort; if VR context is unclear, mark as `unknown`.
13) Panorama ingestion populates DGs, templates, template stacks, and membership mappings.
14) Exports are produced per env: environment.json, inventory.csv, nodes.csv, edges.csv, topology.mmd, agent_context.md.
15) IPv4-only topology inference is explicit for MVP.

---

## 16) Deferred (post-MVP)
- Homebrew + Scoop packaging automation
- Linux support
- Rich policy/rulebase parsing and visualization
- Deeper routing telemetry (neighbors, prefixes, etc.)
- Generated changelog artifacts beyond Git-native diffs
