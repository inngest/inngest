package checkpoint

import (
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

// stepDynamicSeed returns the deterministic seed used to derive an
// executor.step row's dynamic_span_id.
func stepDynamicSeed(op state.GeneratorOpcode, attempt int) []byte {
	return fmt.Appendf(nil, "%s:%d", op.ID, attempt)
}

const PairedTrailingKey = "_paired_trailing"

// isPairedTrailingStepRun reports whether this StepRun is the trailing arm
// of a paired StepPlanned + StepRun emitted by an SDK in checkpointing mode
// for an in-progress step.
func isPairedTrailingStepRun(op state.GeneratorOpcode) bool {
	if op.Op != enums.OpcodeStepRun {
		return false
	}
	opts, ok := op.Opts.(map[string]any)
	if !ok {
		return false
	}
	v, ok := opts[PairedTrailingKey].(bool)
	return ok && v
}

func stepRunAttrs(attrs *meta.SerializableAttrs, op state.GeneratorOpcode, runID ulid.ULID) *meta.SerializableAttrs {
	attrs = attrs.Merge(
		meta.NewAttrSet(
			meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
			meta.Attr(meta.Attrs.DynamicStatus, inngestgo.Ptr(enums.StepStatusCompleted)),
			meta.Attr(meta.Attrs.IsCheckpoint, inngestgo.Ptr(true)),
			meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
			meta.Attr(meta.Attrs.RunID, &runID),
		),
	)

	if isPairedTrailingStepRun(op) {
		attrs = attrs.Merge(meta.NewAttrSet(
			// Omit QueuedAt and StartedAt, as the trailing edge has different values
			// and would override the leading edge's values.
			//
			// Mark this span, so that we don't add QueuedAt and StartedAt later.
			meta.Attr(meta.Attrs.IsPairedTrailing, inngestgo.Ptr(true)),
		))
	} else {
		attrs = attrs.Merge(meta.NewAttrSet(
			meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
			meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
		))
	}
	return attrs
}

func stepPlannedAttrs(attrs *meta.SerializableAttrs, op state.GeneratorOpcode, runID ulid.ULID) *meta.SerializableAttrs {
	return attrs.Merge(
		meta.NewAttrSet(
			meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
			meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
			meta.Attr(meta.Attrs.DynamicStatus, inngestgo.Ptr(enums.StepStatusRunning)),
		),
	)
}

func stepErrorAttrs(attrs *meta.SerializableAttrs, op state.GeneratorOpcode, runID ulid.ULID, status enums.StepStatus) *meta.SerializableAttrs {
	return attrs.Merge(
		meta.NewAttrSet(
			meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
			meta.Attr(meta.Attrs.RunID, &runID),
			meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
			meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
			meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
			meta.Attr(meta.Attrs.DynamicStatus, &status),
			meta.Attr(meta.Attrs.IsCheckpoint, inngestgo.Ptr(true)),
		),
	)
}
