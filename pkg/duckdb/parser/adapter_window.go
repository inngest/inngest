// pkg/duckdb/parser/adapter_window.go
package parser

import "github.com/inngest/inngest/pkg/duckdb/parser/peg"

func (a *adapter) adaptWindowClause(n *peg.Node) []*WindowDef {
	// WindowClause <- 'WINDOW' List(WindowDefinition)
	listSeq := body(body(n).Children[1])
	defs := []*WindowDef{a.adaptNamedWindowDefinition(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		defs = append(defs, a.adaptNamedWindowDefinition(body(tail).Children[1]))
	}
	return defs
}

func (a *adapter) adaptNamedWindowDefinition(n *peg.Node) *WindowDef {
	// WindowDefinition <- Identifier 'AS' WindowFrameDefinition
	seq := body(n)
	return &WindowDef{baseExpr: a.at(n), Name: literalText(seq.Children[0]), Spec: a.adaptWindowFrameDefinition(seq.Children[2])}
}
