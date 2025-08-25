package executor

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

// Finalize performs run finalization, which involves sending the function
// finished/failed event and deleting state.
func (e *executor) Finalize(ctx context.Context, opts execution.FinalizeOpts) error {
	ctx = context.WithoutCancel(ctx)
	l := logger.StdlibLogger(ctx)

	err := e.tracerProvider.UpdateSpan(&tracing.UpdateSpanOptions{
		EndTime:    time.Now(),
		Debug:      &tracing.SpanDebugData{Location: "executor.finalize"},
		Metadata:   &opts.Metadata,
		TargetSpan: tracing.RunSpanRefFromMetadata(&opts.Metadata),
		Status:     opts.Status(),
		Attributes: finalizeSpanAttributes(opts),
	})
	if err != nil {
		// TODO This should be a warning/error once these spans are critical.
		l.Debug(
			"error updating run span end time",
			"error", err,
			"run_id", opts.Metadata.ID.RunID,
			"target_span", tracing.RunSpanRefFromMetadata(&opts.Metadata),
		)
	}

	// Report the status of the size.
	if e.preDeleteStateSizeReporter != nil {
		e.preDeleteStateSizeReporter(ctx, opts.Metadata)
	}

	// Delete the function state in every case.
	_, err = e.smv2.Delete(ctx, opts.Metadata.ID)
	if err != nil {
		l.Error(
			"error deleting state in finalize",
			"error", err,
			"run_id", opts.Metadata.ID.RunID,
		)
	}

	metrics.IncrRunFinalizedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"reason": opts.Optional.Reason,
		},
	})

	e.finalizeRemoveJobs(ctx, opts)
	return e.finalizeEvents(ctx, opts)
}

// finalizeRemoveJobs removes any other jobs for a finalized run, as the function is
// marked as finished and no other jobs need to execute.
func (e *executor) finalizeRemoveJobs(ctx context.Context, opts execution.FinalizeOpts) {
	l := logger.StdlibLogger(ctx)

	// e.assignedQueueShard ??

	shard, err := e.shardFinder(
		ctx,
		opts.Metadata.ID.Tenant.AccountID,
		nil,
	)
	if err != nil {
		return
	}

	// We may be cancelling an in-progress run.  If that's the case, we want to delete any
	// outstanding jobs from the queue, if possible.
	//
	// XXX: Remove this typecast and normalize queue interface to a single package
	q, ok := e.queue.(redis_state.QueueManager)
	if !ok {
		return
	}
	// Find all items for the current function run.
	jobs, err := q.RunJobs(
		ctx,
		shard.Name,
		opts.Metadata.ID.Tenant.EnvID,
		opts.Metadata.ID.FunctionID,
		opts.Metadata.ID.RunID,
		1000,
		0,
	)
	if err != nil {
		l.Error(
			"error fetching run jobs",
			"error", err,
			"run_id", opts.Metadata.ID.RunID,
		)
	}

	for _, j := range jobs {
		qi, _ := j.Raw.(*queue.QueueItem)
		if qi == nil {
			continue
		}

		jobID := queue.JobIDFromContext(ctx)
		if jobID != "" && qi.ID == jobID {
			// Do not dequeue the current job that we're working on.
			continue
		}

		err := q.Dequeue(ctx, shard, *qi)
		if err != nil && !errors.Is(err, redis_state.ErrQueueItemNotFound) {
			l.Error(
				"error dequeueing run job",
				"error", err,
				"run_id", opts.Metadata.ID.RunID.String(),
			)
		}
	}
}

func (e *executor) finalizeEvents(ctx context.Context, opts execution.FinalizeOpts) error {
	if e.finishHandler == nil {
		// the finishHandler handles sending finalization events.
		return nil
	}

	var (
		fnSlug = opts.Optional.FnSlug
		evts   = opts.Optional.InputEvents
		err    error
	)

	l := logger.StdlibLogger(ctx)

	// If there are no input events, fetch them.
	if len(evts) == 0 {
		evts, err = e.smv2.LoadEvents(ctx, opts.Metadata.ID)
		if err != nil {
			l.Error(
				"error loading run events to finalize",
				"error", err,
				"run_id", opts.Metadata.ID.RunID,
			)
			return err
		}
	}

	// Find the function slug.
	if fnSlug == "" {
		fn, err := e.fl.LoadFunction(ctx, opts.Metadata.ID.Tenant.EnvID, opts.Metadata.ID.FunctionID)
		if err != nil {
			return err
		}
		fnSlug = fn.Function.Slug
	}

	// Parse events for the fail handler before deleting state.
	inputEvents := make([]event.Event, len(evts))
	for n, e := range evts {
		evt, err := event.NewEvent(e)
		if err != nil {
			return err
		}
		inputEvents[n] = *evt
	}

	// Prepare events that we must send
	now := time.Now()
	base := &functionFinishedData{
		FunctionID: fnSlug,
		RunID:      opts.Metadata.ID.RunID,
		Events:     inputEvents,
	}
	base.setResponse(opts.Response)

	// We'll send many events - some for each items in the batch.  This ensures that invoke works
	// for batched functions.
	freshEvents := []event.Event{}
	for n, runEvt := range inputEvents {
		if runEvt.Name == event.FnFailedName || runEvt.Name == event.FnFinishedName {
			// Don't recursively trigger internal finish handlers.
			continue
		}

		invokeID := correlationID(runEvt)
		if invokeID == nil && n > 0 {
			// We only send function finish events for either the first event in a batch or for
			// all events with a correlation ID.
			continue
		}

		// Copy the base data to set the event.
		copied := *base
		copied.Event = runEvt.Map()
		copied.InvokeCorrelationID = invokeID
		data := copied.Map()

		// Add an `inngest/function.finished` event.
		freshEvents = append(freshEvents, event.Event{
			ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
			Name:      event.FnFinishedName,
			Timestamp: now.UnixMilli(),
			Data:      data,
		})

		if opts.Response.Err != nil {
			// Legacy - send inngest/function.failed, except for when the function has been cancelled.
			if !strings.Contains(*opts.Response.Err, state.ErrFunctionCancelled.Error()) {
				freshEvents = append(freshEvents, event.Event{
					ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
					Name:      event.FnFailedName,
					Timestamp: now.UnixMilli(),
					Data:      data,
				})
			}

			// Add function cancelled event
			if *opts.Response.Err == state.ErrFunctionCancelled.Error() {
				freshEvents = append(freshEvents, event.Event{
					ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
					Name:      event.FnCancelledName,
					Timestamp: now.UnixMilli(),
					Data:      data,
				})
			}
		}
	}

	return e.finishHandler(ctx, opts.Metadata.ID, freshEvents)
}

func finalizeSpanAttributes(f execution.FinalizeOpts) *meta.SerializableAttrs {
	return tracing.DriverResponseAttrs(&f.Response, f.Optional.OutputSpanRef)
}
