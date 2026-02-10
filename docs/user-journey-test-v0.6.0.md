# NetSec StateKit User Journey Test Packet v0.6.0

## Purpose

Validate operator feel and experience for environment setup, TSF ingest provenance, Mermaid topology views (current + historical), and export discoverability.

## Preconditions

- `netsec-sk` is available on `PATH`.
- `git` is available on `PATH`.
- You have two TSF archives for the same environment (`<tsf1.tgz>`, `<tsf2.tgz>`), with the second producing a state change.

## Test Steps

### 1) Initialize repository and git identity

```bash
repo="$(mktemp -d)"
netsec-sk init --repo "$repo"
git -C "$repo" config user.email "tester@example.com"
git -C "$repo" config user.name "Tester"
```

Expected:

- `Initialized repository: <absolute_path>`
- no command errors

### 2) Create representative environments and verify deterministic list

```bash
netsec-sk env create prod --repo "$repo"
netsec-sk env create development --repo "$repo"
netsec-sk env create cloud --repo "$repo"
netsec-sk env create lab --repo "$repo"
netsec-sk env create home --repo "$repo"
netsec-sk env create customera --repo "$repo"
netsec-sk env create customerb --repo "$repo"
netsec-sk env list --repo "$repo"
```

Expected `env list` (lexical order):

```text
cloud
customera
customerb
development
home
lab
prod
```

### 3) Ingest TSFs into `prod`

```bash
netsec-sk ingest --repo "$repo" --env prod <tsf1.tgz>
netsec-sk ingest --repo "$repo" --env prod <tsf2.tgz>
```

Expected:

- Each run prints deterministic ingest summary line.
- At least one run reports `committed=1`.

### 4) Validate state history provenance readability

```bash
netsec-sk history state --repo "$repo" --env prod
```

Expected:

- header:
  - `COMMITTED_AT_UTC    GIT_COMMIT    TSF_ID    TSF_ORIGINAL_NAME    CHANGED_SCOPE`
- rows sorted by `COMMITTED_AT_UTC`, then `GIT_COMMIT`.
- each row includes commit hash + TSF provenance + non-empty `CHANGED_SCOPE`.

### 5) Export full deterministic bundle

```bash
netsec-sk export --repo "$repo" --env prod
ls -1 "$repo/envs/prod/exports"
```

Expected artifacts:

- `environment.json`
- `inventory.csv`
- `nodes.csv`
- `edges.csv`
- `topology.mmd`
- `agent_context.md`

### 6) Validate current topology output UX

```bash
netsec-sk topology --repo "$repo" --env prod
```

Expected:

- Mermaid text output beginning with `graph TD`.

### 7) Validate historical topology from history hash

```bash
history_commit="$(netsec-sk history state --repo "$repo" --env prod | sed -n '2p' | awk -F '\t' '{print $2}')"
status_before="$(git -C "$repo" status --short)"
netsec-sk topology --repo "$repo" --env prod --at-commit "$history_commit"
status_after="$(git -C "$repo" status --short)"
```

Expected:

- historical command outputs Mermaid text (`graph TD`).
- `status_before` equals `status_after` (no working tree mutation).

### 8) Validate error UX quality

```bash
netsec-sk topology --repo "$repo" --env prod --at-commit badhash
netsec-sk topology --repo "$repo" --env missing --at-commit "$history_commit"
```

Expected:

- invalid hash: one-line `ERROR E_USAGE ...`
- missing historical artifact/content: one-line `ERROR E_IO ...`
- messages are deterministic and actionable.

## Pass Criteria

- All steps complete without ambiguity.
- Output formatting is readable and stable across reruns.
- History-to-topology historical lookup feels direct and predictable.
