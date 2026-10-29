package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
	prefix := subscriptionMessagePrefix + subscriptionKeyHash(key) + ":"

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

// subscriptionStepID is the step ID of the wait for message seq. The SDK gives
// its blocking next() the same ID.
func subscriptionStepID(key string, reader string, seq int, attempt int) string {
	id := key
	if reader != "" {
		id = fmt.Sprintf("%s:%s", key, reader)
	}
	id = fmt.Sprintf("%s:%d", id, seq)
	if attempt > 0 {
		id = fmt.Sprintf("%s:%d", id, attempt)
	}
	sum := sha1.Sum([]byte(id))
	return hex.EncodeToString(sum[:])
}

// subscriptionParkIndexName names the index of the parked markers of one run
// and subscription. It starts with pauses.SubscriptionParkIndexPrefix; it is a
// key, not an event name. A delivery reads
// this one small index to find the marker without reading the run's stack.
func subscriptionParkIndexName(runID string, key string) string {
	return pauses.SubscriptionParkIndexPrefix + ":" + runID + ":" + subscriptionKeyHash(key)
}

// subscriptionParkID is the ID of the marker of one park. It is made of the park
// step's ID, which is unique to the park, so the cleanup of an earlier marker can
// never delete a later one, however many messages were saved in between.
func subscriptionParkID(runID string, key string, stepID string) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription-park:%s:%s", runID, key, stepID))
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

// parkState is what a park op finds in the run's stack.
type parkState int

const (
	// parkWaiting: the awaited message is not saved yet.
	parkWaiting parkState = iota
	// parkReady: the awaited message is saved, so the run can wake now.
	parkReady
	// parkAnswered: the park step itself is saved, so an earlier copy of this
	// park was woken.
	parkAnswered
)

// parkStateOf reads a park op's situation from the stack. The executor asks it
// twice: before writing the marker (a copy that was already answered stops
// there) and after (a message saved while the marker was being written wakes
// the run now; one saved later finds the marker through its own delivery).
func parkStateOf(stack []string, key string, stepID string, seq int) parkState {
	if slices.Contains(stack, stepID) {
		return parkAnswered
	}
	if len(subscriptionMessageIDs(stack, key)) >= seq {
		return parkReady
	}
	return parkWaiting
}

