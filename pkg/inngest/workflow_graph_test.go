package inngest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGraph_create(t *testing.T) {
	w := Workflow{
		Steps: []Step{
			{
				ID:   "first",
				Name: "My first step!",
			},
			{
				ID:   "#2",
				Name: "Second",
			},
			{
				ID:   "parallel #2",
				Name: "Parallel #2",
			},
		},
		Edges: []Edge{
			{
				Outgoing: TriggerName,
				Incoming: "first",
			},
			{
				Outgoing: "first",
				Incoming: "#2",
			},
			{
				Outgoing: "first",
				Incoming: "parallel #2",
			},
		},
	}

	_, err := NewGraph(w)
	require.NoError(t, err)
}

func TestGraph_lookup(t *testing.T) {
	w := Workflow{
		Steps: []Step{
			{
				ID:   "first",
				Name: "My first step!",
			},
			{
				ID:   "#2",
				Name: "Second",
			},
			{
				ID:   "parallel #2",
				Name: "Parallel #2",
			},
		},
		Edges: []Edge{
			{
				Outgoing: TriggerName,
				Incoming: "first",
			},
			{
				Outgoing: "first",
				Incoming: "#2",
			},
			{
				Outgoing: "first",
				Incoming: "parallel #2",
			},
		},
	}

	g, err := NewGraph(w)
	require.NoError(t, err)

	// Nodes from trigger
	edges := g.EdgesFrom(LookupVertex{TriggerName})
	require.Equal(t, 1, len(edges))
	require.Equal(t, "first", edges[0].Target().(Vertex).Step.ID)

	// a helper func.
	from := g.From(TriggerName)
	require.Equal(t, 1, len(from))
	require.Equal(t, "first", from[0].Incoming.Step.ID)

	// Nodes from first vertex.
	edges = g.EdgesFrom(LookupVertex{"first"})
	require.Equal(t, 2, len(edges))
	ids := []string{
		edges[0].Target().(Vertex).Step.ID,
		edges[1].Target().(Vertex).Step.ID,
	}
	require.ElementsMatch(t, []string{"#2", "parallel #2"}, ids)
}
