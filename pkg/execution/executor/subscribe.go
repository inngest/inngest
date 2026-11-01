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
// state and satisfied by the first message it wants.

// inboxMessageValue is the value saved for a message step.
type inboxMessageValue struct {
	Data  map[string]any `json:"data"`
	Inbox string         `json:"inngest/inbox"`
	Src   []string       `json:"src"`
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

	// A listen op adds sources to an inbox that is open; its messages belong to
	// that inbox, and its pauses are named by the listen op.
	inbox := gen.ID
	if opts.Inbox != "" {
		inbox = opts.Inbox
	}

	pauseIDs := []string{}

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
			Expression:  source.If,
			MaxAttempts: runCtx.MaxAttempts(),
			Subscribe:   &state.PauseSubscribe{Inbox: inbox},
			CreatedAt:   now,
		}

		idx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: source.Event}

		_, err := util.WithRetry(ctx, "pause.handleGeneratorSubscribe", func(ctx context.Context) (int, error) {
			return e.pm.Write(ctx, idx, &pause)
		}, util.NewRetryConf(util.WithRetryConfRetryableErrors(pauses.WritePauseRetryableError)))
		if err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
			return err
		}

		pauseIDs = append(pauseIDs, pause.ID.String())
	}

	// The open record names the pauses of the sources, so that the SDK can tell
	// which messages came through them.
	record, err := json.Marshal(map[string]any{"data": map[string]any{"pauses": pauseIDs}})
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

	e.emitStepSpan(ctx, runCtx, &gen, nil, nil)

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

func (e *executor) handleGeneratorSubscribeWait(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup) error {
	opts, err := gen.SubscribeWaitOpts()
	if err != nil {
		return fmt.Errorf("unable to parse subscribe wait opts: %w", err)
	}

	md := runCtx.Metadata()

	// The timer is queued before the wait is recorded, so a recorded wait
	// always has one. Queuing it again is a no-op.
	timer := opts.Timeout != ""
	if timer {
		if err := e.queueInboxTimer(ctx, runCtx, gen, edge, group, opts.Timeout); err != nil {
			return err
		}
	}

	wake, err := e.smv2.RecordInboxWait(ctx, md.ID, sv2.InboxWait{
		ID:                  gen.ID,
		Inbox:               opts.Inbox,
		Want:                opts.Want,
		Seen:                opts.Seen,
		Timer:               timer,
		Incoming:            edge.Edge.Incoming,
		GroupID:             runCtx.GroupID(),
		MaxAttempts:         runCtx.MaxAttempts(),
		ParallelMode:        gen.ParallelMode(),
		ParallelCoalesceKey: group.ParallelCoalesceKey,
	})
	if err != nil {
		return err
	}

	if err := runCtx.ReleaseCapacityLease(); err != nil {
		logger.StdlibLogger(ctx).ReportError(err, "could not release capacity lease early")
	}

	if wake == nil {
		return nil
	}

	return e.wakeInboxRead(ctx, md.ID, *wake)
}

// inboxTimerJobID is the id of the queue item that times a read out. The wake
// that follows a timeout has an id of its own, so it is never deduped against
// the timer that is still running when it is queued.
func inboxTimerJobID(md *sv2.Metadata, wait string) string {
	return fmt.Sprintf("%s-%s-timer", md.IdempotencyKey(), wait)
}

// queueInboxTimer queues the item that times a read out when its timeout
// passes. It is a pause timeout item, like the one of waitForEvent.
func (e *executor) queueInboxTimer(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, timeout string) error {
	expires, err := state.WaitForEventOpts{Timeout: timeout}.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse the timeout of an inbox read: %w", err)
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
		Subscribe:           &state.PauseSubscribe{},
		CreatedAt:           e.now(),
	}

	err = e.queue.Enqueue(ctx, queue.Item{
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
	id := sv2.IDFromPause(pause)

	wake, err := e.smv2.TimeoutInboxWait(ctx, id, pause.DataKey, []byte("null"))
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
			suffix := wake.MessageStep
			if wake.TimedOut {
				suffix = "timeout"
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

	// The timer that is running is the one that found this read; any other
	// timer is no longer needed.
	if wait.Timer && !wake.TimedOut {
		if err := e.dequeueInboxTimer(ctx, md, wait.ID); err != nil {
			return err
		}
	}

	return e.smv2.DeleteInboxWait(ctx, id, wait.ID)
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
	src      []string
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
		// Sources are listed in a stable order, so a redelivery saves the same
		// value.
		slices.Sort(arrival.src)

		value, err := json.Marshal(inboxMessageValue{
			Data:  evt.GetEvent().Map(),
			Inbox: arrival.inbox,
			Src:   arrival.src,
		})
		if err != nil {
			goerr = errors.Join(goerr, fmt.Errorf("error encoding inbox message: %w", err))

			continue
		}

		stepID := inboxMessageStepID(arrival.inbox, evt.GetInternalID())

		wakes, err := e.smv2.SaveInboxMessage(ctx, arrival.id, sv2.InboxMessage{
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
		if errors.Is(err, state.ErrRunNotFound) {
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

		for _, wake := range wakes {
			if err := e.wakeInboxRead(ctx, arrival.id, wake); err != nil {
				goerr = errors.Join(goerr, err)
			}
		}
	}

	return goerr
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
