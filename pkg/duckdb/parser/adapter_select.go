// pkg/duckdb/parser/adapter_select.go
package parser

import "github.com/inngest/inngest/pkg/duckdb/parser/peg"

// adaptSelectStatementInternal is the mutually-recursive counterpart to
// adaptExpression: a subquery is an expression containing a SELECT, whose
// WHERE/target-list contain expressions. This task's cut supports a single
// plain SELECT with a single-table FROM; every other case panics naming the
// task that adds it (see this task's top-level scope note) rather than
// silently producing a wrong AST.
func (a *adapter) adaptSelectStatementInternal(n *peg.Node) *SelectStatement {
	// SelectStatementInternal <- WithClause? SelectSetOpChain ResultModifiers?
	seq := body(n)
	var with *WithClause
	if w, ok := present(seq.Children[0]); ok {
		with = a.adaptWithClause(w)
	}
	stmt := a.adaptSelectSetOpChain(seq.Children[1])
	stmt.With = with
	if mods, ok := present(seq.Children[2]); ok {
		mseq := body(mods)
		if o, ok := present(mseq.Children[0]); ok {
			stmt.OrderBy = a.adaptOrderByClause(o)
		}
		if l, ok := present(mseq.Children[1]); ok {
			stmt.Limit = a.adaptLimitOffset(l)
		}
	}
	return stmt
}

func (a *adapter) adaptSelectAtom(n *peg.Node) *SelectStatement {
	// SelectAtom <- SelectParens / SelectStatementType
	alt := choice(body(n))
	if alt.Name == "SelectParens" {
		return a.adaptParenSelect(body(alt)) // SelectParens <- Parens(SelectStatementInternal)
	}
	// SelectStatementType <- OptionalParensSimpleSelect / ValuesClause / DescribeStatement / TableStatement / PivotStatement / UnpivotStatement
	stAlt := choice(body(alt))
	switch stAlt.Name {
	case "OptionalParensSimpleSelect":
		// OptionalParensSimpleSelect <- SimpleSelectParens / SimpleSelect
		spAlt := choice(body(stAlt))
		if spAlt.Name == "SimpleSelectParens" {
			return a.adaptSimpleSelect(body(body(spAlt)).Children[1]) // Parens(SimpleSelect)
		}
		return a.adaptSimpleSelect(spAlt)
	case "ValuesClause":
		return a.adaptValuesClause(stAlt)
	case "PivotStatement":
		return a.adaptPivotStatement(stAlt)
	case "UnpivotStatement":
		return a.adaptUnpivotStatement(stAlt)
	default: // DescribeStatement / TableStatement
		panic("duckdb/parser: DESCRIBE/TABLE as a standalone statement is not supported")
	}
}

// adaptParenSelect reads a `Parens(SelectStatementInternal)`-shaped node (n
// is the Parens KindRule node) — used by every subquery-producing
// expression form in adapter_expr.go, and by SelectParens above.
func (a *adapter) adaptParenSelect(n *peg.Node) *SelectStatement {
	return a.adaptSelectStatementInternal(body(n).Children[1])
}

func (a *adapter) adaptSimpleSelect(n *peg.Node) *SelectStatement {
	// SimpleSelect <- SelectFrom WhereClause? GroupByClause? HavingClause? WindowClause? QualifyClause? SampleClause?
	seq := body(n)
	cols, from, distinct := a.adaptSelectFrom(seq.Children[0])
	stmt := &SelectStatement{baseExpr: a.at(n), Columns: cols, From: from, Distinct: distinct}
	if w, ok := present(seq.Children[1]); ok {
		stmt.Where = a.adaptExpression(body(w).Children[1]) // WhereClause <- 'WHERE' Expression
	}
	if g, ok := present(seq.Children[2]); ok {
		stmt.GroupBy = a.adaptGroupByClause(g)
	}
	if h, ok := present(seq.Children[3]); ok {
		stmt.Having = a.adaptExpression(body(h).Children[1]) // HavingClause <- 'HAVING' Expression
	}
	if w, ok := present(seq.Children[4]); ok {
		stmt.Windows = a.adaptWindowClause(w)
	}
	if q, ok := present(seq.Children[5]); ok {
		stmt.Qualify = a.adaptExpression(body(q).Children[1]) // QualifyClause <- 'QUALIFY' Expression
	}
	if _, ok := present(seq.Children[6]); ok {
		panic("duckdb/parser: TABLESAMPLE is not yet supported")
	}
	return stmt
}

func (a *adapter) adaptSelectFrom(n *peg.Node) ([]*SelectItem, *FromClause, *DistinctClause) {
	// SelectFrom <- SelectFromClause / FromSelectClause
	alt := choice(body(n))
	if alt.Name == "SelectFromClause" {
		// SelectFromClause <- SelectClause FromClause?
		seq := body(alt)
		cols, distinct := a.adaptSelectClause(seq.Children[0])
		var from *FromClause
		if f, ok := present(seq.Children[1]); ok {
			from = a.adaptFromClause(f)
		}
		return cols, from, distinct
	}
	// FromSelectClause <- FromClause SelectClause?
	seq := body(alt)
	from := a.adaptFromClause(seq.Children[0])
	var cols []*SelectItem
	var distinct *DistinctClause
	if c, ok := present(seq.Children[1]); ok {
		cols, distinct = a.adaptSelectClause(c)
	}
	return cols, from, distinct
}

