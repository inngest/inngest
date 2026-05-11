package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

// cancelationGracePeriod is the amount of time we add when marking a cancelled function as finished.  This
// allows any in-flight steps to complete and report their status, which prevents orphaned steps and ensures
// that the function's final status is correct.
const cancelationGracePeriod = 10 * time.Second

// Finalize performs run finalization, which involves sending the function
// finished/failed event and deleting state.
func (e *executor) Finalize(ctx context.Context, opts execution.FinalizeOpts) error {
	ctx = context.WithoutCancel(ctx)
	l := logger.StdlibLogger(ctx)

	var endTimeOffset time.Duration
	status := opts.Status()
	if status == enums.StepStatusCancelled {
		endTimeOffset = cancelationGracePeriod
	}

	err := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		EndTime:       e.now(),
		EndTimeOffset: endTimeOffset,
		Debug:         &tracing.SpanDebugData{Location: "executor.finalize"},
		Metadata:      &opts.Metadata,
		TargetSpan:    tracing.RunSpanRefFromMetadata(&opts.Metadata),
		Status:        opts.Status(),
		Attributes:    finalizeSpanAttributes(opts),
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

	// Release any manual-release semaphores held by this run.  Manual-release semaphores
	// (e.g. function concurrency) are acquired when the start job is dequeued but are NOT
	// released when individual step leases complete — they persist for the lifetime of the
	// run.  We must release them here, before state deletion, so that the semaphore info
	// from run metadata is still available.  The run ID is used as the idempotency key to
	// guarantee safe retries.
	if e.semaphoreManager == nil && len(opts.Metadata.Config.Semaphores) > 0 {
		l.Error(
			"semaphore manager is nil but run holds semaphores, leading to deadlock",
			"run_id", opts.Metadata.ID.RunID,
			"semaphores", len(opts.Metadata.Config.Semaphores),
		)
	}

	if e.semaphoreManager != nil && len(opts.Metadata.Config.Semaphores) > 0 {
		for _, sem := range opts.Metadata.Config.Semaphores {
			if sem.Release != constraintapi.SemaphoreReleaseManual {
				continue
			}
			// Retry semaphore release — a failure here means the semaphore is permanently
			// held, which deadlocks all future runs waiting on capacity.
			_, releaseErr := util.WithRetry(ctx, "release-semaphore", func(ctx context.Context) (struct{}, error) {
				return struct{}{}, e.semaphoreManager.ReleaseSemaphore(
					ctx,
					opts.Metadata.ID.Tenant.AccountID,
					sem.ID,
					sem.UsageValue,
					opts.Metadata.ID.RunID.String(),
					sem.Weight,
				)
			}, util.NewRetryConf())
			if releaseErr != nil {
				l.Error(
					"error releasing semaphore on finalize after retries",
					"error", releaseErr,
					"run_id", opts.Metadata.ID.RunID,
					"semaphore", sem.ID,
				)
			}
		}
	}

	// Load defers BEFORE Delete since they live in state and won't survive the
	// deletion. Retry on transient failures so the events get a chance to
	// publish even when Redis is briefly unavailable. Defer-related failures
	// are best-effort: log and continue with no defer events rather than
	// blocking Finalize. The downstream cleanup (Delete, finalizeRemoveJobs,
	// finalizeEvents for function.X) must still run regardless.
	loadDefersStart := e.now()
	defers, deferErr := util.WithRetry(ctx, "state.LoadDefers",
		func(ctx context.Context) (map[string]sv2.Defer, error) {
			return e.smv2.LoadDefers(ctx, opts.Metadata.ID)
		},
		util.NewRetryConf(),
	)
	metrics.HistogramDefersLoadDuration(ctx, e.now().Sub(loadDefersStart), metrics.HistogramOpt{
		PkgName: pkgName,
	})
	if deferErr != nil {
		l.Error(
			"error loading defers to finalize; continuing without defer events",
			"error", deferErr,
			"run_id", opts.Metadata.ID.RunID,
		)
	}
	metrics.HistogramDefersPerRun(ctx, int64(len(defers)), metrics.HistogramOpt{
		PkgName: pkgName,
	})

	// Build defer events from the loaded map BEFORE Delete (resolves fnSlug
	// using the in-memory function loader, not state). The actual publish
	// happens in finalizeEvents so all finalize-time events go through a
	// single finishHandler call.
	deferEvents, err := e.buildDeferEvents(ctx, opts, defers)
	if err != nil {
		l.Error(
			"error building deferred schedule events; continuing without defer events",
			"error", err,
			"run_id", opts.Metadata.ID.RunID,
		)
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
	// any parent function that invoked this run. Defer events are published as
	// part of the same finishHandler call.
	return e.finalizeEvents(ctx, opts, deferEvents)
}

// buildDeferEvents constructs the inngest/deferred.schedule events for every
// AfterRun defer in `defers`. It does no publishing — the events are returned
// for the caller (Finalize) to fold into the single finishHandler call inside
// finalizeEvents.
//
// Per-defer validation failures (Validate, status filter, malformed Input)
// log and skip the bad record. They are not fatal to the batch.
func (e *executor) buildDeferEvents(
	ctx context.Context,
	opts execution.FinalizeOpts,
	defers map[string]sv2.Defer,
) ([]event.Event, error) {
	if len(defers) == 0 {
		return nil, nil
	}

	fnSlug := opts.Optional.FnSlug
	if fnSlug == "" {
		fnSlug = opts.Metadata.Config.FunctionSlug()
	}
	if fnSlug == "" {
		return nil, fmt.Errorf("function slug missing from run metadata for deferred events")
	}

	now := e.now()
	var events []event.Event

	for _, d := range defers {
		if err := d.Validate(); err != nil {
			logger.StdlibLogger(ctx).Error(
				"invalid defer",
				"error", err,
				"run_id", opts.Metadata.ID.RunID,
			)
			metrics.IncrDefersFinalizedCounter(ctx, "invalid", metrics.CounterOpt{PkgName: pkgName})
			continue
		}

		// The GraphQL `defers` field surfaces both SCHEDULED and ABORTED rows;
		// Rejected and any future enum value are out of contract and skipped
		// rather than persisted as an unknown string.
		if e.deferStore != nil {
			var deferRowStatus cqrs.RunDeferStatus
			switch d.ScheduleStatus {
			case enums.DeferStatusAfterRun:
				deferRowStatus = cqrs.RunDeferStatusScheduled
			case enums.DeferStatusAborted:
				deferRowStatus = cqrs.RunDeferStatusAborted
			}
			if deferRowStatus != "" {
				if err := e.deferStore.InsertRunDefer(ctx, opts.Metadata.ID.RunID, d.HashedID, d.UserlandID, d.FnSlug, deferRowStatus); err != nil {
					logger.StdlibLogger(ctx).Error(
						"error persisting run defer",
						"error", err,
						"run_id", opts.Metadata.ID.RunID,
						"defer_id", d.HashedID,
					)
				}
			}
		}

		// TODO: what about an immediate execution mode?
		if d.ScheduleStatus != enums.DeferStatusAfterRun {
			metrics.IncrDefersFinalizedCounter(ctx, d.ScheduleStatus.String(), metrics.CounterOpt{PkgName: pkgName})
			continue
		}

		// Deterministic event ID so any duplicate-publish path dedupes on the
		// runner side (runner.go uses event.ID as the schedule idempotency key).
		// Time prefix is the parent run's start so the ULID stays well-formed.
		seed := []byte(opts.Metadata.ID.RunID.String() + d.HashedID)
		eventID, err := util.DeterministicULID(ulid.Time(opts.Metadata.ID.RunID.Time()), seed)
		if err != nil {
			// Unreachable
			logger.StdlibLogger(ctx).Error(
				"error generating deferred event ID",
				"error", err,
				"run_id", opts.Metadata.ID.RunID,
				"unreachable", true,
			)
			metrics.IncrDefersFinalizedCounter(ctx, "invalid", metrics.CounterOpt{PkgName: pkgName})
			continue
		}

		data := map[string]any{}
		if len(d.Input) > 0 {
			if err := json.Unmarshal(d.Input, &data); err != nil {
				logger.StdlibLogger(ctx).Error(
					"deferred input is not a JSON object",
					"error", err,
					"run_id", opts.Metadata.ID.RunID,
				)
				metrics.IncrDefersFinalizedCounter(ctx, "invalid", metrics.CounterOpt{PkgName: pkgName})
				continue
			}
			if data == nil {
				// Reachable if the input is `null`. We need to set it to an
				// empty map to avoid panicking later
				data = make(map[string]any)
			}
		}

		// Local variable name avoids shadowing the imported `meta` package
		// (see top of file). A future addition that uses meta.NewAttrSet
		// or similar inside this loop would otherwise fail to compile in
		// a non-obvious way.
		deferredMeta := event.DeferredScheduleMetadata{
			FnSlug:       d.FnSlug,
			ParentFnSlug: fnSlug,
			ParentRunID:  opts.Metadata.ID.RunID.String(),
			DeferID:      d.HashedID,
		}
		if err := deferredMeta.Validate(); err != nil {
			logger.StdlibLogger(ctx).Error(
				"invalid deferred event metadata",
				"error", err,
				"run_id", opts.Metadata.ID.RunID,
			)
			metrics.IncrDefersFinalizedCounter(ctx, "invalid", metrics.CounterOpt{PkgName: pkgName})
			continue
		}
		data[consts.InngestEventDataPrefix] = deferredMeta

		events = append(events, event.Event{
			ID:        eventID.String(),
			Name:      consts.FnDeferScheduleName,
			Timestamp: now.UnixMilli(),
			Data:      data,
		})
		metrics.IncrDefersFinalizedCounter(ctx, "after_run", metrics.CounterOpt{PkgName: pkgName})
	}

	return events, nil
}

// finalizeRemoveJobs removes any other jobs for a finalized run, as the function is
// marked as finished and no other jobs need to execute.
func (e *executor) finalizeRemoveJobs(ctx context.Context, opts execution.FinalizeOpts) {
	l := logger.StdlibLogger(ctx)

	shard, err := e.shards.Resolve(ctx, opts.Metadata.ID.Tenant.AccountID, nil)
	if err != nil {
		return
	}

	// We may be cancelling an in-progress run.  If that's the case, we want to delete any
	// outstanding jobs from the queue, if possible.
	//
	// XXX: Remove this typecast and normalize queue interface to a single package
	// Find all items for the current function run.
	jobs, err := shard.RunJobs(
		ctx,
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

		err := shard.Dequeue(ctx, *qi)
		if err != nil && !errors.Is(err, queue.ErrQueueItemNotFound) {
			l.Error(
				"error dequeueing run job",
				"error", err,
				"run_id", opts.Metadata.ID.RunID.String(),
			)
		}
	}
}

func (e *executor) finalizeEvents(ctx context.Context, opts execution.FinalizeOpts, extraEvents []event.Event) error {
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

	// Append extra events (e.g. inngest/deferred.schedule) AFTER the invoke
	// goroutine loop so they aren't dispatched to HandleInvokeFinish.
	freshEvents = append(freshEvents, extraEvents...)

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

	compactHeaders := headers.Compact(headers.Redact(h))

	rawAttrs := meta.NewAttrSet()
	meta.AddAttr(rawAttrs, meta.Attrs.IsFunctionOutput, inngestgo.Ptr(true))
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseHeaders, &compactHeaders)
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseStatusCode, &res.StatusCode)
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseOutputSize, inngestgo.Ptr(len(res.Body)))
	// XXX: We always wrap trace output with {"data":T} or {"error":T} for consistency with steps.
	meta.AddAttr(rawAttrs, meta.Attrs.StepOutput, inngestgo.Ptr(util.DataWrap([]byte(res.Body))))

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
