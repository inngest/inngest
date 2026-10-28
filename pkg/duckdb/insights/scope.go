package insights

import (
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// ValidationError is returned by validate (Task 3) and resolveScope for
// any query this package rejects before it reaches DuckDB. End is the
// offending node's own End() -- zero (equal to Pos) only for the handful of
// construction sites with no single node to point at (e.g. a query-wide
// rejection) -- so Diagnostic() can underline the whole offending
// identifier/clause in the editor instead of a single-character caret.
type ValidationError struct {
	Pos     parser.Position
	End     parser.Position
	Message string
}

func (e *ValidationError) Error() string {
	if (e.Pos == parser.Position{}) {
		return "insights: " + e.Message
	}
	return fmt.Sprintf("insights: %s (at %s)", e.Message, e.Pos)
}

type scopeEntry struct {
	alias string
	table logicalTable
}

// tableScope is the set of logical tables a single SELECT's FROM clause
// resolves to, keyed by the name queries actually use to reference them
// (an explicit alias, or the logical table's own name).
type tableScope struct {
	entries []scopeEntry
	// outer is the scope this one may correlate into when a name isn't
	// found locally — non-nil only for an expression-position subquery
	// (validateWithCTEs threads its own scope down whenever it recurses
	// into a nested *parser.SelectStatement — correlatable per standard
	// SQL with no special syntax) or a LATERAL FROM-clause subquery
	// (addSubquery seeds it with the scope built so far from *earlier*
	// FROM items only, matching LATERAL's left-to-right visibility). nil
	// for a CTE body, a non-LATERAL FROM subquery, or the top-level
	// query — none of those can correlate, matching DuckDB's own rule
	// that a derived table needs LATERAL to see a preceding FROM item at
	// all.
	outer *tableScope
}

// resolveScope walks from — bare table references, JOINs of them, and
// FROM-clause subqueries — building a tableScope, or returns a
// *ValidationError for any unknown table or unsupported FROM-clause shape.
// ctes is whatever CTEs are visible at this lexical point, checked before
// the static logicalTables registry, so a CTE can shadow a real table
// name, matching standard SQL.
// diags is passed straight through to any FROM-clause subquery's own
// deriveTable call (addSubquery below) -- nil from every caller except
// stageValidate's own top-level walk (validateWithCTEs), matching that
// function's own doc comment on why a re-resolution passes nil instead.
func resolveScope(from *parser.FromClause, ctes map[string]logicalTable, diags *[]Diagnostic) (*tableScope, error) {
	s := &tableScope{}
	if from == nil {
		return s, nil
	}
	for _, ref := range from.Refs {
		if err := s.addRef(ref, ctes, diags); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *tableScope) addRef(ref parser.TableRef, ctes map[string]logicalTable, diags *[]Diagnostic) error {
	switch r := ref.(type) {
	case *parser.BaseTableRef:
		return s.addBaseTable(r, ctes)
	case *parser.JoinRef:
		if err := s.addRef(r.Left, ctes, diags); err != nil {
			return err
		}
		return s.addRef(r.Right, ctes, diags)
	case *parser.TableSubqueryRef:
		return s.addSubquery(r, ctes, diags)
	case *parser.TableFunctionRef:
		return s.addTableFunction(r)
	default:
		return &ValidationError{Pos: ref.Pos(), End: ref.End(), Message: "unsupported FROM clause shape"}
	}
}

// addTableFunction admits exactly one FROM-clause table function: UNNEST,
// which only ever expands array/list data already present in a query's own
// scope — unlike read_csv/read_parquet/ATTACH and the rest of DuckDB's
// table-function surface, it can't reach the filesystem or catalog, which
// is why table functions are otherwise excluded from the FROM clause.
// UNNEST's own argument expression is validated like any other, scoped
// against the query's own FROM list — no LATERAL needed, unlike a general
// correlated subquery.
//
// The parser can't yet represent UNNEST's optional column-alias list ("AS
// alias(colname)" panics as unimplemented), so the produced column is
// always named "unnest", matching DuckDB's own default. Its element type
// isn't tracked at this granularity, so it's reported as the same generic
// JSON bucket DuckDBToColumnType gives any composite type.
func (s *tableScope) addTableFunction(r *parser.TableFunctionRef) error {
	name := strings.ToUpper(strings.Join(r.Name, "."))
	if name != "UNNEST" {
		return &ValidationError{Pos: r.Pos(), End: r.End(), Message: fmt.Sprintf("table function %q is not allowed", strings.Join(r.Name, "."))}
	}
	if len(r.Args) != 1 {
		return &ValidationError{Pos: r.Pos(), End: r.End(), Message: "UNNEST in a FROM clause takes exactly one argument"}
	}

	alias := r.Alias
	if alias == "" {
		alias = "unnest"
	}
	// Unnesting an explicitly-declared array (e.g.
	// attributes -> '_inngest.defer.parent_run_ids', whose "[*]" pathHints
	// entry says each element is a run ID) yields elements of that same
	// hint — resolveArrayElementHint already knows how to trace r.Args[0]
	// back to one; s is the scope its argument is evaluated against.
	tbl := logicalTable{
		name:        alias,
		columnOrder: []string{"unnest"},
		columns:     map[string]knownColumn{"unnest": {colType: ColumnTypeJSON, pathHints: rootHint(resolveArrayElementHint(r.Args[0], s))}},
	}
	s.entries = append(s.entries, scopeEntry{alias: alias, table: tbl})
	return nil
}

func (s *tableScope) addBaseTable(r *parser.BaseTableRef, ctes map[string]logicalTable) error {
	if len(r.Name) != 1 {
		return &ValidationError{Pos: r.Pos(), End: r.End(), Message: fmt.Sprintf("unknown table %q", strings.Join(r.Name, "."))}
	}
	tbl, ok := ctes[strings.ToLower(r.Name[0])]
	if !ok {
		tbl, ok = lookupLogicalTable(r.Name[0])
	}
	if !ok {
		return &ValidationError{Pos: r.Pos(), End: r.End(), Message: fmt.Sprintf("unknown table %q", r.Name[0])}
	}
	alias := r.Alias
	if alias == "" {
		alias = r.Name[0]
	}
	s.entries = append(s.entries, scopeEntry{alias: alias, table: tbl})
	return nil
}

// addSubquery validates r.Select — independently, unless r.Lateral, in
// which case s itself (built so far from strictly earlier FROM items)
// becomes its outer scope — and exposes its inferred output columns under
// r.Alias, which DuckDB itself requires a FROM-clause subquery to have.
func (s *tableScope) addSubquery(r *parser.TableSubqueryRef, ctes map[string]logicalTable, diags *[]Diagnostic) error {
	if r.Alias == "" {
		return &ValidationError{Pos: r.Pos(), End: r.End(), Message: "a subquery in FROM must have an alias"}
	}
	var outer *tableScope
	if r.Lateral {
		outer = s
	}
	tbl, err := deriveTable(r.Select, r.Alias, nil, ctes, outer, diags)
	if err != nil {
		return err
	}
	s.entries = append(s.entries, scopeEntry{alias: r.Alias, table: tbl})
	return nil
}

func (s *tableScope) lookup(alias string) (logicalTable, bool) {
	e, ok := s.lookupEntry(alias)
	return e.table, ok
}

func (s *tableScope) lookupEntry(alias string) (scopeEntry, bool) {
	for _, e := range s.entries {
		if strings.EqualFold(e.alias, alias) {
			return e, true
		}
	}
	if s.outer != nil {
		return s.outer.lookupEntry(alias)
	}
	return scopeEntry{}, false
}

// starEntries returns the scope entries a star with this qualifier expands
// over: every local entry, in FROM order, for a bare "*", or just the
// qualified one for "t.*" (false when t isn't in scope).
func (s *tableScope) starEntries(qualifier []string) ([]scopeEntry, bool) {
	switch len(qualifier) {
	case 0:
		return s.entries, true
	case 1:
		e, ok := s.lookupEntry(qualifier[0])
		if !ok {
			return nil, false
		}
		return []scopeEntry{e}, true
	default:
		return nil, false
	}
}

// excludeName is one parsed EXCLUDE list entry: column, optionally
// qualified by a table alias ("t.col"), which then only excludes that
// table's own copy of the column.
type excludeName struct {
	qualifier, column string
}

// parseExcludeName parses one raw EXCLUDE entry (parser.SplitExcludeName).
// ok is false for anything past "t.col".
func parseExcludeName(raw string) (excludeName, bool) {
	switch parts := parser.SplitExcludeName(raw); len(parts) {
	case 1:
		return excludeName{column: parts[0]}, true
	case 2:
		return excludeName{qualifier: parts[0], column: parts[1]}, true
	default:
		return excludeName{}, false
	}
}

// matches reports whether this EXCLUDE entry removes column name of the
// scope entry aliased alias.
func (x excludeName) matches(alias, name string) bool {
	if !strings.EqualFold(x.column, name) {
		return false
	}
	return x.qualifier == "" || strings.EqualFold(x.qualifier, alias)
}

// uniqueColumn returns name's knownColumn only when exactly one table in
// scope has a column by that name — used by buildColumnHints, where an
// ambiguous match must resolve to no hint rather than a guess. Falls back
// to the outer scope only when this scope has no local match at all —
// mirrors columnCount's own precedence exactly, so the two never disagree
// about which scope "owns" a name.
func (s *tableScope) uniqueColumn(name string) (knownColumn, bool) {
	var found knownColumn
	for _, e := range s.entries {
		if col, ok := e.table.column(name); ok {
			found = col
		}
	}
	switch s.localColumnCount(name) {
	case 0:
		if s.outer != nil {
			return s.outer.uniqueColumn(name)
		}
		return knownColumn{}, false
	case 1:
		return found, true
	default:
		return knownColumn{}, false
	}
}

// localColumnCount is columnCount without the correlation fallback —
// columnCount and uniqueColumn both need to know "does *this* scope have
// any match at all" before deciding whether to check s.outer.
func (s *tableScope) localColumnCount(name string) int {
	count := 0
	for _, e := range s.entries {
		if _, ok := e.table.column(name); ok {
			count++
		}
	}
	return count
}

// columnCount reports how many tables have a column by this name — 0
// (unknown), 1 (fine), or >1 (ambiguous, needs a table qualifier) —
// resolved at the nearest scope in the correlation chain that has any
// match at all. Ambiguity is only ever computed within that one scope,
// never merged across a correlation boundary — standard SQL name
// resolution: a bare name resolves to the closest enclosing query block
// that has it at all, and only that block's own tables decide whether
// it's ambiguous. validate uses this to give an accurate error message
// instead of collapsing "unknown" and "ambiguous" together.
func (s *tableScope) columnCount(name string) int {
	if n := s.localColumnCount(name); n > 0 {
		return n
	}
	if s.outer != nil {
		return s.outer.columnCount(name)
	}
	return 0
}
