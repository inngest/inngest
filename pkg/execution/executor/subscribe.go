package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
)

// A step.subscribe subscription keeps its pause when an event matches and saves
// the event as a step of the run. The step ID is made of the subscription and
// the event ID, so saving the same event twice is one message and two events
// never contend for a slot. The run's stack (the order steps were saved in) is
// the order of the messages: the SDK reads them from there.

// subscriptionMessagePrefix starts the ID of every message step. The SDK looks
// for it in the stack.
const subscriptionMessagePrefix = "inngest-sub:"

func subscriptionKeyHash(key string) string {
	sum := sha1.Sum([]byte(key))
	return hex.EncodeToString(sum[:])
}

// subscriptionMessageID is the step ID of the message an event is in a
// subscription.
func subscriptionMessageID(key string, eventID string) string {
	return subscriptionMessagePrefix + subscriptionKeyHash(key) + ":" + eventID
}

// subscriptionMessageIDs returns the message step IDs of a subscription in
// stack order.
func subscriptionMessageIDs(stack []string, key string) []string {
	return subscriptionMessageIDsOfHash(stack, subscriptionKeyHash(key))
}

func subscriptionMessageIDsOfHash(stack []string, hash string) []string {
	prefix := subscriptionMessagePrefix + hash + ":"

	var ids []string
	for _, id := range stack {
		if strings.HasPrefix(id, prefix) {
			ids = append(ids, id)
		}
	}

	return ids
}

// subscriptionClosedID is the step that marks a subscription as closed. It
// starts with the message prefix, so the SDK does not wait for it, and its hash
// part is not a message of any subscription.
func subscriptionClosedID(key string) string {
	return subscriptionMessagePrefix + "closed:" + subscriptionKeyHash(key)
}

// subscriptionExpiryJobID is the queue job that expires a subscription pause.
func subscriptionExpiryJobID(md *sv2.Metadata, pauseID uuid.UUID) string {
	return fmt.Sprintf("%s-subscription-expiry-%s", md.IdempotencyKey(), pauseID)
}

// subscriptionPauseID is the ID of the persistent pause of one event (stream) of a
// subscription.
func subscriptionPauseID(runID string, key string, stream int) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription:%s:%d", runID, key, stream))
}

