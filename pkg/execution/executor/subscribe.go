package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util/strtimeout"
)

// A step.subscribe inbox keeps a pause for each of its sources. A matching event
// is delivered to the inbox's field in the run's inbox hash (see
// state/redis_state/lua/subscriptionState.lua): it is buffered until the SDK asks
// for it, or, when a next op is waiting, saved as a step of the run with a
// discovery queued. The order of the message steps in the run's stack is the order
// of the messages.

// subscriptionMessagePrefix starts the ID of every message step. The SDK looks
// for it in the stack.
const subscriptionMessagePrefix = "inngest-sub:"

// subscriptionField is the field of an inbox in the run's inbox hash.
func subscriptionField(key string) string {
	sum := sha1.Sum([]byte(key))
	return hex.EncodeToString(sum[:])
}

// subscriptionMessageID is the step ID of the message an event is in an inbox.
// The same event is the same message, so a redelivery saves nothing new.
func subscriptionMessageID(key string, eventID string) string {
	return subscriptionMessagePrefix + subscriptionField(key) + ":" + eventID
}

// subscriptionClosedID is the step the inbox saves when a limit closes it. The
// SDK reads it from the stack as the end of the inbox.
func subscriptionClosedID(key string) string {
	return subscriptionMessagePrefix + "closed:" + subscriptionField(key)
}

// subscriptionPauseID is the ID of the pause of one source of an inbox.
func subscriptionPauseID(runID string, key string, stream int) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription:%s:%d", runID, key, stream))
}

// subscriptionExpiryJobID is the queue job that expires a subscription pause.
func subscriptionExpiryJobID(md *sv2.Metadata, pauseID uuid.UUID) string {
	return fmt.Sprintf("%s-subscription-expiry-%s", md.IdempotencyKey(), pauseID)
}

// subscriptionLimitError is the error a run fails with when a step.subscribe op
// exceeds a limit or is malformed.
func subscriptionLimitError(format string, args ...any) error {
	return state.WrapInStandardError(
		state.ErrSubscriptionLimit,
		state.InngestErrSubscriptionLimit,
		fmt.Sprintf(format, args...),
		"",
	)
}

func validateSubscribe(sub *state.SubscribeOpts) error {
	if sub.Key == "" {
		return subscriptionLimitError("The subscription ID is empty.")
	}
	if len(sub.Key) > consts.MaxSubscriptionKeyLength {
		return subscriptionLimitError("The subscription ID is longer than %d characters.", consts.MaxSubscriptionKeyLength)
	}
	if len(sub.Streams) > consts.MaxSubscriptionPausesPerRun {
		return subscriptionLimitError("A subscription can listen to at most %d events.", consts.MaxSubscriptionPausesPerRun)
	}
	seen := map[int]bool{}
	for _, source := range sub.Streams {
		if source.Event == "" || source.Stream < 0 || source.Stream >= consts.MaxSubscriptionPausesPerRun || seen[source.Stream] {
			return subscriptionLimitError("A subscription lists a source twice or without an event.")
		}
		seen[source.Stream] = true
	}

	return nil
}

func (e *executor) subscriptionStore() (sv2.SubscriptionStore, error) {
	store, ok := e.smv2.(sv2.SubscriptionStore)
	if !ok {
		return nil, errors.New("the run state does not support step.subscribe")
	}

	return store, nil
}

// subscriptionLimitArgs are the limits the state script holds a message to when it
// becomes a step.
func (e *executor) subscriptionLimitArgs(id sv2.ID) []string {
	stepLimit, stateLimit := 0, 0
	if e.steplimit != nil {
		stepLimit = e.steplimit(id)
	}
	if stepLimit == 0 {
		stepLimit = consts.DefaultMaxStepLimit
	}
	if e.stateSizeLimit != nil {
		stateLimit = e.stateSizeLimit(id)
	}
	if stateLimit == 0 {
		stateLimit = consts.DefaultMaxStateSizeLimit
	}

	return []string{
		strconv.Itoa(consts.MaxSubscriptionMessages),
		strconv.Itoa(consts.MaxSubscriptionMessagesPerRun),
		strconv.Itoa(stepLimit),
		strconv.Itoa(stateLimit),
	}
}

