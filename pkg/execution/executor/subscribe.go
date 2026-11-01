package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

// An inbox is opened by an OpcodeSubscribe op: one long-lived pause per source,
// then a record that the inbox is open. A matching event is saved as a message
// step of the inbox, and wakes a read only while one is waiting. A read with
// nothing unread reports an OpcodeSubscribeWait op, which is recorded in run
// state and satisfied by the first message of the inbox.

// inboxMessageValue is the value saved for a message step.
type inboxMessageValue struct {
	Data  map[string]any `json:"data"`
	Inbox string         `json:"inngest/inbox"`
}

// An arrival that does not fit in the run's step or state size bounds saves
// nothing and queues one call for the run instead. The call finds the run at its
// step bound, or carries a mark that the state did not fit, and ends the run
// with the usual error.
const (
	inboxOverflowKey   = "inboxOverflow"
	inboxOverflowState = "state"
)

// inboxMessageStepID is the id of the step a message is saved as.
func inboxMessageStepID(inbox string, eventID ulid.ULID) string {
	sum := sha1.Sum([]byte(inbox + eventID.String()))

	return hex.EncodeToString(sum[:])
}

// sourcePauseID is the id of the pause of one source of an inbox. It is
// deterministic, so opening an inbox again is a no-op.
func sourcePauseID(runID ulid.ULID, inbox string, source state.SubscribeSource) uuid.UUID {
	expr := ""
	if source.If != nil {
		expr = *source.If
	}

	return inngest.DeterministicSha1UUID(runID.String() + inbox + source.Event + expr)
}

func (e *executor) handleGeneratorSubscribe(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	opts, err := gen.SubscribeOpts()
	if err != nil {
		return fmt.Errorf("unable to parse subscribe opts: %w", err)
	}

	md := runCtx.Metadata()
	now := e.now()
	opcode := gen.Op.String()

	inbox := gen.ID

	sources := []inboxSourcePause{}

	for _, source := range opts.Sources {
		if source.If != nil {
			if err := expressions.Validate(ctx, expressions.DefaultRestrictiveValidationPolicy(), *source.If); err != nil && errors.Is(err, expressions.ErrCompileFailed) {
				return state.WrapInStandardError(
					err,
					"InvalidExpression",
					"Subscribe source If expression failed to compile",
					err.Error(),
				)
			}
		}

		// The pause matches on the trigger's values, but its id hashes the
		// expression as written: the same on every retry and replay, and the
		// same as before sources could reference the trigger.
		matchIf, err := e.interpolateTriggerEvent(ctx, runCtx, source.If)
		if err != nil {
			return fmt.Errorf("error interpolating subscribe source expression: %w", err)
		}

		eventName := source.Event
		pause := state.Pause{
			ID:          sourcePauseID(md.ID.RunID, gen.ID, source),
			WorkspaceID: md.ID.Tenant.EnvID,
			Identifier:  sv2.NewPauseIdentifier(md.ID),
			GroupID:     runCtx.GroupID(),
			Outgoing:    gen.ID,
			Incoming:    edge.Edge.Incoming,
			StepName:    gen.UserDefinedName(),
			Opcode:      &opcode,
			// The pause store needs a finite expiry; an inbox ends at close or
			// when the run does.
			Expires:     state.Time(now.Add(consts.MaxWaitForEventTimeout)),
			Event:       &eventName,
			Expression:  matchIf,
			MaxAttempts: runCtx.MaxAttempts(),
			Subscribe:   &state.PauseSubscribe{Inbox: inbox},
			CreatedAt:   now,
		}

		idx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: source.Event}

		_, err = util.WithRetry(ctx, "pause.handleGeneratorSubscribe", func(ctx context.Context) (int, error) {
			return e.pm.Write(ctx, idx, &pause)
		}, util.NewRetryConf(util.WithRetryConfRetryableErrors(pauses.WritePauseRetryableError)))
		if err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
			return err
		}

		sources = append(sources, inboxSourcePause{ID: pause.ID.String(), Event: source.Event})
	}

	// The open record names the pauses of the sources, so that a close can
	// delete them.
	record, err := json.Marshal(map[string]any{"data": map[string]any{"pauses": sources}})
	if err != nil {
		return fmt.Errorf("error encoding the record of an inbox: %w", err)
	}

	hasPendingSteps, err := e.smv2.SaveStep(ctx, md.ID, gen.ID, record)
	if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
		err = nil
	}
	if err != nil {
		return err
	}

	e.emitInboxOpenSpan(ctx, runCtx, gen)

	if err := runCtx.ReleaseCapacityLease(); err != nil {
		logger.StdlibLogger(ctx).ReportError(err, "could not release capacity lease early")
	}

	groupID := uuid.New().String()

	return e.maybeEnqueueDiscoveryStep(
		state.WithGroupID(ctx, groupID),
		runCtx,
		gen,
		edge,
		groupID,
		hasPendingSteps,
		runCtx.LifecycleItem().ParallelCoalesceKey,
	)
}