// handleGeneratorSubscribe handles the WaitForEvent ops of a step.subscribe
// subscription: open, next, poll and close.
func (e *executor) handleGeneratorSubscribe(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, opts *state.WaitForEventOpts, expr *string) error {
	sub := opts.Subscribe
	if err := validateSubscribe(sub); err != nil {
		return err
	}

	md := runCtx.Metadata()
	runID := md.ID.RunID.String()
	pauseID := subscriptionPauseID(runID, sub.Key, sub.Stream)
	idx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: opts.Event}

	// answer completes the op right away with the given output.
	answer := func(output any) error {
		byt, err := json.Marshal(output)
		if err != nil {
			return err
		}
		gen.Op = enums.OpcodeStepRun
		gen.Data = byt
		return e.handleGeneratorStep(ctx, runCtx, gen, edge)
	}

	// messages are the subscription's message step IDs in stack order.
	messages := func() ([]string, error) {
		stack, err := e.smv2.LoadStack(ctx, md.ID)
		if err != nil {
			return nil, err
		}
		return subscriptionMessageIDs(stack, sub.Key), nil
	}

	// deleteStream removes the pause of this op's stream, so its events are no
	// longer saved. Deleting a pause that is gone is not an error.
	deleteStream := func() error {
		pause, err := e.pm.PauseByID(ctx, idx, pauseID)
		if err != nil && !errors.Is(err, state.ErrPauseNotFound) {
			return fmt.Errorf("error loading subscription pause: %w", err)
		}
		if pause != nil {
			if err := e.pm.Delete(ctx, idx, *pause); err != nil {
				return fmt.Errorf("error deleting subscription pause: %w", err)
			}
			if e.exprAggregator != nil {
				_ = e.exprAggregator.RemovePause(ctx, pause)
			}
			e.dequeueSubscriptionExpiry(ctx, md, pauseID)
		}

		return nil
	}

	switch sub.Mode {
	case state.SubscribeOpen:
		// The op's own source is the first stream; a many-source open carries
		// the rest, so an inbox opens in one step.
		sources := []state.SubscribeStream{{Stream: sub.Stream, Event: opts.Event, If: expr}}
		seen := map[int]bool{sub.Stream: true}
		for _, extra := range sub.Streams {
			if extra.Event == "" || seen[extra.Stream] {
				return subscriptionLimitError("A subscription open lists a stream twice or without an event.")
			}
			seen[extra.Stream] = true
			if err := validateSubscribe(&state.SubscribeOpts{Key: sub.Key, Mode: sub.Mode, Stream: extra.Stream}); err != nil {
				return err
			}
			if err := validateWaitExpression(ctx, extra.If); err != nil {
				return err
			}
			interpolated, _, err := interpolateWaitExpression(ctx, runCtx, extra.If)
			if err != nil {
				return err
			}
			sources = append(sources, state.SubscribeStream{Stream: extra.Stream, Event: extra.Event, If: interpolated})
		}
		if len(sources) > consts.MaxSubscriptionPausesPerRun {
			return subscriptionLimitError("A subscription can listen to at most %d events.", consts.MaxSubscriptionPausesPerRun)
		}

		var written []*state.Pause
		for _, source := range sources {
			if err := e.openSubscriptionStream(ctx, runCtx, edge, gen, sub, source, &written); err != nil {
				// The run fails, so the pauses this op wrote go with it.
				for _, pause := range written {
					_ = e.removeSubscriptionPause(ctx, md, pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: *pause.Event}, pause)
				}
				return err
			}
		}

		return answer(map[string]any{"opened": true, "streams": len(sources)})

	case state.SubscribePoll:
		ids, err := messages()
		if err != nil {
			return err
		}
		if len(ids) < sub.Seq {
			return answer(nil)
		}

		saved, err := e.smv2.LoadStepsWithIDs(ctx, md.ID, []string{ids[sub.Seq-1]})
		if err != nil {
			return err
		}
		var stored struct {
			Data any `json:"data"`
		}
		if err := json.Unmarshal(saved[ids[sub.Seq-1]], &stored); err != nil {
			return err
		}
		return answer(stored.Data)

	case state.SubscribeCurrent:
		ids, err := messages()
		if err != nil {
			return err
		}

		return answer(map[string]any{"count": len(ids)})

	case state.SubscribeStop:
		if err := deleteStream(); err != nil {
			return err
		}

		return answer(map[string]any{"stopped": true})

	case state.SubscribeClose:
		if err := deleteStream(); err != nil {
			return err
		}

		if err := e.endSubscription(ctx, md.ID, sub.Key, []byte("true")); err != nil {
			return err
		}

		ids, err := messages()
		if err != nil {
			return err
		}
		return answer(map[string]any{"count": len(ids)})
	}

	return fmt.Errorf("unknown subscription mode %q", sub.Mode)
}