// handleGeneratorSubscribe handles the WaitForEvent ops of a step.subscribe inbox.
func (e *executor) handleGeneratorSubscribe(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, opts *state.WaitForEventOpts) error {
	sub := opts.Subscribe
	if err := validateSubscribe(sub); err != nil {
		return err
	}

	store, err := e.subscriptionStore()
	if err != nil {
		return err
	}

	md := runCtx.Metadata()
	field := subscriptionField(sub.Key)

	answer := func(output any) error {
		return e.answerSubscribeOp(ctx, runCtx, gen, edge, output)
	}

	switch sub.Mode {
	case state.SubscribeOpen:
		sources := make([]state.SubscribeStream, 0, len(sub.Streams))
		for _, source := range sub.Streams {
			if err := validateWaitExpression(ctx, source.If); err != nil {
				return err
			}
			interpolated, _, err := interpolateWaitExpression(ctx, runCtx, source.If)
			if err != nil {
				return err
			}
			sources = append(sources, state.SubscribeStream{Stream: source.Stream, Event: source.Event, If: interpolated})
		}

		var written []*state.Pause
		for _, source := range sources {
			if err := e.openSubscriptionSource(ctx, runCtx, edge, gen, sub, source, &written); err != nil {
				// The run fails, so the pauses this op wrote go with it.
				for _, pause := range written {
					_ = e.removeSubscriptionPause(ctx, md, pause)
				}
				return err
			}
		}

		return answer(map[string]any{"opened": true})

	case state.SubscribeAll:
		total, err := e.subscriptionFlush(ctx, store, md.ID, field, "flush")
		if err != nil {
			return err
		}

		return answer(map[string]any{"total": total})

	case state.SubscribeStop:
		if err := e.removeSubscriptionSources(ctx, md, sub); err != nil {
			return err
		}

		return answer(map[string]any{"stopped": true})

	case state.SubscribeClose:
		if err := e.removeSubscriptionSources(ctx, md, sub); err != nil {
			return err
		}
		total, err := e.subscriptionFlush(ctx, store, md.ID, field, "close")
		if err != nil {
			return err
		}

		return answer(map[string]any{"total": total})

	case state.SubscribeNext:
		if group.SubscribeBusy {
			// The other ops of the response give the run a discovery that runs the
			// function again, and it sends this op again if it still has nothing.
			return nil
		}

		return e.handleSubscribeNext(ctx, runCtx, gen, edge, sub)
	}

	return fmt.Errorf("unknown subscription mode %q", sub.Mode)
}

// answerSubscribeOp completes the op with a step under its own ID.
func (e *executor) answerSubscribeOp(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, output any) error {
	byt, err := json.Marshal(output)
	if err != nil {
		return err
	}
	gen.Op = enums.OpcodeStepRun
	gen.Data = byt

	return e.handleGeneratorStep(ctx, runCtx, gen, edge)
}

// subscriptionFlush runs the flush or close op: the buffered messages become
// steps. It returns how many messages of the inbox are steps of the run.
func (e *executor) subscriptionFlush(ctx context.Context, store sv2.SubscriptionStore, id sv2.ID, field string, op string) (int, error) {
	res, err := store.SubscriptionOp(ctx, id, field, append([]string{op}, e.subscriptionLimitArgs(id)...)...)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 || res[0] == "norun" {
		return 0, state.ErrRunNotFound
	}
	if len(res) < 3 {
		return 0, fmt.Errorf("unexpected subscription op result %v", res)
	}

	return strconv.Atoi(res[2])
}

