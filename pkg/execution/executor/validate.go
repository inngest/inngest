package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/expressions"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

func newRunValidator(
	e *executor,
	f *inngest.Function,
	md sv2.Metadata,
	evts []json.RawMessage,
	item queue.Item,
) *runValidator {
	return &runValidator{
		item: item,
		md:   md,
		f:    f,
		e:    e,
		evts: evts,
	}
}

// runValidator runs checks when executing a queue item for the executor.
type runValidator struct {
	item queue.Item
	md   sv2.Metadata
	f    *inngest.Function
	evts []json.RawMessage

	// stopWithoutRetry prevents
	stopWithoutRetry bool

	e *executor
}

func (r *runValidator) validate(ctx context.Context) error {
	chain := []func(ctx context.Context) error{
		r.checkStepLimit,
		r.checkCancellation,
		r.checkStartTimeout,
		r.checkFinishTimeout,
	}

	for _, step := range chain {
		if err := step(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *runValidator) checkStepLimit(ctx context.Context) error {
	l := logger.From(ctx).With().
		Str("run_id", r.md.ID.RunID.String()).
		Str("workflow_id", r.md.ID.FunctionID.String()).
		Logger()

	var limit int

	if r.e.steplimit != nil {
		limit = r.e.steplimit(r.md.ID)
	}

	if limit == 0 {
		limit = consts.DefaultMaxStepLimit
	}
	if limit > consts.AbsoluteMaxStepLimit {
		return fmt.Errorf("%d is greater than the absolute step limit of %d", limit, consts.AbsoluteMaxStepLimit)
	}

	if limit > 0 && r.md.Metrics.StepCount >= limit {
		// Create a new driver response to map as the function finished error.
		resp := state.DriverResponse{}

		gracefulErr := state.StandardError{
			Error:   state.ErrFunctionOverflowed.Error(),
			Name:    state.InngestErrFunctionOverflowed,
			Message: fmt.Sprintf("The function run exceeded the step limit of %d steps.", limit),
		}.Serialize(execution.StateErrorKey)

		resp.Err = &gracefulErr
		resp.SetFinal()

		if performedFinalization, err := r.e.finalize(ctx, r.md, r.evts, r.f.GetSlug(), resp); err != nil {
			l.Error().Err(err).Msg("error running finish handler")
		} else if performedFinalization {
			for _, e := range r.e.lifecycles {
				go e.OnFunctionFinished(context.WithoutCancel(ctx), r.md, r.item, r.evts, resp)
			}
		} else {
			l.Info().Msg("run cancelled but did not finalize")
		}

		// Stop the function from running, but don't return an error as we don't
		// want the step to retry.
		r.stopWithoutRetry = true
	}
	return nil
}

func (r *runValidator) checkCancellation(ctx context.Context) error {
	if r.e.cancellationChecker != nil {
		evt := event.Event{}
		if err := json.Unmarshal(r.evts[0], &evt); err != nil {
			return fmt.Errorf("error decoding input event in cancellation checker: %w", err)
		}

		cancel, err := r.e.cancellationChecker.IsCancelled(
			ctx,
			r.md.ID.Tenant.EnvID,
			r.md.ID.FunctionID,
			r.md.ID.RunID,
			evt.Map(),
		)
		if err != nil {
			if errors.Is(err, &expressions.CompileError{}) {
				logger.StdlibLogger(ctx).Warn(
					"invalid cancellation expression",
					"error", err.Error(),
					"run_id", r.md.ID.RunID,
					"function_id", r.md.ID.FunctionID,
					"workspace_id", r.md.ID.Tenant.EnvID,
				)
			} else {
				logger.StdlibLogger(ctx).Error(
					"error checking cancellation",
					"error", err.Error(),
					"run_id", r.md.ID.RunID,
					"function_id", r.md.ID.FunctionID,
					"workspace_id", r.md.ID.Tenant.EnvID,
				)
			}
		}
		if cancel != nil {
			err = r.e.Cancel(ctx, r.md.ID, execution.CancelRequest{
				CancellationID: &cancel.ID,
			})
			if err != nil {
				return err
			}

			// Stop the function from running, but don't return an error as we don't
			// want the step to retry.
			r.stopWithoutRetry = true
		}
	}
	return nil
}

func (r *runValidator) checkStartTimeout(ctx context.Context) error {
	if r.f.Timeouts != nil {
		since := time.Since(ulid.Time(r.md.ID.RunID.Time()))
		if r.f.Timeouts.Start > 0 && since > r.f.Timeouts.Start {
			logger.StdlibLogger(ctx).Debug("start timeout reached", "run_id", r.md.ID.RunID.String())
			if err := r.e.Cancel(ctx, r.md.ID, execution.CancelRequest{}); err != nil {
				return err
			}
			// Stop the function from running, but don't return an error as we don't
			// want the step to retry.
			r.stopWithoutRetry = true
		}
	}
	return nil

}

func (r *runValidator) checkFinishTimeout(ctx context.Context) error {
	if r.f.Timeouts != nil && r.f.Timeouts.Finish > 0 {
		started := r.md.Config.StartedAt

		if started.IsZero() || started.Unix() == 0 || time.Since(started) <= r.f.Timeouts.Finish {
			return nil
		}
		logger.StdlibLogger(ctx).Info(
			"finish timeout reached",
			"run_id", r.md.ID.RunID,
			"started_at", started.UTC(),
			"timeout", r.f.Timeouts.Finish.String(),
			"since", time.Since(started).String(),
		)
		if err := r.e.Cancel(ctx, r.md.ID, execution.CancelRequest{}); err != nil {
			return err
		}
		// Stop the function from running, but don't return an error as we don't
		// want the step to retry.
		r.stopWithoutRetry = true
	}
	return nil

}
