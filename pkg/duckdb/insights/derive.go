package insights

import (
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

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
// validation.
//
// The scope output columns are inferred against is the one validation
// itself resolved (validatedLeafScope), never a second resolveScope of the
// same FROM clause: re-resolving re-derives every nested FROM-clause
// subquery, so a subquery nested k deep was derived 2^k times — Transpile
// was exponential in nesting depth (depth 16 took ~0.7s).
func deriveTable(stmt *parser.SelectStatement, name string, explicitCols []string, ctes map[string]logicalTable, outerScope *tableScope, diags *[]Diagnostic) (logicalTable, error) {
	leafScope, err := validatedLeafScope(stmt, ctes, outerScope, diags)
	if err != nil {
		return logicalTable{}, err
	}

	order, cols, err := inferOutputColumns(leftmostOperand(stmt), leafScope, explicitCols)
	if err != nil {
		return logicalTable{}, err
	}
	return logicalTable{name: name, columnOrder: order, columns: cols}, nil
}

// validatedLeafScope validates stmt exactly as validateWithCTEs does and
// returns the scope of its leftmost operand (stmt itself unless it is a
// set operation) — the scope its output columns come from. validateWithCTEs
// discards its operands' scopes for a set operation, so that case repeats
// its two steps here (a WITH clause first, then each operand with the
// merged CTEs) to keep the left operand's.
func validatedLeafScope(stmt *parser.SelectStatement, ctes map[string]logicalTable, outerScope *tableScope, diags *[]Diagnostic) (*tableScope, error) {
	if stmt.SetOp == parser.SetOpNone {
		scope, _, err := validateWithCTEs(stmt, ctes, outerScope, diags)
		return scope, err
	}
	if stmt.With != nil {
		// A statement holding only the WITH clause: validateWithCTEs
		// derives its CTEs and returns them merged, with nothing else to
		// validate.
		_, merged, err := validateWithCTEs(&parser.SelectStatement{With: stmt.With}, ctes, nil, diags)
		if err != nil {
			return nil, err
		}
		ctes = merged
	}
	left, err := validatedLeafScope(stmt.SetLeft, ctes, outerScope, diags)
	if err != nil {
		return nil, err
	}
	if _, _, err := validateWithCTEs(stmt.SetRight, ctes, outerScope, diags); err != nil {
		return nil, err
	}
	return left, nil
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
// name is used) -- a computed expression with no alias is rejected rather
// than guessing a name.
//
// explicitCols renames columns positionally first, then a duplicate name
// (e.g. a join's "*" expanding two same-named columns) is renamed the way
// DuckDB names a derived table's columns (dedupeColumnNames), so the
// derived table keeps one entry per real output column and "SELECT *"
// over it stays positionally aligned.
func inferOutputColumns(stmt *parser.SelectStatement, scope *tableScope, explicitCols []string) ([]string, map[string]knownColumn, error) {
	var names []string
	var kcs []knownColumn
	for _, item := range stmt.Columns {
		if star, ok := item.Expr.(*parser.StarExpr); ok {
			for _, sc := range resolveStarColumns(star, scope) {
				names = append(names, sc.name)
				kcs = append(kcs, sc.col)
			}
			continue
		}

		name, err := outputColumnName(item)
		if err != nil {
			return nil, nil, err
		}
		names = append(names, name)
		kcs = append(kcs, knownColumn{
			colType:   inferType(item.Expr, scope),
			pathHints: rootHint(resolveItemHint(item.Expr, scope)),
		})
	}

	for i, override := range explicitCols {
		if i < len(names) {
			names[i] = override
		}
	}
	names = dedupeColumnNames(names)

	cols := make(map[string]knownColumn, len(names))
	for i, name := range names {
		cols[strings.ToLower(name)] = kcs[i]
	}
	return names, cols, nil
}

// dedupeColumnNames renames each name already taken (case-insensitively)
// by an earlier one to name_1, name_2, ..., the first suffix not yet taken,
// matching DuckDB's derived-table naming (checked empirically):
// (x, x, x, x_1) -> (x, x_1, x_2, x_1_1), and (x, X) -> (x, X_1).
func dedupeColumnNames(names []string) []string {
	out := make([]string, len(names))
	taken := make(map[string]bool, len(names))
	for i, name := range names {
		candidate := name
		for n := 1; taken[strings.ToLower(candidate)]; n++ {
			candidate = fmt.Sprintf("%s_%d", name, n)
		}
		taken[strings.ToLower(candidate)] = true
		out[i] = candidate
	}
	return out
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
