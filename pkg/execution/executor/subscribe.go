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

// subscriptionStepID is the step ID of the wait for message seq. The SDK gives
// its blocking next() the same ID.
func subscriptionStepID(key string, seq int, attempt int) string {
	id := fmt.Sprintf("%s:%d", key, seq)
	if attempt > 0 {
		id = fmt.Sprintf("%s:%d", id, attempt)
	}
	sum := sha1.Sum([]byte(id))
	return hex.EncodeToString(sum[:])
}

// subscriptionParkIndexName names the index of the parked markers of one run
// and subscription. It starts with consts.SubscriptionParkName. A delivery reads
// this one small index to find the marker without reading the run's stack.
func subscriptionParkIndexName(runID string, key string) string {
	return consts.SubscriptionParkName + ":" + runID + ":" + subscriptionKeyHash(key)
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
		if gen.ID != subscriptionStepID(sub.Key, sub.Seq, sub.Attempt) {
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
			return nil
		case parkReady:
			return e.consumePark(ctx, &park)
		}

		// Still waiting. A subscription closed in the same batch may have run its
		// close before this marker existed, and then nobody would end this park. A
		// closed subscription has no pause: end the park now.
		open, err := e.pm.PauseByID(ctx, idx, pauseID)
		if err != nil && !errors.Is(err, state.ErrPauseNotFound) {
			return err
		}
		if open == nil {
			return e.consumePark(ctx, &park)
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

		// A read that is still parked (one the function stopped waiting for, such as
		// a lost race) must not keep the run waiting: end every park now, so the
		// steps they stand for are answered and discovery can follow the close.
		parked, err := e.parkedMarkers(ctx, md.ID, sub.Key)
		if err != nil {
			return err
		}
		for _, marker := range parked {
			if err := e.consumePark(ctx, marker); err != nil {
				return err
			}
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
	return pause.Event != nil && strings.HasPrefix(*pause.Event, consts.SubscriptionParkName)
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
		if err := e.consumePark(ctx, pause); err != nil {
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
// output names that message's position. Another handler may have consumed the
// marker first: the store then answers that it did not consume (DidConsume
// false, or an idempotent success) and nothing more is queued.
func (e *executor) consumePark(ctx context.Context, pause *state.Pause) error {
	awaited := parkSeq(pause)

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

	return e.wakeSubscriber(ctx, id, key)
}

// parkHasTimeout reports whether a park marker has a timeout job in the queue.
func parkHasTimeout(pause *state.Pause) bool {
	timeout, _ := pause.Metadata["timeout"].(bool)
	return timeout
}