// dequeueSubscriptionExpiry removes the queue job that would expire a subscription
// pause, once the pause is closed.
func (e *executor) dequeueSubscriptionExpiry(ctx context.Context, md *sv2.Metadata, pauseID uuid.UUID) {
	e.dequeuePauseJob(ctx, md, subscriptionExpiryJobID(md, pauseID))
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

// deliverSubscription saves an event that matched a subscription pause as a
// message of the run and wakes the run if it is blocked on that message. The
// pause stays.
func (e *executor) deliverSubscription(ctx context.Context, evt event.TrackedEvent, pause *state.Pause, l logger.Logger) error {
	id := sv2.IDFromPause(*pause)
	key := pause.Subscribe.Key
	eventID := evt.GetInternalID().String()

	// The event that started the run is its first message, not a match.
	if pause.TriggeringEventID != nil && *pause.TriggeringEventID == eventID {
		return nil
	}

	// SaveStep writes to the run's hashes whether or not the run still
	// exists, so check first: a finished run cannot take the message. The
	// metadata also tells how much the run holds, for the limits below.
	md, err := e.smv2.LoadMetadata(ctx, id)
	if err != nil {
		return err
	}

	// The message is the event, with the stream that saved it so a reader can
	// tell which source of the subscription it came from.
	saved := evt.GetEvent().Map()
	saved["stream"] = pause.Subscribe.Stream

	data, err := json.Marshal(map[string]any{"data": saved})
	if err != nil {
		return err
	}

	// The same event saves the same bytes under the same ID: a redelivery is
	// the message it already is.
	messageID := subscriptionMessageID(key, eventID)

	// A closed subscription saves nothing more: another stream of it may still
	// have its pause, so remove that one now.
	if slices.Contains(md.Stack, subscriptionClosedID(key)) {
		idx := pauses.Index{WorkspaceID: pause.WorkspaceID, EventName: *pause.Event}
		return e.removeSubscriptionPause(ctx, &md, idx, pause)
	}

	// A message that would break a limit is dropped and the subscription is
	// closed: its pause stops matching, and the reader sees the end after the
	// messages already saved. A redelivery of a saved message is not new.
	if !slices.Contains(md.Stack, messageID) {
		stepLimit, stateLimit := e.subscriptionLimits(id)
		if limit := subscriptionMessageLimit(&md, key, len(data), stepLimit, stateLimit); limit != "" {
			l.Warn("subscription limit reached, closing the subscription", "limit", limit, "key", key)

			// Close first: it is idempotent, so a failure here is retried with the
			// pause still there to find it.
			reason, _ := json.Marshal(map[string]any{"limit": limit})
			if err := e.endSubscription(ctx, id, key, reason); err != nil {
				return err
			}
			idx := pauses.Index{WorkspaceID: pause.WorkspaceID, EventName: *pause.Event}
			if err := e.removeSubscriptionPause(ctx, &md, idx, pause); err != nil {
				return err
			}

			// A read waiting for a message that will not come sees the close.
			return e.wakeSubscription(ctx, &md, pause.Incoming, pause.MaxAttempts, tracing.SpanRefFromPause(pause), key, subscriptionClosedID(key))
		}
	}
	_, err = e.smv2.SaveStep(ctx, id, messageID, data)
	redelivered := err != nil
	if err != nil && !errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
		return err
	}

	l.Debug("subscription message saved", "message_id", messageID, "redelivered", redelivered)

	return e.wakeSubscription(ctx, &md, pause.Incoming, pause.MaxAttempts, tracing.SpanRefFromPause(pause), key, messageID)
}

// subscriptionTriggerExclusion returns the event ID a subscription pause must
// not save as a message: the event that started the run, when the SDK asked
// for it to be the first message itself (an agent inbox puts it back). Without
// the ask the trigger is an ordinary event to the subscription.
func subscriptionTriggerExclusion(sub *state.SubscribeOpts, md *sv2.Metadata) *string {
	if !sub.TriggerIsFirst || len(md.Config.EventIDs) == 0 {
		return nil
	}

	trigger := md.Config.EventIDs[0].String()
	return &trigger
}

// subscriptionClosedPrefix starts the ID of every closed marker. Closed markers
// are not messages, so they never count against a message limit.
const subscriptionClosedPrefix = subscriptionMessagePrefix + "closed:"

// subscriptionLimitError is the error a run fails with when a step.subscribe op
// exceeds a limit or is malformed. It fails the run immediately, like a
// waitForEvent timeout that is too long.
func subscriptionLimitError(format string, args ...any) error {
	return state.WrapInStandardError(
		state.ErrSubscriptionLimit,
		state.InngestErrSubscriptionLimit,
		fmt.Sprintf(format, args...),
		"",
	)
}

// validateSubscribe checks the bounds of a subscription op before it touches any
// state: the sizes and numbers an SDK (or a caller of the API) could set freely.
func validateSubscribe(sub *state.SubscribeOpts) error {
	if sub.Key == "" {
		return subscriptionLimitError("The subscription ID is empty.")
	}
	if len(sub.Key) > consts.MaxSubscriptionKeyLength {
		return subscriptionLimitError("The subscription ID is longer than %d characters.", consts.MaxSubscriptionKeyLength)
	}
	if sub.Stream < 0 || sub.Stream >= consts.MaxSubscriptionPausesPerRun {
		return subscriptionLimitError("A subscription can listen to at most %d events.", consts.MaxSubscriptionPausesPerRun)
	}
	if sub.Mode == state.SubscribePoll && sub.Seq < 1 {
		return subscriptionLimitError("The message number of a subscription read starts at 1.")
	}

	return nil
}