// inboxSourcePause is a source pause as the open record of an inbox names it.
// The event name is part of where the pause is stored.
type inboxSourcePause struct {
	ID    string `json:"id"`
	Event string `json:"event"`
}

// emitInboxOpenSpan draws the row of an open op. It runs until a close
// ends it; an inbox that is never closed is drawn ended with its run when the
// trace is read.
func (e *executor) emitInboxOpenSpan(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode) {
	md := runCtx.Metadata()
	attrs := tracing.GeneratorAttrs(&gen)
	tracing.AddMetadataTenantAttrs(attrs, md.ID)
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, new(enums.StepStatusRunning))

	queuedAt, scheduledAt, startedAt, _ := e.opcodeTiming(ctx, runCtx, &gen)
	tracing.AddTimingAttrs(attrs, queuedAt, scheduledAt, startedAt, time.Time{})

	lifecycleItem := runCtx.LifecycleItem()
	_, err := e.tracerProvider.CreateSpan(ctx, meta.SpanNameStep, &tracing.CreateSpanOptions{
		Seed:       tracing.FinalizedStepDynamicSeed(gen.ID),
		Debug:      &tracing.SpanDebugData{Location: "executor.emitInboxOpenSpan"},
		Metadata:   md,
		QueueItem:  &lifecycleItem,
		Parent:     runCtx.RootSpan(),
		StartTime:  queuedAt,
		Attributes: attrs,
	})
	if err != nil {
		e.log.Debug("error creating the row of an inbox", "error", err)
	}
}

func (e *executor) handleGeneratorSubscribeWait(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup) error {
	opts, err := gen.SubscribeWaitOpts()
	if err != nil {
		return fmt.Errorf("unable to parse subscribe wait opts: %w", err)
	}

	md := runCtx.Metadata()

	// The timer is queued before the wait is recorded, so a recorded wait
	// always has one. Queuing it again is a no-op. A read with no timeout
	// still has one, so the run holds a queue item while it waits: the
	// function's singleton lock is released when the run has none.
	//
	// A read that never waits (`drain()`) is answered at once and needs none.
	drain := opts.Batch != nil && opts.Batch.Drain
	if !drain {
		if err := e.queueInboxTimer(ctx, runCtx, gen, edge, group, opts.Inbox, opts.Timeout); err != nil {
			return err
		}
	}

	var batch *sv2.InboxBatch
	if opts.Batch != nil {
		batch = &sv2.InboxBatch{Drain: opts.Batch.Drain, Unread: opts.Batch.Unread, Max: opts.Batch.Max}
	}

	wake, err := e.smv2.RecordInboxWait(ctx, md.ID, sv2.InboxWait{
		ID:                  gen.ID,
		Inbox:               opts.Inbox,
		Seen:                opts.Seen,
		Timer:               !drain,
		Batch:               batch,
		Incoming:            edge.Edge.Incoming,
		GroupID:             runCtx.GroupID(),
		MaxAttempts:         runCtx.MaxAttempts(),
		ParallelMode:        gen.ParallelMode(),
		ParallelCoalesceKey: group.ParallelCoalesceKey,
	})
	if err != nil {
		return err
	}

	e.emitInboxWaitSpan(ctx, runCtx, gen, group, wake == nil)

	if err := runCtx.ReleaseCapacityLease(); err != nil {
		logger.StdlibLogger(ctx).ReportError(err, "could not release capacity lease early")
	}

	if wake == nil {
		return nil
	}

	return e.wakeInboxRead(ctx, md.ID, *wake)
}

