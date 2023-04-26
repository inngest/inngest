package inngest

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// Graph represents the workflow as a graph for BFS traversal.
type Graph struct {
	dag.AcyclicGraph

	workflow Workflow
	source   Vertex
}

func NewGraph(w Workflow) (Graph, error) {
	var g dag.AcyclicGraph

	// Create a new root vertex, representing the trigger: the root of the workflow.
	source := Vertex{Root: true}

	// Iterate through all actions, creating a vertex for the action in a map
	// so that adding the edges is easy.
	vertices := map[string]Vertex{
		"$root": source,
	}
	for _, s := range w.Steps {
		step := s
		v := Vertex{Step: &step}
		vertices[step.ID] = v
		g.Add(v)
	}

	// Iterate through edges and add them to the graph.  Adding an edge adds
	// both vertices to the graph if they are not yet present, so this adds
	// all of our actions for us.
	for _, e := range w.Edges {
		edge := GraphEdge{
			Edge:     e,
			Outgoing: vertices[e.Outgoing],
			Incoming: vertices[e.Incoming],
		}
		g.Connect(edge)
	}

	return Graph{
		AcyclicGraph: g,
		workflow:     w,
		source:       source,
	}, nil
}

func (g Graph) Workflow() Workflow {
	return g.workflow
}

func (g Graph) From(id string) []GraphEdge {
	ifaces := g.EdgesFrom(LookupVertex{ID: id})
	edges := make([]GraphEdge, len(ifaces))
	for n, i := range ifaces {
		edges[n] = i.(GraphEdge)
	}
	return edges
}

type LookupVertex struct {
	ID string
}

func (l LookupVertex) Hashcode() interface{} {
	return l.ID
}

// Vertex represents an action or the trigger within our workflow graph
type Vertex struct {
	Root bool
	Step *WorkflowStep
}

func (g Vertex) Hashcode() interface{} {
	return g.ID()
}

func (g Vertex) ID() string {
	if g.Step == nil {
		return TriggerName
	}
	return g.Step.ID
}

// Edge inherits functionality from simple.Edge and includes our workflow edge
// connecting two actions.
type GraphEdge struct {
	Edge Edge

	Outgoing Vertex
	Incoming Vertex
}

func (e GraphEdge) Source() dag.Vertex {
	return e.Outgoing
}

func (e GraphEdge) Target() dag.Vertex {
	return e.Incoming
}

func (e GraphEdge) Hashcode() interface{} {
	return fmt.Sprintf("%s-%s", e.Outgoing.ID(), e.Incoming.ID())
}