// handleSubscribeNext runs a next op. Buffered messages are flushed and a
// discovery shows them to the SDK. With none, the inbox waits for this op, with
// its timer when the op has a timeout, and the run is idle until a delivery or the
// timer queues the discovery.
func (e *executor) handleSubscribeNext(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, sub *state.SubscribeOpts) error {
	store, err := e.subscriptionStore()
	if err != nil {
		return err
	}

	md := runCtx.Metadata()
	field := subscriptionField(sub.Key)

	var timeoutAt time.Time
	if sub.Timeout != "" {
		timeoutAt, err = strtimeout.ParseTimeout(sub.Timeout, e.now)
		if err != nil {
			return fmt.Errorf("unable to parse subscription timeout: %w", err)
		}
	}

	res, err := store.SubscriptionOp(ctx, md.ID, field, append([]string{"next", gen.ID}, e.subscriptionLimitArgs(md.ID)...)...)
	if err != nil {
		return err
	}
	if len(res) == 0 || res[0] == "norun" {
		return state.ErrRunNotFound
	}

	if res[0] == "closed" {
		// The inbox closed while this op was on its way: the op ends with null, which
		// the SDK reads as the end.
		return e.answerSubscribeOp(ctx, runCtx, gen, edge, nil)
	}
	if res[0] != "waiting" {
		// Flushed messages, or a limit that ended the inbox: the SDK reads them from
		// the stack once the discovery runs.
		unique := "end"
		if res[0] == "flushed" && len(res) > 2 {
			unique = "total-" + res[2]
		}

		return e.enqueueSubscriptionDiscovery(ctx, md, edge.Edge.Incoming, runCtx.MaxAttempts(), nil, field, gen.ID, gen.ID, unique)
	}
	if timeoutAt.IsZero() {
		return nil
	}

	return e.enqueueSubscriptionTimer(ctx, runCtx, edge, field, gen.ID, timeoutAt)
}

// subscriptionWakeMetaKey is the queue item metadata of a discovery that wakes a
// waiting next op: the inbox field and the op ID.
const subscriptionWakeMetaKey = "subscription_wake"

// subscriptionTimerMetaKey is the queue item metadata of the timer of a next op:
// the inbox field.
const subscriptionTimerMetaKey = "subscription_timer"

// subscriptionDiscoveryIdempotency is how long the job ID of a discovery stays
// taken after its item is gone, so a redelivery cannot queue it twice.
const subscriptionDiscoveryIdempotency = 10 * time.Minute

// enqueueSubscriptionDiscovery queues the discovery that shows the run what an
// inbox saved. The job ID belongs to one next op and to the wait it wakes
// (unique: where the wait began, or the number of messages after a flush), so
// every message of a burst and every redelivery share one discovery, while a
// later wake of the same op, which a view that skips a message sends again,
// queues its own.
func (e *executor) enqueueSubscriptionDiscovery(ctx context.Context, md *sv2.Metadata, incoming string, maxAttempts *int, follows *meta.SpanReference, field string, opID string, outgoing string, unique string) error {
	jobID := fmt.Sprintf("%s-subscription-wake-%s-%s-%s", md.ID.RunID, field, opID, unique)
	item := queue.Item{
		JobID:                 &jobID,
		GroupID:               uuid.New().String(),
		WorkspaceID:           md.ID.Tenant.EnvID,
		Kind:                  queue.KindEdge,
		Identifier:            sv2.V1FromMetadata(*md),
		PriorityFactor:        md.Config.PriorityFactor,
		CustomConcurrencyKeys: md.Config.CustomConcurrencyKeys,
		Semaphores:            stepSemaphores(*md),
		MaxAttempts:           maxAttempts,
		Payload: queue.PayloadEdge{
			Edge: inngest.Edge{Outgoing: outgoing, Incoming: incoming},
		},
		Metadata: map[string]any{subscriptionWakeMetaKey: field + ":" + opID},
	}

	span, err := e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameStepDiscovery,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{item.Metadata},
			FollowsFrom: follows,
			Debug:       &tracing.SpanDebugData{Location: "executor.enqueueSubscriptionDiscovery"},
			Metadata:    md,
			Parent:      tracing.RunSpanRefFromMetadata(md),
			QueueItem:   &item,
		},
	)
	if err != nil {
		e.log.Debug("error creating span for a subscription discovery", "error", err)
	}

	period := subscriptionDiscoveryIdempotency
	err = e.queue.Enqueue(ctx, item, e.now(), queue.EnqueueOpts{IdempotencyPeriod: &period})
	if errors.Is(err, queue.ErrQueueItemExists) {
		span.Drop()
		return nil
	}
	if err != nil {
		span.Drop()
		return retryableEnqueueError(err)
	}
	_ = span.Send()

	return nil
}

