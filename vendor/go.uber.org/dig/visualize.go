// Copyright (c) 2021 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dig

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"go.uber.org/dig/internal/dot"
)

// A VisualizeOption modifies the default behavior of Visualize.
type VisualizeOption interface {
	applyVisualizeOption(*visualizeOptions)
}

type visualizeOptions struct {
	VisualizeError error
}

// VisualizeError includes a visualization of the given error in the output of
// Visualize if an error was returned by Invoke or Provide.
//
//	if err := c.Provide(...); err != nil {
//	  dig.Visualize(c, w, dig.VisualizeError(err))
//	}
//
// This option has no effect if the error was nil or if it didn't contain any
// information to visualize.
func VisualizeError(err error) VisualizeOption {
	return visualizeErrorOption{err}
}

type visualizeErrorOption struct{ err error }

func (o visualizeErrorOption) String() string {
	return fmt.Sprintf("VisualizeError(%v)", o.err)
}

func (o visualizeErrorOption) applyVisualizeOption(opt *visualizeOptions) {
	opt.VisualizeError = o.err
}

func updateGraph(dg *dot.Graph, err error) error {
	var errs []errVisualizer
	// Unwrap error to find the root cause.
	for {
		if ev, ok := err.(errVisualizer); ok {
			errs = append(errs, ev)
		}
		e := errors.Unwrap(err)
		if e == nil {
			break
		}
		err = e
	}

	// If there are no errVisualizers included, we do not modify the graph.
	if len(errs) == 0 {
		return nil
	}

	// We iterate in reverse because the last element is the root cause.
	for i := len(errs) - 1; i >= 0; i-- {
		errs[i].updateGraph(dg)
	}

	// Remove non-error entries from the graph for readability.
	dg.PruneSuccess()

	return nil
}

// Visualize parses the graph in Container c into DOT format and writes it to
// io.Writer w.
func Visualize(c *Container, w io.Writer, opts ...VisualizeOption) error {
	dg := c.createGraph()

	var options visualizeOptions
	for _, o := range opts {
		o.applyVisualizeOption(&options)
	}

	if options.VisualizeError != nil {
		if err := updateGraph(dg, options.VisualizeError); err != nil {
			return err
		}
	}

	visualizeGraph(w, dg)
	return nil
}

func visualizeGraph(w io.Writer, dg *dot.Graph) {
	w.Write([]byte("digraph {\n\trankdir=RL;\n\tgraph [compound=true];\n"))
	for _, g := range dg.Groups {
		visualizeGroup(w, g)
	}
	for idx, c := range dg.Ctors {
		visualizeCtor(w, idx, c)
	}
	for _, f := range dg.Failed.TransitiveFailures {
		fmt.Fprintf(w, "\t%s [color=orange];\n", strconv.Quote(f.String()))
	}
	for _, f := range dg.Failed.RootCauses {
		fmt.Fprintf(w, "\t%s [color=red];\n", strconv.Quote(f.String()))
	}
	w.Write([]byte("}"))
}

func visualizeGroup(w io.Writer, g *dot.Group) {
	fmt.Fprintf(w, "\t%s [%s];\n", strconv.Quote(g.String()), g.Attributes())
	for _, r := range g.Results {
		fmt.Fprintf(w, "\t%s -> %s;\n", strconv.Quote(g.String()), strconv.Quote(r.String()))
	}
}

func visualizeCtor(w io.Writer, index int, c *dot.Ctor) {
	fmt.Fprintf(w, "\tsubgraph cluster_%d {\n", index)
	if c.Package != "" {
		fmt.Fprintf(w, "\t\tlabel = %s;\n", strconv.Quote(c.Package))
	}
	fmt.Fprintf(w, "\t\tconstructor_%d [shape=plaintext label=%s];\n", index, strconv.Quote(c.Name))

	if c.ErrorType != 0 {
		fmt.Fprintf(w, "\t\tcolor=%s;\n", c.ErrorType.Color())
	}
	for _, r := range c.Results {
		fmt.Fprintf(w, "\t\t%s [%s];\n", strconv.Quote(r.String()), r.Attributes())
	}
	fmt.Fprintf(w, "\t}\n")
	for _, p := range c.Params {
		var optionalStyle string
		if p.Optional {
			optionalStyle = " style=dashed"
		}

		fmt.Fprintf(w, "\tconstructor_%d -> %s [ltail=cluster_%d%s];\n", index, strconv.Quote(p.String()), index, optionalStyle)
	}
	for _, p := range c.GroupParams {
		fmt.Fprintf(w, "\tconstructor_%d -> %s [ltail=cluster_%d];\n", index, strconv.Quote(p.String()), index)
	}
}

// CanVisualizeError returns true if the error is an errVisualizer.
func CanVisualizeError(err error) bool {
	for {
		if _, ok := err.(errVisualizer); ok {
			return true
		}
		e := errors.Unwrap(err)
		if e == nil {
			break
		}
		err = e
	}

	return false
}

func (c *Container) createGraph() *dot.Graph {
	return c.scope.createGraph()
}

func (s *Scope) createGraph() *dot.Graph {
	dg := dot.NewGraph()

	s.addNodes(dg)

	return dg
}

func (s *Scope) addNodes(dg *dot.Graph) {
	for _, n := range s.nodes {
		dg.AddCtor(newDotCtor(n), n.paramList.DotParam(), n.resultList.DotResult())
	}

	for _, cs := range s.childScopes {
		cs.addNodes(dg)
	}
}

func newDotCtor(n *constructorNode) *dot.Ctor {
	return &dot.Ctor{
		ID:      n.id,
		Name:    n.location.Name,
		Package: n.location.Package,
		File:    n.location.File,
		Line:    n.location.Line,
	}
}
