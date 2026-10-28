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

// subscriptionStepID is the step ID of the wait for message seq. The SDK gives
// its blocking next() the same ID.
func subscriptionStepID(key string, seq int) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%d", key, seq)))
	return hex.EncodeToString(sum[:])
}

// subscriptionParkID is the ID of the parked marker of a run and subscription.
// The SDK reads one message at a time, so a run has at most one park per
// subscription and a delivery can find it without reading the stack. Consuming
// a marker deletes it by ID after queuing discovery, and the run may park again
// straight away, so two slots alternate by the parity of the awaited message:
// the next park never shares an ID with the one being cleaned up.
func subscriptionParkID(runID string, key string, slot int) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription-park:%s:%d", runID, key, slot%2))
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
	if sub.Key == "" {
		return fmt.Errorf("subscription key is empty")
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
			Subscribe:   &state.PauseSubscribe{Key: sub.Key},
			CreatedAt:   now,
		}
		if len(md.Config.EventIDs) > 0 {
			// The event that started the run is its first message, not a
			// subscription match.
			trigger := md.Config.EventIDs[0].String()
			pause.TriggeringEventID = &trigger
		}

		_, err := e.pm.Write(ctx, idx, &pause)
		if err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
			return fmt.Errorf("error saving subscription pause: %w", err)
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
		if gen.ID != subscriptionStepID(sub.Key, sub.Seq) {
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
		parkIdx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: consts.SubscriptionWaitName}
		opcode := gen.Op.String()
		park := state.Pause{
			ID:           subscriptionParkID(runID, sub.Key, sub.Seq),
			WorkspaceID:  md.ID.Tenant.EnvID,
			Identifier:   sv2.NewPauseIdentifier(md.ID),
			GroupID:      runCtx.GroupID(),
			Outgoing:     gen.ID,
			Incoming:     edge.Edge.Incoming,
			StepName:     gen.UserDefinedName(),
			Opcode:       &opcode,
			Expires:      state.Time(e.now().Add(consts.MaxWaitForEventTimeout)),
			Event:        new(consts.SubscriptionWaitName),
			DataKey:      gen.ID,
			MaxAttempts:  runCtx.MaxAttempts(),
			ParallelMode: gen.ParallelMode(),
			CreatedAt:    e.now(),
			Metadata:     map[string]any{"key": sub.Key, "seq": sub.Seq},
		}
		if _, err := e.pm.Write(ctx, parkIdx, &park); err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
			return fmt.Errorf("error saving subscription park: %w", err)
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
		if parkStateOf(stack, sub.Key, gen.ID, sub.Seq) != parkReady {
			return nil
		}
		return e.consumeWait(ctx, &park)

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

	case state.SubscribeClose:
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

		ids, err := messages()
		if err != nil {
			return err
		}
		return answer(map[string]any{"count": len(ids)})
	}

	return fmt.Errorf("unknown subscription mode %q", sub.Mode)
}

// waitSeq is the message a wait pause waits for, or 0 when the pause does not say.
// The metadata is a JSON number once it has been stored.
func waitSeq(pause *state.Pause) int {
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

// isSubscriptionWaitPause reports whether a pause is the one a run blocks on while it
// waits for a subscription message. It has no timeout job.
func isSubscriptionWaitPause(pause *state.Pause) bool {
	return pause.Event != nil && *pause.Event == consts.SubscriptionWaitName
}

// dequeueSubscriptionExpiry removes the queue job that would expire a subscription
// pause, once the pause is closed.
func (e *executor) dequeueSubscriptionExpiry(ctx context.Context, md *sv2.Metadata, pauseID uuid.UUID) {
	if e.shards == nil {
		return
	}

	shard, err := e.shards.Resolve(ctx, queue.Scope{
		AccountID:  md.ID.Tenant.AccountID,
		EnvID:      md.ID.Tenant.EnvID,
		FunctionID: md.ID.FunctionID,
	}, nil)
	if err != nil {
		logger.StdlibLogger(ctx).Warn("could not find shard to dequeue the subscription expiry", "error", err)
		return
	}

	err = e.queue.Dequeue(ctx, shard.Name(), queue.QueueItem{
		ID:         queue.HashID(ctx, subscriptionExpiryJobID(md, pauseID)),
		FunctionID: md.ID.FunctionID,
		Data: queue.Item{
			Kind:       queue.KindPause,
			Identifier: sv2.V1FromMetadata(*md),
		},
	})
	if err != nil && !errors.Is(err, queue.ErrQueueItemNotFound) {
		logger.StdlibLogger(ctx).Error("error dequeueing the subscription expiry", "error", err)
	}
}

// wakeSubscriber wakes the run if it is parked on this subscription. A new
// message wakes the marker without reading the stack. A redelivered event is a
// message that is already saved, so it only wakes a run whose awaited message
// exists.
func (e *executor) wakeSubscriber(ctx context.Context, id sv2.ID, key string, redelivered bool) error {
	parkIdx := pauses.Index{WorkspaceID: id.Tenant.EnvID, EventName: consts.SubscriptionWaitName}

	for slot := 0; slot < 2; slot++ {
		pause, err := e.pm.PauseByID(ctx, parkIdx, subscriptionParkID(id.RunID.String(), key, slot))
		if errors.Is(err, state.ErrPauseNotFound) || (err == nil && pause == nil) {
			continue
		}
		if err != nil {
			return err
		}

		if redelivered {
			stack, err := e.smv2.LoadStack(ctx, id)
			if err != nil {
				return err
			}
			if len(subscriptionMessageIDs(stack, key)) < waitSeq(pause) {
				continue
			}
		}

		// A marker that was just consumed and not yet deleted is leased: ignored.
		if err := e.consumeWait(ctx, pause); err != nil {
			return err
		}
	}

	return nil
}

// consumeWait consumes a wait pause so the run sees the message it waits for.
// Its output names that message's position.
func (e *executor) consumeWait(ctx context.Context, pause *state.Pause) error {
	awaited := waitSeq(pause)

	err := e.Resume(ctx, *pause, execution.ResumeRequest{
		With:     map[string]any{"data": map[string]any{"seq": awaited}},
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
	// exists, so check first: a finished run cannot take the message.
	if exists, err := e.smv2.Exists(ctx, id); err != nil {
		return err
	} else if !exists {
		return state.ErrRunNotFound
	}

	data, err := json.Marshal(map[string]any{"data": evt.GetEvent().Map()})
	if err != nil {
		return err
	}

	// The same event saves the same bytes under the same ID: a redelivery is
	// the message it already is.
	messageID := subscriptionMessageID(key, eventID)
	_, err = e.smv2.SaveStep(ctx, id, messageID, data)
	redelivered := err != nil
	if err != nil && !errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
		return err
	}

	l.Debug("subscription message saved", "message_id", messageID, "redelivered", redelivered)

	return e.wakeSubscriber(ctx, id, key, redelivered)
}
