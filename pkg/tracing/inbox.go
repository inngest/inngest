package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

// A tick is a small span on the row of its parent: something that happened to
// the parent that is not a row of its own. An inbox draws one under its open
// row for each message that arrives, and one under a read's row for
// each message the read took. The trace API rolls ticks onto their parent's
// row, so no tick is drawn as a row.

// tickSeed names a tick. The same tick drawn again is the same span.
func tickSeed(kind string, runID fmt.Stringer, row string, id string) []byte {
	return fmt.Appendf(nil, "tick:%s:%s:%s:%s", kind, runID, row, id)
}

// InboxArrivalTick draws a message arriving through the sources of the op row.
// id is the message's step id and sources are the pauses it came through.
func InboxArrivalTick(ctx context.Context, tp TracerProvider, md *statev2.Metadata, row string, id string, sources []string, at time.Time) error {
	attrs := meta.NewAttrSet(
		meta.Attr(meta.Attrs.TickID, &id),
		meta.Attr(meta.Attrs.TickSources, &sources),
		meta.Attr(meta.Attrs.TickAt, &at),
	)
	AddMetadataTenantAttrs(attrs, md.ID)

	_, err := tp.CreateSpan(ctx, meta.SpanNameTick, &CreateSpanOptions{
		Seed:       tickSeed("arrival", md.ID.RunID, row, id),
		Debug:      &SpanDebugData{Location: "tracing.InboxArrivalTick"},
		Metadata:   md,
		Parent:     FinalizedStepSpanRefFromMetadataAndStepID(md, row),
		StartTime:  at,
		EndTime:    at,
		Attributes: attrs,
	})

	return err
}

// InboxRead draws a read of an inbox that was answered from state: its row,
// when it has none yet, and a tick under it for each message it took, stamped
// with when it was read. Reporting the same read again draws the same spans;
// the start of a span is the earliest time it was drawn, so that is the time
// the read first happened.
func InboxRead(ctx context.Context, tp TracerProvider, md *statev2.Metadata, op state.GeneratorOpcode, at time.Time) error {
	opts, err := op.SubscribeReadOpts()
	if err != nil {
		return fmt.Errorf("unable to parse subscribe read opts: %w", err)
	}

	readID := op.ID

	// A read whose op has an id of its own names the row it belongs to.
	if opts.Row != "" {
		op.ID = opts.Row
		op.Name = opts.Row
		op.DisplayName = &opts.Row
	}

	row := FinalizedStepSpanRefFromMetadataAndStepID(md, op.ID)

	if opts.Draw {
		attrs := GeneratorAttrs(&op)
		AddMetadataTenantAttrs(attrs, md.ID)
		meta.AddAttr(attrs, meta.Attrs.DynamicStatus, new(enums.StepStatusCompleted))
		AddTimingAttrs(attrs, at, at, at, at)

		if _, err := tp.CreateSpan(ctx, meta.SpanNameStep, &CreateSpanOptions{
			Seed:       FinalizedStepDynamicSeed(op.ID),
			Debug:      &SpanDebugData{Location: "tracing.InboxRead"},
			Metadata:   md,
			Parent:     RunSpanRefFromMetadata(md),
			StartTime:  at,
			EndTime:    at,
			Attributes: attrs,
		}); err != nil {
			return err
		}
	}

	for _, id := range opts.Messages {
		attrs := meta.NewAttrSet(
			meta.Attr(meta.Attrs.TickID, &id),
			meta.Attr(meta.Attrs.TickReadAt, &at),
		)
		AddMetadataTenantAttrs(attrs, md.ID)

		if _, err := tp.CreateSpan(ctx, meta.SpanNameTick, &CreateSpanOptions{
			Seed:       tickSeed("read", md.ID.RunID, readID, id),
			Debug:      &SpanDebugData{Location: "tracing.InboxRead"},
			Metadata:   md,
			Parent:     row,
			StartTime:  at,
			EndTime:    at,
			Attributes: attrs,
		}); err != nil {
			return err
		}
	}

	return nil
}
