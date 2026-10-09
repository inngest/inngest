package insights

import (
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// TableCTEPrefix prefixes the reserved CTE name every reference to a logical
// table is rewritten to (runs -> __inngest_runs). A user CTE may not use it.
const TableCTEPrefix = "__inngest_"

// TableCTEName is the reserved CTE a logical table's references point at.
func TableCTEName(table string) string { return TableCTEPrefix + table }

// remapTables rewrites every logical BaseTableRef reachable from stmt — its
// own FROM clause, every CTE's body, every FROM-clause and expression-position
// subquery's body, and (for a UNION/INTERSECT/EXCEPT statement) both operands
// — into a reference to that table's reserved CTE, e.g. `runs` ->
// `__inngest_runs AS runs`. Mutates stmt in place and returns the logical
// tables referenced, deduped, in first-seen order. The CTEs themselves are
// added at render time (TranspileResult.Render), so their bodies can come
// from the plain table macros (MacroSource) or from a federated executor.
func remapTables(stmt *parser.SelectStatement, tables map[string]logicalTable) []string {
	r := &remapper{catalog: tables, seen: map[string]bool{}}
	r.remap(stmt, nil)
	return r.tables
}

type remapper struct {
	catalog map[string]logicalTable
	tables  []string
	seen    map[string]bool
}

// remap tracks which names are CTEs (not physical logical tables) at each
// point in the tree — a CTE can shadow a real table name, so a bare
// reference to one must be left untouched. Mirrors validateWithCTEs'
// accumulation order: a CTE's own body only sees earlier CTEs, never itself
// or later ones.
func (r *remapper) remap(stmt *parser.SelectStatement, outerCTEs map[string]bool) {
	ctes := outerCTEs
	if stmt.With != nil {
		merged := make(map[string]bool, len(outerCTEs)+len(stmt.With.CTEs))
		for name := range outerCTEs {
			merged[name] = true
		}
		for _, cte := range stmt.With.CTEs {
			r.remap(cte.Select, merged)
			merged[cte.Name] = true
		}
		ctes = merged
	}

	if stmt.SetOp != parser.SetOpNone {
		r.remap(stmt.SetLeft, ctes)
		r.remap(stmt.SetRight, ctes)
		// No return: a set-op node has no FROM of its own, but its own
		// ORDER BY/LIMIT/OFFSET can still hold subqueries, which the
		// subquery walk below must reach.
	}
	if stmt.From != nil {
		for i, ref := range stmt.From.Refs {
			stmt.From.Refs[i] = r.remapRef(ref, ctes)
		}
	}
	// A scalar/IN/EXISTS subquery can appear anywhere in an expression
	// tree, not just the FROM tree handled above. Remapping must mirror
	// validate.go's collectExprs walk exactly, or a subquery's own table
	// references reach DuckDB as bare, unscoped names instead of CTE
	// references — a real env/account scoping bypass.
	v := &subqueryRemapper{r: r, ctes: ctes}
	for _, n := range collectExprs(stmt) {
		parser.Walk(v, n)
	}
}

type subqueryRemapper struct {
	r    *remapper
	ctes map[string]bool
}

func (v *subqueryRemapper) Visit(n parser.Node) parser.Visitor {
	if sub, ok := n.(*parser.SelectStatement); ok {
		v.r.remap(sub, v.ctes)
		return nil
	}
	return v
}

// remapRef rewrites one FROM-tree node: a BaseTableRef becomes a reference
// to its logical table's CTE unless its name is a user CTE, a JoinRef
// recurses into both sides, and a TableSubqueryRef has its own body
// rewritten in place.
func (r *remapper) remapRef(ref parser.TableRef, ctes map[string]bool) parser.TableRef {
	switch x := ref.(type) {
	case *parser.BaseTableRef:
		return r.remapBaseTable(x, ctes)
	case *parser.JoinRef:
		x.Left = r.remapRef(x.Left, ctes)
		x.Right = r.remapRef(x.Right, ctes)
		return x
	case *parser.TableSubqueryRef:
		r.remap(x.Select, ctes)
		return x
	default:
		return ref
	}
}

// remapBaseTable points a logical table reference at its CTE, keeping the
// name the query uses for it (its alias, or the table name). A user CTE
// reference, or (defensively) anything else that isn't a logical table, is
// left as is.
func (r *remapper) remapBaseTable(ref *parser.BaseTableRef, ctes map[string]bool) parser.TableRef {
	if ctes[ref.Name[0]] {
		return ref
	}
	tbl, known := r.catalog[ref.Name[0]]
	if !known {
		return ref
	}
	if !r.seen[tbl.name] {
		r.seen[tbl.name] = true
		r.tables = append(r.tables, tbl.name)
	}
	alias := ref.Alias
	if alias == "" {
		alias = ref.Name[0]
	}
	return &parser.BaseTableRef{Name: []string{TableCTEName(tbl.name)}, Alias: alias}
}
