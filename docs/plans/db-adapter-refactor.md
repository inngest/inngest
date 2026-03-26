# Database Adapter Pattern Refactor

## Problem Statement

The current database layer uses SQLite as the canonical type system. PostgreSQL support
is bolted on via a ~1,300-line normalization layer that converts every Postgres result
into SQLite structs (`ToSQLite()`). Adding a new backend (MySQL, CockroachDB, etc.)
requires duplicating this boilerplate and adding more `isPostgres()` branches in
business logic.

### Current pain points

| Issue | Location | Impact |
|-------|----------|--------|
| SQLite types are the canonical model | `sqlc/sqlite/models.go` used everywhere | Every adapter must convert _to SQLite_, not to domain types |
| ~880 lines of hand-written shim per adapter | `sqlc/postgres/db_normalization.go` (60 wrapper methods) | Linear cost per new backend |
| ~460 lines of field-by-field type mapping | `sqlc/postgres/normalization.go` (`ToSQLite()` on every model) | Fragile, easy to miss a field |
| Dialect branches in business logic | `base_cqrs/cqrs.go` — 22 occurrences of `isPostgres()`/`dialect()` | Grows with each backend |
| Connection pooling mixed into query layer | `NewNormalized()` calls `SetMaxIdleConns` etc. | Untestable, side-effect-heavy |
| Dual sqlc codegen with separate SQL files | `sqlc.yaml` — 2 engines, 2 schemas, 2 query files | Schema drift between dialects |
| Migrations tightly coupled to `golang-migrate` | `base_cqrs.go` `up()` function | Hard to add features like seed data, dry-run |

### What stays the same

- sqlc for type-safe code generation (per-dialect)
- goqu for dynamic query building where needed
- `database/sql` as the common driver interface
- CQRS separation (Reader/Writer interfaces in `pkg/cqrs/`)
- Embedded migration files

---

## Phase 1: Domain Model Types

**Goal:** Decouple the canonical data types from any specific database dialect.

### 1.1 — Define domain models in `pkg/cqrs/models.go`

Extract database-agnostic structs from the existing SQLite models. These become the
single source of truth that all adapters return.

```go
// pkg/cqrs/models.go
package cqrs

import (
    "database/sql"
    "time"
    "github.com/google/uuid"
    "github.com/oklog/ulid/v2"
)

type App struct {
    ID          uuid.UUID
    Name        string
    SdkLanguage string
    SdkVersion  string
    Framework   sql.NullString
    Metadata    string
    Status      string
    Error       sql.NullString
    Checksum    string
    CreatedAt   time.Time
    ArchivedAt  sql.NullTime
    Url         string
    Method      sql.NullString
    AppVersion  sql.NullInt64
}

type Function struct { /* ... */ }
type Event struct { /* ... */ }
type FunctionRun struct { /* ... */ }
type History struct { /* ... */ }
// ... one struct per table
```

### 1.2 — Define the `Querier` interface on domain types

Move the `Querier` interface out of sqlc-generated code into `pkg/cqrs/querier.go`.
It returns `cqrs.App`, not `sqlc.App`.

```go
// pkg/cqrs/querier.go
package cqrs

type Querier interface {
    GetApp(ctx context.Context, id uuid.UUID) (*App, error)
    GetApps(ctx context.Context) ([]*App, error)
    UpsertApp(ctx context.Context, arg UpsertAppParams) (*App, error)
    // ... all 99 current methods, typed with domain models
}
```

### 1.3 — Adapter implementations convert internally

Each adapter's sqlc-generated code stays private. The adapter converts to domain types:

```go
// pkg/cqrs/adapters/sqlite/querier.go
func (a *Adapter) GetApp(ctx context.Context, id uuid.UUID) (*cqrs.App, error) {
    row, err := a.q.GetApp(ctx, id)
    if err != nil { return nil, err }
    return toDomainApp(row), nil
}
```

### Deliverables

- [ ] `pkg/cqrs/models.go` — all domain structs
- [ ] `pkg/cqrs/querier.go` — Querier interface on domain types
- [ ] `pkg/cqrs/params.go` — param structs for write operations (shared across adapters)

---

## Phase 2: Adapter Interface & Registry

**Goal:** Replace string-based driver detection with a typed adapter contract.

### 2.1 — Define `Adapter` interface

