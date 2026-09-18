# pkg/duckdb/parser

A Go parser for DuckDB's `SELECT` SQL dialect. It turns a query string into
a typed, walkable Go AST — and can turn that AST back into SQL text again.

```go
stmt, err := parser.ParseString(`
    SELECT customer_id, sum(amount) AS total
    FROM orders
    WHERE created_at > '2026-01-01'
    GROUP BY customer_id
    HAVING sum(amount) > 100
`)
if err != nil {
    var perr *parser.ParseError
    if errors.As(err, &perr) {
        fmt.Printf("parse error at %s: %s\n", perr.Pos, perr.Message)
    }
    return err
}

// Walk the whole tree without knowing every node type up front.
parser.Walk(myVisitor{}, stmt)

// Or render it straight back to SQL (not necessarily byte-identical —
// whitespace and keyword casing are normalized — but always equivalent).
var buf bytes.Buffer
_ = parser.Write(&buf, stmt)
```

## Why this exists

No Go library speaks DuckDB SQL. The closest thing —
[`clickhouse-sql-parser`](https://github.com/AfterShip/clickhouse-sql-parser)
— parses ClickHouse's dialect, and the two diverge enough (list/struct
literals, `QUALIFY`, `PIVOT`/`UNPIVOT`, DuckDB's own function and operator
set) that reusing it would mean silently mis-parsing or rejecting valid
DuckDB syntax.

DuckDB actually publishes its own grammar as PEG (`.gram`) files — the same
ones its SQL autocomplete feature is built from. Rather than hand-porting
that grammar into some Go PEG library's dialect (a translation that would
quietly drift from upstream over time), this package vendors those `.gram`
files verbatim and interprets them directly with a small custom PEG engine.
When DuckDB adds new syntax, updating this package is a matter of bumping a
pinned commit and fixing whatever adapter code the shape change touches —
not re-deriving a grammar by hand.

## Scope

This parses `SELECT` statements — including CTEs, joins, subqueries, window
functions, `GROUP BY`/`HAVING`/`QUALIFY`, `PIVOT`/`UNPIVOT`, and set
operations. It does **not** handle DDL, `INSERT`/`UPDATE`/`DELETE`,
`PRAGMA`, `COPY`, or any other statement type, and it does no semantic
validation (table/column existence, type checking) — it only tells you
whether a query is syntactically valid DuckDB SQL and, if so, hands you a
tree describing it.

## How it fits together

```
  SQL text
     │
     ▼
┌──────────────────┐   DuckDB's own vendored .gram grammar, interpreted
│ PEG parser       │   directly by a small packrat engine (pkg/duckdb/
│ (peg/, grammar/) │   parser/peg — generic, has no notion of "SQL")
└───────┬──────────┘
        │  untyped parse tree
        ▼
┌──────────────────┐   walks the parse tree into concrete Go types —
│ Adapter          │   *SelectStatement, *BinaryExpr, *Ident, ...
│ (adapter_*.go)   │
└───────┬──────────┘
        │  typed AST
        ▼
   your code — Walk() it, Write() it back to SQL, or just read the
   fields directly
```

## Testing

Besides ordinary unit tests, this package uses golden-file fixtures:
plain `.sql` files in `testdata/queries/`, each with an expected
parsed-tree dump checked into `testdata/queries/parser/*.sql.out`. Running
the tests with `-update` regenerates those dumps — always diff the result
by eye before committing, since a clean run just means "this is what the
parser produces now," not "this is correct."

```
go test ./pkg/duckdb/parser/...
go test ./pkg/duckdb/parser/... -update   # after an intentional output change
```

## Performance

This has been optimized fairly heavily for repeated parsing — the kind of
workload where the same process parses many different queries over its
lifetime, not just one. For that case, use `ParseString` as-is; it already
reuses scratch memory across calls internally. If you're calling into the
lower-level `peg` package directly and doing many parses back-to-back,
see the `peg.Session` type (and this directory's `CLAUDE.md` for the
guardrails around it).
