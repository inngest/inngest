package executor

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
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

// Cancel cancels an in-progress function run, preventing any enqueued or future
// steps from running.
func (s *scheduler) Cancel(ctx context.Context, id sv2.ID, r execution.CancelRequest) error {
	l := s.log.With(
		"run_id", id.RunID.String(),
		"workflow_id", id.FunctionID.String(),
	)

	md, err := s.smv2.LoadMetadata(ctx, id)
	if err == sv2.ErrMetadataNotFound || errors.Is(err, state.ErrRunNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to load run: %w", err)
	}

	ctx = tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
		Identifier: md.ID,
		Attempt:    0,
	})

	// We need events to finalize the function.
	evts, err := s.smv2.LoadEvents(ctx, id)
	if errors.Is(err, state.ErrEventNotFound) {
		// If the event has gone, another thread cancelled the function.
		l.Warn("cancel: events not found but metadata exists, skipping finalize")
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to load run events: %w", err)
	}

	// We need the function slug.
	f, err := s.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		return fmt.Errorf("unable to load function: %w", err)
	}

	if err := s.Finalize(ctx, execution.FinalizeOpts{
		Metadata: md,
		Response: execution.FinalizeResponse{
			Type:           execution.FinalizeResponseDriver,
			DriverResponse: state.DriverResponse{},
		},
		Optional: execution.FinalizeOptional{
			FnSlug:      f.Function.GetSlug(),
			InputEvents: evts,
			Cancel:      true,
			Reason:      "cancel",
		},
	}); err != nil {
		l.Error("error running finish handler", "error", err)
	}
	for _, lc := range s.lifecycles {
		go lc.OnFunctionCancelled(context.WithoutCancel(ctx), md, r, evts)
	}

	return nil
}

