package federate

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// The executor builds every query as a parser AST and renders it with
// parser.Write, so identifiers and values are quoted by the writer and
// clauses compose as nodes rather than spliced SQL text. Values are
// rendered as typed literals, not '?' args, so no clause's args can bind
// to another's placeholders.

func colRef(name string) parser.Expr { return &parser.Ident{Parts: []string{name}} }

func strLit(s string) parser.Expr { return &parser.Literal{Kind: parser.LitString, Text: s} }

func numLit(n int64) parser.Expr {
	return &parser.Literal{Kind: parser.LitNumber, Text: strconv.FormatInt(n, 10)}
}

// typeName parses one of the schema's own type names (VARIANT, UUID,
// STRUCT(...)[]); failing to is a bug in the schema, not the query.
func typeName(s string) *parser.TypeName {
	t, err := parser.ParseType(s)
	if err != nil {
		panic(fmt.Sprintf("federate: bad type %q: %v", s, err))
	}
	return t
}

func cast(x parser.Expr, typ string) parser.Expr {
	return &parser.CastExpr{X: x, Type: typeName(typ)}
}

// lit renders v as a typed SQL literal.
func lit(v any) (parser.Expr, error) {
	switch x := v.(type) {
	case nil:
		return &parser.Literal{Kind: parser.LitNull}, nil
	case string:
		return strLit(x), nil
	case bool:
		if x {
			return &parser.Literal{Kind: parser.LitTrue}, nil
		}
		return &parser.Literal{Kind: parser.LitFalse}, nil
	case int:
		return numLit(int64(x)), nil
	case int32:
		return numLit(int64(x)), nil
	case int64:
		return numLit(x), nil
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return nil, fmt.Errorf("federate: can't render %v as a SQL literal", x)
		}
		return &parser.Literal{Kind: parser.LitNumber, Text: strconv.FormatFloat(x, 'g', -1, 64)}, nil
	case time.Time:
		return timeLit(x), nil
	case uuid.UUID:
		return uuidLit(x.String()), nil
	}
	return nil, fmt.Errorf("federate: can't render %T as a SQL literal", v)
}

func timeLit(t time.Time) parser.Expr {
	return &parser.TypeLiteral{Type: typeName("TIMESTAMPTZ"), Value: t.UTC().Format("2006-01-02 15:04:05.999999-07:00")}
}

func uuidLit(s string) parser.Expr { return cast(strLit(s), "UUID") }

func binary(op string, l, r parser.Expr) parser.Expr {
	return &parser.BinaryExpr{Op: op, Left: l, Right: r}
}

// and joins conds with AND, skipping nils; nil for none.
func and(conds ...parser.Expr) parser.Expr {
	var out parser.Expr
	for _, c := range conds {
		switch {
		case c == nil:
		case out == nil:
			out = c
		default:
			out = binary("AND", out, c)
		}
	}
	return out
}

func item(x parser.Expr, as string) *parser.SelectItem { return &parser.SelectItem{Expr: x, Alias: as} }

func starItem() *parser.SelectItem { return item(&parser.StarExpr{}, "") }

func from(ref parser.TableRef) *parser.FromClause {
	return &parser.FromClause{Refs: []parser.TableRef{ref}}
}

func tableRef(name ...string) *parser.BaseTableRef { return &parser.BaseTableRef{Name: name} }

// selectStar is SELECT * FROM ref.
func selectStar(ref parser.TableRef) *parser.SelectStatement {
	return &parser.SelectStatement{Columns: []*parser.SelectItem{starItem()}, From: from(ref)}
}

func unionAll(l, r *parser.SelectStatement, byName bool) *parser.SelectStatement {
	return &parser.SelectStatement{SetOp: parser.SetOpUnion, SetAll: true, SetByName: byName, SetLeft: l, SetRight: r}
}

var materialized = true

func cte(name string, sel *parser.SelectStatement) *parser.CTE {
	return &parser.CTE{Name: name, Select: sel}
}

func materializedCTE(name string, sel *parser.SelectStatement) *parser.CTE {
	return &parser.CTE{Name: name, Select: sel, Materialized: &materialized}
}

// withCTEs returns stmt with ctes ahead of its own WITH clause's, leaving
// stmt itself unchanged.
func withCTEs(stmt *parser.SelectStatement, ctes ...*parser.CTE) *parser.SelectStatement {
	if len(ctes) == 0 {
		return stmt
	}
	out := *stmt
	w := &parser.WithClause{}
	if stmt.With != nil {
		*w = *stmt.With
	}
	w.CTEs = append(append([]*parser.CTE{}, ctes...), w.CTEs...)
	out.With = w
	return &out
}

// render writes stmt as SQL text.
func render(stmt *parser.SelectStatement) (string, error) {
	var b strings.Builder
	if err := parser.Write(&b, stmt); err != nil {
		return "", fmt.Errorf("federate: rendering query: %w", err)
	}
	return b.String(), nil
}
