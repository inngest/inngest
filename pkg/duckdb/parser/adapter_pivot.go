// pkg/duckdb/parser/adapter_pivot.go
package parser

import "github.com/inngest/inngest/pkg/duckdb/parser/peg"

func (a *adapter) adaptTablePivotClause(source TableRef, n *peg.Node) TableRef {
	// TablePivotClause <- 'PIVOT' Parens(TablePivotClauseBody) TableAlias?
	seq := body(n)
	// TablePivotClauseBody <- TargetList 'FOR' PivotValueList+ PivotGroupByList?
	bodySeq := body(body(seq.Children[1]).Children[1])
	ref := &PivotRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Source: source, Columns: a.adaptTargetList(bodySeq.Children[0])}
	for _, pv := range body(bodySeq.Children[2]).Children {
		ref.For = append(ref.For, a.adaptPivotValueList(pv))
	}
	if gb, ok := present(bodySeq.Children[3]); ok {
		ref.GroupBy = a.adaptPivotGroupByList(gb)
	}
	if alias, ok := present(seq.Children[2]); ok {
		ref.Alias = a.adaptTableAlias(alias)
	}
	return ref
}

func (a *adapter) adaptPivotGroupByList(n *peg.Node) []string {
	// PivotGroupByList <- 'GROUP' 'BY' List(ColIdOrString)
	listSeq := body(body(n).Children[2])
	names := []string{literalText(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		names = append(names, literalText(body(tail).Children[1]))
	}
	return names
}

func (a *adapter) adaptPivotValueList(n *peg.Node) *PivotColumn {
	// PivotValueList <- PivotHeader 'IN' PivotValueTarget ; PivotHeader <- BaseExpression
	// seq.Children[0] is the PivotHeader node itself; adaptBaseExpression
	// expects a BaseExpression node directly, so unwrap PivotHeader's
	// single-ref body first to reach it (same fix as fillFetchValue's).
	seq := body(n)
	pc := &PivotColumn{baseExpr: a.at(n), Header: a.adaptBaseExpression(body(seq.Children[0]))}
	// PivotValueTarget <- PivotEnumTarget / PivotListTarget — the common
	// "FOR col IN (a, b, c)" form is PivotListTarget (a parenthesized
	// TargetList, not a single value each); PivotEnumTarget is only the
	// degenerate unparenthesized "FOR col IN a" spelling.
	target := choice(body(seq.Children[2]))
	if target.Name == "PivotEnumTarget" {
		pc.In = []string{literalText(body(target))} // PivotEnumTarget <- Identifier
		return pc
	}
	// PivotListTarget <- PivotTargetList ; PivotTargetList <- Parens(TargetList)
	// target -> body() -> PivotTargetList node -> body() -> "Parens" node -> body() -> its Seq.
	parensSeq := body(body(body(target)))
	items := a.adaptTargetList(parensSeq.Children[1])
	for _, item := range items {
		pc.In = append(pc.In, a.src[item.Pos().Offset:item.End().Offset])
	}
	return pc
}

func (a *adapter) adaptTableUnpivotClause(source TableRef, n *peg.Node) TableRef {
	// TableUnpivotClause <- 'UNPIVOT' IncludeOrExcludeNulls? Parens(TableUnpivotClauseBody) TableAlias?
	seq := body(n)
	ref := &UnpivotRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Source: source}
	if ie, ok := present(seq.Children[1]); ok {
		if choice(body(ie)).Name == "IncludeNulls" {
			ref.IncludeNulls = true
		} else {
			ref.ExcludeNulls = true
		}
	}
	// TableUnpivotClauseBody <- UnpivotHeader 'FOR' UnpivotValueList+
	bodySeq := body(body(seq.Children[2]).Children[1])
	ref.Header = a.adaptUnpivotHeader(bodySeq.Children[0])
	for _, uv := range body(bodySeq.Children[2]).Children {
		// UnpivotValueList <- UnpivotHeader 'IN' UnpivotTargetList ; UnpivotTargetList <- Parens(TargetList)
		// uvSeq.Children[2] is the UnpivotTargetList node itself, whose own
		// body is a single ExprCall (Parens(...)) — double-unwrap to reach
		// its Seq before indexing.
		uvSeq := body(uv)
		ref.Names = append(ref.Names, a.adaptUnpivotHeader(uvSeq.Children[0])...)
		ref.Columns = append(ref.Columns, a.adaptTargetList(body(body(uvSeq.Children[2])).Children[1])...)
	}
	if alias, ok := present(seq.Children[3]); ok {
		ref.Alias = a.adaptTableAlias(alias)
	}
	return ref
}

