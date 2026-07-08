package defers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
)

const pkgName = "defers"

// SaveFromOp persists a DeferAdd opcode. Returns nil for accepted and
// soft-rejected outcomes; non-nil only for infra errors.
//
// On accepted: emits an executor.defer span with status=AfterRun so the run's
// deferred-runs list is visible in the UI before the parent finalizes.
// On soft-rejected when the rejection is persisted: emits the span with
// status=Rejected. Per_run_count rejections don't emit (nothing was persisted;
// the SDK will retransmit each step until finalize so a span per retransmit
// would be noise).
func SaveFromOp(
	ctx context.Context,
	rs statev2.RunService,
	tp tracing.TracerProvider,
	log logger.Logger,
	md *statev2.Metadata,
	op state.GeneratorOpcode,
) error {
	var (
		// Why the defer was rejected.
		rejectReason string

		// Whether the rejected defer was persisted in the StateStore. Some
		// rejections don't persist (e.g. per_run_count when there are too many
		// defers), so this isn't redundant with rejectReason.
		//
		// Gates defer span creation to prevent abuse, where someone inserts an
		// unbounded number of defer spans.
		rejectionPersisted bool
	)

	var userlandID string
	if op.Userland != nil {
		userlandID = op.Userland.ID
	}

	opts, err := op.DeferAddOpts()
	if err != nil {
		if errors.Is(err, state.ErrDeferInputTooLarge) {
			rejectReason = "per_defer_size"
		} else if errors.Is(err, state.ErrDeferInputInvalid) {
			rejectReason = "invalid_input"
		} else {
			rejectReason = "invalid_opts"
		}
		if opts != nil {
			if err := rs.SaveDefer(ctx, md.ID, statev2.Defer{
				FnSlug:         opts.FnSlug,
				HashedID:       op.ID,
				ScheduleStatus: enums.DeferStatusRejected,
			}); err != nil {
				log.Warn("failed to persist rejected defer; SDK retransmits will not dedupe",
					"step_id", util.SanitizeLogField(op.ID),
					"run_id", md.ID.RunID,
					"error", err,
				)
			} else {
				rejectionPersisted = true
			}
		}
	}

	if rejectReason == "" {
		saveErr := rs.SaveDefer(ctx, md.ID, statev2.Defer{
			FnSlug:         opts.FnSlug,
			HashedID:       op.ID,
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          opts.Input,
		})
		switch {
		case errors.Is(saveErr, statev2.ErrDeferLimitExceeded):
			// Count cap rejects without persisting anything. SDK retransmits absorbed.
			rejectReason = "per_run_count"
		case errors.Is(saveErr, statev2.ErrDeferInputAggregateExceeded):
			// saveDefer.lua already persisted the Rejected record.
			rejectReason = "aggregate_size"
			rejectionPersisted = true
		case saveErr != nil:
			return fmt.Errorf("error saving defer: %w", saveErr)
		}
	}

	if rejectReason != "" {
		log.Warn("defer soft-rejected; run will continue without this deferred run",
			"step_id", util.SanitizeLogField(op.ID),
			"reason", rejectReason,
			"run_id", md.ID.RunID,
		)
		metrics.IncrDefersRejectedCounter(ctx, rejectReason, metrics.CounterOpt{PkgName: pkgName})
	}

	if rejectReason == "" {
		// Create span for the defer that'll schedule after the parent run ends.
		createDeferSpan(ctx, tp, log, md, statev2.Defer{
			FnSlug:         opts.FnSlug,
			HashedID:       op.ID,
			ScheduleStatus: enums.DeferStatusAfterRun,
		}, userlandID)
	} else if rejectionPersisted {
		fnSlug := ""
		if opts != nil {
			fnSlug = opts.FnSlug
		}

		// Create span for the rejected defer.
		createDeferSpan(ctx, tp, log, md, statev2.Defer{
			FnSlug:         fnSlug,
			HashedID:       op.ID,
			ScheduleStatus: enums.DeferStatusRejected,
		}, userlandID)
	}

	return nil
}

// AbortFromOp flips the target defer's status to Aborted. Unknown-target
// aborts are benign — the SDK contract requires shipping aborts even for
// locally-cancelled DeferAdds that never landed — so ErrDeferNotFound returns
// nil with a debug log. Other errors are surfaced.
//
// On success: updates the existing executor.defer span to status=Aborted via
// UpdateSpan. The span identity is reconstructed from (parentRunID, hashedID),
// not stored anywhere.
func AbortFromOp(
	ctx context.Context,
	rs statev2.RunService,
	tp tracing.TracerProvider,
	log logger.Logger,
	md *statev2.Metadata,
	op state.GeneratorOpcode,
) error {
	opts, err := op.DeferAbortOpts()
	if err != nil {
		log.Error("error parsing DeferAbort opts", "error", err)
		return fmt.Errorf("error parsing DeferAbort opts: %w", err)
	}

	if err := rs.SetDeferStatus(ctx, md.ID, opts.TargetHashedID, enums.DeferStatusAborted); err != nil {
		if errors.Is(err, state.ErrDeferNotFound) {
			log.Debug("abort for unknown defer", "hashed_id", opts.TargetHashedID)
			return nil
		}
		log.Error("error aborting defer", "error", err)
		return fmt.Errorf("error aborting defer: %w", err)
	}

	updateDeferSpanStatus(ctx, tp, log, md, opts.TargetHashedID, enums.DeferStatusAborted)
	return nil
}

// createDeferSpan writes the executor.defer span. userlandID is passed
// separately because it is not part of the persisted defer record: it's
// only used here to display the user-typed defer ID in the trace UI.
func createDeferSpan(
	ctx context.Context,
	tp tracing.TracerProvider,
	log logger.Logger,
	md *statev2.Metadata,
	d statev2.Defer,
	userlandID string,
) {
	if tp == nil {
		return
	}
	now := time.Now()
	_, err := tp.CreateSpan(ctx, meta.SpanNameDefer, &tracing.CreateSpanOptions{
		Debug:     &tracing.SpanDebugData{Location: "defers.emitDeferSpan"},
		Metadata:  md,
		Parent:    tracing.RunSpanRefFromMetadata(md),
		StartTime: now,
		EndTime:   now,
		Seed:      tracing.DeferSpanSeed(md.ID.RunID, d.HashedID),
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferHashedID, &d.HashedID),
			meta.Attr(meta.Attrs.DeferUserlandID, &userlandID),
			meta.Attr(meta.Attrs.DeferFnSlug, &d.FnSlug),
			meta.Attr(meta.Attrs.DeferStatus, &d.ScheduleStatus),
		),
	})
	if err != nil {
		log.Error(
			"error emitting executor.defer span",
			"error", err,
			"hashed_id", util.SanitizeLogField(d.HashedID),
			"run_id", md.ID.RunID,
		)
	}
}

func updateDeferSpanStatus(
	ctx context.Context,
	tp tracing.TracerProvider,
	log logger.Logger,
	md *statev2.Metadata,
	hashedID string,
	status enums.DeferStatus,
) {
	if tp == nil {
		return
	}
	err := tp.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		Debug:      &tracing.SpanDebugData{Location: "defers.updateDeferSpanStatus"},
		TargetSpan: tracing.DeferSpanRef(md.ID.RunID, hashedID),
		Metadata:   md,
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DeferStatus, &status),
		),
	})
	if err != nil {
		log.Error(
			"error updating executor.defer span status",
			"error", err,
			"hashed_id", util.SanitizeLogField(hashedID),
			"run_id", md.ID.RunID,
			"status", status.String(),
		)
	}
}
