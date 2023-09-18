package executor

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/oklog/ulid/v2"
	"github.com/xhit/go-str2duration/v2"
)

var metadataCtxKey = metadataCtxType{}

type metadataCtxType struct{}

// WithContextMetadata stores the given function run metadata within the given context.
func WithContextMetadata(ctx context.Context, m state.Metadata) context.Context {
	return context.WithValue(ctx, metadataCtxKey, &m)
}

// GetContextMetadata returns function run metadata stored in context or nil if not present.
func GetContextMetadata(ctx context.Context) *state.Metadata {
	val, _ := ctx.Value(metadataCtxKey).(*state.Metadata)
	return val
}

// GetFunctionMetadata returns a function run's metadata.  This attempts to load metadata
// from context first, to reduce state store reads, falling back to the state.Manager's Metadata()
// method if the metadata does not exist in context.
func GetFunctionRunMetadata(ctx context.Context, sm state.Manager, runID ulid.ULID) (*state.Metadata, error) {
	if val := GetContextMetadata(ctx); val != nil {
		return val, nil
	}
	return sm.Metadata(ctx, runID)
}

// ParseWait parses the given wait string, using data from the function run's state/output within
// interpolation, treating the wait as an expression.
//
// NOTE: This is deprecated and unused.  It should be removed in a followup PR.
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

func sortOps(opcodes []*state.GeneratorOpcode) {
	sort.SliceStable(opcodes, func(i, j int) bool {
		// Ensure that we process waitForEvents first, as these are highest priority:
		// it ensures that wait triggers are saved as soon as possible.
		if opcodes[i].Op == enums.OpcodeWaitForEvent {
			return true
		}
		return opcodes[i].Op < opcodes[j].Op
	})
}
