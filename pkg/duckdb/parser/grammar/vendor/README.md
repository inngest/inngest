# Vendored DuckDB grammar

Pinned to the commit in `VERSION`. Fetched verbatim from
`https://raw.githubusercontent.com/duckdb/duckdb/<VERSION>/src/parser/peg/grammar/...`
— do not hand-edit these files.

## Rule-name overrides

A fixed set of leaf rule names are intercepted by name in DuckDB's own
`MatcherFactory::CreateRootMatcher` (see `matcher_factory.cpp` at the pinned
commit) rather than parsed from their textual `.gram` body — the text below
for these rules is present because DuckDB's own grammar files define it, but
`pkg/duckdb/parser/primitives.go` never evaluates it; it satisfies these
names directly instead:

`Identifier`, `ReservedIdentifier`, `CatalogName`, `SchemaName`,
`ReservedSchemaName`, `TableName`, `ReservedTableName`, `ColumnName`,
`ReservedColumnName`, `IndexName`, `ReservedIndexName`, `SequenceName`,
`FunctionName`, `ReservedFunctionName`, `TableFunctionName`, `TypeName`,
`ReservedTypeName`, `PragmaName`, `SettingName`, `CopyOptionName`,
`NumberLiteral`, `StringLiteral`, `OperatorLiteral`, `UnreservedKeyword`,
`ReservedKeyword`, `ColumnNameKeyword`, `FuncNameKeyword`, `TypeNameKeyword`,
`EndOfInput` (this last one has no `.gram` body at all upstream either).

Of these, `UnreservedKeyword`/`ColumnNameKeyword`/`FuncNameKeyword`/`TypeNameKeyword`
aren't actually overridden upstream (DuckDB really does evaluate a
generated ~300-alternative literal choice rule for each, built from the
matching `keywords/*.list` file) — this port reimplements them as a
constant-time set-membership check against that same `.list` file instead,
which is semantically identical (a PEG ordered choice of exhaustive,
mutually-exclusive single-word literals *is* a set-membership check) and
avoids vendoring the generated choice-rule text. Every other name in the
list above is a genuine upstream override with no equivalent `.gram` body
worth vendoring at all.

`statements/base.gram` isn't fetched from a single upstream URL — see its
own Step 1b (docs/plans/011-duckdb-select-parser-plan.md, Task 1) for why
(it's hand-extracted from a build-generated file, not one of the individual
`statements/*.gram` sources).

## Re-vendoring

1. Resolve the target commit (`git ls-remote https://github.com/duckdb/duckdb.git HEAD`, or a specific pinned SHA if intentionally not tracking `main`).
2. Diff every vendored `statements/*.gram` and `keywords/*.list` file against that commit's copy before overwriting anything — a rule rename or restructure is invisible until you go looking for it, `go test` will only tell you *that* something broke, not *why*.
3. Re-check `base.gram`'s 19 hand-extracted lines against the new commit's `inlined_grammar.gram` (no single stable URL vendors this file — see Task 1's plan entry for why it's hand-extracted at all).
4. Update `VERSION` and overwrite only the files that actually differ.
5. Update `vendor_test.go`'s `wantLines` to the new line counts.
6. `go test ./pkg/duckdb/parser/... -v` and fix whatever it surfaces — almost always a handful of `seq.Children[N]` indices or `case` names in the adapter files, localized to exactly the rules the diff in step 2 flagged.
7. **A green test run is necessary but not sufficient.** A cardinality change (`?`/single -> `*`/`+`, or the reverse) at a grammar position no existing fixture exercises multiple occurrences of will *not* show up as a test failure by itself — `present()`/`repeatChildren()` panic loudly on a `Kind` mismatch (catching the "wrong shape entirely" case), but a `?` silently widened to `*` still parses fine structurally; only the *value* extracted is wrong (or missing) for anyone using more than the previously-allowed cardinality. Read the diff for `?`/`*`/`+` changes specifically, and hand-write a quick verification (not necessarily a permanent fixture) for anything that changed cardinality, rather than trusting `go test`'s green alone.
8. Regenerate golden fixtures with `-update` only after confirming by eye that the new output is still correct.

**Last verified:** 2026-09-02, bumping from `1582849bf9e35de3fa3c330935ede5ce598e28a3` to `f931913e6b03beaf75df0f56695900f765dab426`. Upstream had changed three rules' cardinality (`CatalogReservedSchemaTable`'s `ReservedSchemaQualification` from single-mandatory to `+`; `QualifiedTableFunction`'s and `CatalogReservedSchemaFunctionName`'s `ReservedSchemaQualification`/`SchemaQualification` from `?` to `*` — DuckDB added support for deeply nested catalog/schema chains) and added one new `ColumnReference` alternative (`NestedSchemaTableColumnName`, for 5+-component references). `go test` stayed green through the bump itself, exactly because no existing fixture used more than one schema-qualification level — the cardinality bugs were only caught by writing new, targeted tests (`adapter_revendor_test.go`) for the changed positions specifically, confirming step 7 above isn't a hypothetical concern.
