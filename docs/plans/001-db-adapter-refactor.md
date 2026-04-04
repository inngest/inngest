# Database Adapter Refactor Plan

## Overview

Refactor the CQRS database layer to use a clean adapter pattern, eliminating `isPostgres()` branching and the 1,300-line Postgres normalization layer. Each database dialect gets its own adapter that converts dialect-specific sqlc types into unified domain types.

## Package Structure

```
pkg/db/                     # Domain types, interfaces, adapter contracts
  models.go                 # 14 domain model structs + 4 composite row types
  params.go                 # 25 parameter structs for queries
  querier.go                # Querier interface (all DB operations)
  adapter.go                # Adapter, TxAdapter, DialectHelpers interfaces
  models_test.go            # Domain type tests
  sqlite/                   # SQLite adapter implementation
    adapter.go              # Adapter + TxAdapter
    helpers.go              # DialectHelpers (json_each, time parsing, etc.)
    convert.go              # sqlc SQLite types -> domain types
    querier.go              # Querier implementation wrapping sqlc
  postgres/                 # PostgreSQL adapter implementation
    adapter.go              # Adapter + TxAdapter
    helpers.go              # DialectHelpers (jsonb_array_elements, RFC3339, etc.)
    convert.go              # sqlc Postgres types -> domain types
    querier.go              # Querier implementation wrapping sqlc
  mysql/                    # MySQL stub adapter (not yet implemented)
    adapter.go              # Stub that panics on Q()/Helpers()
```

## Dependency Graph

```
pkg/db ──> pkg/run (for ExprSQLConverter)
         ──> goqu (for dynamic SQL helpers)

pkg/db/sqlite ──> pkg/db (domain types + interfaces)
               ──> pkg/cqrs/base_cqrs/sqlc/sqlite (generated queries)

pkg/db/postgres ──> pkg/db (domain types + interfaces)
                 ──> pkg/cqrs/base_cqrs/sqlc/postgres (generated queries)

pkg/cqrs/base_cqrs ──> pkg/db (via Adapter interface, future)
```

No import cycles: `pkg/db` is a leaf package that doesn't import `pkg/cqrs`.

## Phases

### Phase 1: Domain Types (DONE)
- [x] Define database-agnostic model types (`App`, `Event`, `Function`, etc.)
- [x] Define parameter structs for all write/query operations
- [x] Define `Querier` interface using domain types
- [x] Write unit tests for type correctness

### Phase 2: Adapter Interfaces & Implementations (DONE)
- [x] Define `Adapter`, `TxAdapter`, `DialectHelpers` interfaces
- [x] Implement SQLite adapter (`pkg/db/sqlite/`)
- [x] Implement PostgreSQL adapter (`pkg/db/postgres/`)
- [x] Create MySQL stub adapter (`pkg/db/mysql/`)
- [x] Verify all packages build cleanly

### Phase 3: Wire Adapter into base_cqrs (DONE)
- [x] Update `base_cqrs.New()` to accept `db.Adapter`
- [x] Replace `isPostgres()` calls with `adapter.Helpers()` dispatch
- [x] Remove `spanRunsAdapter` struct (replaced by `DialectHelpers`)
- [x] Remove `NormalizedQueries` wrapper (~882 lines)
- [x] Remove `normalization.go` converters (~459 lines)
- [x] Update `devserver.go` and other instantiation sites

### Phase 4: Integration Tests (DONE)
- [x] Add adapter-level integration tests using SQLite in-memory DB
- [x] Add Postgres integration tests via testcontainers-go (run with TEST_DATABASE=postgres)
- [x] Tests cover: adapter contract, App/Function/Event/Span/History CRUD, transactions

### Phase 5: Migration Tooling (Follow-up)

Current status:
- [x] Switched `base_cqrs` runtime migrations to goose
- [x] Added idempotent baseline migrations for SQLite and Postgres under `pkg/db/{sqlite,postgres}/migrations/`
- [x] Added migration verification tests covering fresh DB setup, idempotency, schema parity, and legacy-to-goose no-op compatibility
- [ ] Remove golang-migrate and delete legacy incremental migration files in a follow-up PR

**Decision: Replace golang-migrate with pressly/goose v3.**

#### 5.1 — Why goose over golang-migrate

