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
//
// This is now purely an internal helper for unionColumnPathHints' own
// reconciliation -- buildColumnPathHints (below) is every other caller's
// (Transpile's, ultimately Execute's/the GQL layer's) single source of
// hint information, root (whole-value) hint included.
func buildColumnHints(stmt *parser.SelectStatement, scope *tableScope, ctes map[string]logicalTable) []ColumnHint {
	if stmt.SetOp != parser.SetOpNone {
		return unionColumnHints(stmt, ctes)
	}
	var hints []ColumnHint
	for _, item := range stmt.Columns {
		if star, ok := item.Expr.(*parser.StarExpr); ok {
			for _, sc := range resolveStarColumns(star, scope) {
				hints = append(hints, sc.col.hint())
			}
			continue
		}
		hints = append(hints, resolveItemHint(item.Expr, scope))
	}
	return hints
}

// buildColumnPathHints returns one column's-worth of pathHints per output
// column, positionally aligned with the *executed* query's own result set
// -- nil for a position with neither a whole-value hint nor any known
// JSON sub-path hint.
//
// Every entry's whole-value (empty-Path) hint is folded in directly
// (withRootHint), from the exact same resolution resolveItemHint/
// unionColumnHints already do for any expression shape this package
// resolves a hint for at all (a bare/qualified column, a JSON sub-path
// access, UNNEST(...)/a single index, an arrayOfStructs field, ...) --
// this is the *only* place a whole-column hint is computed; there's no
// separate ColumnHints-shaped result living alongside this one anymore.
// A star or bare/qualified column reference layers that on top of the
// column's *other* pathHints entries too (sc.col.pathHints/
// resolveColumnPathHints), verbatim from tables.go; anything else (a
// computed expression, an already-indexed/dotted sub-access) gets at
// most the one root entry.
//
// A caller (the GQL layer, ultimately a UI) needing to resolve a specific
// JSON value found inside a column's own data reads the non-root entries
// the same way this package's own resolveArrayElementHint/
// resolveSubPathHint do: a {wc}-only Path describes every element of an
// unnested/indexed array, {wc, seg(field)} one field of every struct
// element, any other Path a direct JSON sub-path off the column's own
// top-level value.
func buildColumnPathHints(stmt *parser.SelectStatement, scope *tableScope, ctes map[string]logicalTable) [][]PathHint {
	if stmt.SetOp != parser.SetOpNone {
		return unionColumnPathHints(stmt, ctes)
	}
	var pathHints [][]PathHint
	for _, item := range stmt.Columns {
		if star, ok := item.Expr.(*parser.StarExpr); ok {
			for _, sc := range resolveStarColumns(star, scope) {
				pathHints = append(pathHints, sc.col.pathHints)
			}
			continue
		}
		pathHints = append(pathHints, exprPathHints(item.Expr, scope))
	}
	return pathHints
}

// exprPathHints returns expr's full pathHints exactly as buildColumnPathHints
// would report them for a SELECT list item of this same shape -- root
// (whole-value) entry included whenever resolveItemHint finds one.
// Factored out of buildColumnPathHints' own per-item loop so a
// hintTransform (preservesArgHint, listElementHint; functions.go) can
// compute "this representative argument's own top-level pathHints"
// without duplicating that combination logic -- MAX(FIRST(run_id)), say,
// resolves by recursing through this same function at each level.
func exprPathHints(expr parser.Expr, scope *tableScope) []PathHint {
	return withRootHint(resolveColumnPathHints(expr, scope), resolveItemHint(expr, scope))
}

// wrapArrayElementPathHints wraps every entry of ph (as returned by
// exprPathHints for a LIST/ARRAY_AGG call's own argument) one array
// -wildcard segment deeper, for listElementHint (functions.go): ph's own
// root (empty-Path) entry -- the argument's own whole-value hint --
// becomes the {wc} entry describing every element of the aggregated
// LIST, and any deeper entry (a JSON sub-path the argument column
// already declared) gets wc prepended the same way, matching how a
// knownColumn.arrayOfStructs field's own pathHints entries are shaped
// (tables.go).
func wrapArrayElementPathHints(ph []PathHint) []PathHint {
	if len(ph) == 0 {
		return nil
	}
	wrapped := make([]PathHint, len(ph))
	for i, p := range ph {
		path := make([]PathSegment, 0, len(p.Path)+1)
		path = append(path, wc)
		path = append(path, p.Path...)
		wrapped[i] = PathHint{Path: path, Hint: p.Hint}
	}
	return wrapped
}