// Finalize performs run finalization, which involves sending the function
// finished/failed event and deleting state.
func (s *scheduler) Finalize(ctx context.Context, opts execution.FinalizeOpts) error {
	ctx = context.WithoutCancel(ctx)
	l := logger.StdlibLogger(ctx)

	var endTimeOffset time.Duration
	status := opts.Status()
	if status == enums.StepStatusCancelled {
		endTimeOffset = cancelationGracePeriod
	}

	err := s.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		EndTime:       s.now(),
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
		opts.Optional.InputEvents, err = s.smv2.LoadEvents(ctx, opts.Metadata.ID)
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
	if s.semaphoreManager == nil && len(opts.Metadata.Config.Semaphores) > 0 {
		l.Error(
			"semaphore manager is nil but run holds semaphores, leading to deadlock",
			"run_id", opts.Metadata.ID.RunID,
			"semaphores", len(opts.Metadata.Config.Semaphores),
		)
	}

	if s.semaphoreManager != nil && len(opts.Metadata.Config.Semaphores) > 0 {
		for _, sem := range opts.Metadata.Config.Semaphores {
			if sem.Release != constraintapi.SemaphoreReleaseManual {
				continue
			}
			// Retry semaphore release — a failure here means the semaphore is permanently
			// held, which deadlocks all future runs waiting on capacity.
			_, releaseErr := util.WithRetry(ctx, "release-semaphore", func(ctx context.Context) (struct{}, error) {
				return struct{}{}, s.semaphoreManager.ReleaseSemaphore(
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

	// Delete the function state in every case.
	err = s.smv2.Delete(ctx, opts.Metadata.ID)
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

	s.finalizeRemoveJobs(ctx, opts)

	// finalizeEvents creates function finished events, and also attempts to fast-resume
	// any parent function that invoked this run.
	return s.finalizeEvents(ctx, opts)
}

// finalizeRemoveJobs removes any other jobs for a finalized run, as the function is
// marked as finished and no other jobs need to execute.
func (s *scheduler) finalizeRemoveJobs(ctx context.Context, opts execution.FinalizeOpts) {
	l := logger.StdlibLogger(ctx)

	if s.shardFinder == nil {
		return
	}

	shard, err := s.shardFinder(
		ctx,
		opts.Metadata.ID.Tenant.AccountID,
		nil,
	)
	if err != nil {
		return
	}

	// We may be cancelling an in-progress run.  If that's the case, we want to delete any
	// outstanding jobs from the queue, if possible.
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

func (s *scheduler) finalizeEvents(ctx context.Context, opts execution.FinalizeOpts) error {
	if s.finishHandler == nil {
		return nil
	}

	var (
		isInvoke bool
		fnSlug   = opts.Optional.FnSlug
		evts     = opts.Optional.InputEvents
	)

	if fnSlug == "" {
		fn, err := s.fl.LoadFunction(ctx, opts.Metadata.ID.Tenant.EnvID, opts.Metadata.ID.FunctionID)
		if err != nil {
			return err
		}
		fnSlug = fn.Function.Slug
	}

	inputEvents := make([]event.Event, len(evts))
	for n, e := range evts {
		evt, err := event.NewEvent(e)
		if err != nil {
			return err
		}
		inputEvents[n] = *evt
	}

	now := s.now()
	base := &functionFinishedData{
		FunctionID: fnSlug,
		RunID:      opts.Metadata.ID.RunID,
		Events:     inputEvents,
	}
	base.setResponse(opts.Response)

	freshEvents := []event.Event{}
	for n, runEvt := range inputEvents {
		if runEvt.Name == event.FnFailedName || runEvt.Name == event.FnFinishedName {
			continue
		}

		invokeID := correlationID(runEvt)
		if invokeID == nil && n > 0 {
			continue
		}

		isInvoke = true

		copied := *base
		copied.Event = runEvt.Map()
		copied.InvokeCorrelationID = invokeID
		data := copied.Map()

		data[consts.InngestEventDataPrefix] = map[string]any{
			"status": opts.Status(),
		}

		freshEvents = append(freshEvents, event.Event{
			ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
			Name:      event.FnFinishedName,
			Timestamp: now.UnixMilli(),
			Data:      data,
		})

		switch opts.Status() {
		case enums.StepStatusCancelled:
			freshEvents = append(freshEvents, event.Event{
				ID:        opts.Metadata.ID.RunID.String(),
				Name:      event.FnCancelledName,
				Timestamp: now.UnixMilli(),
				Data:      data,
			})
		case enums.StepStatusFailed:
			freshEvents = append(freshEvents, event.Event{
				ID:        opts.Metadata.ID.RunID.String(),
				Name:      event.FnFailedName,
				Timestamp: now.UnixMilli(),
				Data:      data,
			})
		}
	}

	// For each event, if this has a correlation ID attempt to resume
	// the invoke parent within a goroutine.  If no fast-resume callback is
	// configured, the regular event handler pub/sub flow handles it.
	if isInvoke && s.invokeFinishHandler != nil {
		for _, evt := range freshEvents {
			tracked := event.BaseTrackedEvent{
				ID:          ulid.MustParse(evt.ID),
				Event:       evt,
				AccountID:   opts.Metadata.ID.Tenant.AccountID,
				WorkspaceID: opts.Metadata.ID.Tenant.EnvID,
			}
			service.Go(func() {
				err := s.invokeFinishHandler(context.WithoutCancel(ctx), tracked)
				if err != nil && !errors.Is(err, ErrNoCorrelationID) {
					logger.From(ctx).Error("error fast resuming invoke", "error", err)
				}
			})
		}
	}

	return s.finishHandler(ctx, opts.Metadata.ID, freshEvents)
}

func finalizeSpanAttributes(f execution.FinalizeOpts) *meta.SerializableAttrs {
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
	meta.AddAttr(rawAttrs, meta.Attrs.StepOutput, inngestgo.Ptr(util.DataWrap(res.Body)))

	return rawAttrs
}

func runCompleteAttrs(gen state.GeneratorOpcode) *meta.SerializableAttrs {
	rawAttrs := meta.NewAttrSet()

	meta.AddAttr(rawAttrs, meta.Attrs.IsFunctionOutput, inngestgo.Ptr(true))
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseStatusCode, inngestgo.Ptr(200))
	meta.AddAttr(rawAttrs, meta.Attrs.ResponseOutputSize, inngestgo.Ptr(len(gen.Data)))
	meta.AddAttr(rawAttrs, meta.Attrs.StepOutput, inngestgo.Ptr(util.DataWrap(gen.Data)))

	rawAttrs = rawAttrs.Merge(tracing.GeneratorAttrs(&gen))

	return rawAttrs
}
