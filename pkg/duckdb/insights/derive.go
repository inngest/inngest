package insights

import "github.com/inngest/inngest/pkg/duckdb/parser"

// deriveTable validates stmt (a CTE body or a FROM-clause subquery) with
// its own scope — built from its own FROM clause plus ctes, per standard
// SQL visibility rules — and infers the logicalTable it exposes to
// whatever references it by name.
//
// outerScope, when non-nil, is what stmt may correlate into for any name
// not found in its own scope (see tableScope.outer). A CTE body is always
// called with outerScope nil (a CTE can never correlate); a FROM-clause
// subquery gets one only when it's LATERAL (scope.go's addSubquery).
// Without an outer scope, a reference to the enclosing query's columns
// simply fails as an ordinary "unknown column" error.
//
// name is the CTE name or subquery alias, used only for error messages.
// explicitCols, if non-empty, is a WITH x(a, b, ...) explicit column list
// overriding inferred names positionally. diags is validateWithCTEs' own
// diagnostics accumulator, passed straight through to stmt's own
// validation -- the second resolveScope call below (for leafScope) always
// passes nil instead, since it re-resolves the very same FROM clause
// validateWithCTEs just validated, and reusing diags there would collect
// (and so surface) every one of that FROM clause's function-call
// diagnostics a second time.
func deriveTable(stmt *parser.SelectStatement, name string, explicitCols []string, ctes map[string]logicalTable, outerScope *tableScope, diags *[]Diagnostic) (logicalTable, error) {
	_, _, err := validateWithCTEs(stmt, ctes, outerScope, diags)
	if err != nil {
		return logicalTable{}, err
	}

	leaf := leftmostOperand(stmt)
	leafScope, err := resolveScope(leaf.From, ctes, nil)
	if err != nil {
		return logicalTable{}, err
	}
	leafScope.outer = outerScope

	order, cols, err := inferOutputColumns(leaf, leafScope, explicitCols)
	if err != nil {
		return logicalTable{}, err
	}
	return logicalTable{name: name, columnOrder: order, columns: cols}, nil
}

// leftmostOperand returns stmt itself, or (for a UNION/INTERSECT/EXCEPT
// statement) its leftmost leaf operand — real SQL takes a set operation's
// output column names from its first operand.
func leftmostOperand(stmt *parser.SelectStatement) *parser.SelectStatement {
	for stmt.SetOp != parser.SetOpNone {
		stmt = stmt.SetLeft
	}
	return stmt
}

// inferOutputColumns computes stmt's projected columns' names, types, and
// hints, for exposing a CTE or subquery as a new scope entry. A star
// expands using resolveStarColumns (columnhints.go), the same resolution
// buildColumnHints itself uses. Any other item needs either an explicit
// alias or to already be a bare/qualified column reference (whose own
// name is used) — a computed expression with no alias is rejected rather
// than guessing a name.
//
// A duplicate output name (e.g. a join's "*" expanding two same-named
// columns from different tables) collapses to one entry, keyed by that
// name — an accepted imprecision, since a derived table only needs "is
// this a known column of it," not an exact positional count.
func inferOutputColumns(stmt *parser.SelectStatement, scope *tableScope, explicitCols []string) ([]string, map[string]knownColumn, error) {
	var order []string
	cols := map[string]knownColumn{}
	add := func(name string, kc knownColumn) {
		if _, exists := cols[name]; !exists {
			order = append(order, name)
		}
		cols[name] = kc
	}

	for _, item := range stmt.Columns {
		if star, ok := item.Expr.(*parser.StarExpr); ok {
			for _, sc := range resolveStarColumns(star, scope) {
				add(sc.name, sc.col)
			}
			continue
		}

		name, err := outputColumnName(item)
		if err != nil {
			return nil, nil, err
		}
		add(name, knownColumn{
			colType:   inferType(item.Expr, scope),
			pathHints: rootHint(resolveItemHint(item.Expr, scope)),
		})
	}

	for i, override := range explicitCols {
		if i >= len(order) || override == order[i] {
			continue
		}
		cols[override] = cols[order[i]]
		delete(cols, order[i])
		order[i] = override
	}

	return order, cols, nil
}

// outputColumnName names one non-star SelectItem: its alias if given, else
// a bare/qualified column's own name — anything else must be aliased.
func outputColumnName(item *parser.SelectItem) (string, error) {
	if item.Alias != "" {
		return item.Alias, nil
	}
	switch e := item.Expr.(type) {
	case *parser.Ident:
		return e.Parts[len(e.Parts)-1], nil
	case *parser.DotExpr:
		return e.Field, nil
	default:
		return "", &ValidationError{Pos: item.Pos(), End: item.End(), Message: "a computed column in a CTE or subquery must have an alias"}
	}
}
