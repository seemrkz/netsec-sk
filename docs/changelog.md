# Changelog

## TASK-00001
- Date: 2026-02-11
- Type: Added
- Summary: Implemented runtime bootstrap with localhost ephemeral bind, runtime metadata emission, and health endpoint.
- Verification:
  - `jq -e '.url and .port and .pid and .started_at and .version' "$HOME/.netsec-sk/runtime/server.json"` -> `true`
  - `BASE_URL="$(jq -r '.url' "$HOME/.netsec-sk/runtime/server.json")" && curl -sS "$BASE_URL/api/health" | jq -e '.version and .started_at and .url'` -> `true`
- Commit proof:
  - `181a68b` `TASK-00001: implement runtime bootstrap and health endpoint`
