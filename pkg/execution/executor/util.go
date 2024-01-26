package executor

import (
	"context"
	"crypto/rand"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
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

// opGroups groups opcodes by their type, ensuring we run `waitForEvent` opcodes
// first. This is used to ensure that we save wait triggers as soon as possible,
// as well as capturing expression errors early.
func opGroups(opcodes []*state.GeneratorOpcode) [][]*state.GeneratorOpcode {
	var waitForEventGroup []*state.GeneratorOpcode
	var otherGroup []*state.GeneratorOpcode

	for _, op := range opcodes {
		if op.Op == enums.OpcodeWaitForEvent {
			waitForEventGroup = append(waitForEventGroup, op)
		} else {
			otherGroup = append(otherGroup, op)
		}
	}

	// filter out any empty groups
	groups := [][]*state.GeneratorOpcode{}
	if len(waitForEventGroup) > 0 {
		groups = append(groups, waitForEventGroup)
	}
	if len(otherGroup) > 0 {
		groups = append(groups, otherGroup)
	}

	return groups
}

func CreateInvokeNotFoundEvent(ctx context.Context, opts execution.InvokeNotFoundHandlerOpts) event.Event {
	now := time.Now()
	data := map[string]interface{}{
		"function_id": opts.FunctionID,
		"run_id":      opts.RunID,
	}

	origEvt := opts.OriginalEvent.GetEvent().Map()
	if dataMap, ok := origEvt["data"].(map[string]interface{}); ok {
		if inngestObj, ok := dataMap[consts.InngestEventDataPrefix].(map[string]interface{}); ok {
			if dataValue, ok := inngestObj[consts.InvokeCorrelationId].(string); ok {
				data[consts.InvokeCorrelationId] = dataValue
			}
		}
	}

	if opts.Err != nil {
		data["error"] = opts.Err
	} else {
		data["result"] = opts.Result
	}

	evt := event.Event{
		ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
		Name:      event.FnFinishedName,
		Timestamp: now.UnixMilli(),
		Data:      data,
	}

	logger.From(ctx).Debug().Interface("event", evt).Msg("function finished event")

	return evt
}
