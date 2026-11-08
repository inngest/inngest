package insights

import (
	"fmt"
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// rewriteArrayOfStructAccess rewrites every col.field reference where col
// resolves to a knownColumn.arrayOfStructs column (e.g. runs.sessions)
// into the equivalent JSONPath wildcard extraction col ->> '$[*].field'
// (see knownColumn.arrayOfStructs). Mutates stmt in place.
//
// Mirrors remapTablesWithCTEs' CTE/SetOp/FROM-subquery recursion shape so
// a match anywhere in the tree gets rewritten against its own lexically
// correct scope — ctes and outerScope carry the same accumulation/
// correlation semantics validateWithCTEs established them with.
func rewriteArrayOfStructAccess(stmt *parser.SelectStatement, ctes map[string]logicalTable, outerScope *tableScope) ([]Diagnostic, error) {
	var diagnostics []Diagnostic

	mergedCTEs := ctes
	if stmt.With != nil {
		merged := make(map[string]logicalTable, len(ctes)+len(stmt.With.CTEs))
		for k, v := range ctes {
			merged[k] = v
		}
		for _, cte := range stmt.With.CTEs {
			diags, err := rewriteArrayOfStructAccess(cte.Select, merged, nil)
			diagnostics = append(diagnostics, diags...)
			if err != nil {
				return diagnostics, err
			}
			tbl, err := deriveTable(cte.Select, cte.Name, cte.ColumnNames, merged, nil, nil)
			if err != nil {
				return diagnostics, err
			}
			merged[strings.ToLower(cte.Name)] = tbl
		}
		mergedCTEs = merged
	}

	if stmt.SetOp != parser.SetOpNone {
		diags, err := rewriteArrayOfStructAccess(stmt.SetLeft, mergedCTEs, outerScope)
		diagnostics = append(diagnostics, diags...)
		if err != nil {
			return diagnostics, err
		}
		diags, err = rewriteArrayOfStructAccess(stmt.SetRight, mergedCTEs, outerScope)
		diagnostics = append(diagnostics, diags...)
		if err != nil {
			return diagnostics, err
		}
		// The set-op node's own ORDER BY/LIMIT/OFFSET resolve against its
		// leftmost operand's scope -- see validateSetOpModifiers.
		leaf := leftmostOperand(stmt)
		scope, err := resolveScope(leaf.From, mergedCTEs, nil)
		if err != nil {
			return diagnostics, err
		}
		scope.outer = outerScope
		r := &arrayOfStructRewriter{scope: scope, ctes: mergedCTEs}
		r.rewriteModifiers(stmt)
		diagnostics = append(diagnostics, r.diagnostics...)
		return diagnostics, r.err
	}

	scope, err := resolveScope(stmt.From, mergedCTEs, nil)
	if err != nil {
		return diagnostics, err
	}
	scope.outer = outerScope

	if stmt.From != nil {
		for i, ref := range stmt.From.Refs {
			diags, err := rewriteArrayOfStructAccessInRef(ref, mergedCTEs, stmt.From.Refs[:i], outerScope)
			diagnostics = append(diagnostics, diags...)
			if err != nil {
				return diagnostics, err
			}
		}
	}

	r := &arrayOfStructRewriter{scope: scope, ctes: mergedCTEs}
	if stmt.From != nil {
		// A FROM-clause UNNEST's argument is validated against the whole
		// FROM scope (collectRefExprs, validate.go), so it's rewritten
		// against it too.
		for _, ref := range stmt.From.Refs {
			r.rewriteTableFunctionArgs(ref)
		}
	}
	for _, item := range stmt.Columns {
		orig := item.Expr
		item.Expr = r.rewrite(item.Expr)
		// An unaliased bare col.field item is named by its last part (see
		// outputColumnName); its computed replacement would lose that name,
		// both as a result column and when deriveTable re-derives a CTE or
		// subquery from this rewritten body, so pin it as an explicit alias.
		if id, ok := orig.(*parser.Ident); ok && item.Alias == "" && item.Expr != orig {
			item.Alias = id.Parts[len(id.Parts)-1]
		}
	}
	if stmt.Where != nil {
		stmt.Where = r.rewrite(stmt.Where)
	}
	if stmt.Having != nil {
		stmt.Having = r.rewrite(stmt.Having)
	}
	if stmt.Qualify != nil {
		stmt.Qualify = r.rewrite(stmt.Qualify)
	}
	if stmt.Distinct != nil {
		for i, e := range stmt.Distinct.On {
			stmt.Distinct.On[i] = r.rewrite(e)
		}
	}
	if stmt.GroupBy != nil {
		for _, item := range stmt.GroupBy.Items {
			r.rewriteGroupByItem(item)
		}
	}
	r.rewriteModifiers(stmt)
	for _, w := range stmt.Windows {
		r.rewriteWindowSpec(w.Spec)
	}
	diagnostics = append(diagnostics, r.diagnostics...)
	if r.err != nil {
		return diagnostics, r.err
	}
	return diagnostics, nil
}

// rewriteArrayOfStructAccessInRef mirrors remapRef's FROM-tree recursion
// shape, rewriting each nested SelectStatement it finds (a FROM-clause
// subquery or either side of a JOIN) with its own scope. earlier is every
// FROM item strictly before ref (a JOIN's left side included, for its
// right side), which a LATERAL subquery correlates into, matching
// addSubquery (scope.go); outer is the enclosing statement's own outer
// scope.
func rewriteArrayOfStructAccessInRef(ref parser.TableRef, ctes map[string]logicalTable, earlier []parser.TableRef, outer *tableScope) ([]Diagnostic, error) {
	switch r := ref.(type) {
	case *parser.JoinRef:
		var diagnostics []Diagnostic
		diags, err := rewriteArrayOfStructAccessInRef(r.Left, ctes, earlier, outer)
		diagnostics = append(diagnostics, diags...)
		if err != nil {
			return diagnostics, err
		}
		withLeft := append(append([]parser.TableRef(nil), earlier...), r.Left)
		diags, err = rewriteArrayOfStructAccessInRef(r.Right, ctes, withLeft, outer)
		diagnostics = append(diagnostics, diags...)
		if err != nil {
			return diagnostics, err
		}
		if r.On != nil {
			// r.On is evaluated against both sides of the join together —
			// resolveScope on r itself gives exactly that combined scope
			// (onScope re-exposes a SEMI/ANTI join's own right side).
			scope, err := resolveScope(&parser.FromClause{Refs: []parser.TableRef{r}}, ctes, nil)
			if err != nil {
				return diagnostics, err
			}
			scope = scope.onScope(r)
			scope.outer = outer
			rw := &arrayOfStructRewriter{scope: scope, ctes: ctes}
			r.On = rw.rewrite(r.On)
			diagnostics = append(diagnostics, rw.diagnostics...)
			if rw.err != nil {
				return diagnostics, rw.err
			}
		}
		return diagnostics, nil
	case *parser.TableSubqueryRef:
		if !r.Lateral {
			return rewriteArrayOfStructAccess(r.Select, ctes, nil)
		}
		scope, err := resolveScope(&parser.FromClause{Refs: earlier}, ctes, nil)
		if err != nil {
			return nil, err
		}
		scope.outer = outer
		return rewriteArrayOfStructAccess(r.Select, ctes, scope)
	default:
		return nil, nil
	}
}

func (r *arrayOfStructRewriter) rewriteTableFunctionArgs(ref parser.TableRef) {
	switch t := ref.(type) {
	case *parser.JoinRef:
		r.rewriteTableFunctionArgs(t.Left)
		r.rewriteTableFunctionArgs(t.Right)
	case *parser.TableFunctionRef:
		for i, a := range t.Args {
			t.Args[i] = r.rewrite(a)
		}
	}
}

// arrayOfStructRewriter recursively rewrites one expression tree against a
// fixed scope, replacing every col.field access into an arrayOfStructs
// column and recursing into any nested SelectStatement via a fresh,
// independently-scoped rewriteArrayOfStructAccess call correlated to this
// scope — mirroring exprValidator's and subqueryRemapper's treatment of
// the same node shape.
type arrayOfStructRewriter struct {
	scope *tableScope
	// ctes is the CTE set visible at scope, handed to every nested
	// subquery so it can still reference an enclosing WITH's names.
	ctes        map[string]logicalTable
	diagnostics []Diagnostic
	err         error
	// boundVars is the stack of list-comprehension variables in scope,
	// innermost last -- each shadows a same-named column, exactly like
	// exprValidator.boundVars, so "s.key" off a bound s is never
	// rewritten.
	boundVars []string
}

func (r *arrayOfStructRewriter) isBound(name string) bool {
	return slices.ContainsFunc(r.boundVars, func(b string) bool { return strings.EqualFold(b, name) })
}

func (r *arrayOfStructRewriter) rewrite(expr parser.Expr) parser.Expr {
	if expr == nil || r.err != nil {
		return expr
	}

	if id, ok := expr.(*parser.Ident); ok {
		if replacement, ok := r.rewriteIdent(id); ok {
			return replacement
		}
		return expr
	}

	switch e := expr.(type) {
	case *parser.BinaryExpr:
		e.Left = r.rewrite(e.Left)
		// A COLLATE's right operand is a collation name (validate only
		// admits an Ident there), never a column reference to rewrite.
		if !strings.EqualFold(e.Op, "COLLATE") {
			e.Right = r.rewrite(e.Right)
		}
	case *parser.QuantifiedExpr:
		// A *SubqueryExpr Right recurses via the SubqueryExpr case.
		e.Left, e.Right = r.rewrite(e.Left), r.rewrite(e.Right)
	case *parser.UnaryExpr:
		e.X = r.rewrite(e.X)
	case *parser.NullTest:
		e.X = r.rewrite(e.X)
	case *parser.IsExpr:
		e.X = r.rewrite(e.X)
	case *parser.DistinctFromExpr:
		e.Left, e.Right = r.rewrite(e.Left), r.rewrite(e.Right)
	case *parser.BetweenExpr:
		e.X, e.Low, e.High = r.rewrite(e.X), r.rewrite(e.Low), r.rewrite(e.High)
	case *parser.InExpr:
		e.X = r.rewrite(e.X)
		for i, item := range e.List {
			e.List[i] = r.rewrite(item)
		}
		if e.Subquery != nil {
			diags, err := rewriteArrayOfStructAccess(e.Subquery, r.ctes, r.scope)
			r.diagnostics = append(r.diagnostics, diags...)
			if err != nil {
				r.err = err
			}
		}
	case *parser.LikeExpr:
		e.X, e.Pattern, e.Escape = r.rewrite(e.X), r.rewrite(e.Pattern), r.rewrite(e.Escape)
	case *parser.CastExpr:
		e.X = r.rewrite(e.X)
	case *parser.CaseExpr:
		e.Operand = r.rewrite(e.Operand)
		for _, w := range e.Whens {
			w.Cond, w.Result = r.rewrite(w.Cond), r.rewrite(w.Result)
		}
		e.Else = r.rewrite(e.Else)
	case *parser.SubqueryExpr:
		diags, err := rewriteArrayOfStructAccess(e.Select, r.ctes, r.scope)
		r.diagnostics = append(r.diagnostics, diags...)
		if err != nil {
			r.err = err
		}
	case *parser.ListExpr:
		for i, el := range e.Elems {
			e.Elems[i] = r.rewrite(el)
		}
	case *parser.StructExpr:
		for _, f := range e.Fields {
			f.Value = r.rewrite(f.Value)
		}
	case *parser.MapExpr:
		for i := range e.Keys {
			e.Keys[i], e.Values[i] = r.rewrite(e.Keys[i]), r.rewrite(e.Values[i])
		}
	case *parser.IntervalExpr:
		e.Value = r.rewrite(e.Value)
	case *parser.FunctionExpr:
		e.Receiver = r.rewrite(e.Receiver)
		for i, a := range e.Args {
			e.Args[i] = r.rewrite(a)
		}
		r.rewriteOrderBy(e.OrderBy)
		r.rewriteOrderBy(e.Within)
		e.Filter = r.rewrite(e.Filter)
		if e.Over != nil {
			r.rewriteWindowSpec(e.Over)
		}
	case *parser.NamedArg:
		e.Value = r.rewrite(e.Value)
	case *parser.GroupingExpr:
		for i, a := range e.Args {
			e.Args[i] = r.rewrite(a)
		}
	case *parser.ListComprehensionExpr:
		// Source is outside the comprehension's own bindings; Expr/Filter
		// see its variables (see exprValidator.checkListComprehension).
		e.Source = r.rewrite(e.Source)
		r.boundVars = append(r.boundVars, e.Vars...)
		e.Expr, e.Filter = r.rewrite(e.Expr), r.rewrite(e.Filter)
		r.boundVars = r.boundVars[:len(r.boundVars)-len(e.Vars)]
	case *parser.DotExpr:
		e.X = r.rewrite(e.X)
	case *parser.SliceExpr:
		e.X, e.Start, e.Stop, e.Step = r.rewrite(e.X), r.rewrite(e.Start), r.rewrite(e.Stop), r.rewrite(e.Step)
	case *parser.LambdaExpr:
		e.Body = r.rewrite(e.Body)
	}
	return expr
}

// rewriteModifiers rewrites stmt's ORDER BY/LIMIT/OFFSET -- the clauses a
// set-op node carries too, alongside its operands.
func (r *arrayOfStructRewriter) rewriteModifiers(stmt *parser.SelectStatement) {
	if stmt.OrderBy != nil {
		for _, item := range stmt.OrderBy.Items {
			item.X = r.rewrite(item.X)
		}
	}
	if stmt.Limit != nil {
		stmt.Limit.Limit = r.rewrite(stmt.Limit.Limit)
		stmt.Limit.Offset = r.rewrite(stmt.Limit.Offset)
	}
}

func (r *arrayOfStructRewriter) rewriteWindowSpec(spec *parser.WindowSpec) {
	if spec == nil {
		return
	}
	for i, p := range spec.PartitionBy {
		spec.PartitionBy[i] = r.rewrite(p)
	}
	r.rewriteOrderBy(spec.OrderBy)
	if f := spec.Frame; f != nil {
		for _, b := range []*parser.FrameBound{f.Start, f.EndBound} {
			if b != nil {
				b.Expr = r.rewrite(b.Expr)
			}
		}
	}
}

func (r *arrayOfStructRewriter) rewriteOrderBy(o *parser.OrderByClause) {
	if o == nil {
		return
	}
	for _, item := range o.Items {
		item.X = r.rewrite(item.X)
	}
}

func (r *arrayOfStructRewriter) rewriteGroupByItem(item parser.GroupByItem) {
	switch g := item.(type) {
	case *parser.GroupByExprItem:
		g.X = r.rewrite(g.X)
	case *parser.GroupByCube:
		for i, e := range g.Items {
			g.Items[i] = r.rewrite(e)
		}
	case *parser.GroupByRollup:
		for i, e := range g.Items {
			g.Items[i] = r.rewrite(e)
		}
	case *parser.GroupingSets:
		for _, set := range g.Sets {
			r.rewriteGroupByItem(set)
		}
	}
}

// rewriteIdent reports whether id is a col.field (or table.col.field)
// reference into a knownColumn.arrayOfStructs column, returning its
// col ->> '$[*].field' replacement when so — see this file's top doc
// comment.
func (r *arrayOfStructRewriter) rewriteIdent(id *parser.Ident) (parser.Expr, bool) {
	if r.isBound(id.Parts[0]) {
		return nil, false
	}
	switch len(id.Parts) {
	case 2:
		// Only a bare column (not table.column) can be arrayOfStructs
		// access here — a table qualifier takes the 3-part branch below,
		// matching checkQualifiedIdent's (validate.go) own precedence.
		if _, isTable := r.scope.lookup(id.Parts[0]); isTable {
			return nil, false
		}
		col, ok := r.scope.uniqueColumn(id.Parts[0])
		if !ok || !col.arrayOfStructs {
			return nil, false
		}
		return r.recordArrayOfStructAccess(id, id.Parts[:1], id.Parts[1]), true
	case 3:
		tbl, ok := r.scope.lookup(id.Parts[0])
		if !ok {
			return nil, false
		}
		col, ok := tbl.column(id.Parts[1])
		if !ok || !col.arrayOfStructs {
			return nil, false
		}
		return r.recordArrayOfStructAccess(id, id.Parts[:2], id.Parts[2]), true
	default:
		return nil, false
	}
}

// recordArrayOfStructAccess builds id's col ->> '$[*].field' replacement
// and records an info diagnostic at id's original position explaining the
// rewrite — see knownColumn.arrayOfStructs' doc comment (tables.go).
func (r *arrayOfStructRewriter) recordArrayOfStructAccess(id *parser.Ident, colParts []string, field string) parser.Expr {
	replacement := arrayOfStructAccess(colParts, field)
	r.diagnostics = append(r.diagnostics, diagnosticAt(id, DiagnosticInfo, "array-of-struct-access-rewritten",
		fmt.Sprintf("%s is an array of structs, not JSON -- rewrote %s to %s.",
			strings.Join(colParts, "."), strings.Join(id.Parts, "."), parser.String(replacement))))
	return replacement
}

func arrayOfStructAccess(colParts []string, field string) parser.Expr {
	return &parser.BinaryExpr{
		Op:   "->>",
		Left: &parser.Ident{Parts: append([]string(nil), colParts...)},
		Right: &parser.Literal{
			Kind: parser.LitString,
			Text: "$[*]." + field,
		},
	}
}
