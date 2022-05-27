package state

import (
	"context"

	"github.com/inngest/inngest-cli/inngest"
)

// EdgeExpressionData returns data from the current state to evaluate the given
// edge's expressions.
func EdgeExpressionData(ctx context.Context, s State, e inngest.Edge) map[string]interface{} {
	// Add the outgoing edge's data as a "response" field for predefined edges.
	var response map[string]interface{}
	if e.Outgoing != "" && e.Outgoing != inngest.TriggerName {
		response, _ = s.ActionID(e.Outgoing)
	}
	data := map[string]interface{}{
		"event":    s.Event(),
		"steps":    s.Actions(),
		"response": response,
	}
	return data
}
