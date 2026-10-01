# pkg/duckdb/insights

A parse-validate-rewrite pipeline that turns a user-written DuckDB `SELECT`
string into env/account-scoped SQL safe to execute against the `--duckdb`
dual-write store — the engine behind the `insights` GQL query,
self-hosted Inngest's equivalent of Cloud's Insights feature.

```go
tr, err := insights.Transpile(
    "SELECT run_id, app_id FROM runs WHERE status = 'Completed'",
    accountID, envID,
)
if err != nil {
    // a *insights.ValidationError (unknown table/column/function, or a
    // parse error) — never partially executed
    return err
}

result, err := insights.Execute(ctx, db, tr)
if err != nil {
    return err
}
// result.Columns: name, ColumnType, and a ColumnHint (APP_ID/FUNCTION_ID/
// RUN_ID/EVENT_ID, or none) for each output column
// result.Rows:    [][]any, one native Go value per cell
```

## Why this exists

`inngest dev --duckdb` accumulates run/event/span/metadata history in a
local DuckDB store, but the only way to read it was through the fixed GQL
resolvers `docs/plans/007`/`008` added (`Query.Runs`, `Query.RunTrace`,
etc.) — no way to ask an ad-hoc question. This package is what lets a user
write real SQL against that data safely: it validates the query against an
explicit table/column/function allowlist *before* anything reaches DuckDB,
then rewrites every table reference so environment/account scoping is
mandatory rather than trusting a `WHERE` clause the caller wrote.

## Scope

Six logical tables, each backed by a parameterized DuckDB table macro
(`pkg/db/duckdb/migrations/000004_insights_views.sql`) rather than the
underlying physical table directly:

| Logical table | What it is |
|---|---|
| `runs` | One row per `run_id` (latest state), plus merged run-scoped metadata as `metadata`/`inngest` |
| `events` | Ingested events |
| `metadata` | One row per `(run_id, span_id)`, with every emission merged per `kind` into `user_metadata`/`internal_metadata` |
| `extended_trace_spans` | SDK-emitted (userland) extended trace spans |
| `steps` | One row per step, latest attempt only |
| `step_attempts` | One row per step *attempt* — every retry, not just the latest |

A query is a `SELECT` (any `WHERE`/`GROUP BY`/`HAVING`/`QUALIFY`/`ORDER BY`/
`LIMIT`/window clause, `JOIN`s of the tables above, `SELECT *`/`t.*`/
`EXCLUDE`), optionally two such selects combined by `UNION`/`INTERSECT`/
`EXCEPT`, plus `WITH`/CTEs and subqueries (a `FROM`-clause subquery or a
scalar/`IN`/`EXISTS` subquery anywhere in an expression) — correlated or
not. An expression-position subquery may always reference its enclosing
query's own columns, and a `LATERAL` FROM-clause subquery may reference
any FROM item that appears before it in the same clause, matching
standard SQL and DuckDB's own rules exactly (a non-`LATERAL` subquery, and
a CTE body, still can't correlate — neither can in real SQL either). Its
output columns/types/hints are derived from its body via a static,
allowlist-only type checker (`typecheck.go`), not by asking DuckDB. `WITH
RECURSIVE`, a self-referencing CTE, and a computed (unaliased)
CTE/subquery column are all rejected. Table functions, `PIVOT`/`UNPIVOT`,
and `VALUES` are still rejected outright.

`account_id`/`env_id` are never queryable columns — each logical table's
macro takes them as call arguments and excludes both from its own
projection, so scoping isn't something a query can omit, override, or leak
around.

## How it fits together

```
   user's SQL string
        │
        ▼
┌──────────────────┐  pkg/duckdb/parser.ParseString
│  Parse           │
└───────┬──────────┘
        │  AST
        ▼
┌──────────────────┐  validate → extractQueryInfo → buildColumnHints →
│  Transpile        │  remapTables → addDefaultLimit — an ordered []stage
│  (this package)   │  list; see this directory's CLAUDE.md before adding one
└───────┬──────────┘
        │  rewritten SQL + args + column hints
        ▼
┌──────────────────┐  runs it, decodes rows via each cell's native
│  Execute          │  driver-decoded Go value (no pre-stringification)
└───────┬──────────┘
        │
        ▼
   pkg/coreapi/graph/resolvers.DuckdbInsightsQuery → GQL InsightsQueryResult
```

Column **type** comes from the executed query's own result set (via a
`DESCRIBE` of the rewritten SQL, not the executed rows — see this
directory's `CLAUDE.md` for why). Column **hint** (`APP_ID`/`FUNCTION_ID`/
`RUN_ID`/`EVENT_ID`, or none) is traced statically from the query's AST
back to a known identifier column or a known JSON path within one (e.g.
`attributes ->> '_inngest.function.id'`) — a computed expression, an
aggregate, or an ambiguous reference across a join gets no hint rather
than a guessed one.

## Testing

```
go test ./pkg/duckdb/insights/...
go test ./pkg/duckdb/insights/... -run TestTranspileGolden -update   # after an intentional Transpile output change
```

Besides ordinary unit tests, `Transpile`'s behavior is captured as
golden-file fixtures — one query per file in `testdata/queries/*.sql`, with
expected output in `testdata/queries/transpile/*.sql.out` — covering the
happy path, every rejection case, and every hint-tracing shape. Integration
tests (`views_test.go`, `execute_test.go`) run against a real `duckdb`
subprocess and skip automatically if none is on `PATH`.