// enqueueSubscriptionTimer queues the timer of a next op: a sleep item that
// resumes the run under the op's ID, if the op is still waiting when it fires.
func (e *executor) enqueueSubscriptionTimer(ctx context.Context, runCtx execution.RunContext, edge queue.PayloadEdge, field string, opID string, at time.Time) error {
	md := runCtx.Metadata()
	jobID := queue.HashID(ctx, fmt.Sprintf("%s-subscription-timer-%s", md.IdempotencyKey(), opID))
	item := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           md.ID.Tenant.EnvID,
		GroupID:               uuid.New().String(),
		Kind:                  queue.KindSleep,
		Identifier:            sv2.V1FromMetadata(*md),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Semaphores:            stepSemaphores(*md),
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload:               queue.PayloadEdge{Edge: inngest.Edge{Outgoing: opID, Incoming: edge.Edge.Incoming}},
		Metadata:              map[string]any{subscriptionTimerMetaKey: field},
	}

	err := e.queue.Enqueue(ctx, item, at, queue.EnqueueOpts{PassthroughJobId: true})
	if err != nil && !errors.Is(err, queue.ErrQueueItemExists) {
		return retryableEnqueueError(err)
	}

	return nil
}

// claimSubscriptionTimeout runs when a sleep item fires. For the timer of a next
// op it reports whether the timer won: only while the op is still waiting does it
// save null under the op's ID. Any other sleep wins.
func (e *executor) claimSubscriptionTimeout(ctx context.Context, id state.Identifier, item queue.Item, opID string) (bool, error) {
	field, _ := item.Metadata[subscriptionTimerMetaKey].(string)
	if field == "" {
		return true, nil
	}

	store, err := e.subscriptionStore()
	if err != nil {
		return false, err
	}
	res, err := store.SubscriptionOp(ctx, sv2.IDFromV1(id), field, "timeout", opID)
	if err != nil {
		return false, err
	}

	return len(res) > 0 && res[0] == "1", nil
}

// clearSubscriptionWait runs at the start of the discovery that wakes a next op,
// before the run's state is read: a message delivered after this is buffered, and
// one delivered before it is in the state this execution reads. Clearing twice, or
// after another op started waiting, is harmless.
func (e *executor) clearSubscriptionWait(ctx context.Context, id state.Identifier, item queue.Item) error {
	value, _ := item.Metadata[subscriptionWakeMetaKey].(string)
	field, opID, ok := strings.Cut(value, ":")
	if !ok {
		return nil
	}

	store, err := e.subscriptionStore()
	if err != nil {
		return err
	}
	_, err = store.SubscriptionOp(ctx, sv2.IDFromV1(id), field, "clear", opID)

	return err
}

