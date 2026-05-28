package defers

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

// emitDeferSpan writes the executor.defer schedule span. Resolvers
// reconstruct parent->child linkage from these spans rather than a
// side-channel row; the span is omitted from the user-facing trace tree in
// loaders/trace.go but stays queryable for linkage.
func emitDeferSpan(ctx context.Context, tp tracing.TracerProvider, md statev2.Metadata, now time.Time, d statev2.Defer) {
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "defers.SaveFromOp"},
		Metadata:  &md,
		Parent:    tracing.RunSpanRefFromMetadata(&md),
		StartTime: now,
		EndTime:   now,
		Seed:      SpanSeed(md.ID.RunID, d.HashedID, SpanSchedule),
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &d.HashedID),
			meta.Attr(meta.Attrs.DeferUserlandID, &d.UserlandID),
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

// emitAbortedDeferSpan writes the executor.defer span carrying the terminal
// aborted status. GetRunDefers collapses it with the schedule span by hashed
// ID and prefers this terminal status.
func emitAbortedDeferSpan(ctx context.Context, tp tracing.TracerProvider, md statev2.Metadata, now time.Time, hashedID string) {
	status := enums.DeferStatusAborted
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "defers.AbortFromOp"},
		Metadata:  &md,
		Parent:    tracing.RunSpanRefFromMetadata(&md),
		StartTime: now,
		EndTime:   now,
		Seed:      SpanSeed(md.ID.RunID, hashedID, SpanAborted),
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &hashedID),
			meta.Attr(meta.Attrs.DeferStatus, &status),
		),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error emitting aborted executor.defer span",
			"error", err,
			"run_id", md.ID.RunID,
			"hashed_id", hashedID,
		)
	}
}

// EmitChildRunIDSpan writes the executor.defer span linking a parent defer
// to its scheduled child run. The child run ID isn't known when the schedule
// span is emitted, so it's recorded here at child-schedule time. parentMD
// must describe the parent run so GetRunDefers(parent) finds it.
func EmitChildRunIDSpan(ctx context.Context, tp tracing.TracerProvider, parentMD statev2.Metadata, now time.Time, hashedID string, childRunID ulid.ULID) {
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "executor.Schedule.deferChildRunID"},
		Metadata:  &parentMD,
		Parent:    tracing.RunSpanRefFromMetadata(&parentMD),
		StartTime: now,
		EndTime:   now,
		Seed:      SpanSeed(parentMD.ID.RunID, hashedID, SpanChildRunID),
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &hashedID),
			meta.Attr(meta.Attrs.DeferChildRunID, &childRunID),
		),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error emitting child-run-id executor.defer span",
			"error", err,
			"run_id", parentMD.ID.RunID,
			"hashed_id", hashedID,
			"child_run_id", childRunID,
		)
	}
}
