package output

import (
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
)

func TextPause(item *state.Pause) error {
	if item == nil {
		fmt.Println("no item found")
		return nil
	}

	w := NewTextWriter()

	if err := w.WriteOrdered(OrderedData(
		"ID", item.ID,
		"WorkspaceID", item.WorkspaceID,
		"Identifier", OrderedData(
			"RunID", item.Identifier.RunID,
			"FunctionID", item.Identifier.FunctionID,
			"AccountID", item.Identifier.AccountID,
		),
		"Outgoing", item.Outgoing,
		"Incoming", item.Incoming,
		"StepName", item.StepName,
		"Opcode", item.Opcode,
		"Expires", fmt.Sprintf("%d (%s)", time.Time(item.Expires).UTC().UnixMilli(), time.Time(item.Expires).UTC().Format(time.RFC3339)),
		"Event", item.Event,
		"Expression", item.Expression,
		"InvokeCorrelationID", item.InvokeCorrelationID,
		"InvokeTargetFnID", item.InvokeTargetFnID,
		"SignalID", item.SignalID,
		"ReplaceSignalOnConflict", item.ReplaceSignalOnConflict,
		"OnTimeout", item.OnTimeout,
		"DataKey", item.DataKey,
		"Cancel", item.Cancel,
		"MaxAttempts", item.MaxAttempts,
		"GroupID", item.GroupID,
		"TriggeringEventID", item.TriggeringEventID,
		"Metadata", item.Metadata,
		"ParallelMode", item.ParallelMode,
		"CreatedAt", fmt.Sprintf("%d (%s)", item.CreatedAt.UTC().UnixMilli(), item.CreatedAt.UTC().Format(time.RFC3339)),
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}
