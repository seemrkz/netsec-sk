# netsec-sk

NetSec StateKit (`netsec-sk`) is a local Go CLI that ingests Palo Alto Networks TSFs
(firewall + Panorama), extracts facts-only state, and stores deterministic snapshots in a
Git-backed repo for change tracking and analysis.

## Status

- Project phase: spec and planning completed; implementation tasks are defined.
- Canonical spec: `docs/spec.v0.5.0.md`
- Implementation plan: `docs/plan.v0.1.0.md`
- Constitution/governance: `docs/AGENTS.md`
- Legacy draft spec (superseded): `docs/netsec-sk_mvp_spec_v0.4.md`

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
- No production code is implemented yet in this repository.