| Criteria             | golang-migrate v4.16.2                                                                                                  | pressly/goose v3                                                  |
|----------------------|-------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------|
| Dirty-state recovery | Manual and fragile — current code re-runs the dirty version, which fails on non-idempotent DDL (`base_cqrs.go:203-207`) | Built-in dirty-state handling                                     |
| Go-coded migrations  | Not supported (SQL-only)                                                                                                | Native — `goose.AddMigrationNoTx()` for Go funcs alongside SQL    |
| Transaction control  | Per-migration opt-out (`NoTxWrap`)                                                                                      | Per-migration `-- +goose NO TRANSACTION` annotation               |
| Programmatic API     | Minimal                                                                                                                 | `goose.Provider` — clean API, easier test setup                   |
| Maintenance          | Slow — last release Aug 2023, 200+ open issues                                                                          | Active — regular releases                                         |
| Version table        | Configurable (`migrations` currently)                                                                                   | `goose_db_version` by default, configurable via `WithTableName()` |

#### 5.2 — Consolidate & re-number migrations

Current state: 72 migration files (36 per dialect), with numbering that **diverges at 000007** between SQLite and Postgres (SQLite has `function_runs_workspace_id` at 007; Postgres has a different `connect` at 007 and a Postgres-only `add_pg_indexes` at 012).

**Approach — squash to an idempotent baseline:**

1. Create a single `000001_baseline.sql` per dialect containing the full current schema using `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS`. Source from the existing `sqlc/{dialect}/schema.sql` files which represent ground truth.
2. Delete the 72 incremental migration files (000001–000018 × up/down × 2 dialects).
3. Future migrations start at `000002` with aligned numbering across both dialects.

Because the baseline is fully idempotent, no bridge from golang-migrate is needed. On an existing database, every `CREATE IF NOT EXISTS` statement is a no-op and goose simply records the baseline as applied. On a fresh database, it creates everything. The orphaned golang-migrate `migrations` table can be dropped in the baseline or left inert.

#### 5.3 — Implementation steps

- [x] **Step 1: Add goose dependency** — `go get github.com/pressly/goose/v3`
- [x] **Step 2: Create baseline migration files**
  ```
  pkg/db/
    sqlite/migrations/000001_baseline.sql
    postgres/migrations/000001_baseline.sql
  ```
  Each file uses goose annotations:
  ```sql
  -- +goose Up
  CREATE TABLE IF NOT EXISTS apps ( ... );
  CREATE TABLE IF NOT EXISTS events ( ... );
  -- ... all tables, indexes ...

  -- +goose Down
  DROP TABLE IF EXISTS ... ;
  ```
- [x] **Step 3: Rewrite `up()` in `base_cqrs.go`** — Replace the current 75-line function (lines 140-215) with goose's `Provider` API (~15 lines). This eliminates:
  - Fragile dirty-state recovery logic
  - `NoTxWrap` workaround for SQLite
  - Separate `source.Driver` / `database.Driver` plumbing
  - Manual `migrate.ErrNoChange` handling
- [x] **Step 4: Update embed directive** — Implemented via adapter-local embedded migration files under `pkg/db/{sqlite,postgres}`.
- [x] **Step 5: Update test utilities** — Existing `base_cqrs.New()` call sites remain unchanged; compatibility testing uses a dedicated harness.
- [x] **Step 6: Add dedicated migration tests** — Added:
  - `TestBaselineOnFreshDB` — applies baseline to empty in-memory SQLite, verifies all tables exist
  - `TestBaselineOnFreshPostgres` — same, on testcontainer Postgres
  - `TestMigrationIdempotency` — runs `up()` twice on the same DB, second call is a no-op
  - `TestSchemaMatchesSqlc` — compares table/column list after migrations against expected schema from `sqlc/{dialect}/schema.sql`
- [ ] **Step 7: Clean up** — Remove `github.com/golang-migrate/migrate/v4` from `go.mod`/`go.sum`, delete the old 72 incremental migration files. Deferred intentionally so this PR can prove legacy migrations followed by goose baseline produce no schema changes.

#### 5.4 — Risks & mitigations

| Risk                                 | Mitigation                                                                                          |
|--------------------------------------|-----------------------------------------------------------------------------------------------------|
| Existing databases miss the baseline | Baseline uses `IF NOT EXISTS` throughout — fully idempotent, safe on any DB state                   |
| goose API instability                | Pin to a specific v3.x tag; goose v3 API has been stable since 2023                                 |
| SQLite transaction limitations       | goose supports `-- +goose NO TRANSACTION` annotation, replacing golang-migrate's `NoTxWrap`         |
| Two-dialect maintenance persists     | Inherent to SQLite + Postgres DDL differences; squashed baseline reduces surface from 72 files to 2 |

