package state

import (
	"context"

	"github.com/inngest/inngest-cli/inngest"
)

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
