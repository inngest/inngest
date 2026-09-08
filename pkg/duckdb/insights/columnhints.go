package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// buildColumnHints returns one ColumnHint per column the *executed* query
// will return, left to right (positional, not name-based). StarExpr items
// are expanded using each in-scope table's static columnOrder, never by
// asking the database. ctes is needed here only for a top-level
// UNION/INTERSECT/EXCEPT statement, whose operands resolve their own scope
// independently.
func buildColumnHints(stmt *parser.SelectStatement, scope *tableScope, ctes map[string]logicalTable) []ColumnHint {
	if stmt.SetOp != parser.SetOpNone {
		return unionColumnHints(stmt, ctes)
	}
	var hints []ColumnHint
	for _, item := range stmt.Columns {
		if star, ok := item.Expr.(*parser.StarExpr); ok {
			for _, sc := range resolveStarColumns(star, scope) {
				hints = append(hints, sc.col.hint)
			}
			continue
		}
		hints = append(hints, resolveItemHint(item.Expr, scope))
	}
	return hints
}

// unionColumnHints reconciles a UNION/INTERSECT/EXCEPT statement's two
// operands' hints positionally: a position gets a hint only when both
// sides independently agree on it; a mismatch resolves to HintNone rather
// than guessing which side is "right". Recurses naturally for a multi-way
// UNION ("a UNION b UNION c" parses as SetLeft = "a UNION b", SetRight =
// c): buildColumnHints re-enters this function when SetLeft is itself a
// SetOp statement (its From is always nil; each leaf level resolves its
// own real scope here).
func unionColumnHints(stmt *parser.SelectStatement, ctes map[string]logicalTable) []ColumnHint {
	leftScope, err := resolveScope(stmt.SetLeft.From, ctes)
	if err != nil {
		return nil
	}
	rightScope, err := resolveScope(stmt.SetRight.From, ctes)
	if err != nil {
		return nil
	}
	left := buildColumnHints(stmt.SetLeft, leftScope, ctes)
	right := buildColumnHints(stmt.SetRight, rightScope, ctes)

	n := min(len(right), len(left))
	hints := make([]ColumnHint, n)
	for i := range n {
		if left[i] == right[i] {
			hints[i] = left[i]
		}
	}
	return hints
}

// starColumn pairs a resolved star-expansion column's name with its known
// definition — shared by buildColumnHints and deriveTable so the
// qualifier/exclude resolution logic lives in exactly one place.
type starColumn struct {
	name string
	col  knownColumn
}

// resolveStarColumns resolves a StarExpr (bare "*", "t.*", or either with
// "EXCLUDE (...)") to the columns it expands to, in the same left-to-right
// order DuckDB itself would project them — every in-scope table's own
// columnOrder, in scope order for a bare "*", or just the qualified
// table's for "t.*".
func resolveStarColumns(star *parser.StarExpr, scope *tableScope) []starColumn {
	excluded := map[string]bool{}
	for _, c := range star.Exclude {
		excluded[c] = true
	}

	var tables []logicalTable
	if len(star.Qualifier) == 1 {
		if tbl, ok := scope.lookup(star.Qualifier[0]); ok {
			tables = []logicalTable{tbl}
		}
	} else {
		for _, name := range scope.names() {
			tbl, _ := scope.lookup(name)
			tables = append(tables, tbl)
		}
	}

	var out []starColumn
	for _, tbl := range tables {
		for _, name := range tbl.columnOrder {
			if excluded[name] {
				continue
			}
			out = append(out, starColumn{name: name, col: tbl.columns[name]})
		}
	}
	return out
}

// resolveItemHint traces expr back to a known column or a known JSON
// sub-path on a known column, returning HintNone when it isn't a bare/
// aliased/qualified reference to exactly one.
//
// UNNEST(...) and a single-index "[n]" access both extract exactly one
// array element, so both defer to resolveArrayElementHint rather than
// inheriting whatever hint the array itself carries — a column/path's own
// hint describes its own value, not necessarily "each element".
func resolveItemHint(expr parser.Expr, scope *tableScope) ColumnHint {
	if id, ok := expr.(*parser.Ident); ok {
		return resolveIdentHint(id, scope)
	}
	if col, path, ok := jsonPathAccess(expr); ok {
		return resolveSubPathHint(col, path, scope)
	}
	if f, ok := expr.(*parser.FunctionExpr); ok && isUnnest(f) {
		return resolveArrayElementHint(f.Args[0], scope)
	}
	if s, ok := expr.(*parser.SliceExpr); ok && isSingleIndex(s) {
		return resolveArrayElementHint(s.X, scope)
	}
	return HintNone
}

// resolveArrayElementHint resolves the hint for one element of the array x
// evaluates to — x is UNNEST(...)'s argument, a single-index "[n]" access's
// operand, or a FROM-clause UNNEST's argument, all of which extract
// exactly one array element.
//
// This only ever consults an explicit "<path>[*]" pathHints entry — there
// is no fallback to x's own whole-value hint. That whole-value hint can
// describe something else entirely (a JSON object with a hinted sub-field,
// or a plain scalar that isn't an array at all), so only a column/path
// that explicitly declares "[*]" is asserted to hold multiple values of
// that hinted type.
func resolveArrayElementHint(x parser.Expr, scope *tableScope) ColumnHint {
	if id, ok := x.(*parser.Ident); ok {
		return arrayElementHintForIdent(id, scope)
	}
	if col, path, ok := jsonPathAccess(x); ok {
		known, ok := resolveKnownColumn(col, scope)
		if !ok {
			return HintNone
		}
		return known.pathHints[normalizeJSONPath(path)+"[*]"]
	}
	return HintNone
}