#### 5.5 — Estimated scope

| Step                               | Effort                                                         |
|------------------------------------|----------------------------------------------------------------|
| Step 1: Add goose dep              | Trivial                                                        |
| Step 2: Create baseline migrations | Small — copy from existing `schema.sql`, add goose annotations |
| Step 3: Rewrite `up()`             | Small — ~15 lines replacing ~75 lines                          |
| Step 4: Update embed               | Trivial                                                        |
| Step 5: Update tests               | Small                                                          |
| Step 6: Add migration tests        | Medium                                                         |
| Step 7: Clean up                   | Small                                                          |

### Phase 6: Database Regression Test Suite

PR #3945 exposed a critical gap: the goose migration switchover introduced a single schema regression (`NOT NULL constraint failed: apps.created_at`) that cascaded into **30+ test failures** across 5 packages and both dialects. The existing tests caught the symptom but only at the integration/E2E level — there was no fast, isolated test that would have flagged the schema mismatch before it reached those layers.

#### 6.1 — CI failure analysis (PR #3945)

**26 of 48 CI checks failed.** Broken down by suite:

| Suite         | Failed / Total | Dialects                      |
|---------------|----------------|-------------------------------|
| Go Tests      | 8 / 8          | All SQLite and all Postgres   |
| Go SDK E2E    | 12 / 10+2 pg   | All SQLite + Postgres split:1 |
| TS SDK E2E    | 2 / 2          | SQLite-only (default)         |
| API E2E       | 2 / 4          | SQLite-only                   |
| Execution E2E | 2 / 2          | SQLite (default)              |

**Three distinct root causes identified:**

**Root Cause A — `NOT NULL constraint failed: apps.created_at` (1299)** — Dominant failure. The goose baseline defines `created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP` but the `UpsertApp` SQL query does not supply a value for `created_at` on INSERT, relying on the DEFAULT. However, goose wraps SQLite migrations in a transaction by default, and SQLite's `DEFAULT CURRENT_TIMESTAMP` may behave differently under goose's transaction handling vs golang-migrate's `NoTxWrap: true`. This single error cascades into every test that calls `UpsertApp`, which is effectively all database-touching tests.

