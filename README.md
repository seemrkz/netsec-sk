# netsec-sk

NetSec StateKit (`netsec-sk`) is a local Go CLI that ingests Palo Alto Networks TSFs
(firewall + Panorama), extracts facts-only state, and stores deterministic snapshots in a
Git-backed repo for change tracking and analysis.

## Status

- Project phase: prototype-focused spec and implementation planning completed; build tasks are defined.
- Canonical spec: `docs/spec.v0.5.1.md`
- Implementation plan: `docs/plan.v0.3.0.md`
- Constitution/governance: `docs/AGENTS.md`
- Legacy draft spec (superseded): `docs/netsec-sk_mvp_spec_v0.4.md`

## Quickstart

Prereqs: `git` installed and available on `PATH`.

Install (pick one):

- From GitHub (requires Go 1.22+): `go install github.com/seemrkz/netsec-sk/cmd/netsec-sk@main`
- From a release artifact: download a `netsec-sk_*` binary from `dist/release/` (or GitHub Releases), then put it on your `PATH`.

Initialize a state repo (this is where `envs/` and snapshots live):

```sh
repo="$(mktemp -d)"
netsec-sk init --repo "$repo"
git -C "$repo" config user.name "NetSecSK"
git -C "$repo" config user.email "netsec-sk@example.com"
netsec-sk env create prod --repo "$repo"
```

Ingest TSFs and export derived views:

```sh
netsec-sk ingest --repo "$repo" --env prod /path/to/tsf1.tgz /path/to/tsf2.tgz
netsec-sk export --repo "$repo" --env prod
netsec-sk devices --repo "$repo" --env prod
netsec-sk show device <DEVICE_ID> --repo "$repo" --env prod
netsec-sk topology --repo "$repo" --env prod
netsec-sk open --repo "$repo" --env prod
```

Note: if you omit `--repo`, the default repo path is `./default`.

## MVP Summary

- Ingest `.tgz` and `.tar.gz` TSFs (mixed firewall + Panorama batches).
- Multi-environment state isolation under `envs/<env_id>/...`.
- One atomic Git commit per TSF when state changes.
- Facts-only snapshots plus exports per environment:
  - `environment.json`
  - `inventory.csv`
  - `nodes.csv`
  - `edges.csv`
  - `topology.mmd`
  - `agent_context.md`
- Optional reverse DNS enrichment for newly discovered devices (`--rdns`).

## Planned CLI Surface (MVP)

- `init`
- `open`
- `help` / `help <command>`
- `env list`
- `env create <env_id>`
- `ingest <paths...> [--env X] [--rdns] [--keep-extract]`
- `export [--env X]`
- `devices`
- `panorama`
- `show device <device_id>`
- `show panorama <panorama_id>`
- `topology`

## Repository Layout

- `docs/` specification, amendment, and plan artifacts.
- `docs/amendments/` applied amendment records.

Runtime state repo layout (created/managed by CLI):

- `envs/<env_id>/state/` authoritative snapshots and commit ledger.
- `envs/<env_id>/exports/` derived JSON/CSV/Mermaid/agent outputs.
- `envs/<env_id>/overrides/` optional topology override inputs.
- `.netsec-state/` tool metadata, locks, and extraction workspace (gitignored).

## Notes

- Git is a hard requirement for ingest flows.
- MVP topology inference is IPv4-only and VR-aware.
- Prototype code scaffolding exists; ingest-to-state runtime is being completed under `docs/plan.v0.3.0.md`.