// handleGeneratorSubscribe handles the WaitForEvent ops of a step.subscribe
// subscription: open, next, poll and close.
func (e *executor) handleGeneratorSubscribe(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, opts *state.WaitForEventOpts, expr *string) error {
	sub := opts.Subscribe
	if err := validateSubscribe(sub, opts.Event); err != nil {
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
			Event:       &opts.Event,
			Expression:  expr,
			MaxAttempts: runCtx.MaxAttempts(),
			Subscribe:   &state.PauseSubscribe{Key: sub.Key, Stream: sub.Stream},
			CreatedAt:   now,

			TriggeringEventID: subscriptionTriggerExclusion(sub, md),
		}

		_, err := e.pm.Write(ctx, idx, &pause)
		if err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
			return fmt.Errorf("error saving subscription pause: %w", err)
		}
		if err == nil {
			// Count after writing, so opens in the same response each see the
			// ones before them and at most the limit are kept. An open that was
			// already written passed this check before.
			ids, err := e.pm.PauseIDsForRun(ctx, md.ID.RunID)
			if err != nil {
				return fmt.Errorf("error counting the run's pauses: %w", err)
			}
			if len(ids) > consts.MaxSubscriptionPausesPerRun {
				if err := e.removeSubscriptionPause(ctx, md, idx, &pause); err != nil {
					return err
				}
				return subscriptionLimitError("A run can hold at most %d pauses (subscriptions, parked reads and waitForEvents) when it opens a subscription.", consts.MaxSubscriptionPausesPerRun)
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

		return answer(map[string]any{"opened": true})

	case state.SubscribePark:
		if gen.ID != subscriptionStepID(sub.Key, sub.Reader, sub.Seq, sub.Attempt) {
			return fmt.Errorf("subscription step ID does not match message %d", sub.Seq)
		}

		stack, err := e.smv2.LoadStack(ctx, md.ID)
		if err != nil {
			return err
		}
		// A park that was already woken and answered is sent again: nothing to do.
		if parkStateOf(stack, sub.Key, gen.ID, sub.Seq) == parkAnswered {
			return nil
		}

		// The marker says "this run is waiting for message Seq". It is a pause
		// only so that a delivery can find and consume it atomically.
		parkExpires := e.now().Add(consts.MaxWaitForEventTimeout)
		if sub.Timeout != "" {
			expires, err := (&state.WaitForEventOpts{Timeout: sub.Timeout}).Expires()
			if err != nil {
				return fmt.Errorf("unable to parse the park timeout: %w", err)
			}
			parkExpires = expires
		}

		parkEvent := subscriptionParkIndexName(runID, sub.Key)
		parkIdx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: parkEvent}
		opcode := gen.Op.String()
		park := state.Pause{
			ID:           subscriptionParkID(runID, sub.Key, gen.ID),
			WorkspaceID:  md.ID.Tenant.EnvID,
			Identifier:   sv2.NewPauseIdentifier(md.ID),
			GroupID:      runCtx.GroupID(),
			Outgoing:     gen.ID,
			Incoming:     edge.Edge.Incoming,
			StepName:     gen.UserDefinedName(),
			Opcode:       &opcode,
			Expires:      state.Time(parkExpires),
			Event:        &parkEvent,
			DataKey:      gen.ID,
			MaxAttempts:  runCtx.MaxAttempts(),
			ParallelMode: gen.ParallelMode(),
			// Like waitForEvent: the wake joins the batch's shared discovery.
			ParallelCoalesceKey: group.ParallelCoalesceKey,
			CreatedAt:           e.now(),
			Metadata:            map[string]any{"key": sub.Key, "seq": sub.Seq, "timeout": sub.Timeout != ""},
		}
		if _, err := e.pm.Write(ctx, parkIdx, &park); err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
			return fmt.Errorf("error saving subscription park: %w", err)
		}

		// A park with a timeout ends unanswered when it expires, like a
		// waitForEvent: a queue item consumes the marker then. Consuming the marker
		// first (a message, or close) removes the item.
		if sub.Timeout != "" {
			jobID := fmt.Sprintf("%s-%s", md.IdempotencyKey(), gen.ID)
			timeout := queue.Item{
				JobID:                 &jobID,
				WorkspaceID:           md.ID.Tenant.EnvID,
				GroupID:               runCtx.GroupID(),
				Kind:                  queue.KindPause,
				Identifier:            sv2.V1FromMetadata(*md),
				PriorityFactor:        runCtx.PriorityFactor(),
				CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
				Payload:               queue.PayloadPauseTimeout{PauseID: park.ID, Pause: park},
				Metadata:              make(map[string]any),
				MaxAttempts:           runCtx.MaxAttempts(),
				ParallelMode:          gen.ParallelMode(),
			}
			if err := e.queue.Enqueue(ctx, timeout, parkExpires, queue.EnqueueOpts{}); err != nil && !errors.Is(err, queue.ErrQueueItemExists) {
				return fmt.Errorf("error enqueueing the park timeout: %w", err)
			}
		}

		// A message may have been saved after the SDK read its state and before
		// the marker existed: its delivery found no marker, so wake now. The
		// stack is read again after the marker is written: a message saved
		// before this read is seen here, and one saved after it finds the marker.
		stack, err = e.smv2.LoadStack(ctx, md.ID)
		if err != nil {
			return err
		}
		// Still waiting: the marker is written, and a delivery wakes it. A copy of
		// this park woken in the meantime has its answer saved: nothing to do.
		switch parkStateOf(stack, sub.Key, gen.ID, sub.Seq) {
		case parkAnswered:
			// An earlier copy was answered between the first read and the write,
			// so the marker just written belongs to a park that is over. Remove it:
			// a later message would consume it and queue a stale discovery.
			if err := e.pm.Delete(ctx, parkIdx, park); err != nil && !errors.Is(err, state.ErrPauseNotFound) {
				return fmt.Errorf("error deleting stale subscription park: %w", err)
			}
			// And the timeout item enqueued for it above.
			if sub.Timeout != "" {
				e.dequeuePauseJob(ctx, md, fmt.Sprintf("%s-%s", md.IdempotencyKey(), gen.ID))
			}
			return nil
		case parkReady:
			return e.consumePark(ctx, &park, false)
		}

		// Still waiting. A subscription closed in the same batch may have run its
		// close before this marker existed, and then nobody would end this park.
		// A closed subscription has its closed step: end the park now.
		if slices.Contains(stack, subscriptionClosedID(sub.Key)) {
			return e.consumePark(ctx, &park, true)
		}

		// Reads parked at once are bounded: a delivery reads every marker. The
		// count includes this one, written above.
		markers, err := e.parkedMarkers(ctx, md.ID, sub.Key)
		if err != nil {
			return err
		}
		waiting := 0
		for _, marker := range markers {
			if marker.Outgoing == "" || !slices.Contains(stack, marker.Outgoing) {
				waiting++
			}
		}
		if waiting > consts.MaxSubscriptionParks {
			if err := e.pm.Delete(ctx, parkIdx, park); err != nil && !errors.Is(err, state.ErrPauseNotFound) {
				return fmt.Errorf("error deleting over-limit subscription park: %w", err)
			}
			if sub.Timeout != "" {
				e.dequeuePauseJob(ctx, md, fmt.Sprintf("%s-%s", md.IdempotencyKey(), gen.ID))
			}
			return subscriptionLimitError("A subscription can have at most %d reads waiting at once.", consts.MaxSubscriptionParks)
		}

		return nil

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

