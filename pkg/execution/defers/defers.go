package defers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
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
) error {
	var (
		rejected     bool
		rejectReason string
	)

	opts, parseErr := op.DeferAddOpts()
	switch {
	case errors.Is(parseErr, state.ErrDeferInputTooLarge):
		rejected = true
		rejectReason = "per_defer_size"
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
	case parseErr != nil:
		// No FnSlug, so no sentinel. SDK retransmits absorbed.
		rejected = true
		rejectReason = "invalid_opts"
	}

	if !rejected {
		var userlandID string
		if op.Userland != nil {
			userlandID = op.Userland.ID
		}
		saveErr := rs.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         opts.FnSlug,
			HashedID:       op.ID,
			UserlandID:     userlandID,
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          opts.Input,
		})
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

// AbortFromOp flips the target defer's status to Aborted. Errors are
// surfaced (no soft-fail).
func AbortFromOp(
	ctx context.Context,
	rs statev2.RunService,
	log logger.Logger,
	id statev2.ID,
	op state.GeneratorOpcode,
) error {
	opts, err := op.DeferAbortOpts()
	if err != nil {
		log.Error("error parsing DeferAbort opts", "error", err)
		return fmt.Errorf("error parsing DeferAbort opts: %w", err)
	}

	if err := rs.SetDeferStatus(ctx, id, opts.TargetHashedID, enums.DeferStatusAborted); err != nil {
		log.Error("error aborting defer", "error", err)
		return fmt.Errorf("error aborting defer: %w", err)
	}

	return nil
}

func sanitizeLogValue(v string) string {
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\r", "")
	return v
}
