package executor

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

// OpcodeGroup is a group of opcodes that can be processed in parallel.
type OpcodeGroup struct {
	// Opcodes is the list of opcodes in the group.
	Opcodes []*state.GeneratorOpcode
	// ShouldStartHistoryGroup indicates whether each item in the group should
	// start a new history group. This is true if the overall list of opcodes
	// received from an SDK Call Request contains more than one opcode.
	ShouldStartHistoryGroup bool
}

// OpcodeGroups are groups opcodes by their type, helping to run `waitForEvent`
// opcodes first. This is used to ensure that we save wait triggers as soon as
// possible, as well as capturing expression errors early.
type OpcodeGroups struct {
	// PriorityGroup is a group of opcodes that should be processed first.
	PriorityGroup OpcodeGroup

	// OtherGroup is a group of opcodes that should be processed after the
	// priority group.
	OtherGroup OpcodeGroup
}

// opGroups groups opcodes by their type.
func opGroups(opcodes []*state.GeneratorOpcode) OpcodeGroups {
	shouldStartHistoryGroup := len(opcodes) > 1

	groups := OpcodeGroups{
		PriorityGroup: OpcodeGroup{
			Opcodes:                 []*state.GeneratorOpcode{},
			ShouldStartHistoryGroup: shouldStartHistoryGroup,
		},
		OtherGroup: OpcodeGroup{
			Opcodes:                 []*state.GeneratorOpcode{},
			ShouldStartHistoryGroup: shouldStartHistoryGroup,
		},
	}

	for _, op := range opcodes {
		if op.Op == enums.OpcodeWaitForEvent {
			groups.PriorityGroup.Opcodes = append(groups.PriorityGroup.Opcodes, op)
		} else {
			groups.OtherGroup.Opcodes = append(groups.OtherGroup.Opcodes, op)
		}
	}

	return groups
}

// All returns a list of all groups in the order they should be processed.
func (g OpcodeGroups) All() []OpcodeGroup {
	return []OpcodeGroup{g.PriorityGroup, g.OtherGroup}
}

func CreateInvokeFailedEvent(ctx context.Context, opts execution.InvokeFailHandlerOpts) event.Event {
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
