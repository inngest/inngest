# pkg/duckdb/insights — Agent Guide

This file supplements the root `CLAUDE.md`. Read that first for repo-wide
conventions (commit types, PR sections). This file covers what's specific to
working in this package.

## What this is

The transpile pipeline behind `insights`, a GQL query that
accepts a raw user-written `SELECT` string, validates and rewrites it
against six logical tables, and executes it against the `--duckdb`
dual-write store — the DuckDB-backed equivalent of Cloud's Insights
feature. Spec: `docs/plans/010-duckdb-insights-query-layer.md`. Build log:
`docs/plans/012-duckdb-insights-query-layer-plan.md` (both marked `Done`).
Feature-status snapshot: `FINDINGS/duckdb-insights.md`.

This package never touches the database except in `execute.go` — parsing,
validation, and rewriting are pure functions over
`pkg/duckdb/parser`'s AST. It has no dependency on `cqrs`/`pkg/db/duckdb`
beyond `duckdb.DuckLakeAlias` (a string constant) and `duckdb.Open`/
`Migrate` in its own integration tests.

## Where things live

```
tables.go        the logical table registry: known columns, hints, JSON pathHints
functions.go     allowedFunctions — the function-name allowlist
scope.go         tableScope — resolves a FROM clause to its in-scope logical tables, CTEs included
derive.go        deriveTable — synthesizes a logicalTable for a CTE/subquery from its validated body
typecheck.go     inferType — static, allowlist-only expression type inference (backs deriveTable)
validate.go      the validate stage — table/column/function allowlisting; validateWithCTEs threads CTE scope
jsonpath.go      jsonPathAccess — recognizes every "->>'/->/json_extract*" shape for hint-tracing
columnhints.go   the buildColumnHints stage
queryinfo.go     the extractQueryInfo stage
remap.go         the remapTables stage — rewrites logical table refs (FROM tree and expression-position subqueries) into macro calls
limit.go         the addDefaultLimit stage
transpile.go     Transpile: the ordered []stage pipeline + TranspileResult
execute.go       Execute: runs the rewritten SQL, decodes rows/columns
columntype.go    ColumnType enum + DuckDBToColumnType
hint.go          ColumnHint enum
testdata/queries/*.sql            golden fixture inputs, one query per file
testdata/queries/transpile/*.out  their expected Transpile() output (goldie)
```

Physical schema this package reads from: six `CREATE MACRO ... AS TABLE`
statements in `pkg/db/duckdb/migrations/000004_insights_views.sql`.

## Architecture, in one pass

```
SQL text --parser.ParseString--> AST --Transpile's []stage pipeline--> rewritten SQL + args
                                                                              |
                                                                    Execute runs it, decodes rows
```

`Transpile` (`transpile.go`) is a thin loop over an ordered `[]stage` list
threading a shared `pipelineState`. **Adding a stage is one new function
plus one new line in that list — never an edit to `Transpile` itself.**
This mirrors the spec's own framing: a future stage the reference
`pkg/insights` (Cloud's ClickHouse equivalent) gains should have an
obvious, analogous place to land here.

The stages, in order: `validate` → `extractQueryInfo` → `buildColumnHints`
→ `remapTables` → `addDefaultLimit`. `validate` resolves and returns the
query's `*tableScope` so later stages never call `resolveScope` a second
time.

## The most important gotcha: `validate` must walk *every* AST field that can carry an expression

`collectExprs` (`validate.go`) exists because `parser.Walk` only recurses
through `Node.Children()` starting from wherever you point it — it does
**not** know to visit every field on `*parser.SelectStatement` on its own.
Every clause that can carry a column/function reference has to be
explicitly added to `collectExprs`'s list.

This has already caused four real bugs, found by systematically re-auditing
`SelectStatement`'s full field list after the first one turned up (see
`validate.go`'s and `validate_test.go`'s comments for each):

- `stmt.Windows` (a named `WINDOW w AS (...)` clause, referenced elsewhere
  as `OVER w`) — an inline `OVER(...)` is reachable via its own
  `FunctionExpr`, but a named window is a separate top-level field.
- `GroupByItem`'s `*parser.GroupingSets` case — its `Sets` field holds more
  `GroupByItem`s that need recursing into, not a bare `Expr` list.
