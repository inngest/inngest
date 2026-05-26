package defers

import (
	"context"
	"time"

	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

// emitDeferSpan writes the executor.defer span for a single defer. The span is
// the storage: resolvers reconstruct parent->child linkage from it rather than
// reading a side-channel row. The "s"-tag seed is sibling to the untagged
// event-ID seed in buildDeferEvents and the "r"-tag child run ID seed in
// util.DeterministicChildRunID — see that helper for the three-tag convention.
func emitDeferSpan(ctx context.Context, tp tracing.TracerProvider, md statev2.Metadata, now time.Time, d statev2.Defer) {
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "defers.SaveFromOp"},
		Metadata:  &md,
		Parent:    tracing.RunSpanRefFromMetadata(&md),
		StartTime: now,
		EndTime:   now,
		Seed:      []byte(md.ID.RunID.String() + d.HashedID + "s"),
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &d.HashedID),
			meta.Attr(meta.Attrs.DeferUserID, &d.UserlandID),
			meta.Attr(meta.Attrs.DeferFnSlug, &d.FnSlug),
			meta.Attr(meta.Attrs.DeferStatus, &d.ScheduleStatus),
		),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error emitting executor.defer span",
			"error", err,
			"run_id", md.ID.RunID,
			"hashed_id", d.HashedID,
		)
	}
}
