package state

import (
	"context"
)

// ExpressionData returns data from the current state to evaluate function expressions
func ExpressionData(ctx context.Context, s State) map[string]interface{} {
	data := map[string]interface{}{
		"event": s.Event(),
	}
	return data
}
