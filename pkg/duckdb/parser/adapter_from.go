// pkg/duckdb/parser/adapter_from.go
package parser

import "github.com/inngest/inngest/pkg/duckdb/parser/peg"

func (a *adapter) adaptTableRef(n *peg.Node) TableRef {
	// TableRef <- InnerTableRef JoinOrPivot*
	seq := body(n)
	result := a.adaptInnerTableRef(seq.Children[0])
	for _, jp := range body(seq.Children[1]).Children {
		result = a.adaptJoinOrPivot(result, jp)
	}
	return result
}

func (a *adapter) adaptInnerTableRef(n *peg.Node) TableRef {
	// InnerTableRef <- ValuesRef / TableFunction / TableSubquery / BaseTableRef / ParensTableRef
	alt := choice(body(n))
	switch alt.Name {
	case "BaseTableRef":
		return a.adaptBaseTableRef(alt)
	case "TableSubquery":
		return a.adaptTableSubquery(alt)
	case "TableFunction":
		return a.adaptTableFunctionRef(alt)
	case "ParensTableRef":
		return a.adaptParensTableRef(alt)
	default: // ValuesRef
		panic("duckdb/parser: VALUES as a table reference is not yet supported (see Task 11)")
	}
}

func (a *adapter) adaptTableSubquery(n *peg.Node) TableRef {
	// TableSubquery <- TableAliasColon? Lateral? SubqueryReference TableAlias?
	seq := body(n)
	if _, ok := present(seq.Children[0]); ok {
		panic("duckdb/parser: the `alias:` table-alias-colon form is not yet supported")
	}
	_, lateral := present(seq.Children[1])
	sub := a.adaptParenSelect(body(seq.Children[2])) // SubqueryReference <- Parens(SelectStatementInternal)
	ref := &TableSubqueryRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Lateral: lateral, Select: sub}
	if alias, ok := present(seq.Children[3]); ok {
		ref.Alias = a.adaptTableAlias(alias)
	}
	return ref
}

func (a *adapter) adaptTableFunctionRef(n *peg.Node) TableRef {
	// TableFunction <- TableFunctionLateralOpt / TableFunctionAliasColon
	alt := choice(body(n))
	if alt.Name == "TableFunctionAliasColon" {
		panic("duckdb/parser: the `alias:` table-alias-colon form is not yet supported")
	}
	// TableFunctionLateralOpt <- Lateral? QualifiedTableFunction TableFunctionArguments WithOrdinality? TableAlias?
	seq := body(alt)
	_, lateral := present(seq.Children[0])
	ref := &TableFunctionRef{
		baseTableRefNode: baseTableRefNode{a.at(n)},
		Lateral:          lateral,
		Name:             a.adaptQualifiedTableFunction(seq.Children[1]),
		Args:             a.adaptTableFunctionArgs(seq.Children[2]),
	}
	_, ref.WithOrdinality = present(seq.Children[3])
	if alias, ok := present(seq.Children[4]); ok {
		ref.Alias = a.adaptTableAlias(alias)
	}
	return ref
}

func (a *adapter) adaptQualifiedTableFunction(n *peg.Node) []string {
	// QualifiedTableFunction <- CatalogQualification? SchemaQualification* TableFunctionName
	// (Task 15's second-commit re-vendor found SchemaQualification widened
	// from '?' to '*' upstream — DuckDB now allows nested schema chains
	// here, e.g. catalog.schema1.schema2.func(...).)
	seq := body(n)
	var parts []string
	if c, ok := present(seq.Children[0]); ok {
		parts = append(parts, literalText(body(c).Children[0]))
	}
	for _, s := range repeatChildren(seq.Children[1]) {
		parts = append(parts, literalText(body(s).Children[0]))
	}
	return append(parts, literalText(seq.Children[2]))
}

func (a *adapter) adaptTableFunctionArgs(n *peg.Node) []Expr {
	// TableFunctionArguments <- Parens(List(FunctionArgument)?) — n's own
	// body is a single ExprCall (Parens(...)), so body(n) only unwraps to
	// the "Parens" KindRule node itself; a second body() reaches its Seq.
	inner, ok := present(body(body(n)).Children[1])
	if !ok {
		return nil
	}
	listSeq := body(inner)
	args := []Expr{a.adaptFunctionArgument(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		args = append(args, a.adaptFunctionArgument(body(tail).Children[1]))
	}
	return args
}

func (a *adapter) adaptParensTableRef(n *peg.Node) TableRef {
	// ParensTableRef <- TableAliasColon? Parens(TableRef) TableAlias? SampleClause?
	seq := body(n)
	if _, ok := present(seq.Children[0]); ok {
		panic("duckdb/parser: the `alias:` table-alias-colon form is not yet supported")
	}
	inner := a.adaptTableRef(body(seq.Children[1]).Children[1])
	ref := &ParensTableRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Ref: inner}
	if alias, ok := present(seq.Children[2]); ok {
		ref.Alias = a.adaptTableAlias(alias)
	}
	if _, ok := present(seq.Children[3]); ok {
		panic("duckdb/parser: TABLESAMPLE is not yet supported")
	}
	return ref
}

