package defers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing"
)

const pkgName = "defers"

// SaveFromOp persists a DeferAdd opcode. Returns nil for accepted and
// soft-rejected outcomes; non-nil only for infra errors.
func SaveFromOp(
	ctx context.Context,
	rs statev2.RunService,
	log logger.Logger,
	id statev2.ID,
	op state.GeneratorOpcode,
	tp tracing.TracerProvider,
	md statev2.Metadata,
	now time.Time,
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

	opts, parseErr := op.DeferAddOpts()
	if parseErr != nil {
		switch {
		case errors.Is(parseErr, state.ErrDeferInputTooLarge):
			rejectReason = "per_defer_size"
		case errors.Is(parseErr, state.ErrDeferInputInvalid):
			rejectReason = "invalid_input"
		default:
			rejectReason = "invalid_opts"
		}
		// Persist a Rejected record so SDK retransmits dedupe.
		if opts != nil && opts.FnSlug != "" {
			if err := rs.SaveDefer(ctx, id, statev2.Defer{
				FnSlug:         opts.FnSlug,
				HashedID:       op.ID,
				UserlandID:     userlandID,
				ScheduleStatus: enums.DeferStatusRejected,
			}); err != nil {
				log.Warn("failed to persist rejected defer; SDK retransmits will not dedupe",
					"step_id", sanitizeLogValue(op.ID),
					"run_id", id.RunID,
					"error", err,
				)
			} else {
				rejectionPersisted = true
			}
		}
	}

	if rejectReason == "" {
		d := statev2.Defer{
			FnSlug:         opts.FnSlug,
			HashedID:       op.ID,
			UserlandID:     userlandID,
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          opts.Input,
		}
		saveErr := rs.SaveDefer(ctx, id, d)
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

		if rejectReason == "" {
			emitDeferSpan(ctx, tp, md, now, d)
		}
	}

	if rejectReason != "" {
		log.Warn("defer soft-rejected; run will continue without this deferred run",
			"step_id", sanitizeLogValue(op.ID),
			"reason", rejectReason,
			"run_id", id.RunID,
		)
		metrics.IncrDefersRejectedCounter(ctx, rejectReason, metrics.CounterOpt{PkgName: pkgName})
	}

	if rejectionPersisted {
		fnSlug := ""
		if opts != nil {
			fnSlug = opts.FnSlug
		}
		emitDeferSpan(ctx, tp, md, now, statev2.Defer{
			FnSlug:         fnSlug,
			HashedID:       op.ID,
			UserlandID:     userlandID,
			ScheduleStatus: enums.DeferStatusRejected,
		})
	}

	return nil
}

func sanitizeLogValue(v string) string {
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\r", "")
	return v
}