Affected packages and tests:
| Failed Package             | Dialect | Test Count                                                                                                                                                                                                                                                                                                               |
|----------------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `pkg/cqrs/base_cqrs`       | SQLite  | ~15 (TestCQRSGetApps, TestCQRSGetAppByChecksum, TestCQRSGetAppByID, TestCQRSGetAppByURL, TestCQRSGetAppByName, TestCQRSGetAllApps, TestCQRSUpsertApp/*, TestCQRSUpdateAppError, TestCQRSUpdateAppURL, TestCQRSDeleteApp, TestCQRSGetFunctionByInternalUUID, TestCQRSGetFunctionsByAppInternalID, TestCQRSInsertFunction) |
| `pkg/db`                   | SQLite  | All (timed out at 600s)                                                                                                                                                                                                                                                                                                  |
| `pkg/devserver`            | Both    | TestRegister_FunctionVersionIncrement, TestRegister_DuplicateAppCleanup/*                                                                                                                                                                                                                                                |
| `tests/execution/executor` | SQLite  | TestInvokeRetrySucceedsIfPauseAlreadyCreated, TestExecutorReturnsResponseWhenNonRetriableError, TestCapacityErrorRetriesWhenAttemptsExhausted, TestExecutorScheduleRateLimit, TestExecutorScheduleBacklogSizeLimit                                                                                                       |
| `tests` (TS SDK E2E)       | SQLite  | TestSDKCancelNotReceived, TestSDKCancelReceived, TestCancelFunctionViaAPI, TestSDKFunctions, TestSDKNoRetry, TestSDKRetry, TestSDKSteps, TestSDKWaitForEvent_WithEvent, TestSDKWaitForEvent_NoEvent ("Expected executor request but timed out" — server can't start due to DB error)                                     |

**Root Cause B — Go SDK Postgres split:1 failures** — Different error: `err with gql: "event not found: ..."` in `TestEvent/found` and `status didn't match` in `TestFnCheckpoint`, `TestEventList/internal_events`. These appear to be query/API-level regressions on Postgres, possibly related to the adapter conversion layer rather than schema.

**Root Cause C — Pre-existing flaky tests (unrelated):**
| Test | Package | Issue |
|---|---|---|
| `TestHeartbeatDuringGatewayDrain_ClosesConnection` | `pkg/connect` | Timing-dependent, "Condition never satisfied" |
| `TestQueueItemProcessWithConstraintChecks` | `pkg/execution/state/redis_state` | Redis constraint check race |

#### 6.2 — Testing gaps identified

1. **No schema validation test** — Nothing compares the migration end-state against the sqlc schema files, which are the source of truth for query codegen. This would have caught Root Cause A instantly.
2. **No INSERT round-trip tests for every table** — The adapter integration tests (`pkg/db/adapter_integration_test.go`) cover App/Function/Event/Span/History but not all columns and constraints
3. **No DEFAULT value verification** — Tests don't verify that columns with `DEFAULT` clauses actually produce correct values when omitted from INSERT. Root Cause A is specifically a DEFAULT not firing.
4. **No cross-dialect parity test** — Nothing verifies that SQLite and Postgres baselines produce equivalent logical schemas
5. **No UpsertApp constraint test** — The `UpsertApp` query relies on `created_at` having a DEFAULT, but no test exercises this path in isolation
6. **No GQL/API-level query regression test** — Root Cause B (`event not found`, `status didn't match`) shows that Postgres query results can silently diverge from expectations. No test validates that the GQL resolvers return correct data for both dialects after adapter conversion.
7. **Migration tests run too late** — Schema problems only surface during integration tests, not during a fast unit-level check
8. **No transaction-mode migration test** — Root Cause A may stem from goose's default transactional migration behavior vs golang-migrate's `NoTxWrap: true`. No test verifies that DEFAULTs work correctly under both modes.

#### 6.3 — Regression test plan

##### Layer 1: Schema Validation (fast, no DB required for comparison)

- [x] **`TestSchemaColumnsMatchSqlc`** — After running migrations, query `PRAGMA table_info(...)` (SQLite) or `information_schema.columns` (Postgres) for every table. Compare column names, types, nullability, and defaults against a parsed representation of the sqlc `schema.sql` files. Fail if any mismatch. This is the single most important test — it would have caught the `apps.created_at` regression instantly.
- [x] **`TestCrossDialectSchemaParity`** — Compare the logical schema (table names, column names, nullability, defaults) between SQLite and Postgres baselines. Flag divergences that aren't expected (e.g., `UUID` vs `CHAR(36)` is expected; a missing column is not).

##### Layer 2: Constraint & Default Verification (requires in-memory SQLite)

- [x] **`TestDefaultValues`** — For every table, INSERT a row with only required columns (omit all columns that have DEFAULTs). SELECT the row back and verify that default-populated columns have correct non-zero values. Tables to cover:
  - `apps` — `created_at`, `metadata`, `method`
  - `events` — `received_at`
  - `function_runs` — `run_started_at`, `trigger_type`
  - `function_finishes` — `output`, `completed_step_count`, `created_at`
  - `history` — `created_at`, `run_started_at`
  - `event_batches` — `executed_at`
  - `trace_runs` — `has_ai`
- [x] **`TestNotNullConstraints`** — For every NOT NULL column without a DEFAULT, verify that INSERT without that column fails with a constraint error. This ensures the schema is strict where it should be.
- [x] **`TestForeignKeyAndPrimaryKey`** — Verify PKs reject duplicate inserts. (No FK constraints currently, but this future-proofs the suite.)

##### Layer 3: Query Round-Trip Tests (requires in-memory SQLite + testcontainer Postgres)

These extend the existing `pkg/db/adapter_integration_test.go` with more comprehensive coverage:

- [x] **`TestUpsertAppRoundTrip`** — INSERT via `UpsertApp`, SELECT back, verify all fields including `created_at` default. Then UPDATE the same app and verify `created_at` is preserved.
- [x] **`TestInsertFunctionRoundTrip`** — Same pattern for functions, covering `archived_at` NULL behavior.
- [x] **`TestInsertEventRoundTrip`** — Verify event round-trip behavior including nullable `account_id` and `workspace_id` decoding.
- [x] **`TestInsertHistoryRoundTrip`** — Cover current history round-trip fields including nullable step metadata and result payloads.
- [x] **`TestInsertSpanRoundTrip`** — Cover JSON fields (`attributes`, `links`, `output`, `input`) and the `status`/`event_ids` columns added in later migrations.
- [x] **`TestWorkerConnectionRoundTrip`** — Cover the `worker_connections` table which has no existing test coverage.
- [x] **`TestTracesAndTraceRunsRoundTrip`** — Cover the OTEL trace tables.
- [x] **`TestEventBatchRoundTrip`** — Cover batch table.

Each test runs against both SQLite and Postgres via `TEST_DATABASE` env var.

##### Layer 3b: GQL/API Query Regression Tests (covers Root Cause B)

These address the Postgres-specific failures where queries return unexpected results:

- [x] **`TestGetEventByID`** — Insert an event, fetch it by ID via the GQL resolver, verify it's found and all fields match. Covers the `event not found` regression.
- [x] **`TestFunctionRunStatusLifecycle`** — Create a run, update its status through the expected lifecycle (Queued → Running → Completed/Failed), verify each status is correctly persisted and queryable. Covers the `status didn't match` regression.
- [x] **`TestEventListFiltering`** — Insert events with various names including internal events (`inngest/*`), verify list queries return correct results with and without internal event filtering.

##### Layer 4: Migration Lifecycle Tests

- [x] **`TestMigrationIdempotency`** — Run `up()` twice; second call is a no-op (already exists, verify it covers both dialects).
- [x] **`TestMigrationFromLegacy`** — Start with golang-migrate at version 18 (using the old `up()` function preserved in test code). Then run goose `up()`. Verify the schema is identical to a fresh goose baseline. This prevents regressions when we eventually delete the legacy migrations.
- [x] **`TestGooseVersionTableExists`** — After migration, verify `goose_db_version` table exists and has the expected version recorded.

#### 6.4 — Implementation location

Phase 6 coverage now lives in both `pkg/db/regression_test.go` and `pkg/db/adapter_integration_test.go`. The tests share the existing `newTestAdapter()` helper, which handles both SQLite in-memory and Postgres testcontainer setup.

Layer 1 (schema validation) and Layer 2 (constraints/defaults) now run as standard package tests on every `go test ./pkg/db` invocation. They're fast with in-memory SQLite and catch the class of bugs seen in PR #3945 before broader integration suites run.

#### 6.5 — Priority order

| Priority | Test                             | Rationale                                          |
|----------|----------------------------------|----------------------------------------------------|
| P0       | `TestSchemaColumnsMatchSqlc`     | Would have caught Root Cause A directly            |
| P0       | `TestDefaultValues`              | Would have caught the `created_at` DEFAULT issue   |
| P0       | `TestUpsertAppRoundTrip`         | Most common failing operation in CI (Root Cause A) |
| P0       | `TestGetEventByID`               | Would have caught Root Cause B on Postgres         |
| P1       | `TestCrossDialectSchemaParity`   | Prevents silent divergence between dialects        |
| P1       | `TestMigrationFromLegacy`        | Critical for safe goose transition                 |
| P1       | `TestNotNullConstraints`         | Guards against constraint relaxation               |
| P1       | `TestFunctionRunStatusLifecycle` | Covers Postgres `status didn't match` regression   |
| P2       | Remaining round-trip tests       | Comprehensive coverage for all tables              |
| P2       | `TestMigrationIdempotency`       | Already partially exists                           |
| P2       | `TestEventListFiltering`         | Covers internal event filtering edge case          |

## Key Type Differences by Dialect

| Field                   | SQLite           | Postgres                    | Domain (pkg/db)        |
|-------------------------|------------------|-----------------------------|------------------------|
| JSON fields             | `interface{}`    | `pqtype.NullRawMessage`     | `[]byte`               |
| Nullable ints           | `sql.NullInt64`  | `sql.NullInt32`             | `sql.NullInt64`        |
| Int widths              | `int64`          | `int16`/`int32`             | `int64`                |
| Nullable strings        | `sql.NullString` | `sql.NullString` / `string` | `sql.NullString`       |
| Event AccountID         | `interface{}`    | `sql.NullString`            | `sql.NullString`       |
| FunctionRun.WorkspaceID | present          | absent                      | present (zero when PG) |

## Risks & Mitigations

1. **Import cycles**: Mitigated by keeping `pkg/db` as a standalone leaf package
2. **Type conversion bugs**: Each adapter's `convert.go` is the single place to audit
3. **Missing fields**: Postgres `FunctionRun` lacks `WorkspaceID`; field stays zero-valued
4. **Integer overflow**: Domain uses `int64`; Postgres narrows to `int32` on write (acceptable for current data ranges)
5. **Breaking changes**: Phase 3 changes are internal to `base_cqrs`; no public API changes