// emitInboxWaitSpan draws the row of a read: waiting if it is, or complete if
// recording it found its message already there. The same row is drawn again
// when the SDK reports the wait again.
func (e *executor) emitInboxWaitSpan(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, group OpcodeGroup, waiting bool) {
	attrs := tracing.GeneratorAttrs(&gen)
	md := runCtx.Metadata()
	tracing.AddMetadataTenantAttrs(attrs, md.ID)

	handledAt := e.opcodeHandledAt(group)
	status := enums.StepStatusCompleted
	if waiting {
		tracing.AddTimingAttrs(attrs, handledAt, handledAt, time.Time{}, time.Time{})
		status = enums.StepStatusWaiting
	} else {
		tracing.AddTimingAttrs(attrs, handledAt, handledAt, handledAt, e.now())
	}
	meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &status)

	lifecycleItem := runCtx.LifecycleItem()
	_, err := e.tracerProvider.CreateSpan(ctx, meta.SpanNameStep, &tracing.CreateSpanOptions{
		Seed:        tracing.FinalizedStepDynamicSeed(gen.ID),
		FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
		Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorSubscribeWait"},
		Metadata:    md,
		QueueItem:   &lifecycleItem,
		Parent:      runCtx.RootSpan(),
		StartTime:   handledAt,
		Attributes:  attrs,
	})
	if err != nil {
		e.log.Debug("error creating span for an inbox read", "error", err)
	}
}

// handleGeneratorSubscribeRead draws a read that was answered from state. It
// is a lazy op: it saves nothing and moves nothing, and reporting it again
// draws the same spans.
func (e *executor) handleGeneratorSubscribeRead(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	if err := tracing.InboxRead(ctx, e.tracerProvider, runCtx.Metadata(), gen, e.now()); err != nil {
		logger.StdlibLogger(ctx).Warn("could not draw an inbox read", "error", err)
	}

	return e.enqueueLazyOpFallback(ctx, runCtx, gen, edge, "SubscribeRead")
}

// handleGeneratorSubscribeClose closes an inbox: it deletes the pauses of the
// inbox's sources, saves a cut at the last message the inbox has as the op's
// result, and ends the inbox's waiting read. The result is saved in the same
// call that takes the read out of the pending set, before its wake is queued,
// so the run that the wake starts finds the close recorded. A close that
// crashed in between is repaired by being handled again.
func (e *executor) handleGeneratorSubscribeClose(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	opts, err := gen.SubscribeCloseOpts()
	if err != nil {
		return fmt.Errorf("unable to parse subscribe close opts: %w", err)
	}

	md := runCtx.Metadata()

	if err := e.deleteSourcePauses(ctx, md, opts.Inbox); err != nil {
		return err
	}

	wake, hasPendingSteps, err := e.smv2.CloseInbox(ctx, md.ID, sv2.InboxClose{
		StepID: gen.ID,
		Inbox:  opts.Inbox,
	})
	if err != nil {
		return err
	}

	e.emitStepSpan(ctx, runCtx, &gen, nil, nil)
	e.endInboxRow(ctx, md, opts.Inbox)

	if err := runCtx.ReleaseCapacityLease(); err != nil {
		logger.StdlibLogger(ctx).ReportError(err, "could not release capacity lease early")
	}

	// The wake continues the run; a close that ended no read does it itself.
	if wake != nil {
		return e.wakeInboxRead(ctx, md.ID, *wake)
	}

	groupID := uuid.New().String()

	return e.maybeEnqueueDiscoveryStep(
		state.WithGroupID(ctx, groupID),
		runCtx,
		gen,
		edge,
		groupID,
		hasPendingSteps,
		runCtx.LifecycleItem().ParallelCoalesceKey,
	)
}

