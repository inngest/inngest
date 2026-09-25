// pkg/duckdb/parser/adapter_clauses.go
package parser

import "github.com/inngest/inngest/pkg/duckdb/parser/peg"

// adaptTargetList reads a bare TargetList node (List(AliasedExpression)) —
// shared by SELECT's own target list and (from Task 13) PIVOT/UNPIVOT's
// USING/ON clauses.
func (a *adapter) adaptTargetList(n *peg.Node) []*SelectItem {
	listSeq := body(body(n))
	items := []*SelectItem{a.adaptAliasedExpression(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		items = append(items, a.adaptAliasedExpression(body(tail).Children[1]))
	}
	return items
}

func (a *adapter) adaptDistinctClause(n *peg.Node) *DistinctClause {
	// DistinctClause <- DistinctOn / DistinctAll
	alt := choice(body(n))
	if alt.Name == "DistinctAll" {
		return nil // 'SELECT ALL' is the explicit-default (no dedup) marker — equivalent to no DISTINCT clause at all.
	}
	// DistinctOn <- 'DISTINCT' DistinctOnTargets?
	seq := body(alt)
	dc := &DistinctClause{baseExpr: a.at(alt)}
	if targets, ok := present(seq.Children[1]); ok {
		// DistinctOnTargets <- 'ON' Parens(List(Expression))
		dc.On = a.adaptExprList(body(targets).Children[1])
	}
	return dc
}

func (a *adapter) adaptGroupByClause(n *peg.Node) *GroupByClause {
	// GroupByClause <- 'GROUP' 'BY' GroupByExpressions ; GroupByExpressions <- GroupByList / GroupByAll
	seq := body(n)
	if choice(body(seq.Children[2])).Name == "GroupByAll" {
		return &GroupByClause{baseExpr: a.at(n), All: true}
	}
	// GroupByList <- List(GroupByExpression)
	listSeq := body(body(choice(body(seq.Children[2]))))
	gc := &GroupByClause{baseExpr: a.at(n), Items: []GroupByItem{a.adaptGroupByExpression(listSeq.Children[0])}}
	for _, tail := range listSeq.Children[1].Children {
		gc.Items = append(gc.Items, a.adaptGroupByExpression(body(tail).Children[1]))
	}
	return gc
}

func (a *adapter) adaptGroupByExpression(n *peg.Node) GroupByItem {
	// GroupByExpression <- EmptyGroupingItem / CubeOrRollupClause / GroupingSetsClause / GroupByBaseExpression
	alt := choice(body(n))
	switch alt.Name {
	case "EmptyGroupingItem":
		return &GroupByEmpty{groupByItemNode{a.at(alt)}}
	case "CubeOrRollupClause":
		// CubeOrRollup Parens(List(Expression)?)
		seq := body(alt)
		items, _ := a.adaptOptExprList(seq.Children[1])
		if choice(body(seq.Children[0])).Name == "CubeKeyword" {
			return &GroupByCube{groupByItemNode{a.at(alt)}, items}
		}
		return &GroupByRollup{groupByItemNode{a.at(alt)}, items}
	case "GroupingSetsClause":
		// 'GROUPING' 'SETS' Parens(List(GroupByExpression))
		gsSeq := body(alt)
		listSeq := body(body(gsSeq.Children[2]).Children[1])
		sets := []GroupByItem{a.adaptGroupByExpression(listSeq.Children[0])}
		for _, tail := range listSeq.Children[1].Children {
			sets = append(sets, a.adaptGroupByExpression(body(tail).Children[1]))
		}
		return &GroupingSets{groupByItemNode{a.at(alt)}, sets}
	default: // GroupByBaseExpression <- Expression
		return &GroupByExprItem{groupByItemNode{a.at(alt)}, a.adaptExpression(body(alt))}
	}
}

func (a *adapter) adaptLimitOffset(n *peg.Node) *LimitClause {
	// LimitOffset <- LimitOffsetClause / OffsetFetchClause / OffsetLimitClause / FetchOnlyClause
	alt := choice(body(n))
	lc := &LimitClause{baseExpr: a.at(n)}
	switch alt.Name {
	case "LimitOffsetClause":
		// LimitClause OffsetClause?
		s := body(alt)
		a.fillLimitValue(lc, s.Children[0])
		if o, ok := present(s.Children[1]); ok {
			lc.Offset = a.adaptOffsetClause(o)
		}
	case "OffsetLimitClause":
		// OffsetClause LimitClause?
		s := body(alt)
		lc.Offset = a.adaptOffsetClause(s.Children[0])
		if l, ok := present(s.Children[1]); ok {
			a.fillLimitValue(lc, l)
		}
	case "OffsetFetchClause":
		// OffsetClause FetchClause
		s := body(alt)
		lc.Offset = a.adaptOffsetClause(s.Children[0])
		a.fillFetchValue(lc, s.Children[1])
	default: // FetchOnlyClause <- FetchClause
		a.fillFetchValue(lc, body(alt))
	}
	return lc
}

func (a *adapter) fillLimitValue(lc *LimitClause, n *peg.Node) {
	// LimitClause <- 'LIMIT' LimitValue ; LimitValue <- LimitAll / LimitLiteralPercent / LimitExpression
	limitAlt := choice(body(body(n).Children[1]))
	switch limitAlt.Name {
	case "LimitAll":
		lc.All = true
	case "LimitLiteralPercent":
		// NumberLiteral 'PERCENT'
		numNode := body(limitAlt).Children[0]
		lc.Limit = &Literal{baseExpr: a.at(numNode), Kind: LitNumber, Text: numNode.Text}
		lc.Percent = true
	default: // LimitExpression <- Expression '%'?
		s := body(limitAlt)
		lc.Limit = a.adaptExpression(s.Children[0])
		if _, ok := present(s.Children[1]); ok {
			lc.Percent = true
		}
	}
}

func (a *adapter) fillFetchValue(lc *LimitClause, n *peg.Node) {
	// FetchClause <- 'FETCH' FirstOrNext FetchValue RowOrRows 'ONLY' ; FetchValue <- Expression
	// body(n).Children[2] is the FetchValue node itself; adaptExpression
	// expects an Expression node directly (its own body() call unwraps
	// Expression -> LambdaArrowExpression), so unwrap FetchValue's single-ref
	// body first to reach it.
	lc.Limit = a.adaptExpression(body(body(n).Children[2]))
}

func (a *adapter) adaptOffsetClause(n *peg.Node) Expr {
	// OffsetClause <- 'OFFSET' OffsetValue ; OffsetValue <- Expression RowOrRows?
	return a.adaptExpression(body(body(n).Children[1]).Children[0])
}