// parkSeq is the message a wait pause waits for, or 0 when the pause does not say.
// The metadata is a JSON number once it has been stored.
func parkSeq(pause *state.Pause) int {
	switch seq := pause.Metadata["seq"].(type) {
	case int:
		return seq
	case int64:
		return int(seq)
	case float64:
		return int(seq)
	}

	return 0
}

// isSubscriptionParkPause reports whether a pause is the one a run blocks on while it
// waits for a subscription message. It has no timeout job.
func isSubscriptionParkPause(pause *state.Pause) bool {
	return pause.Event != nil && strings.HasPrefix(*pause.Event, pauses.SubscriptionParkIndexPrefix)
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

// wakeSubscriber wakes the run if it is parked on this subscription. A delivery
// to a busy run finds no marker and costs nothing more. When there is a marker it
// reads the stack once: a marker whose park is already answered is skipped, and
// so is one that awaits a message that is not saved yet (another delivery of the
// same burst saved its message first and the run has since parked for the
// next).
func (e *executor) wakeSubscriber(ctx context.Context, id sv2.ID, key string) error {
	parked, err := e.parkedMarkers(ctx, id, key)
	if err != nil {
		return err
	}
	if len(parked) == 0 {
		return nil
	}

	stack, err := e.smv2.LoadStack(ctx, id)
	if err != nil {
		return err
	}

	for _, pause := range markersToWake(parked, stack, key) {
		if err := e.consumePark(ctx, pause, false); err != nil {
			return err
		}
	}

	return nil
}

// markersToWake picks the markers a delivery may consume: those whose awaited
// message is saved and whose park is not answered yet.
func markersToWake(parked []*state.Pause, stack []string, key string) []*state.Pause {
	saved := len(subscriptionMessageIDs(stack, key))

	var ready []*state.Pause
	for _, pause := range parked {
		if pause.Outgoing != "" && slices.Contains(stack, pause.Outgoing) {
			continue
		}
		if saved < parkSeq(pause) {
			continue
		}
		ready = append(ready, pause)
	}

	return ready
}

// parkedMarkers returns the markers of the run's parks on a subscription.
// Normally one: the current park. A marker already consumed and not yet deleted
// may show too; consuming it again is idempotent.
func (e *executor) parkedMarkers(ctx context.Context, id sv2.ID, key string) ([]*state.Pause, error) {
	parkIdx := pauses.Index{WorkspaceID: id.Tenant.EnvID, EventName: subscriptionParkIndexName(id.RunID.String(), key)}

	iter, err := e.pm.PausesSince(ctx, parkIdx, time.Time{})
	if err != nil {
		return nil, err
	}

	var parked []*state.Pause
	for iter.Next(ctx) {
		if pause := iter.Val(ctx); pause != nil {
			parked = append(parked, pause)
		}
	}
	if err := iter.Error(); err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	return parked, nil
}

// consumePark consumes a marker so the run sees the message it waits for. Its
// output names that message's position, or says the subscription was closed
// (closed), so the reader can tell the end of a subscription from a message
// that is missing. Another handler may have consumed the
// marker first: the store then answers that it did not consume (DidConsume
// false, or an idempotent success) and nothing more is queued.
func (e *executor) consumePark(ctx context.Context, pause *state.Pause, closed bool) error {
	answer := map[string]any{"seq": parkSeq(pause)}
	if closed {
		answer = map[string]any{"closed": true}
	}

	err := e.Resume(ctx, *pause, execution.ResumeRequest{
		With:     map[string]any{"data": answer},
		StepName: pause.StepName,
	})
	if errors.Is(err, state.ErrPauseLeased) || errors.Is(err, state.ErrPauseNotFound) {
		return nil
	}
	return err
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
			return e.removeSubscriptionPause(ctx, &md, idx, pause)
		}
	}
	_, err = e.smv2.SaveStep(ctx, id, messageID, data)
	redelivered := err != nil
	if err != nil && !errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
		return err
	}

	l.Debug("subscription message saved", "message_id", messageID, "redelivered", redelivered)

	return e.wakeSubscriber(ctx, id, key)
}

