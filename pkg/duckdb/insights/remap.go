package insights

import (
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// remapTables rewrites every logical BaseTableRef reachable from stmt —
// its own FROM clause, every CTE's body, every FROM-clause subquery's
// body, and (for a UNION/INTERSECT/EXCEPT statement) both operands — into
// a call to its physical table macro, e.g. `runs` ->
// `inngest.insights_runs(?, ?) AS runs`. Mutates stmt in place; returns the
// (account_id, env_id) argument pairs to bind to each call's '?'
// placeholders, in the same left-to-right order the placeholders appear in
// the rewritten SQL text — CTE bodies first, in declaration order, then
// the FROM tree itself.
func remapTables(stmt *parser.SelectStatement, accountID, envID uuid.UUID) []any {
	return remapTablesWithCTEs(stmt, accountID, envID, nil)
}

// remapTablesWithCTEs is remapTables' real implementation, tracking which
// names are CTEs (not physical logical tables) at each point in the tree —
// a CTE can shadow a real table name, so a bare reference to one must be
// left untouched: its body is rewritten separately, where it's defined,
// not where it's referenced. Mirrors validateWithCTEs' accumulation order:
// a CTE's own body only sees earlier CTEs, never itself or later ones.
func remapTablesWithCTEs(stmt *parser.SelectStatement, accountID, envID uuid.UUID, outerCTEs map[string]bool) []any {
	ctes := outerCTEs
	var args []any
	if stmt.With != nil {
		merged := make(map[string]bool, len(outerCTEs)+len(stmt.With.CTEs))
		for name := range outerCTEs {
			merged[name] = true
		}
		for _, cte := range stmt.With.CTEs {
			args = append(args, remapTablesWithCTEs(cte.Select, accountID, envID, merged)...)
			merged[cte.Name] = true
		}
		ctes = merged
	}

	if stmt.SetOp != parser.SetOpNone {
		args = append(args, remapTablesWithCTEs(stmt.SetLeft, accountID, envID, ctes)...)
		args = append(args, remapTablesWithCTEs(stmt.SetRight, accountID, envID, ctes)...)
		return args
	}
	if stmt.From != nil {
		for i, ref := range stmt.From.Refs {
			var rewritten parser.TableRef
			rewritten, args = remapRef(ref, accountID, envID, args, ctes)
			stmt.From.Refs[i] = rewritten
		}
	}
	// A scalar/IN/EXISTS subquery can appear anywhere in an expression
	// tree, not just the FROM tree handled above. remapping must mirror
	// validate.go's collectExprs walk exactly, or a subquery's own table
	// references reach DuckDB as bare, unscoped names instead of macro
	// calls — a real env/account scoping bypass, caught only by
	// hand-inspecting a regenerated golden fixture.
	args = append(args, remapSubqueriesIn(stmt, accountID, envID, ctes)...)
	return args
}

// remapSubqueriesIn finds every nested *parser.SelectStatement reachable
// from stmt's own expression-bearing clauses (collectExprs — the same set
// validate.go walks) and rewrites each one's body in place via a fresh,
// independent remapTablesWithCTEs call, mirroring exprValidator.Visit's
// treatment of the same node shape. Doesn't recurse into a found
// subquery's Children() itself afterward — remapTablesWithCTEs already
// fully processes that subquery, including any subqueries nested inside
// it.
func remapSubqueriesIn(stmt *parser.SelectStatement, accountID, envID uuid.UUID, ctes map[string]bool) []any {
	v := &subqueryRemapper{accountID: accountID, envID: envID, ctes: ctes}
	for _, n := range collectExprs(stmt) {
		parser.Walk(v, n)
	}
	return v.args
}

type subqueryRemapper struct {
	accountID, envID uuid.UUID
	ctes             map[string]bool
	args             []any
}

func (v *subqueryRemapper) Visit(n parser.Node) parser.Visitor {
	if sub, ok := n.(*parser.SelectStatement); ok {
		v.args = append(v.args, remapTablesWithCTEs(sub, v.accountID, v.envID, v.ctes)...)
		return nil
	}
	return v
}

// remapRef rewrites one FROM-tree node: a BaseTableRef becomes a macro
// call (remapBaseTable) unless its name is a CTE, a JoinRef recurses into
// both sides, and a TableSubqueryRef has its own body rewritten in place
// (its wrapping parens/alias are untouched — only what's inside needs
// macro calls).
func remapRef(ref parser.TableRef, accountID, envID uuid.UUID, args []any, ctes map[string]bool) (parser.TableRef, []any) {
	switch r := ref.(type) {
	case *parser.BaseTableRef:
		sub, callArgs, ok := remapBaseTable(r, accountID, envID, ctes)
		if ok {
			args = append(args, callArgs...)
		}
		return sub, args
	case *parser.JoinRef:
		r.Left, args = remapRef(r.Left, accountID, envID, args, ctes)
		r.Right, args = remapRef(r.Right, accountID, envID, args, ctes)
		return r, args
	case *parser.TableSubqueryRef:
		args = append(args, remapTablesWithCTEs(r.Select, accountID, envID, ctes)...)
		return r, args
	default:
		return ref, args
	}
}

// remapBaseTable builds r's macro-call replacement. ok is false when r's
// name is a CTE (left as-is; its body is rewritten where it's defined —
// see remapTablesWithCTEs) or, defensively, anything else that isn't a
// physical logical table (validate should have already rejected this
// before Transpile ever calls remapTables).
func remapBaseTable(r *parser.BaseTableRef, accountID, envID uuid.UUID, ctes map[string]bool) (ref parser.TableRef, callArgs []any, ok bool) {
	if ctes[r.Name[0]] {
		return r, nil, false
	}
	tbl, known := logicalTables[r.Name[0]]
	if !known {
		return r, nil, false
	}
	alias := r.Alias
	if alias == "" {
		alias = r.Name[0]
	}
	call := &parser.TableFunctionRef{
		Name: []string{duckdb.DuckLakeAlias, tbl.view},
		Args: []parser.Expr{
			&parser.Parameter{Kind: parser.ParamAnonymous},
			&parser.Parameter{Kind: parser.ParamAnonymous},
		},
		Alias: alias,
	}
	return call, []any{accountID.String(), envID.String()}, true
}
