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

## TASK-00002
- Date: 2026-02-11
- Type: Added
- Summary: Implemented environment lifecycle APIs for create/list/delete with deterministic `meta.json` persistence and soft-delete to trash.
- Verification:
  - `CREATE_OUT="$(curl -sS -X POST "$BASE_URL/api/environments" -H 'Content-Type: application/json' -d '{"name":"plan-e2e-env","description":"deterministic"}')" && echo "$CREATE_OUT" | jq -e '.env_id and .name=="plan-e2e-env"'` -> `true`
  - `ENV_ID="$(echo "$CREATE_OUT" | jq -r '.env_id')" && curl -sS "$BASE_URL/api/environments" | jq -e --arg id "$ENV_ID" '.environments | any(.env_id == $id)'` -> `true`
  - `curl -sS -X DELETE "$BASE_URL/api/environments/$ENV_ID" | jq -e '.soft_deleted == true and .soft_deleted_at'` -> `true`
  - `curl -sS "$BASE_URL/api/environments" | jq -e --arg id "$ENV_ID" '.environments | all(.env_id != $id)'` -> `true`
  - `test -f "$HOME/.netsec-sk/trash/$ENV_ID/meta.json"` -> exit `0`
  - `curl -sS -X POST "$BASE_URL/api/environments" -H 'Content-Type: application/json' -d '{"name":""}' | jq -e '.code == "ERR_ENV_NAME_REQUIRED"'` -> `true`
- Commit proof:
  - Pending
