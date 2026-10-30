package insights

import (
	"fmt"
	"reflect"
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
	// hiddenBy is the SEMI/ANTI join whose right side this entry is on:
	// DuckDB only exposes that join's left side, so the entry is only
	// visible inside hiddenBy's own ON condition (onScope) and nowhere
	// else (a later join, "*", a bare or qualified reference, a LATERAL
	// subquery).
	hiddenBy *parser.JoinRef
}

func (e scopeEntry) hidden() bool { return e.hiddenBy != nil }

// usingColumn is one table's copy of a column a JOIN ... USING (or a
// NATURAL join) merged: entry indexes tableScope.entries, column is
// lowercased.
type usingColumn struct {
	entry  int
	column string
}

// tableScope is the set of logical tables a single SELECT's FROM clause
// resolves to, keyed by the name queries actually use to reference them
// (an explicit alias, or the logical table's own name).
type tableScope struct {
	entries []scopeEntry
	// using maps every column copy a USING/NATURAL join merged to its
	// merged set's id: DuckDB projects each set once from a bare "*" (at
	// the first copy's position) and resolves a bare reference to it
	// unambiguously. A multi-way "a JOIN b USING (k) JOIN c USING (k)"
	// merges all three copies into one set.
	using     map[usingColumn]int
	nextUsing int
	// usingCols is each using set's merged output column, by set id: the
	// left copy for an INNER/LEFT/SEMI/ANTI join, the right copy for a
	// RIGHT join, and COALESCE(left, right) for a FULL join (see
	// mergeJoinColumn).
	usingCols map[int]knownColumn
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
		lo := len(s.entries)
		if err := s.addRef(r.Left, ctes, diags); err != nil {
			return err
		}
		mid := len(s.entries)
		if err := s.addRef(r.Right, ctes, diags); err != nil {
			return err
		}
		return s.addJoinColumns(r, lo, mid, len(s.entries))
	case *parser.TableSubqueryRef:
		return s.addSubquery(r, ctes, diags)
	case *parser.TableFunctionRef:
		return s.addTableFunction(r)
	default:
		return &ValidationError{Pos: ref.Pos(), End: ref.End(), Message: "unsupported FROM clause shape"}
	}
}

// addJoinColumns applies r's own column semantics once both its sides'
// entries are in scope (left is entries[lo:mid], right entries[mid:hi]):
// each USING (or NATURAL join's shared) column must exist, unambiguously,
// on both sides -- the same binder errors DuckDB reports -- and is merged
// into one using set; a SEMI/ANTI join's right side is then hidden from
// everything but r's own ON condition.
func (s *tableScope) addJoinColumns(r *parser.JoinRef, lo, mid, hi int) error {
	typ := strings.ToUpper(r.Type)
	names := r.Using
	if strings.HasPrefix(typ, "NATURAL") {
		names = nil
		seen := map[string]bool{}
		for _, e := range s.entries[lo:mid] {
			if e.hidden() {
				continue
			}
			for _, name := range e.table.columnOrder {
				key := strings.ToLower(name)
				if seen[key] {
					continue
				}
				seen[key] = true
				if len(s.sideColumns(mid, hi, name)) > 0 {
					names = append(names, name)
				}
			}
		}
	}

	for _, name := range names {
		left, err := s.joinSideColumn(r, lo, mid, name, "left")
		if err != nil {
			return err
		}
		right, err := s.joinSideColumn(r, mid, hi, name, "right")
		if err != nil {
			return err
		}
		s.mergeUsing(left, right, typ)
	}

	if strings.Contains(typ, "SEMI") || strings.Contains(typ, "ANTI") {
		for i := mid; i < hi; i++ {
			// An entry an inner SEMI/ANTI join already hid stays hidden
			// by that join, so r's own ON can't see it either.
			if !s.entries[i].hidden() {
				s.entries[i].hiddenBy = r
			}
		}
	}
	return nil
}

// onScope returns the scope r's ON condition is checked against: s itself,
// except that a SEMI/ANTI join's own right side (hidden from everything
// else) is visible.
func (s *tableScope) onScope(r *parser.JoinRef) *tableScope {
	var out *tableScope
	for i, e := range s.entries {
		if e.hiddenBy != r {
			continue
		}
		if out == nil {
			c := *s
			c.entries = append([]scopeEntry(nil), s.entries...)
			out = &c
		}
		out.entries[i].hiddenBy = nil
	}
	if out == nil {
		return s
	}
	return out
}

// sideColumns returns every visible copy of column name among
// entries[lo:hi], collapsing copies an earlier USING/NATURAL join already
// merged into one (the first).
func (s *tableScope) sideColumns(lo, hi int, name string) []usingColumn {
	var out []usingColumn
	sets := map[int]bool{}
	for i := lo; i < hi; i++ {
		if s.entries[i].hidden() {
			continue
		}
		if _, ok := s.entries[i].table.column(name); !ok {
			continue
		}
		uc := usingColumn{entry: i, column: strings.ToLower(name)}
		if id, ok := s.using[uc]; ok {
			if sets[id] {
				continue
			}
			sets[id] = true
		}
		out = append(out, uc)
	}
	return out
}

