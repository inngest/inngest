package insights

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// validate rejects any query outside this package's explicit allowlist. On
// success it returns stmt's own resolved scope (nil for a top-level
// UNION/INTERSECT/EXCEPT statement, which has no FROM clause of its own —
// each operand is validated, and scoped, independently), so callers never
// need to call resolveScope a second time, plus every non-fatal Diagnostic
// (e.g. a function called with too few arguments) noted along the way. See
// validateWithCTEs for CTEs.
func validate(stmt *parser.SelectStatement) (*tableScope, map[string]logicalTable, []Diagnostic, error) {
	var diags []Diagnostic
	scope, ctes, err := validateWithCTEs(stmt, nil, nil, &diags)
	return scope, ctes, diags, err
}

// validateWithCTEs is validate's real implementation, parameterized by
// outerCTEs — the CTEs already visible at this lexical point (nil/empty at
// the true top level) — outerScope, the enclosing query's own resolved
// scope when stmt is an expression-position subquery (scalar/IN/EXISTS),
// so a correlated reference to an outer column resolves instead of
// failing as unknown (nil for the true top level and for every CTE body,
// which can never correlate) — and diags, the accumulator every nested
// call appends its own function-call diagnostics into.
//
// diags is a pointer, not a second return value threaded through every
// caller's own tuple, because this function's own call graph (CTEs,
// FROM-clause subqueries via resolveScope/addSubquery/deriveTable,
// expression-position subqueries) is deep and mutually recursive across
// several files -- passing nil from every call site that isn't part of
// stageValidate's own top-level walk (columnhints.go, rewrite.go,
// derive.go's own leafScope re-resolution) means those re-resolutions
// don't re-collect (and duplicate) diagnostics validate already gathered
// once for the same nodes.
//
// Returns the merged CTE set back out so a caller like deriveTable can
// pass it down into a nested CTE/subquery's own validation, and so
// stageValidate can hand it to stageBuildColumnHints without recomputing.
//
// CTEs are handled before the SetOp check below, not after: a statement
// can have both With != nil and SetOp != SetOpNone on the same node ("WITH
// x AS (...) SELECT ... FROM x UNION SELECT ... FROM y" attaches its WITH
// clause to the outermost/UNION node) — checking SetOp first would skip
// CTE processing for exactly that shape.
func validateWithCTEs(stmt *parser.SelectStatement, outerCTEs map[string]logicalTable, outerScope *tableScope, diags *[]Diagnostic) (*tableScope, map[string]logicalTable, error) {
	ctes := outerCTEs
	if stmt.With != nil {
		if stmt.With.Recursive {
			return nil, nil, &ValidationError{Pos: stmt.With.Pos(), End: stmt.With.End(), Message: "recursive CTEs are not supported"}
		}
		merged := make(map[string]logicalTable, len(outerCTEs)+len(stmt.With.CTEs))
		maps.Copy(merged, outerCTEs)

		for _, cte := range stmt.With.CTEs {
			// deriveTable validates cte.Select using merged as it stands so
			// far — cte.Name is deliberately not yet in merged, so a CTE
			// referencing its own name (self-reference without RECURSIVE)
			// fails as "unknown table" by construction. outerScope is
			// always nil here: a CTE body can never correlate.
			tbl, err := deriveTable(cte.Select, cte.Name, cte.ColumnNames, merged, nil, diags)
			if err != nil {
				return nil, nil, err
			}
			merged[cte.Name] = tbl
		}
		ctes = merged
	}

	if stmt.SetOp != parser.SetOpNone {
		if _, _, err := validateWithCTEs(stmt.SetLeft, ctes, outerScope, diags); err != nil {
			return nil, nil, err
		}
		if _, _, err := validateWithCTEs(stmt.SetRight, ctes, outerScope, diags); err != nil {
			return nil, nil, err
		}
		return nil, ctes, nil
	}
	if stmt.Values != nil {
		return nil, nil, &ValidationError{Pos: stmt.Pos(), End: stmt.End(), Message: "VALUES statements are not supported"}
	}

	scope, err := resolveScope(stmt.From, ctes, diags)
	if err != nil {
		return nil, nil, err
	}
	scope.outer = outerScope

	// GROUP BY/ORDER BY may reference a SELECT-list alias instead of a
	// real column — DuckDB resolves those names against the SELECT list
	// too, not just the FROM scope. Skip re-validating a bare alias-only
	// reference here rather than rejecting it as unknown; a real column of
	// the same name still takes priority. Scoped to validation only —
	// collectExprs itself stays untouched since queryinfo.go and remap.go
	// reuse it for traversals that must still see every expression.
	skipAliasExprs := groupByOrderByAliasExprs(stmt, scope)

	v := &exprValidator{scope: scope, ctes: ctes, diags: diags}
	for _, n := range collectExprs(stmt) {
		if skipAliasExprs[n] {
			continue
		}
		parser.Walk(v, n)
		if v.err != nil {
			return nil, nil, v.err
		}
	}
	return scope, ctes, nil
}

