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
- [ ] Evaluate goose as replacement for golang-migrate
- [ ] Consolidate migration files per dialect
- [ ] Add MySQL schema + migrations when ready

## Key Type Differences by Dialect

| Field | SQLite | Postgres | Domain (pkg/db) |
|-------|--------|----------|-----------------|
| JSON fields | `interface{}` | `pqtype.NullRawMessage` | `[]byte` |
| Nullable ints | `sql.NullInt64` | `sql.NullInt32` | `sql.NullInt64` |
| Int widths | `int64` | `int16`/`int32` | `int64` |
| Nullable strings | `sql.NullString` | `sql.NullString` / `string` | `sql.NullString` |
| Event AccountID | `interface{}` | `sql.NullString` | `sql.NullString` |
| FunctionRun.WorkspaceID | present | absent | present (zero when PG) |

## Risks & Mitigations

1. **Import cycles**: Mitigated by keeping `pkg/db` as a standalone leaf package
2. **Type conversion bugs**: Each adapter's `convert.go` is the single place to audit
3. **Missing fields**: Postgres `FunctionRun` lacks `WorkspaceID`; field stays zero-valued
4. **Integer overflow**: Domain uses `int64`; Postgres narrows to `int32` on write (acceptable for current data ranges)
5. **Breaking changes**: Phase 3 changes are internal to `base_cqrs`; no public API changes
