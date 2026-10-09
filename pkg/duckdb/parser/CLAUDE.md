# pkg/duckdb/parser — Agent Guide

This file supplements the root `CLAUDE.md`. Read that first for repo-wide
conventions (commit types, PR sections). This file covers what's specific to
working in this package.

## What this is

A hand-built parser for DuckDB's `SELECT` dialect: `ParseString(sql) ->
*SelectStatement`, a typed AST, plus a generic visitor (`Walk`/`Visitor`)
and a renderer (`Write`) that turns the AST back into SQL text. It exists
because no Go library parses DuckDB SQL specifically — see
`docs/plans/009-duckdb-select-parser.md` for the full rationale and
`docs/plans/011-duckdb-select-parser-plan.md` for the task-by-task build
log. Scope is deliberately narrow: read-only `SELECT` only, no DDL/DML, no
semantic validation — see that plan's Non-Goals section before adding
anything that looks like scope creep.

## Where things live

```
peg/            generic packrat PEG engine — knows nothing about SQL
grammar/        loads the vendored DuckDB .gram files into a peg.Grammar
grammar/vendor/ the vendored .gram/keyword files themselves — DO NOT hand-edit
primitives.go   the leaf-token matchers (identifiers, literals, keywords) peg calls into
parser.go       ParseString — the public entry point
ast.go          the typed AST node definitions
adapter_*.go    walks the peg CST into the typed AST, one file per grammar area
visitor.go      Node/Visitor/Walk/Dump — generic tree walking
write.go        Write — renders the AST back to SQL text
errors.go       ParseError
testdata/       golden-fixture SQL inputs + expected Dump() output
```

## Architecture, in one pass

```
SQL text --peg.Parser.Parse--> *peg.Node CST --adapter--> typed AST (*SelectStatement)
```

1. **`peg/`** is a generic packrat PEG engine with no notion of "SQL" — it
   interprets a `peg.Grammar` (rules + expressions) against an input string
   and primitive functions you supply, building a CST of `*peg.Node`.
2. **`grammar/`** loads DuckDB's own vendored `.gram` grammar files
   (`grammar/vendor/statements/*.gram`) into a `peg.Grammar` at
   `sync.Once`-guarded startup. This package's grammar *is* DuckDB's
   grammar, interpreted directly — not hand-transcribed. That's the whole
   point: re-vendoring a newer DuckDB commit should only ever require
   fixing adapter call sites, not re-deriving the grammar.
3. **`primitives.go`** supplies the leaf matchers (`Identifier`,
   `NumberLiteral`, keyword sets, etc.) that DuckDB's own grammar expects to
   be supplied out-of-band — see `grammar/vendor/README.md`'s "Rule-name
   overrides" section for exactly which rule names these are and why.
4. **`adapter_*.go`** walks the resulting CST into the typed AST in
   `ast.go`. This is where DuckDB grammar knowledge — rule names, sequence
   child indices, cardinality — actually lives in Go form. If a re-vendor
   changes a rule's shape, this is what breaks.
5. **`visitor.go`** / **`write.go`** operate purely on the typed AST, no
   `peg` dependency at all.

## The two parse APIs — read this before touching hot paths

`peg.Parser` has two ways to run a parse, and mixing them up is a
use-after-reuse bug, not a compile error:

- **`Parser.Parse(input, root)`** builds a fresh evaluator every call. The
  returned `*peg.Node` is valid indefinitely. Use this for one-off/rare
  parses, or anywhere the tree might outlive the call.
- **`Parser.AcquireSession()` / `Session.Parse(...)` / `Session.Release()`**
  reuses one evaluator's scratch buffers (node/child/env slabs, memo maps,
  skip cache) across calls via a `sync.Pool` — the tree returned by
  `Session.Parse` is only valid until the *next* `Session.Parse` or
  `Session.Release` on that same session, because reuse is exactly what
  overwrites that memory. `ParseString` uses this: it acquires a session,
  `defer`s `Release()`, and fully walks the CST into the owned AST before
  that defer fires. **Never** call `Session.Release()` before you're done
  reading the tree, and never call it twice.

If you're adding a new parse entry point, default to `Parse`, not
`Session` — only reach for `Session` if you've profiled and the caller
genuinely parses many inputs back-to-back (see `parser_bench_test.go`'s
`BenchmarkPegParseSession` vs `BenchmarkPegParse` for what the difference
looks like in practice).

## Performance history

This package's hot path (`peg/packrat.go`'s `evaluator`) has been through
several rounds of allocation/latency tuning — bump-allocated node/children
slabs instead of one heap alloc per CST node, primitives writing into that
same arena, a split ref/call memo map, interned rule-name IDs instead of
map[string], and the `Session` pooling described above. If you're touching
`peg/packrat.go`, `primitives.go`, or `adapter_expr.go`'s `positionAt`/`pos`,
run the benchmarks before and after:

```
go test ./pkg/duckdb/parser/... -bench=BenchmarkPegParse -benchtime=300ms -run=^$ -count=6
go test ./pkg/duckdb/parser/... -bench=BenchmarkParseString -benchtime=300ms -run=^$ -count=6
```

`BenchmarkPegParse` isolates the raw grammar-matching cost;
`BenchmarkParseString` is the real entry point (matching + adapt). A
handful of things worth knowing if you're optimizing further:
- `-race` first, always — the `Session` pool and the `Parser`-level
  `dispatch` table (built once, lazily, via `sync.Once`) are both shared,
  concurrently-accessed state.
- `peg.Node.Value` is a plain `string` + `HasValue bool`, not `any` —
  deliberately, since every primitive in this package only ever decodes to
  a string, and boxing into an interface on every primitive match (tried
  speculatively very often during backtracking) was a real, measured cost.
  See `adapter_empty_literal_test.go` for the edge case (`''`/`""` decoding
  to a legitimately-empty string) that makes `HasValue` necessary instead
  of just checking `Value != ""`.
- Profiling this benchmark on a busy machine, most CPU samples land in
  GC/scheduler frames (`runtime.kevent`, `pthread_cond_wait`, etc.), not
  this package's own code — `go tool pprof -focus='parser\.'` to see what's
  actually ours. Bytes/op, not just allocs/op, is what moves wall-clock
  time at this point; a lower alloc *count* with a higher byte count is
  worse, not better.

## Testing

- **Unit tests** are colocated per adapter file
  (`adapter_expr_test.go`, `primitives_test.go`, etc.).
- **Golden fixtures**: `testdata/queries/*.sql` in, `Dump()`'d AST out
  (`testdata/queries/parser/*.sql.out`), via `goldie`. Regenerate with
  `go test ./pkg/duckdb/parser/... -update` — **always read the diff before
  committing** a regenerated fixture; a green `-update` run doesn't mean
  the new output is *correct*.
- **`peg/` package tests** are grammar-agnostic — they exercise the packrat
  engine itself against small synthetic test grammars.

## Re-vendoring DuckDB's grammar

Full procedure and the most recent bump's notes are in
`grammar/vendor/README.md`. The one thing worth repeating here because it's
easy to miss: **a green `go test` run after a re-vendor is necessary but not
sufficient.** A cardinality change (`?` -> `*`/`+`, or the reverse) at a
grammar position no existing fixture exercises multiple/absent occurrences
of will not fail any test by itself — `present()`/`repeatChildren()` only
panic on a `Kind` mismatch, not a widened-but-still-valid cardinality. Read
the upstream diff for `?`/`*`/`+` changes specifically and hand-verify
anything that changed, per that README's step 7.