// isSubscriptionMessageID reports whether a step ID in the stack is a message.
func isSubscriptionMessageID(id string) bool {
	return strings.HasPrefix(id, subscriptionMessagePrefix) && !strings.HasPrefix(id, subscriptionClosedPrefix)
}

// subscriptionMessageLimit reports which limit saving one more message of the
// subscription would break, or "" when it fits. The stack and metrics are the
// run's. A message is only refused for a limit that cannot lift again: counts
// and sizes only grow.
func subscriptionMessageLimit(md *sv2.Metadata, key string, size int, stepLimit int, stateLimit int) string {
	prefix := subscriptionMessagePrefix + subscriptionKeyHash(key) + ":"

	inRun, inSubscription := 0, 0
	for _, id := range md.Stack {
		if !isSubscriptionMessageID(id) {
			continue
		}
		inRun++
		if strings.HasPrefix(id, prefix) {
			inSubscription++
		}
	}

	switch {
	case inSubscription >= consts.MaxSubscriptionMessages:
		return fmt.Sprintf("the subscription saved %d messages", consts.MaxSubscriptionMessages)
	case inRun >= consts.MaxSubscriptionMessagesPerRun:
		return fmt.Sprintf("the run's subscriptions saved %d messages", consts.MaxSubscriptionMessagesPerRun)
	case inRun >= stepLimit/2:
		return fmt.Sprintf("messages took half of the run's step limit of %d", stepLimit)
	case md.Metrics.StepCount >= stepLimit:
		return fmt.Sprintf("the run reached its step limit of %d", stepLimit)
	case md.Metrics.StateSize+size > stateLimit/4*3:
		return fmt.Sprintf("the run's state reached three quarters of its limit of %d bytes", stateLimit)
	}

	return ""
}

// subscriptionLimits are the run's step and state size limits as the executor
// enforces them for steps.
func (e *executor) subscriptionLimits(id sv2.ID) (stepLimit int, stateLimit int) {
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

	return stepLimit, stateLimit
}

// endSubscription marks a subscription closed. The data of the closed step is
// "true", or says which limit closed it. A read waiting on the subscription sees
// the step on its next discovery.
func (e *executor) endSubscription(ctx context.Context, id sv2.ID, key string, data []byte) error {
	if _, err := e.smv2.SaveStep(ctx, id, subscriptionClosedID(key), data); err != nil &&
		!errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
		return err
	}

	return nil
}

// A step.subscribe read with nothing to read ends its request with a waiting
// thread (see handleGeneratorSubscribeWait), and the run is then idle. That is
// the only time a delivery wakes it: the executor keeps one flag per run and
// subscription, set when the run goes idle waiting and cleared by whoever wakes
// it. A run that is busy has no flag, so its messages cost nothing and no
// discovery overlaps the function's own requests.

// subscriptionWakeJobID is the job ID of one wake of a subscription in a run.
// The token belongs to one wait, so a wake never meets the item of the wake
// before it, which the queue keeps until that wake's request has been handled.
func subscriptionWakeJobID(runID string, hash string, token string) string {
	return fmt.Sprintf("%s-subscription-wake-%s-%s", runID, hash, token)
}

// subscriptionWaiter is the run state's waiting flag.
func (e *executor) subscriptionWaiter() (sv2.SubscriptionWaiter, error) {
	waiter, ok := e.smv2.(sv2.SubscriptionWaiter)
	if !ok {
		return nil, errors.New("the run state does not support step.subscribe")
	}

	return waiter, nil
}

// clearSubscriptionWaiting drops the waiting flags of a run.
func (e *executor) clearSubscriptionWaiting(ctx context.Context, id sv2.ID) error {
	waiter, err := e.subscriptionWaiter()
	if err != nil {
		return err
	}

	return waiter.ClearSubscriptionWaiting(ctx, id)
}