// unionColumnPathHints is buildColumnPathHints' UNION/INTERSECT/EXCEPT
// counterpart: reconciling two operands' full pathHints isn't attempted
// (unionColumnHints' own doc comment explains why even the simpler
// whole-value case only trusts an exact agreement), so each position gets
// at most the one root entry unionColumnHints already agrees on.
func unionColumnPathHints(stmt *parser.SelectStatement, ctes map[string]logicalTable) [][]PathHint {
	hints := unionColumnHints(stmt, ctes)
	pathHints := make([][]PathHint, len(hints))
	for i, h := range hints {
		pathHints[i] = rootHint(h)
	}
	return pathHints
}

// resolveColumnPathHints returns the full []PathHint for a bare or
// table-qualified reference to exactly one known table column, a call to
// a function whose returnType (functions.go) reports one (LIST/
// ARRAY_AGG/UNNEST today), a JSON sub-path access (jsonPathAccess,
// projected through its base's own pathHints), or a single-index "[n]"
// access -- nil for anything else (some other computed expression, an
// unknown/ambiguous reference). Mirrors resolveIdentHint's own
// case-1/case-2 structure (a bare column always wins over an unqualified
// table lookup) for the Ident case, returning the column's pathHints
// instead of its single hint.
//
// Every non-Ident case above is expressed as a []PathHint-to-[]PathHint
// transform over some base's own (recursively resolved) pathHints --
// projectJSONSubPathHints/projectPathHints (pathhint.go) for a JSON
// sub-path access or a single-index access (prefix {wc}, unwrapping one
// array level), a function's own returnType for anything else -- so this
// composes to arbitrary nesting: list(inputs) -> '$[*][*].meta.sessions'
// recurses through the FunctionExpr case (listReturnType) and then this
// same jsonPathAccess case again for its own base.
func resolveColumnPathHints(expr parser.Expr, scope *tableScope) []PathHint {
	if base, path, ok := jsonPathAccess(expr); ok {
		return projectJSONSubPathHints(resolveColumnPathHints(base, scope), queryPath(path))
	}
	if f, ok := expr.(*parser.FunctionExpr); ok {
		_, ph, _ := functionReturnType(strings.ToLower(strings.Join(f.Name, ".")), f.Args, scope)
		return ph
	}
	if s, ok := expr.(*parser.SliceExpr); ok && isSingleIndex(s) {
		return projectPathHints(resolveColumnPathHints(s.X, scope), []PathSegment{wc})
	}
	id, ok := expr.(*parser.Ident)
	if !ok {
		return nil
	}
	switch len(id.Parts) {
	case 1:
		col, ok := scope.uniqueColumn(id.Parts[0])
		if !ok {
			return nil
		}
		return col.pathHints
	case 2:
		if tbl, ok := scope.lookup(id.Parts[0]); ok {
			col, ok := tbl.columns[id.Parts[1]]
			if !ok {
				return nil
			}
			return col.pathHints
		}
		return nil
	default:
		return nil
	}
}

// projectJSONSubPathHints is resolveColumnPathHints' jsonPathAccess case:
// baseHints (base's own pathHints) projected through this access's own
// queryPath (projectPathHints) -- plus, when path contains a wildcard
// segment anywhere, a duplicate {wc}-tagged sibling for every resulting
// root (whole-value) entry.
//
// DuckDB's own JSONPath wildcard extraction always collects its matches
// into a JSON array -- confirmed empirically against a real duckdb,
// including for multiple "[*]"s in one path, which flatten into a single
// combined result array rather than nesting further
// (list(inputs) -> '$[*][*].meta' returns one flat array of .meta
// values, not an array of arrays of them). This package's own
// convention (established well before this function --
// TestBuildColumnHintsRunInputsMetaSessions et al) *also* reports that
// array's element hint directly at this expression's own root level, for
// a UI's convenience: a bare `inputs ->> '$[*].meta.sessions'` SELECT
// item shows HintSession without requiring a further UNNEST. Reporting
// only that root entry would make the hint silently vanish the moment a
// caller *does* still UNNEST/index into the expression, even though the
// underlying value is genuinely still array-shaped -- the {wc} sibling
// added here is exactly what unnestReturnType/a single-index access
// (via resolveColumnPathHints' own cases above) needs to find there.
//
// A literal (non-"$"-prefixed) path -- a plain top-level JSON key, not a
// JSONPath expression at all (queryPath's own doc comment) -- never
// triggers this: only a real "[*]" wildcard implies DuckDB collected
// multiple matches into an array. attributes -> '_inngest.defer.parent_run_ids'
// (TestBuildColumnHintsUnnestOfHintedArray) doesn't need this special
// case at all -- attributes' own declared entry already carries an
// explicit trailing wc past that literal key, so projectPathHints alone
// already leaves a {wc} entry (not a root one) for UNNEST to find.
func projectJSONSubPathHints(baseHints []PathHint, path []PathSegment) []PathHint {
	projected := projectPathHints(baseHints, path)
	if !pathHasWildcard(path) {
		return projected
	}
	out := make([]PathHint, len(projected), len(projected)+1)
	copy(out, projected)
	for _, ph := range projected {
		if len(ph.Path) == 0 {
			out = append(out, PathHint{Path: []PathSegment{wc}, Hint: ph.Hint})
		}
	}
	return out
}

