package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
)

// subscriptionStepID is the state step ID of message seq of a subscription.
// The SDK gives its seq-th next() step the same ID, so a message that arrived
// before the SDK asks for it is already the step's memoized output.
func subscriptionStepID(key string, seq int) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%d", key, seq)))
	return hex.EncodeToString(sum[:])
}

// subscriptionAckID is the step ID of the ack of message seq.
func subscriptionAckID(key string, seq int) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:ack:%d", key, seq)))
	return hex.EncodeToString(sum[:])
}

// subscriptionWaiterID is the step ID that records that the run is blocked
// waiting for message seq.
func subscriptionWaiterID(key string, seq int) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:wait:%d", key, seq)))
	return hex.EncodeToString(sum[:])
}

// subscriptionEventMarkerID is the step ID that records an event as handled by
// a subscription, so an event that is handled twice is saved once.
func subscriptionEventMarkerID(key string, eventID string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:event:%s", key, eventID)))
	return hex.EncodeToString(sum[:])
}

// subscriptionEventSlotID is the step ID that records which slot an event was
// saved to.
func subscriptionEventSlotID(key string, eventID string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:slot:%s", key, eventID)))
	return hex.EncodeToString(sum[:])
}

// subscriptionHints remembers the next free message slot per subscription
// pause, so saving a message does not probe from slot 1 every time. It is only
// a hint: probing is the source of truth, so a stale or missing hint is safe.
var subscriptionHints sync.Map

func subscriptionPauseID(runID string, key string) uuid.UUID {
	return inngest.DeterministicSha1UUID(runID + ":subscription:" + key)
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
	pauseID := subscriptionPauseID(runID, sub.Key)
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
		jobID := fmt.Sprintf("%s-%s-expiry", md.IdempotencyKey(), gen.ID)
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

		if len(md.Config.EventIDs) > 0 {
			// Mark the trigger as handled, in case it also matches.
			marker := subscriptionEventMarkerID(sub.Key, md.Config.EventIDs[0].String())
			if _, err := e.smv2.SaveStep(ctx, md.ID, marker, []byte(`{"data":0}`)); err != nil &&
				!errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
				return fmt.Errorf("error marking trigger event: %w", err)
			}
		}
		return answer(map[string]any{"opened": true})

	case state.SubscribeNext:
		if gen.ID != subscriptionStepID(sub.Key, sub.Seq) {
			return fmt.Errorf("subscription step ID does not match message %d", sub.Seq)
		}

		saved, err := e.smv2.LoadStepsWithIDs(ctx, md.ID, []string{gen.ID})
		if err != nil {
			return err
		}
		raw, ok := saved[gen.ID]
		if !ok {
			// Not here yet. Record that the run is blocked on it, so the
			// delivery of message Seq queues discovery. Discovery is only
			// queued for a blocked run: while one is in flight, a second
			// request would run its steps twice.
			waiter := subscriptionWaiterID(sub.Key, sub.Seq)
			if _, err := e.smv2.SaveStep(ctx, md.ID, waiter, []byte(`{"data":1}`)); err != nil &&
				!errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
				return err
			}

			// The message may have arrived between the check and the marker.
			saved, err = e.smv2.LoadStepsWithIDs(ctx, md.ID, []string{gen.ID})
			if err != nil {
				return err
			}
			if raw, ok = saved[gen.ID]; !ok {
				return nil
			}
		}

		// The message arrived before the SDK asked for it. Queue discovery by
		// resuming with the saved data; saving it again is idempotent.
		pause, err := e.pm.PauseByID(ctx, idx, pauseID)
		if err != nil || pause == nil {
			return fmt.Errorf("subscription pause not found: %w", err)
		}
		return e.resumeSubscriber(ctx, pause, gen.ID, raw)

	case state.SubscribeAck:
		if gen.ID != subscriptionAckID(sub.Key, sub.Seq) {
			return fmt.Errorf("subscription step ID does not match ack %d", sub.Seq)
		}
		return answer(true)

	case state.SubscribePoll:
		id := subscriptionStepID(sub.Key, sub.Seq)
		saved, err := e.smv2.LoadStepsWithIDs(ctx, md.ID, []string{id})
		if err != nil {
			return err
		}
		raw, ok := saved[id]
		if !ok {
			return answer(nil)
		}
		var stored struct {
			Data any `json:"data"`
		}
		if err := json.Unmarshal(raw, &stored); err != nil {
			return err
		}
		return answer(stored.Data)

	case state.SubscribeClose:
		pause, err := e.pm.PauseByID(ctx, idx, pauseID)
		if err == nil && pause != nil {
			if err := e.pm.Delete(ctx, idx, *pause); err != nil {
				return fmt.Errorf("error deleting subscription pause: %w", err)
			}
			if e.exprAggregator != nil {
				_ = e.exprAggregator.RemovePause(ctx, pause)
			}
		}
		subscriptionHints.Delete(pauseID)

		count, err := e.countSubscriptionMessages(ctx, md.ID, sub.Key)
		if err != nil {
			return err
		}
		return answer(map[string]any{"count": count})
	}

	return fmt.Errorf("unknown subscription mode %q", sub.Mode)
}