// handleGeneratorSubscribeWait runs when the function's request ended with a
// read that has nothing to read. Nothing is in flight now, so the run is marked
// as waiting, and then the messages are counted once more: one saved since the
// request read its state found no flag to take, so this is what wakes the run.
func (e *executor) handleGeneratorSubscribeWait(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	byt, err := json.Marshal(gen.Opts)
	if err != nil {
		return err
	}
	var opts struct {
		Subscribe struct {
			Key string `json:"key"`
			// Seq is the number of messages the request saw, plus one.
			Seq int `json:"seq"`
		} `json:"subscribe"`
		Busy bool `json:"busy"`
		// Timer is the step ID of the read's timer, when it has one.
		Timer string `json:"timer"`
	}
	if err := json.Unmarshal(byt, &opts); err != nil || opts.Subscribe.Key == "" || opts.Busy {
		return nil
	}

	waiter, err := e.subscriptionWaiter()
	if err != nil {
		return err
	}

	md := runCtx.Metadata()
	key := opts.Subscribe.Key
	hash := subscriptionKeyHash(key)
	token := subscriptionWaitToken(opts.Timer, opts.Subscribe.Seq)
	if err := waiter.SetSubscriptionWaiting(ctx, md.ID, hash, token); err != nil {
		if errors.Is(err, state.ErrRunNotFound) {
			return nil
		}
		return err
	}

	stack, err := e.smv2.LoadStack(ctx, md.ID)
	if err != nil {
		return err
	}
	if len(subscriptionMessageIDsOfHash(stack, hash)) < opts.Subscribe.Seq && !slices.Contains(stack, subscriptionClosedID(key)) {
		return nil
	}

	return e.wakeSubscription(ctx, md, edge.Edge.Incoming, runCtx.MaxAttempts(), nil, key, gen.ID)
}

// wakeSubscription wakes the run if it is waiting on the subscription: it reads
// the flag and, only if it is set, queues a discovery under the job ID of that
// wait. The flag stays set; the discovery clears it when it starts.
func (e *executor) wakeSubscription(ctx context.Context, md *sv2.Metadata, incoming string, maxAttempts *int, follows *meta.SpanReference, key string, outgoing string) error {
	waiter, err := e.subscriptionWaiter()
	if err != nil {
		return err
	}

	hash := subscriptionKeyHash(key)
	token, waiting, err := waiter.GetSubscriptionWaiting(ctx, md.ID, hash)
	if err != nil || !waiting {
		return err
	}

	return e.enqueueSubscriptionWake(ctx, md, incoming, maxAttempts, follows, hash, token, outgoing)
}

// subscriptionWakeIdempotency is how long a wake's job ID stays taken after its
// item is gone, so a late delivery that read the flag of that wait cannot queue
// a second discovery for it.
const subscriptionWakeIdempotency = 10 * time.Minute

// subscriptionWakeMetaKey is the queue item metadata that names the wait a wake
// belongs to. The queue replaces the job ID the item carries with a hash, so the
// token cannot be read back from it.
const subscriptionWakeMetaKey = "subscription_wake"

// subscriptionWakeMeta is the metadata value of a wake: the subscription hash
// and the token of the wait.
func subscriptionWakeMeta(hash string, token string) string {
	return hash + ":" + token
}

// parseSubscriptionWakeMeta splits the metadata value of a wake.
func parseSubscriptionWakeMeta(v string) (hash string, token string, ok bool) {
	hash, token, ok = strings.Cut(v, ":")
	return hash, token, ok && hash != "" && token != ""
}

// startSubscriptionWake runs at the start of every execution, before the run's
// state is read. A wake clears the flag of its own subscription by hash, so a
// message saved after this point finds no flag, even when the wake is a rerun
// of an earlier one. Any other execution runs the function while
// the run is no longer idle, so every flag is stale.
func (e *executor) startSubscriptionWake(ctx context.Context, id sv2.ID, item queue.Item) error {
	waiter, ok := e.smv2.(sv2.SubscriptionWaiter)
	if !ok {
		return nil
	}

	if v, _ := item.Metadata[subscriptionWakeMetaKey].(string); v != "" {
		hash, _, ok := parseSubscriptionWakeMeta(v)
		if !ok {
			return nil
		}

		return waiter.ClearSubscriptionWaitingFor(ctx, id, hash)
	}

	return waiter.ClearSubscriptionWaiting(ctx, id)
}