```go
// pkg/cqrs/adapter.go
package cqrs

type Dialect string

const (
    DialectSQLite   Dialect = "sqlite"
    DialectPostgres Dialect = "postgres"
    DialectMySQL    Dialect = "mysql" // stubbed
)

type Adapter interface {
    // Identity
    Dialect() Dialect

    // Query layer — returns domain types
    Querier() Querier

    // Transaction support
    WithTx(ctx context.Context) (TxAdapter, error)

    // Connection lifecycle
    Close() error
}

type TxAdapter interface {
    Adapter
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

### 2.2 — Dialect-specific SQL helpers

Move the `spanRunsAdapter` pattern into the adapter:

```go
// pkg/cqrs/adapter.go
type DialectHelpers interface {
    GoquDialect() string
    ExprConverter() run.ExprSQLConverter
    BuildEventJoin(q *sq.SelectDataset) *sq.SelectDataset
    ParseEventIDs(raw *string) []string
    ParseTime(s string) (time.Time, error)
}
```

Each adapter implements `DialectHelpers`. The `wrapper` struct calls `adapter.BuildEventJoin()`
instead of checking `isPostgres()`.

### 2.3 — Refactor `wrapper` to use `Adapter`

```go
// pkg/cqrs/base_cqrs/cqrs.go
type wrapper struct {
    adapter cqrs.Adapter
    helpers cqrs.DialectHelpers
    db      *sql.DB
}

// Before:  if w.isPostgres() { ... } else { ... }
// After:   w.helpers.BuildEventJoin(query)
```

### 2.4 — Adapter constructors

```go
// pkg/cqrs/adapters/sqlite/adapter.go
func New(db *sql.DB) *Adapter { ... }

// pkg/cqrs/adapters/postgres/adapter.go
func New(db *sql.DB) *Adapter { ... }

// pkg/cqrs/adapters/mysql/adapter.go  (stub)
func New(db *sql.DB) *Adapter {
    panic("mysql adapter not yet implemented")
}
```

### Deliverables

- [ ] `pkg/cqrs/adapter.go` — Adapter + TxAdapter + DialectHelpers interfaces
- [ ] `pkg/cqrs/adapters/sqlite/` — SQLite adapter (wraps existing sqlc sqlite)
- [ ] `pkg/cqrs/adapters/postgres/` — Postgres adapter (wraps existing sqlc postgres)
- [ ] `pkg/cqrs/adapters/mysql/` — Stub adapter with `panic("not implemented")`
- [ ] Refactored `wrapper` struct — zero `isPostgres()` calls

---

## Phase 3: Connection Lifecycle Extraction

**Goal:** Separate connection pool management from query construction.

### 3.1 — `OpenDB` function

```go
// pkg/cqrs/dbpool.go
package cqrs

type PoolConfig struct {
    MaxIdleConns    int
    MaxOpenConns    int
    ConnMaxIdle     time.Duration
    ConnMaxLifetime time.Duration
}

type DBConfig struct {
    Dialect  Dialect
    DSN      string
    Pool     PoolConfig
    Persist  bool       // SQLite: persist to disk
    ForTest  bool       // create isolated DB per test
    Dir      string     // SQLite: directory for db file
}

func OpenDB(cfg DBConfig) (*sql.DB, error) { ... }
```

### 3.2 — Remove pool config from `NewNormalized`

`NewNormalizedOpts` currently bundles pool settings. After this phase, connection
pooling is configured once in `OpenDB` and never touched again by the query layer.

### Deliverables

- [ ] `pkg/cqrs/dbpool.go` — unified `OpenDB` with pool config
- [ ] Remove `NewNormalizedOpts` pool fields
- [ ] Update `devserver.go` and test setup to use `OpenDB`

---

## Phase 4: Migration System Improvements

**Goal:** Make migrations adapter-owned and prepare for goose adoption.

### 4.1 — Adapter-owned migrations (current system)

Each adapter provides its own migration source:

```go
type MigrationProvider interface {
    MigrationSource() (source.Driver, error)
    MigrationDriver(db *sql.DB) (database.Driver, error)
}
```

Migration files move from `base_cqrs/migrations/{sqlite,postgres}/` into each
adapter package: `adapters/sqlite/migrations/`, `adapters/postgres/migrations/`.

### 4.2 — Goose migration framework (follow-up)

Replace `golang-migrate` with [pressly/goose](https://github.com/pressly/goose):

**Why goose over golang-migrate:**

| Feature | golang-migrate | goose |
|---------|---------------|-------|
| Go-based migrations | No (SQL only) | Yes — `func Up(tx *sql.Tx)` |
| Seed data support | Manual | Built-in via Go migrations |
| Dry-run mode | No | `goose status` |
| Embedded FS | Via `iofs` adapter | Native `embed.FS` support |
| Dialect awareness | Separate drivers | Built-in dialect registry |
| Versioning | Sequential numbers | Timestamps or sequential |
| Down migrations | Required | Optional |
| Transaction control | All-or-nothing | Per-migration control |
| Active maintenance | Moderate | Active, widely adopted |

**Migration path from golang-migrate to goose:**

1. Keep existing numbered migrations as-is (goose supports sequential numbering)
2. Add a `goose_db_version` table alongside existing `migrations` table
3. Write a one-time "bridge" migration that reads `golang-migrate` state and
   seeds the goose version table to match
4. All new migrations use goose format
5. Remove `golang-migrate` dependency once bridge is validated

**Goose integration with adapter pattern:**

```go
// pkg/cqrs/adapters/sqlite/migrations.go
//go:embed migrations/*.sql
var migrationFS embed.FS