// deliverSubscription hands an event that matched a subscription pause to its
// inbox. The pause stays.
func (e *executor) deliverSubscription(ctx context.Context, evt event.TrackedEvent, pause *state.Pause) error {
	id := sv2.IDFromPause(*pause)
	key := pause.Subscribe.Key
	field := subscriptionField(key)
	messageID := subscriptionMessageID(key, evt.GetInternalID().String())

	md, err := e.smv2.LoadMetadata(ctx, id)
	if err != nil {
		return err
	}

	// The message is the event, with the source that delivered it so a reader can
	// tell which one it came from.
	saved := evt.GetEvent().Map()
	saved["stream"] = pause.Subscribe.Stream
	data, err := json.Marshal(map[string]any{"data": saved})
	if err != nil {
		return err
	}

	store, err := e.subscriptionStore()
	if err != nil {
		return err
	}
	res, err := store.SubscriptionOp(ctx, id, field, append([]string{"deliver", messageID, string(data)}, e.subscriptionLimitArgs(id)...)...)
	if err != nil {
		return err
	}
	if len(res) == 0 || res[0] == "norun" {
		return state.ErrRunNotFound
	}

	switch res[0] {
	case "wake":
		return e.enqueueSubscriptionDiscovery(ctx, &md, pause.Incoming, pause.MaxAttempts, tracing.SpanRefFromPause(pause), field, res[1], messageID, "at-"+res[2])
	case "limit":
		e.log.Warn("subscription limit reached, closing the subscription", "limit", res[1], "key", key)

		if err := e.removeSubscriptionPause(ctx, &md, pause); err != nil {
			return err
		}
		// A next op waiting for a message that will not come sees the end.
		if len(res) > 3 && res[2] != "" {
			return e.enqueueSubscriptionDiscovery(ctx, &md, pause.Incoming, pause.MaxAttempts, tracing.SpanRefFromPause(pause), field, res[2], subscriptionClosedID(key), "at-"+res[3])
		}
	}

	return nil
}

// removeSubscriptionSources deletes the pauses of the sources an op lists.
func (e *executor) removeSubscriptionSources(ctx context.Context, md *sv2.Metadata, sub *state.SubscribeOpts) error {
	for _, source := range sub.Streams {
		idx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: source.Event}
		pause, err := e.pm.PauseByID(ctx, idx, subscriptionPauseID(md.ID.RunID.String(), sub.Key, source.Stream))
		if errors.Is(err, state.ErrPauseNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("error loading subscription pause: %w", err)
		}
		if err := e.removeSubscriptionPause(ctx, md, pause); err != nil {
			return err
		}
	}

	return nil
}

// removeSubscriptionPause deletes the pause of a source so its events are no
// longer matched, with its expiry job.
func (e *executor) removeSubscriptionPause(ctx context.Context, md *sv2.Metadata, pause *state.Pause) error {
	idx := pauses.Index{WorkspaceID: pause.WorkspaceID, EventName: *pause.Event}
	if err := e.pm.Delete(ctx, idx, *pause); err != nil && !errors.Is(err, state.ErrPauseNotFound) {
		return fmt.Errorf("error deleting subscription pause: %w", err)
	}
	if e.exprAggregator != nil {
		_ = e.exprAggregator.RemovePause(ctx, pause)
	}
	e.dequeuePauseJob(ctx, md, subscriptionExpiryJobID(md, pause.ID))

	return nil
}

// dequeuePauseJob removes a pause job from the run's queue partition.
func (e *executor) dequeuePauseJob(ctx context.Context, md *sv2.Metadata, jobID string) {
	if e.shards == nil {
		return
	}

	shard, err := e.shards.Resolve(ctx, queue.Scope{
		AccountID:  md.ID.Tenant.AccountID,
		EnvID:      md.ID.Tenant.EnvID,
		FunctionID: md.ID.FunctionID,
	}, nil)
	if err != nil {
		logger.StdlibLogger(ctx).Warn("could not find shard to dequeue a subscription job", "error", err)
		return
	}

	err = e.queue.Dequeue(ctx, shard.Name(), queue.QueueItem{
		ID:         queue.HashID(ctx, jobID),
		FunctionID: md.ID.FunctionID,
		Data: queue.Item{
			Kind:       queue.KindPause,
			Identifier: sv2.V1FromMetadata(*md),
		},
	})
	if err != nil && !errors.Is(err, queue.ErrQueueItemNotFound) {
		logger.StdlibLogger(ctx).Error("error dequeueing a subscription job", "error", err)
	}
}

