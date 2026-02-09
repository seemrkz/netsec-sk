# Changelog

## TASK-00001

- Date: 2026-02-09
- Type: Added
- Summary: Implemented CLI root entrypoint, global flags, and centralized error/exit framework.
- Files:
  - `go.mod`
  - `cmd/netsec-sk/main.go`
  - `internal/cli/root.go`
  - `internal/cli/errors.go`
  - `internal/cli/root_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/cli -run TestGlobalFlags` -> `ok   github.com/seemrkz/netsec-sk/internal/cli (cached)`
  - `go test ./internal/cli -run TestExitCodeMapping` -> `ok   github.com/seemrkz/netsec-sk/internal/cli (cached)`

## TASK-00002

- Date: 2026-02-09
- Type: Added
- Summary: Implemented `init` repository bootstrap with Git prerequisite checks and deterministic base layout creation.
- Files:
  - `internal/repo/git_check.go`
  - `internal/repo/layout.go`
  - `internal/repo/init.go`
  - `internal/repo/init_test.go`
  - `internal/cli/root.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/repo -run TestInitCreatesBaseLayout` -> `ok   github.com/seemrkz/netsec-sk/internal/repo`
  - `go test ./internal/repo -run TestInitFailsWithoutGit` -> `ok   github.com/seemrkz/netsec-sk/internal/repo`

## TASK-00003

- Date: 2026-02-09
- Type: Added
- Summary: Implemented environment ID normalization/validation and `env list`/`env create` command contracts.
- Files:
  - `internal/env/validate.go`
  - `internal/env/service.go`
  - `internal/env/service_test.go`
  - `internal/cli/root.go`
  - `internal/cli/root_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/env -run TestEnvIDValidation` -> `ok   github.com/seemrkz/netsec-sk/internal/env`
  - `go test ./internal/cli -run TestEnvCommandOutputs` -> `ok   github.com/seemrkz/netsec-sk/internal/cli`

## TASK-00004

- Date: 2026-02-09
- Type: Added
- Summary: Implemented ingest runtime skeleton for deterministic input ordering, lock stale/active handling, extract workspace lifecycle cleanup, and ingest-time environment auto-create.
- Files:
  - `internal/ingest/orchestrator.go`
  - `internal/ingest/lock.go`
  - `internal/ingest/ingest_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/ingest -run TestInputOrdering` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`
  - `go test ./internal/ingest -run TestLockStaleRules` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`
  - `go test ./internal/ingest -run TestExtractCleanup` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`

## TASK-00005

- Date: 2026-02-09
- Type: Added
- Summary: Implemented TSF internal identity derivation and environment-scoped duplicate detection from `.netsec-state/ingest.ndjson`.
- Files:
  - `internal/tsf/identity.go`
  - `internal/tsf/identity_test.go`
  - `internal/ingest/orchestrator.go`
  - `internal/ingest/ingest_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/tsf -run TestIdentityDerivation` -> `ok   github.com/seemrkz/netsec-sk/internal/tsf`
  - `go test ./internal/ingest -run TestDuplicateDetection` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`

## TASK-00006

- Date: 2026-02-09
- Type: Added
- Summary: Implemented firewall and panorama parser skeletons with deterministic classifier and partial/fatal parse taxonomy boundaries.
- Files:
  - `internal/parse/classifier.go`
  - `internal/parse/snapshots.go`
  - `internal/parse/parse_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/parse -run TestFirewallSnapshotRequiredFields` -> `ok   github.com/seemrkz/netsec-sk/internal/parse`
  - `go test ./internal/parse -run TestPanoramaSnapshotRequiredFields` -> `ok   github.com/seemrkz/netsec-sk/internal/parse`
  - `go test ./internal/parse -run TestParseErrorClassification` -> `ok   github.com/seemrkz/netsec-sk/internal/parse`

## TASK-00007

- Date: 2026-02-09
- Type: Added
- Summary: Implemented optional RDNS enrichment for newly discovered firewalls with deterministic 1s timeout, single retry policy, and status mapping.
- Files:
  - `internal/enrich/rdns.go`
  - `internal/enrich/rdns_test.go`
  - `internal/ingest/orchestrator.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/enrich -run TestRDNSOnlyForNewDevices` -> `ok   github.com/seemrkz/netsec-sk/internal/enrich`
  - `go test ./internal/enrich -run TestRDNSTimeoutRetry` -> `ok   github.com/seemrkz/netsec-sk/internal/enrich`

## TASK-00008

- Date: 2026-02-09
- Type: Added
- Summary: Implemented canonical state normalization/hash and unchanged-state comparison against `latest.json`.
- Files:
  - `internal/state/hash.go`
  - `internal/state/compare.go`
  - `internal/state/state_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/state -run TestHashCanonicalization` -> `ok   github.com/seemrkz/netsec-sk/internal/state`
  - `go test ./internal/state -run TestUnchangedStateSkip` -> `ok   github.com/seemrkz/netsec-sk/internal/state`

## TASK-00009

- Date: 2026-02-09
- Type: Added
- Summary: Implemented IPv4-only, VR-aware shared-subnet topology inference and deterministic override merge as `manual_override` edges.
- Files:
  - `internal/topology/infer.go`
  - `internal/topology/infer_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/topology -run TestInferSharedSubnetEdges` -> `ok   github.com/seemrkz/netsec-sk/internal/topology`
  - `go test ./internal/topology -run TestOverrideMerge` -> `ok   github.com/seemrkz/netsec-sk/internal/topology`

## TASK-00010

- Date: 2026-02-09
- Type: Added
- Summary: Implemented deterministic export writers for `environment.json`, CSV outputs, Mermaid topology, and `agent_context.md` heading contract.
- Files:
  - `internal/export/writers.go`
  - `internal/export/writers_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/export -run TestEnvironmentJSONSchema` -> `ok   github.com/seemrkz/netsec-sk/internal/export`
  - `go test ./internal/export -run TestCSVHeadersAndOrdering` -> `ok   github.com/seemrkz/netsec-sk/internal/export`
  - `go test ./internal/export -run TestAgentContextTemplate` -> `ok   github.com/seemrkz/netsec-sk/internal/export`

## TASK-00011

- Date: 2026-02-09
- Type: Added
- Summary: Implemented commit allowlist/subject builders plus ingest and commit NDJSON ledger appenders.
- Files:
  - `internal/commit/committer.go`
  - `internal/commit/committer_test.go`
  - `internal/state/commits_ledger.go`
  - `internal/ingest/orchestrator.go`
  - `internal/ingest/ingest_test.go`
  - `docs/plan.v0.1.0.md`
- Verification:
  - `go test ./internal/commit -run TestAtomicCommitAllowlist` -> `ok   github.com/seemrkz/netsec-sk/internal/commit`
  - `go test ./internal/commit -run TestCommitMessageFormat` -> `ok   github.com/seemrkz/netsec-sk/internal/commit`
  - `go test ./internal/ingest -run TestIngestLedgerAllAttempts` -> `ok   github.com/seemrkz/netsec-sk/internal/ingest`
