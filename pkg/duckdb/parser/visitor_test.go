package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeLeaf/fakeBranch exist only for this test, standing in for real AST
// node types before Task 7 introduces any.
type fakeLeaf struct {
	pos Position
	val string
}

func (f *fakeLeaf) Pos() Position    { return f.pos }
func (f *fakeLeaf) End() Position    { return f.pos } // real node types have a distinct span; this fake doesn't need one
func (f *fakeLeaf) Children() []Node { return nil }
func (f *fakeLeaf) String() string   { return fmt.Sprintf("Leaf(%s)", f.val) }

type fakeBranch struct {
	pos      Position
	name     string
	children []Node
}

func (f *fakeBranch) Pos() Position    { return f.pos }
func (f *fakeBranch) End() Position    { return f.pos }
func (f *fakeBranch) Children() []Node { return f.children }
func (f *fakeBranch) String() string   { return fmt.Sprintf("Branch(%s)", f.name) }

func buildFakeTree() Node {
	return &fakeBranch{name: "root", children: []Node{
		&fakeLeaf{val: "a"},
		&fakeBranch{name: "mid", children: []Node{
			&fakeLeaf{val: "b"},
		}},
		&fakeLeaf{val: "c"},
	}}
}

type countingVisitor struct{ count *int }

func (c countingVisitor) Visit(n Node) Visitor {
	*c.count++
	return c
}

func TestWalkVisitsEveryNode(t *testing.T) {
	var count int
	Walk(countingVisitor{count: &count}, buildFakeTree())
	require.Equal(t, 5, count) // root, a, mid, b, c
}

type stopAtVisitor struct{ name string }

func (s stopAtVisitor) Visit(n Node) Visitor {
	if b, ok := n.(*fakeBranch); ok && b.name == s.name {
		return nil // don't descend into this subtree
	}
	return s
}

// composedVisitor runs both visitors' Visit on every node reached by the
// gate visitor, used only so this test can both filter (via stopAtVisitor)
// and count (via countingVisitor) in one Walk.
type composedVisitor struct {
	gate  Visitor
	inner Visitor
}

func compose(gate, inner Visitor) Visitor { return composedVisitor{gate: gate, inner: inner} }

func (c composedVisitor) Visit(n Node) Visitor {
	// n itself is always counted; the gate only decides whether Walk
	// descends into n's children.
	descend := c.gate.Visit(n) != nil
	c.inner.Visit(n)
	if !descend {
		return nil
	}
	return c
}

func TestWalkCanStopDescending(t *testing.T) {
	var count int
	Walk(compose(stopAtVisitor{name: "mid"}, countingVisitor{count: &count}), buildFakeTree())
	require.Equal(t, 4, count) // root, a, mid, c (not b)
}

func TestDefaultVisitorVisitsEverything(t *testing.T) {
	var count int
	Walk(compose(DefaultVisitor{}, countingVisitor{count: &count}), buildFakeTree())
	require.Equal(t, 5, count)
}

func TestDump(t *testing.T) {
	got := Dump(buildFakeTree())
	want := "Branch(root)\n  Leaf(a)\n  Branch(mid)\n    Leaf(b)\n  Leaf(c)\n"
	require.Equal(t, want, got)
}

func TestPositionString(t *testing.T) {
	require.Equal(t, "3:7", Position{Line: 3, Column: 7}.String())
}