func (a *Adapter) Migrate(db *sql.DB) error {
    goose.SetBaseFS(migrationFS)
    return goose.Up(db, "migrations")
}
```

### 4.3 — MySQL migration stub

```
pkg/cqrs/adapters/mysql/migrations/
    00001_create_apps_table.sql   // skeleton only
```

### Deliverables

- [ ] Move migration files into adapter packages
- [ ] Adapter implements `MigrationProvider`
- [ ] **Follow-up PR:** Add goose dependency, write bridge migration, convert new migrations

---

## Phase 5: Cleanup & Deletion

**Goal:** Remove all legacy normalization code.

### Files to delete

| File | Lines | Reason |
|------|-------|--------|
| `sqlc/postgres/db_normalization.go` | 882 | Replaced by postgres adapter's Querier impl |
| `sqlc/postgres/normalization.go` | 459 | Replaced by domain model converters in adapter |
| `sqlc/postgres/augmented.go` | ~100 | Merged into adapter |

### Code to remove from `base_cqrs/cqrs.go`

- `isPostgres()` method and all 22 call sites
- `dialect()` method
- `sqliteSpanRunsAdapter` / `postgresSpanRunsAdapter` global vars
- `spanRunsAdapter()` method
- `NewQueries()` function (replaced by adapter constructors)

**Estimated net deletion: ~1,500 lines**

### Deliverables

- [ ] Delete normalization files
- [ ] Remove dialect branches from `cqrs.go`
- [ ] Remove `NewQueries()` / `NewNormalized()` entry points
- [ ] Verify no imports reference deleted packages

---

## Testing Plan

### Unit Test Strategy

Each adapter gets its own test suite that validates the `Querier` contract:

```go
// pkg/cqrs/adapters/cqrstest/suite.go
package cqrstest

// QuerierSuite runs the full Querier contract against any adapter.
func QuerierSuite(t *testing.T, newAdapter func(t *testing.T) cqrs.Adapter) {
    t.Run("Apps", func(t *testing.T) {
        t.Run("UpsertAndGet", func(t *testing.T) { ... })
        t.Run("GetByChecksum", func(t *testing.T) { ... })
        t.Run("Delete", func(t *testing.T) { ... })
    })
    t.Run("Functions", func(t *testing.T) { ... })
    t.Run("Events", func(t *testing.T) { ... })
    t.Run("FunctionRuns", func(t *testing.T) { ... })
    t.Run("History", func(t *testing.T) { ... })
    t.Run("Traces", func(t *testing.T) { ... })
    t.Run("Transactions", func(t *testing.T) { ... })
    // ...
}
```

Each adapter imports the shared suite:

```go
// pkg/cqrs/adapters/sqlite/adapter_test.go
func TestSQLiteAdapter(t *testing.T) {
    cqrstest.QuerierSuite(t, func(t *testing.T) cqrs.Adapter {
        db := openTestSQLite(t)
        return sqlite.New(db)
    })
}

// pkg/cqrs/adapters/postgres/adapter_test.go
func TestPostgresAdapter(t *testing.T) {
    cqrstest.QuerierSuite(t, func(t *testing.T) cqrs.Adapter {
        db := openTestPostgres(t)  // uses testcontainers
        return postgres.New(db)
    })
}
```

### CI Matrix

Update `.github/workflows/go.yaml`:

```yaml
strategy:
  matrix:
    database: [sqlite, postgres]
    experimentalKeyQueues: [false, true]
    enableConstraintAPI: [false, true]
```

No change to the existing matrix shape. The `TEST_DATABASE` env var maps to adapter
selection in `initCQRS()` test helper:

```go
func initCQRS(t *testing.T) cqrs.Adapter {
    switch os.Getenv("TEST_DATABASE") {
    case "postgres":
        return newPostgresAdapter(t)
    default:
        return newSQLiteAdapter(t)
    }
}
```

When MySQL is implemented, extend:

```yaml
    database: [sqlite, postgres, mysql]
```

And add a MySQL testcontainer:

```go
    case "mysql":
        return newMySQLAdapter(t)  // testcontainers with mysql:8