// deleteSourcePauses deletes the pauses of an inbox's sources, which its open
// record names. A pause that is already gone is a close that ran before.
func (e *executor) deleteSourcePauses(ctx context.Context, md *sv2.Metadata, inbox string) error {
	saved, err := e.smv2.LoadStepsWithIDs(ctx, md.ID, []string{inbox})
	if err != nil {
		return fmt.Errorf("error loading the record of an inbox to close: %w", err)
	}

	var record struct {
		Data struct {
			Pauses []inboxSourcePause `json:"pauses"`
		} `json:"data"`
	}
	if raw, ok := saved[inbox]; ok {
		if err := json.Unmarshal(raw, &record); err != nil {
			return fmt.Errorf("error decoding the record of an inbox to close: %w", err)
		}
	}

	for _, source := range record.Data.Pauses {
		id, err := uuid.Parse(source.ID)
		if err != nil {
			return fmt.Errorf("invalid source pause id %q: %w", source.ID, err)
		}

		idx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: source.Event}

		pause, err := e.pm.PauseByID(ctx, idx, id)
		if errors.Is(err, state.ErrPauseNotFound) || (err == nil && pause == nil) {
			continue
		}
		if err != nil {
			return fmt.Errorf("error loading a source pause to close: %w", err)
		}

		if err := e.pm.Delete(ctx, idx, *pause); err != nil {
			return fmt.Errorf("error deleting a source pause: %w", err)
		}

		if e.exprAggregator != nil {
			if err := e.exprAggregator.RemovePause(ctx, pause); err != nil {
				logger.StdlibLogger(ctx).Warn("error removing a source pause from the aggregator", "error", err)
			}
		}
	}

	return nil
}

// inboxTimerJobID is the id of the queue item that times a read out. The wake
// that follows a timeout has an id of its own, so it is never deduped against
// the timer that is still running when it is queued.
func inboxTimerJobID(md *sv2.Metadata, wait string) string {
	return fmt.Sprintf("%s-%s-timer", md.IdempotencyKey(), wait)
}

// queueInboxTimer queues the item that times a read out when its timeout
// passes. It is a pause timeout item, like the one of waitForEvent.
//
// A read with no timeout (an empty `timeout`) is timed at the furthest a wait
// can be: its timer does nothing when it fires, and exists to keep the run's
// queue item.
func (e *executor) queueInboxTimer(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, inbox string, timeout string) error {
	untimed := timeout == ""

	expires := e.now().Add(consts.MaxWaitForEventTimeout)
	if !untimed {
		var err error

		expires, err = state.WaitForEventOpts{Timeout: timeout}.Expires()
		if err != nil {
			return fmt.Errorf("unable to parse the timeout of an inbox read: %w", err)
		}
	}

	md := runCtx.Metadata()
	opcode := gen.Op.String()
	jobID := inboxTimerJobID(md, gen.ID)

	pause := state.Pause{
		ID:                  inngest.DeterministicSha1UUID(md.ID.RunID.String() + gen.ID),
		WorkspaceID:         md.ID.Tenant.EnvID,
		Identifier:          sv2.NewPauseIdentifier(md.ID),
		GroupID:             runCtx.GroupID(),
		Outgoing:            gen.ID,
		Incoming:            edge.Edge.Incoming,
		StepName:            gen.UserDefinedName(),
		Opcode:              &opcode,
		Expires:             state.Time(expires),
		DataKey:             gen.ID,
		MaxAttempts:         runCtx.MaxAttempts(),
		ParallelMode:        gen.ParallelMode(),
		ParallelCoalesceKey: group.ParallelCoalesceKey,
		Subscribe:           &state.PauseSubscribe{Inbox: inbox, Untimed: untimed},
		CreatedAt:           e.now(),
	}

	err := e.queue.Enqueue(ctx, queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           md.ID.Tenant.EnvID,
		GroupID:               runCtx.GroupID(),
		Kind:                  queue.KindPause,
		Identifier:            sv2.V1FromMetadata(*md),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Payload: queue.PayloadPauseTimeout{
			PauseID: pause.ID,
			Pause:   pause,
		},
		Metadata:     make(map[string]any),
		ParallelMode: gen.ParallelMode(),
	}, expires, queue.EnqueueOpts{})
	if err != nil && !errors.Is(err, queue.ErrQueueItemExists) {
		return fmt.Errorf("error enqueueing the timer of an inbox read: %w", err)
	}

	return nil
}