// arrayElementHintForIdent resolves id's explicit "[*]" pathHints entry —
// a bare column or a table-qualified one — mirroring resolveIdentHint's own
// case-1/case-2 precedence (a bare column always wins over an unqualified
// table lookup).
func arrayElementHintForIdent(id *parser.Ident, scope *tableScope) ColumnHint {
	switch len(id.Parts) {
	case 1:
		col, ok := scope.uniqueColumn(id.Parts[0])
		if !ok {
			return HintNone
		}
		return col.pathHints["[*]"]
	case 2:
		if tbl, ok := scope.lookup(id.Parts[0]); ok {
			col, ok := tbl.columns[id.Parts[1]]
			if !ok {
				return HintNone
			}
			return col.pathHints["[*]"]
		}
		return HintNone
	default:
		return HintNone
	}
}

func isUnnest(f *parser.FunctionExpr) bool {
	return len(f.Args) == 1 && len(f.Name) == 1 && strings.EqualFold(f.Name[0], "UNNEST")
}

// isSingleIndex reports whether s is a single-element index ("expr[n]"),
// not a range slice ("expr[a:b]" / "expr[a:b:c]") -- only a single index
// extracts exactly one array element, so only that shape can inherit its
// operand's hint.
func isSingleIndex(s *parser.SliceExpr) bool {
	return !s.HasStep && s.Stop == nil
}

func resolveIdentHint(id *parser.Ident, scope *tableScope) ColumnHint {
	switch len(id.Parts) {
	case 1:
		col, ok := scope.uniqueColumn(id.Parts[0])
		if !ok {
			return HintNone
		}
		return col.hint
	case 2:
		if tbl, ok := scope.lookup(id.Parts[0]); ok {
			col, ok := tbl.columns[id.Parts[1]]
			if !ok {
				return HintNone
			}
			return col.hint
		}
		// Not a table qualifier: check whether Parts[0] is itself an
		// arrayOfStructs column (e.g. sessions.field), matching
		// arrayOfStructRewriter.rewriteIdent's precedence: a bare column
		// name always wins over treating it as an unqualified table.
		if hint := arrayOfStructsFieldHint(id.Parts[0], id.Parts[1], scope); hint != HintNone {
			return hint
		}
		// Or a quoted-dot JSON sub-path access (e.g.
		// attributes."_inngest.function.slug"): DuckDB parses
		// this identically to a plain 2-part column reference, not as a
		// jsonPathAccess shape, so it needs its own fallback
		// to the same pathHints resolveSubPathHint uses.
		return resolveSubPathHint(&parser.Ident{Parts: []string{id.Parts[0]}}, id.Parts[1], scope)
	case 3:
		tbl, ok := scope.lookup(id.Parts[0])
		if !ok {
			return HintNone
		}
		col, ok := tbl.columns[id.Parts[1]]
		if !ok || !col.arrayOfStructs {
			return HintNone
		}
		return col.pathHints["[*]."+id.Parts[2]]
	default:
		return HintNone
	}
}

// arrayOfStructsFieldHint resolves colName.field's hint when colName is a
// knownColumn.arrayOfStructs column in scope — see rewriteIdent
// (rewrite.go), which rewrites this exact shape into the JSONPath
// wildcard form the "[*].field" pathHints key convention (tables.go)
// mirrors.
func arrayOfStructsFieldHint(colName, field string, scope *tableScope) ColumnHint {
	col, ok := scope.uniqueColumn(colName)
	if !ok || !col.arrayOfStructs {
		return HintNone
	}
	return col.pathHints["[*]."+field]
}

func resolveSubPathHint(id *parser.Ident, path string, scope *tableScope) ColumnHint {
	col, ok := resolveKnownColumn(id, scope)
	if !ok || col.pathHints == nil {
		return HintNone
	}
	return col.pathHints[normalizeJSONPath(path)]
}

// resolveKnownColumn resolves id to its knownColumn, whether id is a bare
// column name (scope.uniqueColumn — errors on ambiguity across a JOIN
// rather than guessing) or a table-qualified one (e.g.
// extended_trace_spans.attributes, needed once two joined tables both
// declare that column name and a bare reference would be ambiguous) —
// mirroring resolveIdentHint's own case-1/case-2 lookup, but for
// jsonPathAccess's callers rather than a plain column reference.
func resolveKnownColumn(id *parser.Ident, scope *tableScope) (knownColumn, bool) {
	switch len(id.Parts) {
	case 1:
		return scope.uniqueColumn(id.Parts[0])
	case 2:
		tbl, ok := scope.lookup(id.Parts[0])
		if !ok {
			return knownColumn{}, false
		}
		col, ok := tbl.columns[id.Parts[1]]
		return col, ok
	default:
		return knownColumn{}, false
	}
}

// normalizeJSONPath strips path's optional leading "$" — DuckDB accepts a
// JSONPath operand either way ('$[*].field' or '[*].field'), but this
// package's pathHints keys are always stored without it (tables.go), so
// every lookup site normalizes through here rather than each caller
// re-deriving the same rule.
func normalizeJSONPath(path string) string {
	return strings.TrimPrefix(path, "$")
}