func pathHasWildcard(path []PathSegment) bool {
	for _, s := range path {
		if s.Wildcard {
			return true
		}
	}
	return false
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
	leftScope, err := resolveScope(stmt.SetLeft.From, ctes, nil)
	if err != nil {
		return nil
	}
	rightScope, err := resolveScope(stmt.SetRight.From, ctes, nil)
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
// Every non-Ident shape (a JSON sub-path access, UNNEST(...), a
// single-index "[n]" access, any other function call) defers to
// resolveColumnPathHints' own recursive resolution and just reads its
// root/whole-value entry, if it has one -- UNNEST(...)/a single-index
// access aren't special-cased here at all: they extract exactly one array
// element, which resolveColumnPathHints' own projectPathHints({wc}, ...)
// case already expresses as "strip one leading wc," landing on a root
// entry exactly when the element itself isn't still array-shaped. A
// column/path's own whole-value hint is deliberately not inherited by
// either of those two shapes for the same reason -- it describes the
// array itself, not necessarily "each element" -- which is why this is a
// projection (only a path that explicitly declares a leading wc
// contributes anything), not a fallback.
func resolveItemHint(expr parser.Expr, scope *tableScope) ColumnHint {
	if id, ok := expr.(*parser.Ident); ok {
		return resolveIdentHint(id, scope)
	}
	return rootHintOf(resolveColumnPathHints(expr, scope))
}

// resolveArrayElementHint resolves the hint for one element of the array x
// evaluates to -- x is a FROM-clause UNNEST's argument (scope.go's
// addTableFunction; an expression-position UNNEST(...)/single-index
// access instead reaches the same projection through resolveItemHint's
// generic resolveColumnPathHints fallback above, now that UNNEST has its
// own returnType, functions.go's unnestReturnType). Kept as its own named
// entry point since scope.go has no FunctionExpr/SliceExpr of its own to
// hand resolveColumnPathHints -- just r.Args[0] directly.
func resolveArrayElementHint(x parser.Expr, scope *tableScope) ColumnHint {
	return rootHintOf(projectPathHints(resolveColumnPathHints(x, scope), []PathSegment{wc}))
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
		return col.hint()
	case 2:
		if tbl, ok := scope.lookup(id.Parts[0]); ok {
			col, ok := tbl.columns[id.Parts[1]]
			if !ok {
				return HintNone
			}
			return col.hint()
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
		return col.lookupPath([]PathSegment{wc, seg(id.Parts[2])})
	default:
		return HintNone
	}
}

// arrayOfStructsFieldHint resolves colName.field's hint when colName is a
// knownColumn.arrayOfStructs column in scope — see rewriteIdent
// (rewrite.go), which rewrites this exact shape into the JSONPath
// wildcard form the {wc, seg(field)} pathHints entry (tables.go) mirrors.
func arrayOfStructsFieldHint(colName, field string, scope *tableScope) ColumnHint {
	col, ok := scope.uniqueColumn(colName)
	if !ok || !col.arrayOfStructs {
		return HintNone
	}
	return col.lookupPath([]PathSegment{wc, seg(field)})
}

// resolveSubPathHint resolves a JSON sub-path access's hint: base's own
// declared/computed pathHints (resolveColumnPathHints -- a bare/qualified
// column's knownColumn.pathHints, or a LIST()/ARRAY_AGG() call's own
// result pathHints, functions.go's listReturnType) looked up at path,
// exactly matching whatever nesting the query itself named (queryPath).
func resolveSubPathHint(base parser.Expr, path string, scope *tableScope) ColumnHint {
	return lookupPathIn(resolveColumnPathHints(base, scope), queryPath(path))
}