// openSubscriptionSource saves the pause of one source and queues its expiry. A
// pause this call wrote is appended to written.
func (e *executor) openSubscriptionSource(ctx context.Context, runCtx execution.RunContext, edge queue.PayloadEdge, gen state.GeneratorOpcode, sub *state.SubscribeOpts, source state.SubscribeStream, written *[]*state.Pause) error {
	md := runCtx.Metadata()
	pauseID := subscriptionPauseID(md.ID.RunID.String(), sub.Key, source.Stream)
	idx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: source.Event}
	now := e.now()

	pause := state.Pause{
		ID:          pauseID,
		WorkspaceID: md.ID.Tenant.EnvID,
		Identifier:  sv2.NewPauseIdentifier(md.ID),
		GroupID:     runCtx.GroupID(),
		Outgoing:    gen.ID,
		Incoming:    edge.Edge.Incoming,
		StepName:    gen.UserDefinedName(),
		Opcode:      new(gen.Op.String()),
		// A subscription lives as long as the run.
		Expires:     state.Time(now.Add(consts.MaxWaitForEventTimeout)),
		Event:       &source.Event,
		Expression:  source.If,
		MaxAttempts: runCtx.MaxAttempts(),
		Subscribe:   &state.PauseSubscribe{Key: sub.Key, Stream: source.Stream},
		CreatedAt:   now,
	}

	_, err := e.pm.Write(ctx, idx, &pause)
	if err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
		return fmt.Errorf("error saving subscription pause: %w", err)
	}
	if err == nil {
		*written = append(*written, &pause)

		// Count after writing, so sources opened in the same op each see the ones
		// before them and at most the limit are kept.
		ids, err := e.pm.PauseIDsForRun(ctx, md.ID.RunID)
		if err != nil {
			return fmt.Errorf("error counting the run's pauses: %w", err)
		}
		if len(ids) > consts.MaxSubscriptionPausesPerRun {
			return subscriptionLimitError("A run can hold at most %d pauses (subscriptions and waitForEvents) when it opens a subscription.", consts.MaxSubscriptionPausesPerRun)
		}
	}

	// A queued item that fires when the pause expires. It keeps the run holding
	// its function singleton while it waits: the singleton lock is released once
	// the run has no queued items.
	jobID := subscriptionExpiryJobID(md, pauseID)
	expiry := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           md.ID.Tenant.EnvID,
		GroupID:               runCtx.GroupID(),
		Kind:                  queue.KindPause,
		Identifier:            sv2.V1FromMetadata(*md),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Payload:               queue.PayloadPauseTimeout{PauseID: pauseID, Pause: pause},
		Metadata:              make(map[string]any),
	}
	if err := e.queue.Enqueue(ctx, expiry, pause.Expires.Time(), queue.EnqueueOpts{}); err != nil && !errors.Is(err, queue.ErrQueueItemExists) {
		return fmt.Errorf("error enqueueing subscription expiry: %w", err)
	}

	return nil
}

// subscribeBusy reports whether a response holds an op that is not only a wait.
// Ops that only park the run (sleep, a signal, an invoke, an event wait, a
// subscription next) leave it idle.
func subscribeBusy(ops []*state.GeneratorOpcode) bool {
	hasNext, busy := false, false
	for _, op := range ops {
		if op == nil {
			continue
		}
		if isSubscribeNext(op) {
			hasNext = true
		} else if !isParkingOp(op) {
			busy = true
		}
	}

	return hasNext && busy
}

func isSubscribeNext(op *state.GeneratorOpcode) bool {
	if op.Op != enums.OpcodeWaitForEvent {
		return false
	}
	opts, err := op.WaitForEventOpts()

	return err == nil && opts.Subscribe != nil && opts.Subscribe.Mode == state.SubscribeNext
}

func isParkingOp(op *state.GeneratorOpcode) bool {
	switch op.Op {
	case enums.OpcodeSleep, enums.OpcodeInvokeFunction, enums.OpcodeWaitForSignal:
		return true
	case enums.OpcodeWaitForEvent:
		opts, err := op.WaitForEventOpts()

		return err == nil && opts.Subscribe == nil
	}

	return false
}