// parkHasTimeout reports whether a park marker has a timeout job in the queue.
func parkHasTimeout(pause *state.Pause) bool {
	timeout, _ := pause.Metadata["timeout"].(bool)
	return timeout
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
func validateSubscribe(sub *state.SubscribeOpts, event string) error {
	if sub.Key == "" {
		return subscriptionLimitError("The subscription ID is empty.")
	}
	if len(sub.Key) > consts.MaxSubscriptionKeyLength {
		return subscriptionLimitError("The subscription ID is longer than %d characters.", consts.MaxSubscriptionKeyLength)
	}
	if len(sub.Reader) > consts.MaxSubscriptionKeyLength {
		return subscriptionLimitError("The subscription reader name is longer than %d characters.", consts.MaxSubscriptionKeyLength)
	}
	if sub.Stream < 0 || sub.Stream >= consts.MaxSubscriptionPausesPerRun {
		return subscriptionLimitError("A subscription can listen to at most %d events.", consts.MaxSubscriptionPausesPerRun)
	}
	if sub.Attempt < 0 {
		return subscriptionLimitError("The subscription attempt is negative.")
	}
	if (sub.Mode == state.SubscribePark || sub.Mode == state.SubscribePoll) && sub.Seq < 1 {
		return subscriptionLimitError("The message number of a subscription read starts at 1.")
	}
	if sub.Mode == state.SubscribeOpen && strings.HasPrefix(event, pauses.SubscriptionParkIndexPrefix) {
		return subscriptionLimitError("A subscription cannot listen to the event %q: the name is reserved.", event)
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

// endSubscription marks a subscription closed and ends every read parked on it.
// The data of the closed step is "true", or says which limit closed it.
func (e *executor) endSubscription(ctx context.Context, id sv2.ID, key string, data []byte) error {
	// Mark the subscription closed, so a park written after this still ends.
	if _, err := e.smv2.SaveStep(ctx, id, subscriptionClosedID(key), data); err != nil &&
		!errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
		return err
	}

	// A read that is still parked (one the function stopped waiting for, such as
	// a lost race) must not keep the run waiting: end every park now, so the
	// steps they stand for are answered and discovery can follow the close.
	parked, err := e.parkedMarkers(ctx, id, key)
	if err != nil {
		return err
	}
	for _, marker := range parked {
		if err := e.consumePark(ctx, marker, true); err != nil {
			return err
		}
	}

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