func (a *adapter) adaptUnpivotHeader(n *peg.Node) []string {
	// UnpivotHeader <- UnpivotHeaderSingle / UnpivotHeaderList
	alt := choice(body(n))
	if alt.Name == "UnpivotHeaderSingle" {
		return []string{literalText(body(alt))} // <- ColIdOrString
	}
	// UnpivotHeaderList <- Parens(List(ColIdOrString))
	listSeq := body(body(alt).Children[1])
	names := []string{literalText(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		names = append(names, literalText(body(tail).Children[1]))
	}
	return names
}

func (a *adapter) adaptPivotStatement(n *peg.Node) *SelectStatement {
	// PivotStatement <- PivotKeyword TableRef PivotOn? PivotUsing? PivotGroupByList?
	seq := body(n)
	ref := &PivotRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Source: a.adaptTableRef(seq.Children[1])}
	if on, ok := present(seq.Children[2]); ok {
		// PivotOn <- 'ON' PivotColumnList ; PivotColumnList <- List(PivotColumnEntry)
		// body(on).Children[1] is the PivotColumnList node itself, whose own
		// body is a single ExprCall (List(...)) — double-unwrap to reach its Seq.
		listSeq := body(body(body(on).Children[1]))
		ref.For = append(ref.For, a.adaptPivotColumnEntry(listSeq.Children[0]))
		for _, tail := range listSeq.Children[1].Children {
			ref.For = append(ref.For, a.adaptPivotColumnEntry(body(tail).Children[1]))
		}
	}
	if using, ok := present(seq.Children[3]); ok {
		ref.Columns = a.adaptTargetList(body(using).Children[1]) // PivotUsing <- 'USING' TargetList
	}
	if gb, ok := present(seq.Children[4]); ok {
		ref.GroupBy = a.adaptPivotGroupByList(gb)
	}
	return &SelectStatement{baseExpr: a.at(n), From: &FromClause{baseExpr: a.at(n), Refs: []TableRef{ref}}}
}

func (a *adapter) adaptPivotColumnEntry(n *peg.Node) *PivotColumn {
	// PivotColumnEntry <- PivotColumnSubquery / PivotValueList / PivotColumnExpression
	alt := choice(body(n))
	switch alt.Name {
	case "PivotValueList":
		return a.adaptPivotValueList(alt)
	case "PivotColumnExpression":
		return &PivotColumn{baseExpr: a.at(alt), Header: a.adaptExpression(body(alt))}
	default:
		panic("duckdb/parser: a PIVOT column subquery entry is not yet supported")
	}
}

func (a *adapter) adaptUnpivotStatement(n *peg.Node) *SelectStatement {
	// UnpivotStatement <- UnpivotKeyword TableRef 'ON' TargetList IntoNameValues?
	seq := body(n)
	ref := &UnpivotRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Source: a.adaptTableRef(seq.Children[1]), Columns: a.adaptTargetList(seq.Children[3])}
	if _, ok := present(seq.Children[4]); ok {
		panic("duckdb/parser: UNPIVOT ... INTO NAME ... VALUE ... is not yet supported")
	}
	return &SelectStatement{baseExpr: a.at(n), From: &FromClause{baseExpr: a.at(n), Refs: []TableRef{ref}}}
}
