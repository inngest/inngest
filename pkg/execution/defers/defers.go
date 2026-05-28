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
		rejected     bool
		rejectReason string
	)

	var userlandID string
	if op.Userland != nil {
		userlandID = op.Userland.ID
	}

	opts, parseErr := op.DeferAddOpts()
	if parseErr != nil {
		rejected = true
		if errors.Is(parseErr, state.ErrDeferInputTooLarge) {
			rejectReason = "per_defer_size"
		} else {
			rejectReason = "invalid_opts"
		}
		// Best-effort sentinel so SDK retransmits dedupe.
		if opts != nil && opts.FnSlug != "" {
			if rerr := rs.SaveRejectedDefer(ctx, id, opts.FnSlug, op.ID); rerr != nil {
				log.Warn("failed to save rejected defer sentinel; SDK retransmits will not dedupe",
					"step_id", sanitizeLogValue(op.ID),
					"run_id", id.RunID,
					"error", rerr,
				)
			}
		}
	}

	if !rejected {
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
			// Count cap binds the sentinel write too. SDK retransmits absorbed.
			rejected = true
			rejectReason = "per_run_count"
		case errors.Is(saveErr, statev2.ErrDeferInputAggregateExceeded):
			// saveDefer.lua already wrote the Rejected sentinel.
			rejected = true
			rejectReason = "aggregate_size"
		case saveErr != nil:
			return fmt.Errorf("error saving defer: %w", saveErr)
		}

		if !rejected {
			emitDeferSpan(ctx, tp, md, now, d)
		}
	}

	if rejected {
		log.Warn("defer soft-rejected; run will continue without this deferred run",
			"step_id", sanitizeLogValue(op.ID),
			"reason", rejectReason,
			"run_id", id.RunID,
		)
		metrics.IncrDefersRejectedCounter(ctx, rejectReason, metrics.CounterOpt{PkgName: pkgName})
	}

	return nil
}

func sanitizeLogValue(v string) string {
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\r", "")
	return v
}