// groupByOrderByAliasExprs returns the set of GROUP BY/ORDER BY top-level
// expressions that are bare references to a SELECT-list alias with no
// matching real column in scope. See validateWithCTEs's call site.
func groupByOrderByAliasExprs(stmt *parser.SelectStatement, scope *tableScope) map[parser.Node]bool {
	aliases := selectAliases(stmt)
	if len(aliases) == 0 {
		return nil
	}

	skip := map[parser.Node]bool{}
	if stmt.GroupBy != nil {
		for _, item := range stmt.GroupBy.Items {
			for _, e := range groupByItemExprs(item) {
				if isAliasOnlyReference(e, aliases, scope) {
					skip[e] = true
				}
			}
		}
	}
	if stmt.OrderBy != nil {
		for _, item := range stmt.OrderBy.Items {
			if isAliasOnlyReference(item.X, aliases, scope) {
				skip[item.X] = true
			}
		}
	}
	return skip
}

func selectAliases(stmt *parser.SelectStatement) map[string]bool {
	aliases := make(map[string]bool, len(stmt.Columns))
	for _, item := range stmt.Columns {
		if item.Alias != "" {
			aliases[item.Alias] = true
		}
	}
	return aliases
}

// isAliasOnlyReference reports whether e is a bare identifier matching a
// SELECT-list alias with no same-named column anywhere in scope.
func isAliasOnlyReference(e parser.Expr, aliases map[string]bool, scope *tableScope) bool {
	id, ok := e.(*parser.Ident)
	if !ok || len(id.Parts) != 1 {
		return false
	}
	return aliases[id.Parts[0]] && scope.columnCount(id.Parts[0]) == 0
}

// collectExprs gathers every top-level expression stmt's clauses carry,
// every named WINDOW definition, plus every JOIN's ON/USING in its FROM
// tree — everything validate needs to expression-check with the FROM
// scope already resolved. Returns []parser.Node rather than []parser.Expr
// since a *WindowDef isn't itself an Expr, only a Node — parser.Walk
// accepts either. Doesn't descend into nested SelectStatements itself;
// parser.Walk plus exprValidator.Visit's *parser.SelectStatement case does
// that.
//
// stmt.Windows must be walked explicitly: an inline OVER(PARTITION BY ...)
// on a FunctionExpr is already reachable via that function's own
// Children(), but a named "WINDOW w AS (...)" clause referenced elsewhere
// as "OVER w" is a separate top-level field Walk never reaches on its
// own — without this, PARTITION BY/ORDER BY inside a named window
// definition could reference any column, known or not, with zero
// validation (a real bug, found by auditing SelectStatement's full field
// list).
func collectExprs(stmt *parser.SelectStatement) []parser.Node {
	var nodes []parser.Node
	for _, item := range stmt.Columns {
		nodes = append(nodes, item.Expr)
	}
	if stmt.Where != nil {
		nodes = append(nodes, stmt.Where)
	}
	if stmt.Having != nil {
		nodes = append(nodes, stmt.Having)
	}
	if stmt.Qualify != nil {
		nodes = append(nodes, stmt.Qualify)
	}
	if stmt.Distinct != nil {
		for _, e := range stmt.Distinct.On {
			nodes = append(nodes, e)
		}
	}
	if stmt.GroupBy != nil {
		for _, item := range stmt.GroupBy.Items {
			for _, e := range groupByItemExprs(item) {
				nodes = append(nodes, e)
			}
		}
	}
	if stmt.OrderBy != nil {
		for _, item := range stmt.OrderBy.Items {
			nodes = append(nodes, item.X)
		}
	}
	if stmt.Limit != nil {
		if stmt.Limit.Limit != nil {
			nodes = append(nodes, stmt.Limit.Limit)
		}
		if stmt.Limit.Offset != nil {
			nodes = append(nodes, stmt.Limit.Offset)
		}
	}
	for _, w := range stmt.Windows {
		nodes = append(nodes, w)
	}
	for _, e := range collectFromExprs(stmt.From) {
		nodes = append(nodes, e)
	}
	return nodes
}