```

### Integration / E2E Tests

- Dev server tests (`pkg/devserver/api_test.go`) continue using SQLite in-memory
- Executor tests (`tests/execution/executor/executor_test.go`) run against the matrix
- No changes needed to E2E SDK tests (they test the API layer, not the DB)

### Migration Tests

```go
// pkg/cqrs/adapters/sqlite/migrate_test.go
func TestSQLiteMigrations(t *testing.T) {
    db := openFreshSQLite(t)
    adapter := sqlite.New(db)
    require.NoError(t, adapter.Migrate(db))
    // Verify schema version
    // Run all migrations up, then all down, then up again
}
```

### Test for MySQL Stub

```go
func TestMySQLStubPanics(t *testing.T) {
    assert.Panics(t, func() {
        mysql.New(nil)
    })
}
```

---

## Risk Assessment

### High Risk

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Domain model drift from sqlc models** | Medium | High — silent data loss if a field is missed | Generate domain model conversion with `go generate`; add compile-time interface checks (`var _ Querier = (*sqliteAdapter)(nil)`) |
| **Breaking the Postgres normalization during refactor** | Medium | High — production data path | Phase the work: keep `db_normalization.go` working in parallel until the new adapter passes the full test suite, then swap |
| **Migration ordering when moving files** | Low | High — corrupted DB state | Never renumber migrations; keep exact same filenames in new locations; add a CI check that migration checksums match before/after |

### Medium Risk

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **goqu query differences across dialects** | Medium | Medium — wrong results for specific queries | The shared test suite (`QuerierSuite`) catches this; run it against every adapter in CI |
| **Transaction semantics differ (SQLite WAL vs Postgres MVCC)** | Low | Medium — subtle concurrency bugs | Transaction tests with concurrent readers/writers in the shared suite |
| **Performance regression from extra conversion layer** | Low | Low — field copies are cheap | Benchmark critical paths (bulk event insert, trace queries) before/after |

### Low Risk

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **MySQL stub accidentally used in prod** | Very Low | Medium | `panic()` in constructor; CI does not include MySQL in matrix until implemented |
| **goose migration bridge misses a version** | Low | Medium | Bridge migration reads `schema_migrations` table and asserts versions match; test with both fresh and migrated DBs |

### Migration Safety: Parallel Running Strategy

During the transition, both old and new code paths must produce identical results.
Recommended approach:

1. **Phase 1-2:** New adapter code lives alongside old code. `NewCQRS()` continues
   to work unchanged. New `adapter.New()` constructors are only used in tests.

2. **Phase 3:** Feature flag (`CQRS_USE_ADAPTER=true`) switches `devserver.go` and
   other entry points to use the new adapter path. Both paths run in CI.

3. **Phase 4:** Once the adapter path passes all tests for 2+ weeks in CI, remove
   the old path (Phase 5 cleanup).

4. **Rollback:** At any point, removing the feature flag reverts to the old path.
   No database schema changes are involved — this is purely a code-level refactor.

---

## Final Directory Structure

```
pkg/cqrs/
    adapter.go              # Adapter, TxAdapter, DialectHelpers interfaces
    cqrs.go                 # Manager, TxManager (unchanged)
    models.go               # Domain model types (new)
    params.go               # Shared write-operation param structs (new)
    querier.go              # Querier interface on domain types (new)
    dbpool.go               # OpenDB + PoolConfig (new)
    apps.go                 # AppManager interface (unchanged)
    events.go               # EventManager interface (unchanged)
    ...

    adapters/
        cqrstest/
            suite.go        # Shared adapter conformance test suite

        sqlite/
            adapter.go      # Adapter + Querier implementation
            helpers.go      # DialectHelpers (goqu, CEL, JSON parsing)
            convert.go      # sqlc row -> domain model converters
            migrations/     # moved from base_cqrs/migrations/sqlite/
            sqlc/           # sqlc-generated code (private)

        postgres/
            adapter.go
            helpers.go
            convert.go
            migrations/     # moved from base_cqrs/migrations/postgres/
            sqlc/           # sqlc-generated code (private)

        mysql/
            adapter.go      # Stub: panic("not implemented")
            helpers.go      # Stub
            migrations/     # Skeleton migration files

    base_cqrs/
        cqrs.go             # wrapper struct (refactored to use Adapter)
        history_driver.go   # (refactored to use Adapter)
        history_reader.go   # (refactored to use Adapter)
```

---

## Summary of Phases

| Phase | Scope | Est. Files Changed | Risk |
|-------|-------|--------------------|------|
| 1 — Domain Models | New `models.go`, `querier.go`, `params.go` | ~5 new files | Low |
| 2 — Adapter Interface | New interface + 3 adapter packages + refactor wrapper | ~12 files | Medium |
| 3 — Connection Lifecycle | New `dbpool.go`, update constructors | ~6 files | Low |
| 4 — Migrations | Move files, add goose prep (follow-up) | ~8 files | Medium |
| 5 — Cleanup | Delete normalization, remove branches | ~5 files deleted | Low (gated on test suite) |
