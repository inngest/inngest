package parser

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

// adaptType builds a TypeName from a Type node (common.gram):
//
//	Type <- TypeVariations ArrayBounds*
//
// TypeVariations has a dozen alternatives that differ mostly in which
// keywords they spell, so collectType reads them by walking the matched
// subtree for the few rule names that carry structure (Expression,
// NumberLiteral, Type, ColIdType, QualifiedTypeName, TimeZone, Interval)
// and treating every other keyword literal as part of the name.
func (a *adapter) adaptType(n *peg.Node) *TypeName {
	seq := body(n)
	alt := choice(body(seq.Children[0]))
	var t *TypeName
	if alt.Name == "SetofType" {
		// SetofType <- 'SETOF' Type. DuckDB parses it as the inner type,
		// so the inner type's own array bounds and these compose.
		t = a.adaptType(body(alt).Children[1])
		t.Setof = true
	} else {
		t = &TypeName{}
		var words []string
		a.collectType(t, alt, &words)
		if len(t.Name) == 0 {
			t.Name, t.Keyword = []string{strings.Join(words, " ")}, true
		}
	}
	t.baseExpr = a.at(n)
	for _, b := range repeatChildren(seq.Children[1]) {
		// ArrayBounds <- ArrayKeywordWithBounds / SquareBracketsArray / ArrayKeyword
		t.Arrays = append(t.Arrays, ArrayBound{Size: a.arrayBoundSize(b)})
	}
	return t
}

func (a *adapter) collectType(t *TypeName, n *peg.Node, words *[]string) {
	switch n.Kind {
	case peg.KindLiteral:
		if isWordLiteral(n.Text) {
			*words = append(*words, strings.ToUpper(n.Text))
		}
		return
	case peg.KindPrimitive:
		// FloatType <- 'FLOAT' Parens(NumberLiteral)?, IntervalNumber
		if n.Name == "NumberLiteral" {
			t.Modifiers = append(t.Modifiers, &Literal{baseExpr: a.at(n), Kind: LitNumber, Text: n.Text})
		}
		return
	case peg.KindRule:
		switch n.Name {
		case "Expression": // TypeModifiers <- Parens(List(Expression)?), BitType, GeometryType
			t.Modifiers = append(t.Modifiers, a.adaptTypeModifier(n))
			return
		case "Type": // MapType, TupleType
			t.Params = append(t.Params, a.adaptType(n))
			return
		case "ColIdType": // ColIdType <- ColId Type (RowType, UnionType)
			seq := body(n)
			t.Fields = append(t.Fields, &TypeField{baseExpr: a.at(n), Name: literalText(seq.Children[0]), Type: a.adaptType(seq.Children[1])})
			return
		case "QualifiedTypeName":
			t.Name = a.qualifiedTypeName(n)
			return
		case "TimeZone": // TimeZone <- WithOrWithout 'TIME' 'ZONE'
			t.TimeZone = keywordWords(n)
			return
		case "IntervalToIntervalAsType", "Interval":
			t.Interval = keywordWords(n)
			return
		}
	}
	for _, c := range n.Children {
		a.collectType(t, c, words)
	}
}

// qualifiedTypeName reads every name part under a QualifiedTypeName:
//
//	QualifiedTypeName <- CatalogReservedSchemaTypeName / SchemaReservedTypeName / TypeNameAsQualifiedName
//
// Each part is an identifier primitive (CatalogName, SchemaName,
// ReservedSchemaName, ReservedTypeName, TypeName), so collecting them in
// order gives the dotted name.
func (a *adapter) qualifiedTypeName(n *peg.Node) []string {
	var parts []string
	var walk func(*peg.Node)
	walk = func(n *peg.Node) {
		if n.Kind == peg.KindPrimitive {
			parts = append(parts, n.Value)
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return parts
}

// keywordWords joins every keyword literal under n, uppercased, with
// single spaces (`with time  zone` -> `WITH TIME ZONE`).
func keywordWords(n *peg.Node) string {
	var words []string
	var walk func(*peg.Node)
	walk = func(n *peg.Node) {
		if n.Kind == peg.KindLiteral && isWordLiteral(n.Text) {
			words = append(words, strings.ToUpper(n.Text))
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(words, " ")
}

// arrayBoundSize returns an ArrayBounds node's `[n]` size, or nil for `[]`
// and a bare `ARRAY`.
func (a *adapter) arrayBoundSize(n *peg.Node) Expr {
	if n.Kind == peg.KindRule && n.Name == "Expression" {
		return a.adaptTypeModifier(n)
	}
	for _, c := range n.Children {
		if e := a.arrayBoundSize(c); e != nil {
			return e
		}
	}
	return nil
}

// adaptTypeModifier adapts a type modifier or array size, which must be a
// plain number or string literal. A TypeName's Exprs are Walk-visible, but
// nothing downstream type-checks them, and a modifier that could hide a
// subquery or function call (`x::VARCHAR((SELECT ...))`) has no legitimate
// use, so it's rejected rather than passed through.
func (a *adapter) adaptTypeModifier(n *peg.Node) Expr {
	e := a.adaptExpression(n)
	lit, ok := e.(*Literal)
	if !ok || lit.Kind != LitNumber && lit.Kind != LitString {
		a.unsupported(n, "A type modifier other than a number or string literal")
	}
	return lit
}
