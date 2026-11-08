package parser

import (
	"fmt"
	"strings"
)

type Position struct {
	Offset int
	Line   int
	Column int
}

func (p Position) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Column) }

// Node is implemented by every concrete AST type. Children returns this
// node's immediate child AST nodes in a stable, deterministic order,
// omitting absent optional children (a nil *WhereClause contributes no
// entry, not a nil Node in the slice) so a caller never has to nil-check.
//
// Pos/End mirror go/ast.Node's Pos()/End(): together they give every node's
// full source span (not just a single point), which is what actually makes
// a node locatable in the original text — a caller (sub-project 2's
// table/column-reference rewriting, or a diagnostic pointing at an
// offending substring) can slice src[n.Pos().Offset:n.End().Offset] to
// recover exactly the text that produced any node, e.g. an *Ident's `a.b.c`
// or a *BaseTableRef's table name, not just where it starts.
type Node interface {
	fmt.Stringer
	Children() []Node
	Pos() Position
	End() Position
}

// Visitor mirrors go/ast.Visitor: Visit is called for n, then for each of
// n's children if Visit returned non-nil (using the returned Visitor,
// which is usually just v itself).
type Visitor interface {
	Visit(n Node) (w Visitor)
}

func Walk(v Visitor, n Node) {
	if n == nil {
		return
	}
	if v = v.Visit(n); v == nil {
		return
	}
	for _, c := range n.Children() {
		Walk(v, c)
	}
}

// DefaultVisitor visits every node with no side effects and never stops
// descending — a trivial base for callers that only need Walk's traversal
// order (e.g. via a closure-based Visitor wrapping it) rather than
// selective per-type behavior.
type DefaultVisitor struct{}

func (DefaultVisitor) Visit(Node) Visitor { return DefaultVisitor{} }

// Dump renders a full indented tree of n using each node's own String()
// for its own fields (no children) — this is what parser_test.go's golden
// fixtures assert against starting in Task 8.
func Dump(n Node) string {
	var b strings.Builder
	dump(&b, n, 0)
	return b.String()
}

func dump(b *strings.Builder, n Node, depth int) {
	if n == nil {
		return
	}
	b.WriteString(strings.Repeat("  ", depth))
	b.WriteString(n.String())
	b.WriteByte('\n')
	for _, c := range n.Children() {
		dump(b, c, depth+1)
	}
}
