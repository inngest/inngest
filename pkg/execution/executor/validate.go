package executor

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

func newRunValidator(item queue.Item, s state.State, f *inngest.Function, e *executor) *runValidator {
	return &runValidator{
		item: item,
		s:    s,
		md:   s.Metadata(),
		f:    f,
		e:    e,
	}
}

// runValidator runs checks when executing a queue item for the executor.
type runValidator struct {
	item queue.Item
	s    state.State
	md   state.Metadata
	f    *inngest.Function

	// stopWithoutRetry prevents
	stopWithoutRetry bool

	e *executor
}

func (r *runValidator) validate(ctx context.Context) error {
	chain := []func(ctx context.Context) error{
		r.checkCancelled,
		r.checkStepLimit,
		r.checkCancellation,
		r.checkStartTimeout,
		r.checkFinishTimeout,
		r.updateScheduledStatus,
	}

	for _, step := range chain {
		if err := step(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *runValidator) checkCancelled(ctx context.Context) error {
	if r.md.Status == enums.RunStatusCancelled {
		return state.ErrFunctionCancelled
	}
	return nil
}

func (r *runValidator) checkStepLimit(ctx context.Context) error {
	if r.e.steplimit != 0 && len(r.s.Actions()) >= int(r.e.steplimit) {
		// Update this function's state to overflowed, if running.
		if r.md.Status == enums.RunStatusRunning {
			// XXX: Update error to failed, set error message
			if err := r.e.sm.SetStatus(ctx, r.md.Identifier, enums.RunStatusFailed); err != nil {
				return err
			}

			// Create a new driver response to map as the function finished error.
			resp := state.DriverResponse{}
			resp.SetError(state.ErrFunctionOverflowed)
			resp.SetFinal()

			if err := r.e.runFinishHandler(ctx, r.md.Identifier, r.s, resp); err != nil {
				logger.From(ctx).Error().Err(err).Msg("error running finish handler")
			}

			for _, e := range r.e.lifecycles {
				go e.OnFunctionFinished(context.WithoutCancel(ctx), r.md.Identifier, r.item, resp, r.s)
			}
		}

		// Stop the function from running, but don't return an error as we don't
		// want the step to retry.
		r.stopWithoutRetry = true
	}
	return nil
}

func (r *runValidator) checkCancellation(ctx context.Context) error {
	if r.e.cancellationChecker != nil {
		cancel, err := r.e.cancellationChecker.IsCancelled(
			ctx,
			r.md.Identifier.WorkspaceID,
			r.md.Identifier.WorkflowID,
			r.md.Identifier.RunID,
			r.s.Event(),
		)
		if err != nil {
			logger.StdlibLogger(ctx).Error(
				"error checking cancellation",
				"error", err.Error(),
				"run_id", r.md.Identifier.RunID,
				"function_id", r.md.Identifier.WorkflowID,
				"workspace_id", r.md.Identifier.WorkspaceID,
			)
		}
		if cancel != nil {
			err = r.e.Cancel(ctx, r.md.Identifier.RunID, execution.CancelRequest{
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

// updateScheduledStatus pushes a metadata mutation to update the function's scheduled status to
// running.
func (r *runValidator) updateScheduledStatus(ctx context.Context) error {
	if r.md.Status == enums.RunStatusScheduled {
		return r.e.sm.SetStatus(ctx, r.md.Identifier, enums.RunStatusRunning)
	}
	return nil
}

func (r *runValidator) checkStartTimeout(ctx context.Context) error {
	if r.f.Timeouts != nil {
		since := time.Since(ulid.Time(r.md.Identifier.RunID.Time()))
		if r.f.Timeouts.Start > 0 && since > r.f.Timeouts.Start {
			logger.StdlibLogger(ctx).Debug("start timeout reached", "run_id", r.s.RunID())
			if err := r.e.Cancel(ctx, r.md.Identifier.RunID, execution.CancelRequest{}); err != nil {
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
		if r.md.StartedAt.IsZero() || r.md.StartedAt.Unix() == 0 || time.Since(r.md.StartedAt) <= r.f.Timeouts.Finish {
			return nil
		}
		logger.StdlibLogger(ctx).Info(
			"finish timeout reached",
			"run_id", r.s.RunID(),
			"started_at", r.md.StartedAt.UTC(),
			"timeout", r.f.Timeouts.Finish.String(),
			"since", time.Since(r.md.StartedAt).String(),
		)
		if err := r.e.Cancel(ctx, r.md.Identifier.RunID, execution.CancelRequest{}); err != nil {
			return err
		}
		// Stop the function from running, but don't return an error as we don't
		// want the step to retry.
		r.stopWithoutRetry = true
	}
	return nil

}
