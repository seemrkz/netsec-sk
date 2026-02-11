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
  - `48475b1` `TASK-00002: implement environment create list delete APIs`

## TASK-00003
- Date: 2026-02-11
- Type: Added
- Summary: Implemented environment state and commits read endpoints with disk-backed responses and deterministic commit ordering.
- Verification:
  - `curl -sS "$BASE_URL/api/environments/$ENV_ID/state" | jq -e --arg id "$ENV_ID" '.state.schema_version == "1.0.0" and .state.env.env_id == $id'` -> `true`
  - `API_COUNT="$(curl -sS "$BASE_URL/api/environments/$ENV_ID/commits" | jq '.commits | length')" && FILE_COUNT="$(wc -l < "$HOME/.netsec-sk/environments/$ENV_ID/commits.ndjson")" && test "$API_COUNT" -eq "$FILE_COUNT"` -> exit `0` (`API_COUNT=2`, `FILE_COUNT=2`)
  - `curl -sS "$BASE_URL/api/environments/$ENV_ID/commits" | jq -e '.commits as $c | ($c | length) < 2 or ([range(0; ($c|length)-1)] | all($c[.].timestamp >= $c[.+1].timestamp))'` -> `true`
  - `curl -sS "$BASE_URL/api/environments/$MISSING_ID/state" | jq -e '.code == "ERR_ENV_NOT_FOUND"'` -> `true`
  - `curl -sS "$BASE_URL/api/environments/$DELETED_ID/state" | jq -e '.code == "ERR_ENV_ALREADY_DELETED"'` -> `true`
  - `curl -sS "$BASE_URL/api/environments/$NEW_ID/state" | jq -e '.code == "ERR_ENV_STATE_NOT_FOUND"'` -> `true`
- Commit proof:
  - `4f33ebe` `TASK-00003: implement environment state and commits read APIs`

## TASK-00004
- Date: 2026-02-11
- Type: Added
- Summary: Implemented ingest orchestration endpoints with stage/progress/status tracking and completed final-record responses.
- Verification:
  - `INGEST_ID="$(curl -sS -X POST -F "file=@$FIREWALL_TSF" "$BASE_URL/api/environments/$ENV_ID/ingests" | jq -r '.ingest_id')" && test -n "$INGEST_ID"` -> exit `0`
  - `curl -sS "$BASE_URL/api/ingests/$INGEST_ID" | jq -e '.ingest_id and .env_id and .status and .stage and .progress.pct >= 0 and .progress.pct <= 100'` -> `true`
  - `curl -sS "$BASE_URL/api/ingests/$INGEST_ID" | jq -e 'if .status == "completed" then (.final_record.ingest_id == .ingest_id) else true end'` -> `true`
- Commit proof:
  - Pending