// dequeueInboxTimer takes a read's timer off the queue. A timer that is already
// running finds the read gone and does nothing.
func (e *executor) dequeueInboxTimer(ctx context.Context, md sv2.Metadata, wait string) error {
	if e.shards == nil {
		return nil
	}

	shard, err := e.shards.Resolve(ctx, queue.Scope{
		AccountID:  md.ID.Tenant.AccountID,
		EnvID:      md.ID.Tenant.EnvID,
		FunctionID: md.ID.FunctionID,
	}, nil)
	if err != nil {
		return fmt.Errorf("could not find the shard of an inbox read's timer: %w", err)
	}

	err = e.queue.Dequeue(ctx, shard.Name(), queue.QueueItem{
		ID:         queue.HashID(ctx, inboxTimerJobID(&md, wait)),
		FunctionID: md.ID.FunctionID,
		Data: queue.Item{
			Kind:       queue.KindPause,
			Identifier: sv2.V1FromMetadata(md),
		},
	})
	if err != nil && !errors.Is(err, queue.ErrQueueItemNotFound) {
		return fmt.Errorf("error dequeueing the timer of an inbox read: %w", err)
	}

	return nil
}

// timeoutInboxRead is what the timer of a read does when it runs: if the read
// is still waiting it gets a null and the run is woken, otherwise nothing.
func (e *executor) timeoutInboxRead(ctx context.Context, pause state.Pause) error {
	// The timer of a read with no timeout only holds the run's queue item.
	if pause.Subscribe != nil && pause.Subscribe.Untimed {
		return nil
	}

	id := sv2.IDFromPause(pause)

	wake, err := e.smv2.TimeoutInboxWait(ctx, id, pause.Subscribe.Inbox, pause.DataKey, []byte("null"))
	if err != nil {
		if errors.Is(err, state.ErrRunNotFound) {
			return nil
		}

		return err
	}
	if wake == nil {
		return nil
	}

	return e.wakeInboxRead(ctx, id, *wake)
}

