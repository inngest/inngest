package executor

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

// Finalize performs run finalization, which involves sending the function
// finished/failed event and deleting state.
func (e *executor) Finalize(ctx context.Context, opts execution.FinalizeOpts) error {
	ctx = context.WithoutCancel(ctx)
	l := logger.StdlibLogger(ctx)

	err := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		EndTime:    e.now(),
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

	// If there are no input events, fetch them.
	if len(opts.Optional.InputEvents) == 0 {
		opts.Optional.InputEvents, err = e.smv2.LoadEvents(ctx, opts.Metadata.ID)
		if err != nil {
			l.Warn(
				"error loading run events to finalize",
				"error", err,
				"run_id", opts.Metadata.ID.RunID,
			)
		}
	}

	// Delete the function state in every case.
	err = e.smv2.Delete(ctx, opts.Metadata.ID)
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

	// finalizeEvents creates function finished events, and also attempts to fast-resume
	// any parent function that invoked this run.
	return e.finalizeEvents(ctx, opts)
}

// finalizeRemoveJobs removes any other jobs for a finalized run, as the function is
// marked as finished and no other jobs need to execute.
func (e *executor) finalizeRemoveJobs(ctx context.Context, opts execution.FinalizeOpts) {
	l := logger.StdlibLogger(ctx)

	// XXX: can we use e.assignedQueueShard here?
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
		// Track whether this run was an invoke.
		isInvoke bool
		fnSlug   = opts.Optional.FnSlug
		evts     = opts.Optional.InputEvents
	)

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
	now := e.now()
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

		isInvoke = true

		// Copy the base data to set the event.
		copied := *base
		copied.Event = runEvt.Map()
		copied.InvokeCorrelationID = invokeID
		data := copied.Map()

		// Add a status field.
		data[consts.InngestEventDataPrefix] = map[string]any{
			"status": opts.Status(),
		}

		// Add an `inngest/function.finished` event.
		freshEvents = append(freshEvents, event.Event{
			ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
			Name:      event.FnFinishedName,
			Timestamp: now.UnixMilli(),
			Data:      data,
		})

		switch opts.Status() {
		case enums.StepStatusCancelled:
			freshEvents = append(freshEvents, event.Event{
				ID:        opts.Metadata.ID.RunID.String(), // using the RunID as the ID prevents duped runs for parallel steps
				Name:      event.FnCancelledName,
				Timestamp: now.UnixMilli(),
				Data:      data,
			})
		case enums.StepStatusFailed:
			// Legacy - send inngest/function.failed, except for when the function has been cancelled.
			freshEvents = append(freshEvents, event.Event{
				ID:        opts.Metadata.ID.RunID.String(), // using the RunID as the ID prevents duped runs for parallel steps
				Name:      event.FnFailedName,
				Timestamp: now.UnixMilli(),
				Data:      data,
			})
		}
	}

	// For each event, if this has a correlation ID attempt to resume
	// the invoke parent within a goroutine.
	//
	// Note that sending the event will trigger the event handler pub/sub
	// listener which _also_ attempts to do this;  however, this introduces
	// some small delay due to message stream latency.
	if isInvoke {
		for _, evt := range freshEvents {
			tracked := event.BaseTrackedEvent{
				ID:          ulid.MustParse(evt.ID),
				Event:       evt,
				AccountID:   opts.Metadata.ID.Tenant.AccountID,
				WorkspaceID: opts.Metadata.ID.Tenant.EnvID,
			}
			service.Go(func() {
				err := e.HandleInvokeFinish(context.WithoutCancel(ctx), tracked)
				if err != nil && !errors.Is(err, ErrNoCorrelationID) {
					logger.From(ctx).Error("error fast resuming invoke", "error", err)
				}
			})
		}
	}

	return e.finishHandler(ctx, opts.Metadata.ID, freshEvents)
}

func finalizeSpanAttributes(f execution.FinalizeOpts) *meta.SerializableAttrs {
	// We're explicitly not setting any output span reference here and passing
	// `nil` instead. We do this because we need to be setting the function
	// output twice - once for the execution itself and once for the run span -
	// in order to appropriately filter this in Cloud and other data stores.

	switch f.Response.Type {
	case execution.FinalizeResponseAPI:
		return apiAttributes(f.Response.APIResponse)
	case execution.FinalizeResponseRunComplete:
		return runCompleteAttrs(f.Response.RunComplete)
	case execution.FinalizeResponseDriver:
		return tracing.DriverResponseAttrs(&f.Response.DriverResponse, nil)
	}

	panic("unknown finalize response type")
}

func apiAttributes(res apiresult.APIResult) *meta.SerializableAttrs {
	h := http.Header{}
	for k, v := range res.Headers {
		h.Set(k, v)
	}

	rawAttrs := meta.NewAttrSet()
	meta.AddAttr(rawAttrs, meta.Attrs.IsFunctionOutput, inngestgo.Ptr(true))
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseHeaders, &h)
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseStatusCode, &res.StatusCode)
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseOutputSize, inngestgo.Ptr(len(res.Body)))
	// XXX: We always wrap trace output with {"data":T} or {"error":T} for consistency with steps.
	meta.AddAttr(rawAttrs, meta.Attrs.StepOutput, inngestgo.Ptr(util.DataWrap(res.Body)))

	return rawAttrs
}

func runCompleteAttrs(gen state.GeneratorOpcode) *meta.SerializableAttrs {
	rawAttrs := meta.NewAttrSet()

	meta.AddAttr(rawAttrs, meta.Attrs.IsFunctionOutput, inngestgo.Ptr(true))
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseStatusCode, inngestgo.Ptr(200)) // Must be to have this code.  It's an async fn.
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseOutputSize, inngestgo.Ptr(len(gen.Data)))
	// XXX: We always wrap trace output with {"data":T} or {"error":T} for consistency with steps.
	meta.AddAttr(rawAttrs, meta.Attrs.StepOutput, inngestgo.Ptr(util.DataWrap(gen.Data)))

	rawAttrs = rawAttrs.Merge(tracing.GeneratorAttrs(&gen))

	return rawAttrs
}
