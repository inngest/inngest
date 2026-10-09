// pkg/duckdb/parser/adapter_cte.go
package parser

import "github.com/inngest/inngest/pkg/duckdb/parser/peg"

func (a *adapter) adaptWithClause(n *peg.Node) *WithClause {
	// WithClause <- 'WITH' Recursive? List(WithStatement)
	seq := body(n)
	_, recursive := present(seq.Children[1])
	listSeq := body(seq.Children[2])
	wc := &WithClause{baseExpr: a.at(n), Recursive: recursive, CTEs: []*CTE{a.adaptWithStatement(listSeq.Children[0])}}
	for _, tail := range listSeq.Children[1].Children {
		wc.CTEs = append(wc.CTEs, a.adaptWithStatement(body(tail).Children[1]))
	}
	return wc
}

func (a *adapter) adaptWithStatement(n *peg.Node) *CTE {
	// WithStatement <- ColIdOrString InsertColumnList? UsingKey? 'AS' Materialized? CTEBody
	seq := body(n)
	cte := &CTE{baseExpr: a.at(n), Name: literalText(seq.Children[0])}
	// seq.Children[1] (InsertColumnList?) always resolves absent — see Step 3.
	if _, ok := present(seq.Children[2]); ok {
		panic("duckdb/parser: USING KEY on a CTE is not yet supported")
	}
	if m, ok := present(seq.Children[4]); ok {
		// Materialized <- 'NOT'? 'MATERIALIZED'
		_, not := present(body(m).Children[0])
		val := !not
		cte.Materialized = &val
	}
	// CTEBody <- CTESelectBody / CTEDMLBody ; CTESelectBody <- Parens(SelectStatementInternal)
	bodyAlt := choice(body(seq.Children[5]))
	if bodyAlt.Name != "CTESelectBody" {
		panic("duckdb/parser: a non-SELECT CTE body is not supported")
	}
	cte.Select = a.adaptParenSelect(body(bodyAlt))
	return cte
}

func (a *adapter) adaptValuesClause(n *peg.Node) *SelectStatement {
	// ValuesClause <- 'VALUES' List(ValuesExpressions) ; ValuesExpressions <- Parens(List(Expression))
	seq := body(n)
	listSeq := body(seq.Children[1])
	rows := [][]Expr{a.adaptExprList(body(listSeq.Children[0]))}
	for _, tail := range listSeq.Children[1].Children {
		rows = append(rows, a.adaptExprList(body(body(tail).Children[1])))
	}
	return &SelectStatement{baseExpr: a.at(n), Values: rows}
}

func (a *adapter) adaptSelectSetOpChain(n *peg.Node) *SelectStatement {
	// SelectSetOpChain <- IntersectChain SelectSetOpChainTail*
	seq := body(n)
	left := a.adaptIntersectChain(seq.Children[0])
	for _, tail := range body(seq.Children[1]).Children {
		// SelectSetOpChainTail <- SetopClause IntersectChain
		tseq := body(tail)
		op, all, byName := a.adaptSetopClause(tseq.Children[0])
		right := a.adaptIntersectChain(tseq.Children[1])
		left = &SelectStatement{baseExpr: a.at(tail), SetOp: op, SetAll: all, SetByName: byName, SetLeft: left, SetRight: right}
	}
	return left
}

func (a *adapter) adaptSetopClause(n *peg.Node) (op SetOp, all, byName bool) {
	// SetopClause <- SetopType DistinctOrAll? ByName? ; SetopType <- SetopUnion / SetopExcept
	seq := body(n)
	if choice(body(seq.Children[0])).Name == "SetopUnion" {
		op = SetOpUnion
	} else {
		op = SetOpExcept
	}
	if d, ok := present(seq.Children[1]); ok {
		all = choice(body(d)).Name == "AllKeyword"
	}
	_, byName = present(seq.Children[2])
	return
}

func (a *adapter) adaptIntersectChain(n *peg.Node) *SelectStatement {
	// IntersectChain <- SelectAtom IntersectChainTail*
	seq := body(n)
	left := a.adaptSelectAtom(seq.Children[0])
	for _, tail := range body(seq.Children[1]).Children {
		// IntersectChainTail <- SetIntersectClause SelectAtom ; SetIntersectClause <- 'INTERSECT' DistinctOrAll?
		tseq := body(tail)
		all := false
		if d, ok := present(body(tseq.Children[0]).Children[1]); ok {
			all = choice(body(d)).Name == "AllKeyword"
		}
		left = &SelectStatement{baseExpr: a.at(tail), SetOp: SetOpIntersect, SetAll: all, SetLeft: left, SetRight: a.adaptSelectAtom(tseq.Children[1])}
	}
	return left
}
