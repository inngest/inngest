package defers

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

// emitDeferSpan writes the executor.defer span for a single defer. The span is
// the storage: resolvers reconstruct parent->child linkage from it rather than
// reading a side-channel row. The "s"-tag seed is sibling to the untagged
// event-ID seed in buildDeferEvents, the "a"-tag abort-span seed, and the
// "c"-tag child-run-id span seed in util.DeterministicChildRunIDDeferSpanSeed —
// see those helpers for the tag convention.
//
// The span is parented to the run root so the linkage query
// (GetSpansByRunIDsAndName) can find it by run ID. It is omitted from the
// user-facing trace tree in loaders/trace.go (it is not a real execution step),
// but stays persisted and queryable for linkage reconstruction.
func emitDeferSpan(ctx context.Context, tp tracing.TracerProvider, md statev2.Metadata, now time.Time, d statev2.Defer) {
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "defers.SaveFromOp"},
		Metadata:  &md,
		Parent:    tracing.RunSpanRefFromMetadata(&md),
		StartTime: now,
		EndTime:   now,
		Seed:      util.DeterministicDeferSpanSeed(md.ID.RunID, d.HashedID),
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

// emitAbortedDeferSpan writes a SECOND executor.defer span carrying the
// terminal defer.status = aborted attribute when a defer is aborted.
//
// We emit a distinct span rather than updating the original schedule span
// (e.g. via tracing.UpdateSpan) because the linkage query
// GetSpansByRunIDsAndName filters spans by name = "executor.defer"; it never
// reads the "EXTEND" fragments that UpdateSpan produces, so an extend-based
// status update would never surface to GetRunDefers. The abort span uses the
// "a"-tag seed (vs. the schedule span's "s" tag) so it gets a distinct dynamic
// span ID and lands as a separate row. GetRunDefers collapses the two rows by
// hashed ID, preferring the terminal status — see run_linkage.go.
func emitAbortedDeferSpan(ctx context.Context, tp tracing.TracerProvider, md statev2.Metadata, now time.Time, hashedID string) {
	status := enums.DeferStatusAborted
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "defers.AbortFromOp"},
		Metadata:  &md,
		Parent:    tracing.RunSpanRefFromMetadata(&md),
		StartTime: now,
		EndTime:   now,
		Seed:      util.DeterministicAbortedDeferSpanSeed(md.ID.RunID, hashedID),
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

// EmitChildRunIDSpan writes a THIRD executor.defer span carrying
// defer.child_run_id, linking the parent's defer (parentMD.ID.RunID + hashedID)
// to the child run that was scheduled for it.
//
// The child run ID isn't known when the schedule span is emitted, so it's
// recorded here at child-schedule time. As with the abort span, this is a
// distinct executor.defer span (not an UpdateSpan extension) so the linkage
// query GetSpansByRunIDsAndName — which filters by name and never sees EXTEND
// fragments — surfaces it. GetRunDefers collapses the rows by hashed ID and
// carries the child run ID onto the defer; GetRunDeferredFrom queries these
// spans by defer.child_run_id for the reverse link.
//
// parentMD must describe the PARENT run (its ID.RunID is the parent), so the
// span is attributed to the parent and discoverable by GetRunDefers(parent).
func EmitChildRunIDSpan(ctx context.Context, tp tracing.TracerProvider, parentMD statev2.Metadata, now time.Time, hashedID string, childRunID ulid.ULID) {
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "executor.Schedule.deferChildRunID"},
		Metadata:  &parentMD,
		Parent:    tracing.RunSpanRefFromMetadata(&parentMD),
		StartTime: now,
		EndTime:   now,
		Seed:      util.DeterministicChildRunIDDeferSpanSeed(parentMD.ID.RunID, hashedID),
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