func (a *adapter) adaptJoinOrPivot(left TableRef, n *peg.Node) TableRef {
	// JoinOrPivot <- JoinClause / TablePivotClause / TableUnpivotClause
	alt := choice(body(n))
	switch alt.Name {
	case "JoinClause":
		return a.adaptJoinClause(left, alt)
	case "TablePivotClause":
		return a.adaptTablePivotClause(left, alt)
	default: // TableUnpivotClause
		return a.adaptTableUnpivotClause(left, alt)
	}
}

func (a *adapter) adaptJoinClause(left TableRef, n *peg.Node) TableRef {
	// JoinClause <- JoinByClause / RegularJoinClause / JoinWithoutOnClause / NearestJoinClause
	alt := choice(body(n))
	switch alt.Name {
	case "RegularJoinClause":
		return a.adaptRegularJoinClause(left, alt)
	case "JoinWithoutOnClause":
		return a.adaptJoinWithoutOnClause(left, alt)
	case "JoinByClause":
		panic("duckdb/parser: JOIN BY (TYPE ...) is not supported")
	default: // NearestJoinClause
		panic("duckdb/parser: NEAREST ... BY (vector-search joins) is not supported")
	}
}

func (a *adapter) adaptRegularJoinClause(left TableRef, n *peg.Node) TableRef {
	// RegularJoinClause <- Asof? JoinType? 'JOIN' TableRef JoinQualifier
	seq := body(n)
	_, asof := present(seq.Children[0])
	typ := "INNER"
	if t, ok := present(seq.Children[1]); ok {
		typ = a.adaptJoinType(t)
	}
	if asof {
		typ = "ASOF " + typ
	}
	jr := &JoinRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Left: left, Right: a.adaptTableRef(seq.Children[3]), Type: typ}
	a.fillJoinQualifier(jr, seq.Children[4])
	return jr
}

func (a *adapter) fillJoinQualifier(jr *JoinRef, n *peg.Node) {
	// JoinQualifier <- OnClause / UsingClause
	alt := choice(body(n))
	if alt.Name == "OnClause" {
		jr.On = a.adaptExpression(body(alt).Children[1]) // 'ON' Expression
		return
	}
	// UsingClause <- 'USING' Parens(List(ColumnName))
	listSeq := body(body(body(alt).Children[1]).Children[1])
	jr.Using = []string{literalText(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		jr.Using = append(jr.Using, literalText(body(tail).Children[1]))
	}
}

func (a *adapter) adaptJoinType(n *peg.Node) string {
	// JoinType <- FullJoin / LeftJoin / RightJoin / SemiJoin / AntiJoin / InnerJoin
	alt := choice(body(n))
	outer := func(kw string) string {
		if _, ok := present(body(alt).Children[1]); ok {
			return kw + " OUTER"
		}
		return kw
	}
	switch alt.Name {
	case "FullJoin":
		return outer("FULL")
	case "LeftJoin":
		return outer("LEFT")
	case "RightJoin":
		return outer("RIGHT")
	case "SemiJoin":
		return "SEMI"
	case "AntiJoin":
		return "ANTI"
	default:
		return "INNER"
	}
}

func (a *adapter) adaptJoinWithoutOnClause(left TableRef, n *peg.Node) TableRef {
	// JoinWithoutOnClause <- JoinPrefix 'JOIN' InnerTableRef
	seq := body(n)
	return &JoinRef{
		baseTableRefNode: baseTableRefNode{a.at(n)},
		Left:             left,
		Right:            a.adaptInnerTableRef(seq.Children[2]),
		Type:             a.adaptJoinPrefix(seq.Children[0]),
	}
}

func (a *adapter) adaptJoinPrefix(n *peg.Node) string {
	// JoinPrefix <- CrossJoinPrefix / NaturalJoinPrefix / PositionalJoinPrefix
	alt := choice(body(n))
	switch alt.Name {
	case "CrossJoinPrefix":
		return "CROSS"
	case "PositionalJoinPrefix":
		return "POSITIONAL"
	default: // NaturalJoinPrefix <- 'NATURAL' JoinType?
		seq := body(alt)
		if t, ok := present(seq.Children[1]); ok {
			return "NATURAL " + a.adaptJoinType(t)
		}
		return "NATURAL"
	}
}