// groupByItemExprs recurses into GroupingSets: each of its Sets is itself
// a GroupByItem, not a bare Expr — an earlier version returned nil for
// GroupingSets entirely, letting any column inside a "GROUP BY GROUPING
// SETS ((a), (b))" clause through with zero validation (same class of bug
// as the named-WINDOW gap above).
func groupByItemExprs(item parser.GroupByItem) []parser.Expr {
	switch g := item.(type) {
	case *parser.GroupByExprItem:
		return []parser.Expr{g.X}
	case *parser.GroupByCube:
		return g.Items
	case *parser.GroupByRollup:
		return g.Items
	case *parser.GroupingSets:
		var exprs []parser.Expr
		for _, set := range g.Sets {
			exprs = append(exprs, groupByItemExprs(set)...)
		}
		return exprs
	default:
		return nil
	}
}

// collectFromExprs walks ref's JOIN tree collecting every ON expression —
// USING column lists aren't Exprs (they're plain strings), so they're
// checked directly against scope in resolveScope's caller-agnostic
// addRef; here we only need ON.
func collectFromExprs(from *parser.FromClause) []parser.Expr {
	if from == nil {
		return nil
	}
	var exprs []parser.Expr
	for _, ref := range from.Refs {
		exprs = append(exprs, collectRefExprs(ref)...)
	}
	return exprs
}

func collectRefExprs(ref parser.TableRef) []parser.Expr {
	switch r := ref.(type) {
	case *parser.JoinRef:
		var exprs []parser.Expr
		if r.On != nil {
			exprs = append(exprs, r.On)
		}
		exprs = append(exprs, collectRefExprs(r.Left)...)
		exprs = append(exprs, collectRefExprs(r.Right)...)
		return exprs
	case *parser.TableFunctionRef:
		// UNNEST's own argument (addTableFunction, scope.go) needs the
		// same validation as any other expression -- without this it
		// would reach remapTables/DuckDB completely unchecked.
		return r.Args
	default:
		return nil
	}
}

// exprValidator is a parser.Visitor checking every Ident/FunctionExpr/
// LambdaExpr/StarExpr it finds against scope and allowedFunctions,
// recursively (and independently — see deriveTable) validating any nested
// SelectStatement (a scalar/IN/EXISTS subquery) it encounters
// mid-expression, and rejecting any unsupported FROM-clause node type —
// kept as a defensive default even though nothing valid reaches that path
// today.
type exprValidator struct {
	scope *tableScope
	ctes  map[string]logicalTable
	err   error
	// diags is the same accumulator pointer validateWithCTEs was given --
	// checkFunction appends any Diagnostic a function's own returnType
	// rule produces about its call. May be nil (see validateWithCTEs'
	// doc comment on when callers pass nil deliberately), in which case
	// diagnostic collection is just skipped.
	diags *[]Diagnostic
	// boundVars is a stack of list-comprehension loop variables currently
	// in scope, innermost last. Shadows any real column of the same name,
	// matching DuckDB's own comprehension scoping.
	boundVars []string
}

func (v *exprValidator) isBound(name string) bool {
	return slices.Contains(v.boundVars, name)
}

func (v *exprValidator) Visit(n parser.Node) parser.Visitor {
	if v.err != nil {
		return nil
	}
	switch x := n.(type) {
	case *parser.Ident:
		v.err = v.checkIdent(x)
	case *parser.FunctionExpr:
		v.err = v.checkFunction(x)
	case *parser.LambdaExpr:
		v.err = v.checkLambda(x)
	case *parser.StarExpr:
		v.err = v.checkStar(x)
	case *parser.ListComprehensionExpr:
		v.err = v.checkListComprehension(x)
		return nil
	case *parser.SelectStatement:
		// A subquery (scalar, IN, EXISTS) — validated with
		// its own FROM scope plus any CTEs visible here, chained to this
		// statement's own scope (v.scope) as its outer scope: a
		// correlated reference to one of our own columns resolves rather
		// than failing, matching standard SQL — no LATERAL keyword
		// needed for this position, unlike a FROM-clause subquery (see
		// tableScope.outer's doc comment). Nothing references its output
		// columns by name in this position, so there's nothing to
		// infer/expose here, unlike a FROM-clause subquery
		// (tableScope.addSubquery, scope.go). Returning nil (not v) stops
		// parser.Walk from also descending into x's own Children() using
		// our own scope, which would be wrong twice over.
		if _, _, err := validateWithCTEs(x, v.ctes, v.scope, v.diags); err != nil {
			v.err = err
		}
		return nil
	case *parser.TableFunctionRef, *parser.ParensTableRef,
		*parser.PivotRef, *parser.UnpivotRef:
		v.err = &ValidationError{Pos: n.Pos(), End: n.End(), Message: "unsupported FROM clause shape"}
	}
	if v.err != nil {
		return nil
	}
	return v
}

