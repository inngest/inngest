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

// OpcodeGroups groups opcodes by processing priority. The priority group runs
// first so waitForEvent triggers are saved immediately (capturing expression
// errors early) and lazy ops like DeferAdd/DeferAbort — which the SDK
// piggybacks onto other ops — drain before RunComplete can finalize and delete
// state.
type OpcodeGroups struct {
	// PriorityGroup is a group of opcodes that should be processed first.
	PriorityGroup OpcodeGroup

	// OtherGroup is a group of opcodes that should be processed after the
	// priority group.
	OtherGroup OpcodeGroup
}

// nonLazyOpCount returns the count used to gate parallel-step behavior — see
// enums.OpcodeIsLazy. Nil entries are skipped to mirror handleGeneratorGroup.
func nonLazyOpCount(opcodes []*state.GeneratorOpcode) int {
	n := 0
	for _, op := range opcodes {
		if op == nil {
			continue
		}
		if !enums.OpcodeIsLazy(op.Op) {
			n++
		}
	}
	return n
}

// opGroups groups opcodes by their type.
func opGroups(opcodes []*state.GeneratorOpcode) OpcodeGroups {
	shouldStartHistoryGroup := nonLazyOpCount(opcodes) > 1

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
		if op == nil {
			continue
		}
		if enums.OpcodeIsPriority(op.Op) {
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

// NonLazyIDs returns the step IDs of "non-lazy" op. Distinguishing between lazy
// and non-lazy ops is important because lazy ops do not have their own queue
// items. This lack of queue items is critical for things like tracking pending
// steps
func (g OpcodeGroups) NonLazyIDs() []string {
	ids := []string{}
	for _, group := range g.All() {
		for _, op := range group.Opcodes {
			if enums.OpcodeIsLazy(op.Op) {
				continue
			}
			ids = append(ids, op.ID)
		}
	}
	return ids
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

	logger.StdlibLogger(ctx).Debug("function finished event", "event", evt)

	return evt
}