// wakeInboxRead queues the step after a read that a message satisfied, then
// deletes the read's record. The record outlives the queued wake, so a crash in
// between is repaired by the event being delivered again.
func (e *executor) wakeInboxRead(ctx context.Context, id sv2.ID, wake sv2.InboxWake) error {
	wait := wake.Wait

	md, err := e.smv2.LoadMetadata(ctx, id)
	if err != nil {
		if errors.Is(err, state.ErrRunNotFound) {
			return nil
		}

		return fmt.Errorf("error loading metadata to wake an inbox read: %w", err)
	}

	ctx = tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
		Identifier:  id,
		Attempt:     0,
		MaxAttempts: wait.MaxAttempts,
		GroupID:     new(wait.GroupID),
	})

	if shouldEnqueueDiscovery(wake.HasPending, wait.ParallelMode) {
		if err := e.maybeResetForceStepPlan(ctx, &md); err != nil {
			return fmt.Errorf("error resetting force step plan: %w", err)
		}

		// A parallel batch shares one discovery job; a lone read has its own,
		// named by the read and the message that woke it.
		var jobID string
		if wait.ParallelCoalesceKey != "" && wait.ParallelMode != enums.ParallelModeRace {
			jobID = fmt.Sprintf("%s-%s-discover", md.ID.RunID, wait.ParallelCoalesceKey)
		} else {
			suffix := "message"
			switch {
			case wake.TimedOut:
				suffix = "timeout"
			case wake.Closed:
				suffix = "close"
			}

			jobID = fmt.Sprintf("%s-%s-%s", md.IdempotencyKey(), wait.ID, suffix)
		}

		nextItem := queue.Item{
			JobID:                 &jobID,
			GroupID:               uuid.New().String(),
			WorkspaceID:           id.Tenant.EnvID,
			Kind:                  queue.KindEdge,
			Identifier:            sv2.V1FromMetadata(md),
			PriorityFactor:        md.Config.PriorityFactor,
			CustomConcurrencyKeys: md.Config.CustomConcurrencyKeys,
			Semaphores:            stepSemaphores(md),
			MaxAttempts:           wait.MaxAttempts,
			Payload: queue.PayloadEdge{
				Edge: inngest.Edge{Outgoing: wait.ID, Incoming: wait.Incoming},
			},
			Metadata: make(map[string]any),
		}

		nextStepSpan, err := e.tracerProvider.CreateDroppableSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:  []map[string]any{nextItem.Metadata},
				Debug:     &tracing.SpanDebugData{Location: "executor.wakeInboxRead"},
				Metadata:  &md,
				Parent:    tracing.RunSpanRefFromMetadata(&md),
				QueueItem: &nextItem,
			},
		)
		if err != nil {
			e.log.Debug("error creating span for next step after an inbox read", "error", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, e.now(), queue.EnqueueOpts{})
		switch {
		case err == nil:
			_ = nextStepSpan.Send()
		case errors.Is(err, queue.ErrQueueItemExists):
			nextStepSpan.Drop()
		default:
			nextStepSpan.Drop()

			return fmt.Errorf("error enqueueing after an inbox read: %w", err)
		}
	}

	e.endInboxWaitSpan(ctx, md, wait.ID, wake)

	// The timer that is running is the one that found this read; any other
	// timer is no longer needed.
	if wait.Timer && !wake.TimedOut {
		if err := e.dequeueInboxTimer(ctx, md, wait.ID); err != nil {
			return err
		}
	}

	return e.smv2.DeleteInboxWait(ctx, id, wait.Inbox, wait.ID)
}

// endInboxRow ends the row of the open op a close names.
func (e *executor) endInboxRow(ctx context.Context, md *sv2.Metadata, row string) {
	err := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		Debug:      &tracing.SpanDebugData{Location: "executor.endInboxRow"},
		Metadata:   md,
		Status:     enums.StepStatusCompleted,
		TargetSpan: tracing.FinalizedStepSpanRefFromMetadataAndStepID(md, row),
		EndTime:    e.now(),
	})
	if err != nil {
		e.log.Debug("error ending the row of an inbox", "error", err)
	}
}

// inboxReadEnd says how a read ended, for its row: a message completed it, its
// timeout passed (timed out, with the wait expired), or its inbox was closed
// with nothing to read (cancelled, as a read that lost a race ends).
func inboxReadEnd(wake sv2.InboxWake) (status enums.StepStatus, expired bool) {
	switch {
	case wake.TimedOut:
		return enums.StepStatusTimedOut, true
	case wake.Closed:
		return enums.StepStatusCancelled, false
	}

	return enums.StepStatusCompleted, false
}

