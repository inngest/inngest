package insights

import "github.com/inngest/inngest/pkg/duckdb/parser"

// QueryInfo captures the *physical* logical tables stmt references, for the
// response envelope's diagnostics. A CTE or FROM-clause subquery's own
// name/alias never appears here — only the real logical tables its body
// references (recursed into, deduped alongside everything else). Must run
// before remapTables, which rewrites away the logical table names this
// reads off stmt.
type QueryInfo struct {
	PrimaryTable string
	Tables       []string
}

func extractQueryInfo(stmt *parser.SelectStatement) QueryInfo {
	info := &QueryInfo{}
	collectQueryInfo(stmt, info, map[string]bool{}, map[string]bool{})
	return *info
}

// collectQueryInfo walks stmt — recursing into CTE bodies, UNION/
// INTERSECT/EXCEPT operands, and FROM-clause subqueries — collecting the
// real logical tables it touches, deduped, in first-seen order. cteNames
// tracks CTE names declared at or above this point purely to exclude them
// from the physical-table list; unlike in validate/remapTables it plays no
// scoping role, since this function never rejects anything.
func collectQueryInfo(stmt *parser.SelectStatement, info *QueryInfo, seen, cteNames map[string]bool) {
	if stmt == nil {
		return
	}
	if stmt.With != nil {
		for _, cte := range stmt.With.CTEs {
			cteNames[cte.Name] = true
			collectQueryInfo(cte.Select, info, seen, cteNames)
		}
	}
	if stmt.SetOp != parser.SetOpNone {
		collectQueryInfo(stmt.SetLeft, info, seen, cteNames)
		collectQueryInfo(stmt.SetRight, info, seen, cteNames)
		return
	}
	if stmt.From != nil {
		for _, ref := range stmt.From.Refs {
			collectFromRef(ref, info, seen, cteNames)
		}
	}
	// A scalar/IN/EXISTS subquery can appear anywhere in an expression
	// tree, not just the FROM tree handled above — mirrors remap.go's
	// remapSubqueriesIn, which walks the same collectExprs set for the
	// same reason: otherwise a table referenced only inside one would
	// never appear in info.Tables.
	v := &queryInfoSubqueryCollector{info: info, seen: seen, cteNames: cteNames}
	for _, n := range collectExprs(stmt) {
		parser.Walk(v, n)
	}
}

type queryInfoSubqueryCollector struct {
	info     *QueryInfo
	seen     map[string]bool
	cteNames map[string]bool
}

func (v *queryInfoSubqueryCollector) Visit(n parser.Node) parser.Visitor {
	if sub, ok := n.(*parser.SelectStatement); ok {
		collectQueryInfo(sub, v.info, v.seen, v.cteNames)
		return nil
	}
	return v
}

func collectFromRef(ref parser.TableRef, info *QueryInfo, seen, cteNames map[string]bool) {
	switch r := ref.(type) {
	case *parser.BaseTableRef:
		if len(r.Name) == 0 {
			return
		}
		name := r.Name[len(r.Name)-1]
		if cteNames[name] || seen[name] {
			return
		}
		seen[name] = true
		info.Tables = append(info.Tables, name)
		if info.PrimaryTable == "" {
			info.PrimaryTable = name
		}
	case *parser.JoinRef:
		collectFromRef(r.Left, info, seen, cteNames)
		collectFromRef(r.Right, info, seen, cteNames)
	case *parser.TableSubqueryRef:
		collectQueryInfo(r.Select, info, seen, cteNames)
	}
}