func (v *exprValidator) checkIdent(id *parser.Ident) error {
	switch len(id.Parts) {
	case 1:
		// A list-comprehension loop variable shadows any real column of
		// the same name -- see checkListComprehension.
		if v.isBound(id.Parts[0]) {
			return nil
		}
		// See tableScope.columnCount's doc comment: a bare identifier
		// present on more than one table in scope (e.g. run_id on both
		// runs and extended_trace_spans in a JOIN) needs a table
		// qualifier, same as DuckDB itself would require — reported as
		// "ambiguous", not folded into "unknown".
		switch v.scope.columnCount(id.Parts[0]) {
		case 0:
			return &ValidationError{Pos: id.Pos(), End: id.End(), Message: fmt.Sprintf("unknown column %q", id.Parts[0])}
		case 1:
			// known, unambiguous
		default:
			return &ValidationError{Pos: id.Pos(), End: id.End(), Message: fmt.Sprintf("ambiguous column %q: qualify with a table name", id.Parts[0])}
		}
	default: // 2 or more parts
		return v.checkQualifiedIdent(id)
	}
	return nil
}

// checkQualifiedIdent handles a 2+-part identifier: id.Parts[0] is either a
// table (real column lookup on Parts[1]), or a bare JSON-typed column —
// DuckDB's dot operator also works as a JSON path extraction (e.g.
// data.function_id, or an arbitrarily deep path), so a JSON-typed column
// qualifies here too rather than requiring the ->> operator. Anything past
// the resolved table-column or JSON column is an unvalidated JSON path —
// this package has no static knowledge of what keys exist inside JSON
// data, matching its precedent for the ->>/-> operators.
func (v *exprValidator) checkQualifiedIdent(id *parser.Ident) error {
	// x.field off a loop variable bound over a list of structs -- like a
	// JSON column's own dotted path, this package has no static knowledge
	// of what fields exist, so anything past the bound name is left
	// unvalidated.
	if v.isBound(id.Parts[0]) {
		return nil
	}
	if tbl, ok := v.scope.lookup(id.Parts[0]); ok {
		col, ok := tbl.columns[id.Parts[1]]
		if !ok {
			return &ValidationError{Pos: id.Pos(), End: id.End(), Message: fmt.Sprintf("unknown column %q on table %q", id.Parts[1], id.Parts[0])}
		}
		if len(id.Parts) > 2 && col.colType != ColumnTypeJSON {
			return &ValidationError{Pos: id.Pos(), End: id.End(), Message: fmt.Sprintf("unsupported identifier %q", strings.Join(id.Parts, "."))}
		}
		return nil
	}

	// Not a table -- an unknown or ambiguous name falls through to the
	// error below, matching case 1's ambiguity handling above.
	if col, ok := v.scope.uniqueColumn(id.Parts[0]); ok && col.colType == ColumnTypeJSON {
		return nil
	}
	if len(id.Parts) == 2 {
		return &ValidationError{Pos: id.Pos(), End: id.End(), Message: fmt.Sprintf("unknown table %q", id.Parts[0])}
	}
	return &ValidationError{Pos: id.Pos(), End: id.End(), Message: fmt.Sprintf("unsupported identifier %q", strings.Join(id.Parts, "."))}
}

// checkFunction rejects any function name outside allowedFunctions, then
// always runs that function's own returnType rule to check for a
// DiagnosticError-severity Diagnostic -- one DuckDB itself would
// unconditionally reject too (a confirmed wrong argument count or type;
// see tooFewArgsDiagnostic/tooManyArgsDiagnostic/wrongArgTypeDiagnostic,
// functions.go), promoted here into a real *ValidationError that rejects
// the query outright, rather than letting it reach DuckDB only to fail
// there instead. This check runs regardless of whether v.diags is nil: a
// genuinely invalid call must reject the query no matter which
// re-derivation pass notices it first, unlike the softer diagnostics
// below.
//
// Any lower-severity Diagnostic (a note, not a rejection) only rides into
// the pipeline's Diagnostics when v.diags is non-nil, position-anchored
// to this call site since returnType itself only ever sees the function's
// resolved name and arguments, never the node itself.
func (v *exprValidator) checkFunction(f *parser.FunctionExpr) error {
	name := strings.ToLower(strings.Join(f.Name, "."))
	info, ok := allowedFunctions[name]
	if !ok {
		return &ValidationError{Pos: f.Pos(), End: f.End(), Message: fmt.Sprintf("function %q is not allowed", strings.Join(f.Name, "."))}
	}
	_, _, diags := info.returnType(name, f.Args, v.scope)
	for _, d := range diags {
		d.Start, d.End = f.Pos(), f.End()
		if d.Severity == DiagnosticError {
			return &ValidationError{Pos: d.Start, End: d.End, Message: d.Message}
		}
		if v.diags != nil {
			*v.diags = append(*v.diags, d)
		}
	}
	return nil
}