func (a *adapter) adaptSelectClause(n *peg.Node) ([]*SelectItem, *DistinctClause) {
	// SelectClause <- 'SELECT' DistinctClause? TargetList?
	seq := body(n)
	var distinct *DistinctClause
	if d, ok := present(seq.Children[1]); ok {
		distinct = a.adaptDistinctClause(d)
	}
	targets, ok := present(seq.Children[2])
	if !ok {
		return nil, distinct
	}
	return a.adaptTargetList(targets), distinct
}

func (a *adapter) adaptAliasedExpression(n *peg.Node) *SelectItem {
	// AliasedExpression <- ColIdExpression / ExpressionAsCollabel / ExpressionOptIdentifier
	alt := choice(body(n))
	switch alt.Name {
	case "ColIdExpression":
		// ColId ':' Expression
		seq := body(alt)
		return &SelectItem{baseExpr: a.at(n), Alias: literalText(seq.Children[0]), Expr: a.adaptExpression(seq.Children[2])}
	case "ExpressionAsCollabel":
		// Expression 'AS' ColLabelOrString
		seq := body(alt)
		return &SelectItem{baseExpr: a.at(n), Expr: a.adaptExpression(seq.Children[0]), Alias: literalText(seq.Children[2])}
	default: // ExpressionOptIdentifier <- Expression Identifier?
		seq := body(alt)
		item := &SelectItem{baseExpr: a.at(n), Expr: a.adaptExpression(seq.Children[0])}
		if id, ok := present(seq.Children[1]); ok {
			item.Alias = literalText(id)
		}
		return item
	}
}

// --- FROM: single plain (optionally aliased) table name only for now ---

func (a *adapter) adaptFromClause(n *peg.Node) *FromClause {
	// FromClause <- 'FROM' List(TableRef)
	seq := body(n)
	listSeq := body(seq.Children[1])
	refs := []TableRef{a.adaptTableRef(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		refs = append(refs, a.adaptTableRef(body(tail).Children[1]))
	}
	return &FromClause{baseExpr: a.at(n), Refs: refs}
}

func (a *adapter) adaptBaseTableRef(n *peg.Node) *BaseTableRef {
	// BaseTableRef <- TableAliasColon? BaseTableName TableAlias? AtClause? SampleClause?
	seq := body(n)
	if _, ok := present(seq.Children[0]); ok {
		panic("duckdb/parser: the `alias:` table-alias-colon form is not yet supported (see Task 10)")
	}
	ref := &BaseTableRef{baseTableRefNode: baseTableRefNode{a.at(n)}, Name: a.adaptBaseTableName(seq.Children[1])}
	if alias, ok := present(seq.Children[2]); ok {
		ref.Alias = a.adaptTableAlias(alias)
	}
	if _, ok := present(seq.Children[3]); ok {
		panic("duckdb/parser: AT (time travel) is not yet supported (see Task 10)")
	}
	if _, ok := present(seq.Children[4]); ok {
		panic("duckdb/parser: TABLESAMPLE is not yet supported (see Task 10)")
	}
	return ref
}

func (a *adapter) adaptBaseTableName(n *peg.Node) []string {
	// BaseTableName <- QualifiedTableName / UnqualifiedBaseTableName
	alt := choice(body(n))
	if alt.Name == "UnqualifiedBaseTableName" {
		return []string{literalText(body(alt))} // UnqualifiedBaseTableName <- TableName
	}
	// QualifiedTableName <- CatalogReservedSchemaTable / SchemaReservedTable
	qAlt := choice(body(alt))
	if qAlt.Name == "SchemaReservedTable" {
		// SchemaReservedTable <- SchemaQualification ReservedTableName
		seq := body(qAlt)
		return []string{literalText(body(seq.Children[0]).Children[0]), literalText(seq.Children[1])}
	}
	// CatalogReservedSchemaTable <- CatalogQualification ReservedSchemaQualification+ ReservedTableName
	// (Task 15's second-commit re-vendor found ReservedSchemaQualification
	// widened from a single mandatory match to '+' upstream — DuckDB now
	// allows nested schema chains here, e.g. catalog.schema1.schema2.table.)
	seq := body(qAlt)
	parts := []string{literalText(body(seq.Children[0]).Children[0])}
	for _, s := range repeatChildren(seq.Children[1]) {
		parts = append(parts, literalText(body(s).Children[0]))
	}
	return append(parts, literalText(seq.Children[2]))
}

func (a *adapter) adaptTableAlias(n *peg.Node) string {
	// TableAlias <- TableAliasAs / TableAliasWithoutAs
	alt := choice(body(n))
	if alt.Name == "TableAliasAs" {
		// 'AS' IdentifierOrStringLiteral ColumnAliases?
		seq := body(alt)
		if _, ok := present(seq.Children[2]); ok {
			panic("duckdb/parser: column aliases on a table alias are not yet supported (see Task 10)")
		}
		return literalText(seq.Children[1])
	}
	// TableAliasWithoutAs <- Identifier ColumnAliases?
	seq := body(alt)
	if _, ok := present(seq.Children[1]); ok {
		panic("duckdb/parser: column aliases on a table alias are not yet supported (see Task 10)")
	}
	return literalText(seq.Children[0])
}
