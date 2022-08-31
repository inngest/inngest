package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/xhit/go-str2duration/v2"
)

func ParseWait(ctx context.Context, wait string, s state.State, outgoingID string) (time.Duration, error) {
	// Attempt to parse a basic duration.
	if dur, err := str2duration.ParseDuration(wait); err == nil {
		return dur, nil
	}

	data := state.EdgeExpressionData(ctx, s, outgoingID)

	// Attempt to parse an expression, eg. "date(event.data.from) - duration(1h)"
	out, _, err := expressions.Evaluate(ctx, wait, data)
	if err != nil {
		return 0, fmt.Errorf("Unable to parse wait as a duration or expression: %s", wait)
	}

	switch typ := out.(type) {
	case time.Time:
		return time.Until(typ), nil
	case time.Duration:
		return typ, nil
	case int:
		// Treat ints and floats as seconds.
		return time.Duration(typ) * time.Second, nil
	case float64:
		// Treat ints and floats as seconds.
		return time.Duration(typ) * time.Second, nil
	}

	return 0, fmt.Errorf("Unable to get duration from expression response: %v", out)
}
