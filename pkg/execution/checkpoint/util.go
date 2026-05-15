package checkpoint

import (
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

func stepRunAttrs(attrs *meta.SerializableAttrs, op state.GeneratorOpcode, runID ulid.ULID) *meta.SerializableAttrs {
	return attrs.Merge(
		meta.NewAttrSet(
			meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
			meta.Attr(meta.Attrs.RunID, &runID),
			meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
			meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
			meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
			meta.Attr(meta.Attrs.DynamicStatus, inngestgo.Ptr(enums.StepStatusCompleted)),
			meta.Attr(meta.Attrs.IsCheckpoint, inngestgo.Ptr(true)),
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