// enqueueSubscriptionWake queues the discovery that gives a waiting run the
// chance to see what its subscription saved. It runs under the run's own mode
// like any discovery after a step. The idempotency period keeps the job ID
// taken after the item is gone, so a stale delivery of the same wait is refused.
func (e *executor) enqueueSubscriptionWake(ctx context.Context, md *sv2.Metadata, incoming string, maxAttempts *int, follows *meta.SpanReference, hash string, token string, outgoing string) error {
	jobID := subscriptionWakeJobID(md.ID.RunID.String(), hash, token)
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
		Metadata: map[string]any{subscriptionWakeMetaKey: subscriptionWakeMeta(hash, token)},
	}

	span, err := e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameStepDiscovery,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{item.Metadata},
			FollowsFrom: follows,
			Debug:       &tracing.SpanDebugData{Location: "executor.enqueueSubscriptionWake"},
			Metadata:    md,
			Parent:      tracing.RunSpanRefFromMetadata(md),
			QueueItem:   &item,
		},
	)
	if err != nil {
		e.log.Debug("error creating span for a subscription wake", "error", err)
	}

	period := subscriptionWakeIdempotency
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

// removeSubscriptionPause deletes the pause of a subscription stream so its
// events are no longer matched, with its expiry job.
func (e *executor) removeSubscriptionPause(ctx context.Context, md *sv2.Metadata, idx pauses.Index, pause *state.Pause) error {
	if err := e.pm.Delete(ctx, idx, *pause); err != nil && !errors.Is(err, state.ErrPauseNotFound) {
		return fmt.Errorf("error deleting subscription pause: %w", err)
	}
	if e.exprAggregator != nil {
		_ = e.exprAggregator.RemovePause(ctx, pause)
	}
	e.dequeueSubscriptionExpiry(ctx, md, pause.ID)

	return nil
}

// openSubscriptionStream saves the pause of one stream and queues its expiry.
// A pause this call wrote is appended to written.
func (e *executor) openSubscriptionStream(ctx context.Context, runCtx execution.RunContext, edge queue.PayloadEdge, gen state.GeneratorOpcode, sub *state.SubscribeOpts, source state.SubscribeStream, written *[]*state.Pause) error {
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

		TriggeringEventID: subscriptionTriggerExclusion(sub, md),
	}

	_, err := e.pm.Write(ctx, idx, &pause)
	if err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
		return fmt.Errorf("error saving subscription pause: %w", err)
	}
	if err == nil {
		*written = append(*written, &pause)

		// Count after writing, so opens in the same response each see the
		// ones before them and at most the limit are kept. An open that was
		// already written passed this check before.
		ids, err := e.pm.PauseIDsForRun(ctx, md.ID.RunID)
		if err != nil {
			return fmt.Errorf("error counting the run's pauses: %w", err)
		}
		if len(ids) > consts.MaxSubscriptionPausesPerRun {
			return subscriptionLimitError("A run can hold at most %d pauses (subscriptions and waitForEvents) when it opens a subscription.", consts.MaxSubscriptionPausesPerRun)
		}
	}

	// A queued item that fires when the subscription expires. It keeps the
	// run holding its function singleton while it waits: the singleton lock
	// is released once the run has no queued items.
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

// markBusySubscriptionWaits tells the waiting reads of a response that the run
// is not idle: anything beside waits and their timers (a step the function ran
// or planned) means a message must not queue a discovery under this wait. Ops
// that only park the run (sleep, a signal, an invoke, an event wait that is not
// a subscription op) leave it idle, so a read beside them stays wakeable.
func markBusySubscriptionWaits(ops []*state.GeneratorOpcode) {
	busy := false
	for _, op := range ops {
		if op == nil || isParkingOp(op) || (op.Op == enums.OpcodeNone && op.ID != "") {
			continue
		}
		busy = true
	}
	if !busy {
		return
	}

	for _, op := range ops {
		if op == nil || op.Op != enums.OpcodeNone || op.ID == "" {
			continue
		}
		if opts, ok := op.Opts.(map[string]any); ok {
			opts["busy"] = true
		}
	}
}

// isParkingOp reports whether the op only waits: nothing runs now, and the run
// resumes when something outside it happens.
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