// endInboxWaitSpan ends the row of a read that a message, its timeout or a
// close woke.
func (e *executor) endInboxWaitSpan(ctx context.Context, md sv2.Metadata, wait string, wake sv2.InboxWake) {
	status, expired := inboxReadEnd(wake)

	var attrs *meta.SerializableAttrs
	if expired {
		attrs = meta.NewAttrSet(meta.Attr(meta.Attrs.StepWaitExpired, &expired))
	}

	err := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		Attributes: attrs,
		Debug:      &tracing.SpanDebugData{Location: "executor.endInboxWaitSpan"},
		Metadata:   &md,
		Status:     status,
		TargetSpan: tracing.FinalizedStepSpanRefFromMetadataAndStepID(&md, wait),
		EndTime:    e.now(),
	})
	if err != nil {
		e.log.Debug("error ending the span of an inbox read", "error", err)
	}
}

// inboxArrivals collects the pauses of inbox sources that one event matched, so
// that each inbox saves the event once however many of its sources matched.
// Both pause loops handle every pause in a goroutine of its own, so the pauses
// are gathered here and saved after the loop has finished.
type inboxArrivals struct {
	mu       sync.Mutex
	arrivals map[string]*inboxArrival
}

type inboxArrival struct {
	id    sv2.ID
	inbox string
	// incoming is the step the inbox's open op leads to.
	incoming string
	// src are the pauses of the sources the event matched. They are drawn on
	// the message's tick, and not saved with the message.
	src []string
}

type inboxArrivalsKey struct{}

func withInboxArrivals(ctx context.Context) (context.Context, *inboxArrivals) {
	a := &inboxArrivals{arrivals: map[string]*inboxArrival{}}

	return context.WithValue(ctx, inboxArrivalsKey{}, a), a
}

func inboxArrivalsFrom(ctx context.Context) *inboxArrivals {
	a, _ := ctx.Value(inboxArrivalsKey{}).(*inboxArrivals)

	return a
}

func (a *inboxArrivals) add(pause *state.Pause) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := pause.Identifier.RunID.String() + pause.Subscribe.Inbox
	arrival, ok := a.arrivals[key]
	if !ok {
		arrival = &inboxArrival{
			id: sv2.ID{
				RunID:      pause.Identifier.RunID,
				FunctionID: pause.Identifier.FunctionID,
				Tenant: sv2.Tenant{
					EnvID:     pause.WorkspaceID,
					AccountID: pause.Identifier.AccountID,
				},
			},
			inbox:    pause.Subscribe.Inbox,
			incoming: pause.Incoming,
		}
		a.arrivals[key] = arrival
	}

	if !slices.Contains(arrival.src, pause.ID.String()) {
		arrival.src = append(arrival.src, pause.ID.String())
	}
}

// saveInboxMessages saves the event as a message of every inbox it matched, and
// wakes the reads that wait for it.
func (e *executor) saveInboxMessages(ctx context.Context, evt event.TrackedEvent, arrivals *inboxArrivals) error {
	var goerr error

	for _, arrival := range arrivals.arrivals {
		value, err := json.Marshal(inboxMessageValue{
			Data:  evt.GetEvent().Map(),
			Inbox: arrival.inbox,
		})
		if err != nil {
			goerr = errors.Join(goerr, fmt.Errorf("error encoding inbox message: %w", err))

			continue
		}

		stepID := inboxMessageStepID(arrival.inbox, evt.GetInternalID())

		wake, err := e.smv2.SaveInboxMessage(ctx, arrival.id, sv2.InboxMessage{
			StepID:       stepID,
			Inbox:        arrival.inbox,
			Value:        value,
			MaxSteps:     e.inboxMaxSteps(arrival.id),
			MaxStateSize: e.inboxMaxStateSize(arrival.id),
		})
		if errors.Is(err, sv2.ErrInboxStepsFull) || errors.Is(err, sv2.ErrInboxStateFull) {
			if err := e.queueInboxOverflow(ctx, arrival, errors.Is(err, sv2.ErrInboxStateFull)); err != nil {
				goerr = errors.Join(goerr, err)
			}

			continue
		}
		// A message that reaches state after its inbox closed or its run ended
		// is not wanted by anyone.
		if errors.Is(err, state.ErrRunNotFound) || errors.Is(err, sv2.ErrInboxClosed) {
			continue
		}
		if errors.Is(err, state.ErrDuplicateResponse) {
			logger.StdlibLogger(ctx).Warn("inbox message already saved with other data", "step_id", stepID, "run_id", arrival.id.RunID.String())

			continue
		}
		if err != nil {
			goerr = errors.Join(goerr, err)

			continue
		}

		e.tickInboxArrival(ctx, arrival, stepID)

		if wake != nil {
			if err := e.wakeInboxRead(ctx, arrival.id, *wake); err != nil {
				goerr = errors.Join(goerr, err)
			}
		}
	}

	return goerr
}