- `stmt.Distinct.On` (`SELECT DISTINCT ON (...)`).
- `*parser.StarExpr`'s `Qualifier`/`Exclude` — `StarExpr.Children()`
  returns `nil` (they're plain `[]string`, not `Expr`/`Ident` nodes), so
  nothing walks into them without an explicit `checkStar` visitor case.

**If you add support for a new clause shape, or `pkg/duckdb/parser` adds a
new field to `SelectStatement`/`GroupByItem`/anything else this package
already partially handles, check whether `collectExprs`/`groupByItemExprs`
needs a new case.** None of the four bugs above were exploitable against
today's fixed six-table schema (DuckDB itself would still error on a truly
nonexistent column), but each violated this feature's own stated contract
("no column reference outside a table's known set reaches DuckDB
unchecked") — don't assume "the database will catch it anyway" is good
enough; write the golden fixture and unit test pair for a rejection case
whenever you touch this file.

## Other non-obvious constraints

- **The `_inngest.` attribute key prefix.** Every OTel span attribute key
  this package reads out of `attributes` (via `spanAttrPathHints` in
  `tables.go`, or the `insights_step_attempts` migration) is stored with a
  `_inngest.` prefix — `pkg/tracing/meta/consts.go`'s `AttrKeyPrefix`,
  applied by every `*Attr` constructor in `serializers.go` before a span's
  raw attributes are ever written. `Attrs.StepID` (declared as
  `StringAttr("step.id")`) is actually stored under the literal key
  `"_inngest.step.id"`. If you add a new path hint or unpack a new
  attribute in the migration, it needs this prefix or it will silently
  read `NULL`.
- **The `col -> 'key'` / lambda-arrow ambiguity.** `pkg/duckdb/parser`
  can't syntactically distinguish DuckDB's lambda arrow from its JSON
  arrow operator — a bare `col -> 'key'` parses as a single-param
  `*parser.LambdaExpr`, not a `BinaryExpr`. This package's function
  allowlist (`functions.go`'s `allowedFunctions`) never includes a
  lambda-taking function (`list_transform`, etc.), so every `LambdaExpr`
  `validate`/`buildColumnHints` ever see is necessarily JSON-arrow-shaped
  column access, not a genuine lambda. Don't "fix" this ambiguity by
  trying to disambiguate lambdas some other way — it's load-bearing.
- **Positional, not name-based, column hints.** `buildColumnHints` expands
  `SELECT *`/`t.*` using each logical table's own static `columnOrder`
  (`tables.go`), not by asking the database. This only works because this
  package also owns every macro's `SELECT` list order (Task 9's migration)
  — **if you add/reorder a column in the migration, `columnOrder` must
  match exactly**, or hints silently shift onto the wrong column.
- **`Execute` gets column metadata from `rows.ColumnTypes()` directly — no
  separate `DESCRIBE` query of its own.** This used to not be true:
  `pkg/db/duckdb`'s driver derived `*sql.Rows.Columns()`/`ColumnTypes()`
  from the first actually-returned row, so a query matching zero rows
  reported no columns at all, and `Execute` ran its own `DESCRIBE <sql>`
  (same bound args) to work around it. That limitation now lives in the
  driver instead: `conn.QueryContext` (`pkg/db/duckdb/conn.go`) always runs
  a query through `sqlExecer.query`, which returns column name/type
  unconditionally — a batched `DESCRIBE` + the real query in one stdin
  round trip for the jsonlines transport (`rows.go`'s `session.query`), or,
  for quack, no extra statement at all (the `PrepareResponse`'s own schema
  metadata, `quack_session.go`'s `quackSession.query`). Don't reintroduce a
  `DESCRIBE <sql>` call here — that would run `DESCRIBE` twice (the driver's
  own plus this package's), which errors as invalid SQL. If you ever need to
  bypass the driver's own type-fetching for a specific query, that's a
  `pkg/db/duckdb` change, not a `pkg/duckdb/insights` one.
- **Every query sent to this driver needs a trailing `;`, or it hangs**
  (not errors) indefinitely — this was a real bug, now fixed once in
  `pkg/db/duckdb/rows.go`'s `session.exec` rather than patched per call
  site, so you shouldn't need to think about it here. If you see a test
  hang rather than fail, this is the first thing to suspect regressed.
- **Scope of supported `SELECT` shapes.** `WITH`/CTEs and subqueries (FROM-
  clause or expression-position — scalar/`IN`/`EXISTS`), correlated or not,
  are supported; table functions/`PIVOT`/`UNPIVOT`/parenthesized-FROM-
  without-a-derived-select are not. `UNION`/`INTERSECT`/`EXCEPT` of two flat
  selects *is* supported (each side independently validated/scoped);
  `buildColumnHints` compares both sides' hints positionally and only
  reconciles a position where they agree (`unionColumnHints`,
  `columnhints.go`) — a mismatched or unknown side becomes `HintNone` at
  that position, not a blanket nil for the whole query.
- **Correlation is a single `outer *tableScope` pointer on `tableScope`
  itself, not a parameter threaded through every resolution function.**
  `tableScope.lookup`/`columnCount`/`uniqueColumn` (`scope.go`) each try
  the local scope first, then fall back to `s.outer` (recursively, so a
  correlation chain composes across any nesting depth) only when the local
  scope has *no* match at all — ambiguity is always computed within one
  scope, never merged across a correlation boundary, matching standard SQL
  name resolution. Because the fallback lives inside `tableScope` itself,
  every consumer (`exprValidator.checkIdent`/`checkLambda`, `inferType`,
  `resolveItemHint`, `resolveStarColumns`) gets correlation for free with
  no signature changes.
  - An **expression-position subquery** (scalar/`IN`/`EXISTS`) always gets
    the enclosing query block's own scope as its outer scope —
    `exprValidator.Visit`'s `*parser.SelectStatement` case
    (`validate.go`) passes `v.scope` down every time it recurses into one.
    No special syntax needed; this matches every real SQL engine.
  - A **`LATERAL` FROM-clause subquery** gets the scope built so far from
    *strictly earlier* FROM items in the same clause as its outer scope —
    `addSubquery` (`scope.go`) passes `s` (the `*tableScope` being built,
    still mid-construction) itself when `r.Lateral`. A later FROM item is
    invisible to it, matching DuckDB's own left-to-right LATERAL
    visibility rule.
  - A **non-`LATERAL` FROM subquery** and a **CTE body** always get `nil`
    — neither can correlate in real SQL either (a non-LATERAL derived
    table can't see a preceding FROM item at all; a CTE can never
    reference the query that consumes it). Don't add an outer scope to
    either path; that would accept SQL DuckDB itself would reject.
  - A CTE's own name is only added to the `ctes` map *after* its body is
    validated, so a self-reference without `RECURSIVE` fails as "unknown
    table". `WITH RECURSIVE` is rejected outright — no fixpoint evaluation
    exists here.
- **A CTE/subquery's output columns come from a real (if limited) static
  type checker, not from asking DuckDB.** `deriveTable` requires every
  computed (non-bare-column) output expression to have an explicit alias,
  then infers each column's type via `inferType` (`typecheck.go`) — an
  exhaustive switch over every expression shape the allowlist can produce.
  Anything it can't determine (an ambiguous column across a join, a `CASE`
  with mismatched branch types, `NULL`) becomes `ColumnTypeUnknown`, never
  a guess. If you extend the parser's allowlist with a new expression
  shape or function, `inferType`/`functionReturnType` need a matching case
  or a CTE/subquery column built from it silently reports `UNKNOWN`.
- **`remapTables` must rewrite expression-position subqueries too, not just
  the `FROM` tree.** `remapRef` handles `BaseTableRef`/`JoinRef`/
  `TableSubqueryRef` (the FROM tree); `remapSubqueriesIn` (`remap.go`)
  separately walks the same `collectExprs` set `validate.go` uses to find
  every scalar/`IN`/`EXISTS` subquery and rewrite its body too. This was a
  real bug during development, caught only by hand-reading a regenerated
  golden fixture: `WHERE run_id IN (SELECT run_id FROM runs)`'s *inner*
  `runs` was left bare/unrewritten, bypassing env/account scoping entirely
  for that subquery. `extractQueryInfo` (`queryinfo.go`) has the same two-
  places-to-walk shape for the same reason, though its own gap is only a
  diagnostics completeness issue, not a scoping bypass. If `pkg/duckdb/
  parser` ever adds another way to embed a `*parser.SelectStatement`
  outside `collectExprs`'s reach, both `remap.go` and `queryinfo.go` need a
  matching case, exactly like `validate.go`'s own `collectExprs` gotcha
  above.
- **`account_id`/`env_id` are macro parameters, never columns.** Every
  logical table's macro excludes both from its own projection — a query
  can't reference them even if `validate` had a bug, because they simply
  aren't in the macro's output schema. Don't add them back to `tables.go`'s
  `columns` map; that would just make `validate` accept a reference DuckDB
  itself would then reject.

## Testing

- **Golden fixtures**: `testdata/queries/*.sql` in (one query per file),
  `Transpile()`'s JSON-serialized output out
  (`testdata/queries/transpile/*.sql.out`), via `goldie`. Regenerate with
  `go test ./pkg/duckdb/insights/... -run TestTranspileGolden -update` —
  **always read the diff by hand before committing** a regenerated
  fixture (`cat` the changed `.out` files); a green `-update` run doesn't
  mean the new output is *correct*. When adding a new SQL construct or
  validation rule, add a fixture for it, not just a `require.Error`/
  `require.NoError` unit test — the golden output captures the full
  rewritten SQL/args/hints, which has caught real bugs a pass/fail
  assertion wouldn't (see the `validate.go` gotcha above).
- **Integration tests** (`views_test.go`, `execute_test.go`, and
  `pkg/coreapi/graph/resolvers/insights_integration_test.go`) run against a
  real `duckdb` subprocess via a locally-duplicated `newTestDuckDB` helper
  (can't import `pkg/cqrs/duckdbquery/testutil_test.go`'s — it's
  package-private). Skip silently if no `duckdb` binary is on `PATH`.
- **Unit tests** are colocated per stage file (`validate_test.go`,
  `columnhints_test.go`, etc.) and don't need a live database.