// subscriptionMessageWonPrefix starts what a message that wins a timed read saves
// as the result of the read's timer step; the wake's token follows. The timer
// saves null when it wins. Whichever is saved first is the winner, and the other
// one sees the duplicate and runs nothing, so a message and its timer never both
// run what follows the read. The token names the wake that won, so a second wake
// of the same read (another token) stops too.
const subscriptionMessageWonPrefix = "subscription-message:"

// subscriptionMessageWon is the value a wake with the token saves.
func subscriptionMessageWon(token string) []byte {
	return []byte(strconv.Quote(subscriptionMessageWonPrefix + token))
}

// subscriptionWaitToken is the token of one wait: unique, and for a timed read
// followed by the ID of the read's timer step and the number of the message the
// read waits for.
func subscriptionWaitToken(timer string, seq int) string {
	token := uuid.NewString()
	if timer == "" {
		return token
	}

	return fmt.Sprintf("%s.%s.%d", token, timer, seq)
}

// subscriptionTimerOfToken is the timer step ID and awaited message number a
// wait token carries, empty for an untimed read.
func subscriptionTimerOfToken(token string) (timer string, seq int) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", 0
	}
	seq, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0
	}

	return parts[1], seq
}

// markSubscriptionTimers tells each waiting read of a response which sleep is
// its timer: the SDK gives the waiting op the ID of the timer plus ":waiting".
func markSubscriptionTimers(ops []*state.GeneratorOpcode) {
	timers := map[string]string{}
	for _, op := range ops {
		if op != nil && op.Op == enums.OpcodeSleep {
			if id := userStepID(op); id != "" {
				timers[id+":waiting"] = op.ID
			}
		}
	}
	if len(timers) == 0 {
		return
	}

	for _, op := range ops {
		if op == nil || op.Op != enums.OpcodeNone || op.ID == "" {
			continue
		}
		timer, ok := timers[userStepID(op)]
		if !ok {
			continue
		}
		if opts, ok := op.Opts.(map[string]any); ok {
			opts["timer"] = timer
		}
	}
}

// timerWinner reads who saved the timer step: the token of the wake of a message
// that won, or empty when the timer (or nothing) did.
func (e *executor) timerWinner(ctx context.Context, id sv2.ID, stepID string) (string, error) {
	steps, err := e.smv2.LoadStepsWithIDs(ctx, id, []string{stepID})
	if err != nil {
		return "", err
	}

	var v string
	if err := json.Unmarshal(steps[stepID], &v); err != nil {
		return "", nil
	}

	winner, _ := strings.CutPrefix(v, subscriptionMessageWonPrefix)

	return winner, nil
}

// claimSubscriptionTimer runs for a wake before it does anything else. For a
// timed read the wake claims the wait by saving the timer's step as won, but
// only when the message the read waits for is saved: a wake for an earlier
// message (a redelivery) or a close would otherwise end the read as a timeout.
// It returns false when the timer got there first: the timer's request runs the
// function, so this one must not. A rerun of the same wake finds its own token
// and carries on; any other wake of the read stops, since the winner runs.
func (e *executor) claimSubscriptionTimer(ctx context.Context, id sv2.ID, item queue.Item) (bool, error) {
	v, _ := item.Metadata[subscriptionWakeMetaKey].(string)
	hash, token, ok := parseSubscriptionWakeMeta(v)
	timer, seq := subscriptionTimerOfToken(token)
	if !ok || timer == "" {
		return true, nil
	}

	stack, err := e.smv2.LoadStack(ctx, id)
	if err != nil {
		return false, err
	}
	if len(subscriptionMessageIDsOfHash(stack, hash)) < seq {
		return true, nil
	}

	_, err = e.smv2.SaveStep(ctx, id, timer, subscriptionMessageWon(token))
	if err != nil && !errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
		return false, err
	}

	// A retried save reports a duplicate as success, so the stored value says who
	// won, whatever the save returned.
	winner, err := e.timerWinner(ctx, id, timer)

	return winner == token, err
}

// userStepID is the ID the user gave the op, empty when the SDK sent none.
func userStepID(op *state.GeneratorOpcode) string {
	if op.Userland == nil {
		return ""
	}

	return op.Userland.ID
}