// tickInboxArrival draws the message as a tick on the row of the inbox. Traces
// are best effort: a failure is logged.
func (e *executor) tickInboxArrival(ctx context.Context, arrival *inboxArrival, stepID string) {
	md, err := e.smv2.LoadMetadata(ctx, arrival.id)
	if err != nil {
		logger.StdlibLogger(ctx).Warn("could not load metadata to draw an inbox arrival", "error", err)

		return
	}

	slices.Sort(arrival.src)

	if err := tracing.InboxArrivalTick(ctx, e.tracerProvider, &md, arrival.inbox, stepID, arrival.src, e.now()); err != nil {
		logger.StdlibLogger(ctx).Warn("could not draw an inbox arrival", "error", err)
	}
}

// inboxMaxSteps is the run's existing step bound, as checkStepLimit reads it.
func (e *executor) inboxMaxSteps(id sv2.ID) int {
	max := 0
	if e.steplimit != nil {
		max = e.steplimit(id)
	}
	if max == 0 {
		max = consts.DefaultMaxStepLimit
	}

	return max
}

// inboxMaxStateSize is the run's existing state size bound, as
// validateStateSize reads it. Zero when state size is not checked.
func (e *executor) inboxMaxStateSize(id sv2.ID) int {
	if e.stateSizeLimit == nil {
		return 0
	}

	max := e.stateSizeLimit(id)
	if max == 0 {
		max = consts.DefaultMaxStateSizeLimit
	}

	return max
}

// queueInboxOverflow queues the one call that ends a run an arrival did not fit
// in. The job id names the run only, so any number of arrivals queue one call,
// and a redelivery queues the same one.
func (e *executor) queueInboxOverflow(ctx context.Context, arrival *inboxArrival, stateFull bool) error {
	md, err := e.smv2.LoadMetadata(ctx, arrival.id)
	if err != nil {
		if errors.Is(err, state.ErrRunNotFound) {
			return nil
		}

		return fmt.Errorf("error loading metadata to end an overflowing run: %w", err)
	}

	jobID := fmt.Sprintf("%s-inbox-overflow", md.ID.RunID)
	item := queue.Item{
		JobID:                 &jobID,
		GroupID:               uuid.New().String(),
		WorkspaceID:           arrival.id.Tenant.EnvID,
		Kind:                  queue.KindEdge,
		Identifier:            sv2.V1FromMetadata(md),
		PriorityFactor:        md.Config.PriorityFactor,
		CustomConcurrencyKeys: md.Config.CustomConcurrencyKeys,
		Semaphores:            stepSemaphores(md),
		Payload: queue.PayloadEdge{
			Edge: inngest.Edge{Outgoing: arrival.inbox, Incoming: arrival.incoming},
		},
		Metadata: make(map[string]any),
	}
	if stateFull {
		item.Metadata[inboxOverflowKey] = inboxOverflowState
	}

	err = e.queue.Enqueue(ctx, item, e.now(), queue.EnqueueOpts{})
	if err != nil && !errors.Is(err, queue.ErrQueueItemExists) {
		return fmt.Errorf("error enqueueing the end of an overflowing run: %w", err)
	}

	return nil
}
