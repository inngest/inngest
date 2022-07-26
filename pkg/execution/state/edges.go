package state

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/logger"
)

var (
	// DefaultEdgeEvaluator returns edges available to traverse based off of the
	// current state, using EdgeExpressionData to return data to use within
	// edge expressions.
	DefaultEdgeEvaluator = edgeEvaluator{
		datagen:   EdgeExpressionData,
		evaluator: expressions.Evaluate,
	}
)

// EdgeExpressionDataGen is a function which generates a map of data to be used within
// expressions, when comparing edges.
type EdgeExpressionDataGen func(ctx context.Context, s State, outgoingID string) map[string]interface{}

// Evaluator is a function which evaluates the current expression, returning whether it's true.
type Evaluator func(ctx context.Context, expression string, input map[string]interface{}) (bool, *time.Time, error)

// EdgeExpressionData returns data from the current state to evaluate the given
// edge's expressions.
func EdgeExpressionData(ctx context.Context, s State, outgoingID string) map[string]interface{} {
	// Add the outgoing edge's data as a "response" field for predefined edges.
	var response map[string]interface{}
	if outgoingID != "" && outgoingID != inngest.TriggerName {
		response, _ = s.ActionID(outgoingID)
	}
	data := map[string]interface{}{
		"event":    s.Event(),
		"steps":    s.Actions(),
		"response": response,
	}
	return data
}

// ChildIterator returns the children availale to process in a DAG based off of
// the given source node and given state.
type EdgeEvaluator interface {
	// AvailableChildren returns all children which are available to execute as a child
	// of the given step ID, based off of the given State.  This does the following:
	//
	// - Transforms the current workflow into a DAG
	// - Iterates through the edges available from the current step
	// - If each edge has an expression conditionally specifying whether it can be traversed,
	//   we evaluate the condition and disregard the edge if the condition is false.
	//
	// Note that any edges which have asynchronous event matching are returned;  it's not
	// children can be executed based off of the current workflow state.  Some children may not
	// be executed due to conditional expressions or asynchronous event conditions.  This is
	// the scheduler's job to manage.
	AvailableChildren(ctx context.Context, state State, stepID string) ([]inngest.Edge, error)
}

// NewEdgeEvaluator returns a new EdgeEvaluator, using the given function to return data for
// variables within the expression.
func NewEdgeEvaluator(eval Evaluator, datagen EdgeExpressionDataGen) EdgeEvaluator {
	// TODO (tonyhb): clean this up with options.
	if eval == nil {
		eval = expressions.Evaluate
	}
	if datagen == nil {
		datagen = EdgeExpressionData
	}

	return edgeEvaluator{
		evaluator: eval,
		datagen:   datagen,
	}
}

type edgeEvaluator struct {
	evaluator Evaluator
	datagen   EdgeExpressionDataGen
}

func (i edgeEvaluator) AvailableChildren(ctx context.Context, state State, stepID string) ([]inngest.Edge, error) {
	w := state.Workflow()

	if len(w.Steps) == 0 {
		return nil, fmt.Errorf("empty workflow returned from state")
	}

	g, err := inngest.NewGraph(w)
	if err != nil {
		return nil, err
	}

	// Handle the outgoing edges from this particular node.
	edges := g.From(stepID)
	if len(edges) == 0 {
		logger.From(ctx).Trace().Msg("no child edges available")
		return nil, nil
	}

	future := []inngest.Edge{}
	for _, edge := range edges {
		logger.From(ctx).Trace().Interface("edge", edge.WorkflowEdge).Msg("evaluating child edge")

		ok, err := i.canTraverseEdge(ctx, state, edge)
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		// We can traverse this edge.  Schedule a new execution from this node.
		// Scheduling executions needs to be done regardless of whether
		// the context has cancelled.
		future = append(future, edge.WorkflowEdge)
	}

	// Sort the edges which are returned according to incoming string order.
	sort.Slice(future, func(i, j int) bool {
		return future[i].Incoming < future[j].Incoming
	})

	return future, nil
}

// canTraverseEdge determines whether the edge can be traversed immediately.  Edges come
// in three flavours:  plain graph edges which link functions in a DAG;  edges with
// expressions which are traversed conditionally based off of workflow state;  and
// asynchronous edges which wait for an event mathing a condition to be traversed (at some
// point in the future, with a TTL).
func (i edgeEvaluator) canTraverseEdge(ctx context.Context, s State, edge inngest.GraphEdge) (bool, error) {
	l := logger.From(ctx).With().Interface("edge", edge.WorkflowEdge).Logger()

	if edge.Outgoing.ID() != inngest.TriggerName && !s.ActionComplete(edge.Outgoing.ID()) {
		l.Trace().Bool("traverse", false).Msg("edge incomplete")
		return false, nil
	}

	exprdata := i.datagen(ctx, s, edge.WorkflowEdge.Outgoing)

	if edge.WorkflowEdge.Metadata != nil && edge.WorkflowEdge.Metadata.If != "" {
		l.Trace().Str("expression", edge.WorkflowEdge.Metadata.If).Msg("evaluating edge expression")

		ok, _, err := i.evaluator(ctx, edge.WorkflowEdge.Metadata.If, exprdata)
		if err != nil || !ok {
			l.Trace().
				Bool("traverse", false).
				Str("expression", edge.WorkflowEdge.Metadata.If).
				Err(err).
				Msg("expression false")
			return ok, err
		}
	}

	l.Trace().Bool("traverse", true).Msg("edge traversable")
	return true, nil
}