// checkStar validates a StarExpr's Qualifier (if given) resolves to a
// known table in scope, and every name in Exclude is a real column of the
// table(s) that star would expand to — StarExpr.Children() returns nil
// (Qualifier/Exclude are plain string slices, not Expr/Ident nodes), so
// without this explicit case neither field is ever checked: "SELECT
// bogus_alias.*" or "SELECT * EXCLUDE (nonexistent_column)" would
// otherwise reach remapTables/DuckDB completely unvalidated.
func (v *exprValidator) checkStar(s *parser.StarExpr) error {
	var tables []logicalTable
	switch len(s.Qualifier) {
	case 0:
		for _, name := range v.scope.names() {
			if tbl, ok := v.scope.lookup(name); ok {
				tables = append(tables, tbl)
			}
		}
	case 1:
		tbl, ok := v.scope.lookup(s.Qualifier[0])
		if !ok {
			return &ValidationError{Pos: s.Pos(), End: s.End(), Message: fmt.Sprintf("unknown table %q", s.Qualifier[0])}
		}
		tables = []logicalTable{tbl}
	default:
		return &ValidationError{Pos: s.Pos(), End: s.End(), Message: fmt.Sprintf("unsupported qualifier %q", strings.Join(s.Qualifier, "."))}
	}

	for _, exclude := range s.Exclude {
		found := false
		for _, tbl := range tables {
			if _, ok := tbl.columns[exclude]; ok {
				found = true
				break
			}
		}
		if !found {
			return &ValidationError{Pos: s.Pos(), End: s.End(), Message: fmt.Sprintf("unknown column %q in EXCLUDE", exclude)}
		}
	}
	return nil
}

// checkLambda treats every LambdaExpr as JSON-arrow-shaped column access
// (col -> 'key'), never a genuine lambda: allowedFunctions never includes
// a lambda-taking function, so a genuine lambda is unreachable through any
// other allowed path anyway.
func (v *exprValidator) checkLambda(l *parser.LambdaExpr) error {
	if len(l.Params) != 1 {
		return &ValidationError{Pos: l.Pos(), End: l.End(), Message: "lambda expressions are not supported"}
	}
	switch v.scope.columnCount(l.Params[0]) {
	case 0:
		return &ValidationError{Pos: l.Pos(), End: l.End(), Message: fmt.Sprintf("unknown column %q", l.Params[0])}
	case 1:
		return nil
	default:
		return &ValidationError{Pos: l.Pos(), End: l.End(), Message: fmt.Sprintf("ambiguous column %q: qualify with a table name", l.Params[0])}
	}
}

// checkListComprehension validates a "[Expr FOR var[, var...] IN Source (IF
// Filter)?]" expression: Source is checked against the enclosing scope,
// then l.Vars are pushed onto v.boundVars for the duration of Expr/Filter
// (which may reference them as if they were columns, shadowing any real
// column of the same name) and popped again once both are checked, so the
// binding never leaks past this comprehension. Returns nil, and the Visit
// call site also returns nil for this node, since generic parser.Walk
// recursion into l's Children() would otherwise re-visit Expr/Source/
// Filter a second time with the unmodified scope.
func (v *exprValidator) checkListComprehension(l *parser.ListComprehensionExpr) error {
	parser.Walk(v, l.Source)
	if v.err != nil {
		return v.err
	}

	v.boundVars = append(v.boundVars, l.Vars...)
	defer func() { v.boundVars = v.boundVars[:len(v.boundVars)-len(l.Vars)] }()

	parser.Walk(v, l.Expr)
	if v.err != nil {
		return v.err
	}
	if l.Filter != nil {
		parser.Walk(v, l.Filter)
	}
	return v.err
}
