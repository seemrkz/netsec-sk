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
  - `0d1267f` `TASK-00004: implement ingest orchestration and status API`

## TASK-00005
- Date: 2026-02-11
- Type: Added
- Summary: Implemented TSF extraction + classification with required identity fields, explicit `not_found` fallback semantics, and panorama-only payload mapping.
- Verification:
  - `tail -n 1 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -e '.device.device_type and .device.serial and .device.hostname'` -> `true`
  - `jq -e '.devices.logical[] | .current.identity | has("hostname") and has("model") and has("serial") and has("panos_version") and has("mgmt_ip")' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"` -> `true`
  - `jq -e '.devices.logical[] | select(.device_type=="panorama") | .current.panorama.managed_device_serials' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"` -> `["PAFW100","PAFW200"]`
- Commit proof:
  - `e701474` `TASK-00005: implement TSF extraction and device classification`

## TASK-00006
- Date: 2026-02-11
- Type: Added
- Summary: Implemented deterministic state persistence with canonical state shape, atomic writes (`tmp` + sync + rename), `state.json.bak`, and `intro.md` regeneration.
- Verification:
  - `jq -e '.schema_version == "1.0.0" and .generated_at and .env.env_id and .devices.logical and .topology.inferred_adjacencies' "$HOME/.netsec-sk/environments/$ENV_ID/state.json"` -> `true`
  - `rg -n '^# |Quick facts|Where to look in state.json|AI Agent notes|/devices/logical|/topology/inferred_adjacencies|/devices/logical\[i\]/current/network' "$HOME/.netsec-sk/environments/$ENV_ID/intro.md"` -> matched expected sections/pointers
  - `tail -c1 "$HOME/.netsec-sk/environments/$ENV_ID/state.json" | od -An -t x1 | rg -q '0a'` -> exit `0`
- Commit proof:
  - `2c22829` `TASK-00006: implement canonical state persistence and intro generation`

## TASK-00007
- Date: 2026-02-11
- Type: Added
- Summary: Implemented ingest/commit log semantics including fingerprint dedupe, no-change handling, and commit-on-change-only behavior.
- Verification:
  - `tail -n 5 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -s -e 'any(.[]; .status=="duplicate") and any(.[]; .status=="error") and all(.[]; .duration_ms_total >= 0 and .duration_ms_compute >= 0)'` -> `true`
  - `COMMIT_LINES_BEFORE="$(wc -l < "$HOME/.netsec-sk/environments/$ENV_ID/commits.ndjson")" ... COMMIT_LINES_AFTER="$(wc -l < "$HOME/.netsec-sk/environments/$ENV_ID/commits.ndjson")" && test "$COMMIT_LINES_BEFORE" -eq "$COMMIT_LINES_AFTER"` -> exit `0` (`3 == 3`)
- Commit proof:
  - `d1532db` `TASK-00007: implement ingest and commit log semantics`

## TASK-00008
- Date: 2026-02-11
- Type: Added
- Summary: Verified deterministic batch sequencing and continue-on-error semantics via lexicographically ordered per-file ingest submissions.
- Verification:
  - `for f in $(find "$BATCH_DIR" -name '*.tgz' -maxdepth 1 | LC_ALL=C sort); do curl -sS -X POST -F "file=@$f" "$BASE_URL/api/environments/$ENV_ID/ingests" >/dev/null; done`
  - `tail -n 3 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -s -e 'length == 3 and any(.[]; .status=="error")'` -> `true`
  - `tail -n 3 "$HOME/.netsec-sk/environments/$ENV_ID/ingest.ndjson" | jq -r '.source.filenames[0]'` -> `a-ok.tgz`, `b-bad.tgz`, `c-ok.tgz`
- Commit proof:
  - `5679c8b` `TASK-00008: record batch sequencing verification evidence`

## TASK-00009
- Date: 2026-02-11
- Type: Added
- Summary: Implemented RMA awaiting-user flow with candidates, decision handling (`link_replacement|treat_as_new_device|canceled`), final ingest RMA/error fields, and runtime payload cleanup/TTL.
- Verification:
  - `curl -sS "$BASE_URL/api/ingests/$A2" | jq -e '.status=="awaiting_user" and .rma_prompt.required==true and (.rma_prompt.candidates|length)>0'` -> `true`
  - `curl -sS -X POST "$BASE_URL/api/ingests/$A2/rma-decision" -H 'Content-Type: application/json' -d '{"decision":"link_replacement","target_logical_device_id":"'$TARGET'"}' | jq -e '.'` -> valid JSON response
  - `curl -sS "$BASE_URL/api/ingests/$A2" | jq -e '.status=="completed" and .final_record.rma.prompted==true and .final_record.rma.decision=="link_replacement"'` -> `true`
  - `test ! -f "$HOME/.netsec-sk/runtime/ingests/$A2.json"` -> exit `0`
  - `curl -sS "$BASE_URL/api/ingests/$CANCEL_INGEST" | jq -e '.final_record.status=="error" and .final_record.error.code=="ERR_USER_ABORTED" and .final_record.rma.decision=="canceled"'` -> `true`
- Commit proof:
  - Pending