func (s *tableScope) joinSideColumn(r *parser.JoinRef, lo, hi int, name, side string) (usingColumn, error) {
	switch cols := s.sideColumns(lo, hi, name); len(cols) {
	case 0:
		return usingColumn{}, &ValidationError{Pos: r.Pos(), End: r.End(), Message: fmt.Sprintf("column %q does not exist on %s side of join", name, side)}
	case 1:
		return cols[0], nil
	default:
		return usingColumn{}, &ValidationError{Pos: r.Pos(), End: r.End(), Message: fmt.Sprintf("ambiguous column %q on %s side of join", name, side)}
	}
}

// mergeUsing puts a (the join's left copy) and b (its right copy), and
// whatever sets either already belongs to, into one using set whose merged
// column is mergeJoinColumn of the two for a join of type typ.
func (s *tableScope) mergeUsing(a, b usingColumn, typ string) {
	if s.using == nil {
		s.using = map[usingColumn]int{}
		s.usingCols = map[int]knownColumn{}
	}
	merged := mergeJoinColumn(s.columnOf(a), s.columnOf(b), typ)
	aID, aOK := s.using[a]
	bID, bOK := s.using[b]
	id := aID
	switch {
	case aOK && bOK:
		for uc, ucID := range s.using {
			if ucID == bID {
				s.using[uc] = aID
			}
		}
		delete(s.usingCols, bID)
	case aOK:
		s.using[b] = aID
	case bOK:
		id = bID
		s.using[a] = bID
	default:
		s.nextUsing++
		id = s.nextUsing
		s.using[a], s.using[b] = id, id
	}
	s.usingCols[id] = merged
}

// columnOf returns uc's column as a reference to it resolves: its using
// set's merged column if it's in one, or just its own table's column.
func (s *tableScope) columnOf(uc usingColumn) knownColumn {
	if id, ok := s.using[uc]; ok {
		return s.usingCols[id]
	}
	col, _ := s.entries[uc.entry].table.column(uc.column)
	return col
}

// mergeJoinColumn is the column DuckDB projects for a USING/NATURAL join's
// merged column (checked against a real duckdb): the left copy for every
// join type but RIGHT (the right copy) and FULL (COALESCE(left, right),
// so it only keeps whatever type/hints both sides agree on).
func mergeJoinColumn(left, right knownColumn, typ string) knownColumn {
	switch {
	case strings.Contains(typ, "RIGHT"):
		return right
	case strings.Contains(typ, "FULL"):
		if reflect.DeepEqual(left, right) {
			return left
		}
		out := knownColumn{colType: ColumnTypeUnknown}
		if left.colType == right.colType {
			out.colType = left.colType
		}
		if reflect.DeepEqual(left.pathHints, right.pathHints) {
			out.pathHints = left.pathHints
		}
		if left.description == right.description {
			out.description = left.description
		}
		out.arrayOfStructs = left.arrayOfStructs && right.arrayOfStructs
		return out
	default:
		return left
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
		if !e.hidden() && strings.EqualFold(e.alias, alias) {
			return e, true
		}
	}
	if s.outer != nil {
		return s.outer.lookupEntry(alias)
	}
	return scopeEntry{}, false
}

// starEntries returns the scope entries a star with this qualifier expands
// over: every local entry, in FROM order, for a bare "*" (callers skip
// hidden ones themselves, since entry indexes key tableScope.using), or
// just the qualified one for "t.*" (false when t isn't in scope).
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

// parseExcludeName parses star's i'th EXCLUDE entry from its decoded
// dotted parts (parser.StarExpr.ExcludeParts, or just Exclude[i] on a
// hand-built StarExpr without them), also returning the entry as written
// for error messages. ok is false for anything past "t.col".
func parseExcludeName(star *parser.StarExpr, i int) (x excludeName, display string, ok bool) {
	parts := []string{star.Exclude[i]}
	if star.ExcludeParts != nil {
		parts = star.ExcludeParts[i]
	}
	display = strings.Join(parts, ".")
	switch len(parts) {
	case 1:
		return excludeName{column: parts[0]}, display, true
	case 2:
		return excludeName{qualifier: parts[0], column: parts[1]}, display, true
	default:
		return excludeName{}, display, false
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
//
// A USING/NATURAL-merged column resolves to its merged column (columnOf),
// the same one a bare "*" projects.
func (s *tableScope) uniqueColumn(name string) (knownColumn, bool) {
	switch cols := s.sideColumns(0, len(s.entries), name); len(cols) {
	case 0:
		if s.outer != nil {
			return s.outer.uniqueColumn(name)
		}
		return knownColumn{}, false
	case 1:
		return s.columnOf(cols[0]), true
	default:
		return knownColumn{}, false
	}
}

// localColumnCount is columnCount without the correlation fallback —
// columnCount and uniqueColumn both need to know "does *this* scope have
// any match at all" before deciding whether to check s.outer. Copies a
// USING/NATURAL join merged count once, matching DuckDB ("SELECT k FROM a
// JOIN b USING (k)" isn't ambiguous).
func (s *tableScope) localColumnCount(name string) int {
	return len(s.sideColumns(0, len(s.entries), name))
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