// countSubscriptionMessages is how many messages the subscription has saved.
func (e *executor) countSubscriptionMessages(ctx context.Context, id sv2.ID, key string) (int, error) {
	const batch = 32
	count := 0
	for {
		ids := make([]string, batch)
		for i := range ids {
			ids[i] = subscriptionStepID(key, count+i+1)
		}
		saved, err := e.smv2.LoadStepsWithIDs(ctx, id, ids)
		if err != nil {
			return 0, err
		}
		for i := range ids {
			if _, ok := saved[ids[i]]; !ok {
				return count + i, nil
			}
		}
		count += batch
	}
}

// resumeSubscriber saves the bytes as the output of the step dataKey and queues
// the discovery request that lets the run see it. It does not consume the pause.
func (e *executor) resumeSubscriber(ctx context.Context, pause *state.Pause, dataKey string, saved []byte) error {
	// ConsumePause stores what it is given as JSON. Handing it the stored
	// message back as a map of raw JSON writes the same bytes, so saving it
	// again is idempotent.
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(saved, &wrapper); err != nil {
		return fmt.Errorf("stored subscription message is not JSON: %w", err)
	}

	clone := *pause
	clone.ID = uuid.New()
	clone.DataKey = dataKey
	clone.Outgoing = dataKey
	clone.Subscribe = nil

	err := e.Resume(ctx, clone, execution.ResumeRequest{
		With:      map[string]any{"data": wrapper["data"]},
		StepName:  pause.StepName,
		EventName: derefString(pause.Event),
	})
	if errors.Is(err, state.ErrPauseLeased) || errors.Is(err, state.ErrPauseNotFound) {
		return nil
	}
	return err
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// deliverSubscription saves an event that matched a subscription pause as the
// next numbered message of the run and queues discovery. The pause stays.
func (e *executor) deliverSubscription(ctx context.Context, evt event.TrackedEvent, pause *state.Pause, l logger.Logger) error {
	id := sv2.IDFromPause(*pause)
	key := pause.Subscribe.Key
	eventMap := evt.GetEvent().Map()

	// SaveStep writes to the run's hashes whether or not the run still
	// exists, so check first: a finished run cannot take the message.
	if exists, err := e.smv2.Exists(ctx, id); err != nil {
		return err
	} else if !exists {
		subscriptionHints.Delete(pause.ID)
		return state.ErrRunNotFound
	}

	data, err := json.Marshal(map[string]any{"data": eventMap})
	if err != nil {
		return err
	}

	// Claim the event first: an event that matches twice (a redelivery, two
	// executors) is saved once. The slot it was saved to is recorded after, so
	// a retry of an interrupted delivery can still wake the run.
	marker := subscriptionEventMarkerID(key, evt.GetInternalID().String())
	slotMarker := subscriptionEventSlotID(key, evt.GetInternalID().String())

	var seq int
	if _, err := e.smv2.SaveStep(ctx, id, marker, []byte(`{"data":0}`)); err != nil {
		if !errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
			return err
		}

		// Already claimed. If it was saved, make sure the run was woken.
		saved, err := e.smv2.LoadStepsWithIDs(ctx, id, []string{slotMarker})
		if err != nil {
			return err
		}
		raw, ok := saved[slotMarker]
		if ok {
			var slot struct {
				Data int `json:"data"`
			}
			if err := json.Unmarshal(raw, &slot); err != nil || slot.Data < 1 {
				return nil
			}
			seq = slot.Data
		}
	}

	if seq == 0 {
		// Not saved yet: either the first delivery, or a retry of one that was
		// interrupted. Probing is idempotent: the slot that already holds this
		// event answers ErrIdempotentResponse, and other slots are skipped.
		seq = 1
		if hint, ok := subscriptionHints.Load(pause.ID); ok {
			seq = hint.(int)
		}

		for ; ; seq++ {
			_, err := e.smv2.SaveStep(ctx, id, subscriptionStepID(key, seq), data)
			if errors.Is(err, state.ErrDuplicateResponse) {
				// Another message has this slot.
				continue
			}
			if err != nil && !errors.Is(err, state.ErrIdempotentResponse) {
				return err
			}
			break
		}
		subscriptionHints.Store(pause.ID, seq+1)

		slotData, _ := json.Marshal(map[string]any{"data": seq})
		if _, err := e.smv2.SaveStep(ctx, id, slotMarker, slotData); err != nil &&
			!errors.Is(err, state.ErrDuplicateResponse) && !errors.Is(err, state.ErrIdempotentResponse) {
			return err
		}
	}

	stepID := subscriptionStepID(key, seq)
	l.Debug("subscription message saved", "seq", seq, "step_id", stepID)

	// Only a run that is blocked on this message needs a discovery request.
	waiter, err := e.smv2.LoadStepsWithIDs(ctx, id, []string{subscriptionWaiterID(key, seq)})
	if err != nil {
		return err
	}
	if _, blocked := waiter[subscriptionWaiterID(key, seq)]; !blocked {
		return nil
	}

	return e.resumeSubscriber(ctx, pause, stepID, data)
}
