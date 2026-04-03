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

- [ ] **Step 1: Add goose dependency** — `go get github.com/pressly/goose/v3`
- [ ] **Step 2: Create baseline migration files**
  ```
  pkg/cqrs/base_cqrs/migrations/
    sqlite/000001_baseline.sql      # from sqlc/sqlite/schema.sql + goose annotations
    postgres/000001_baseline.sql    # from sqlc/postgres/schema.sql + goose annotations
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
- [ ] **Step 3: Rewrite `up()` in `base_cqrs.go`** — Replace the current 75-line function (lines 140-215) with goose's `Provider` API (~15 lines). This eliminates:
  - Fragile dirty-state recovery logic
  - `NoTxWrap` workaround for SQLite
  - Separate `source.Driver` / `database.Driver` plumbing
  - Manual `migrate.ErrNoChange` handling
- [ ] **Step 4: Update embed directive** — Existing `//go:embed **/**/*.sql` already works for `.sql` files. If Go-based migrations are added later, use `goose.AddMigrationNoTx()` registration in `init()`.
- [ ] **Step 5: Update test utilities** — Remove golang-migrate imports. No changes needed to test signatures since `base_cqrs.New()` API is unchanged.
- [ ] **Step 6: Add dedicated migration tests** — Currently there are zero tests for migration logic itself. Add:
  - `TestBaselineOnFreshDB` — applies baseline to empty in-memory SQLite, verifies all tables exist
  - `TestBaselineOnFreshPostgres` — same, on testcontainer Postgres
  - `TestMigrationIdempotency` — runs `up()` twice on the same DB, second call is a no-op
  - `TestSchemaMatchesSqlc` — compares table/column list after migrations against expected schema from `sqlc/{dialect}/schema.sql`
- [ ] **Step 7: Clean up** — Remove `github.com/golang-migrate/migrate/v4` from `go.mod`/`go.sum`, delete the old 72 incremental migration files

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
