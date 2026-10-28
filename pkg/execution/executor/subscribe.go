package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// subscriptionWaitEvent names the index of the pauses that block a run waiting
// for a message. No event has this name, so events never match them: the
// delivery of a message consumes them.
const subscriptionWaitEvent = "inngest/subscription.wait"

// subscriptionWaitPauseID is the ID of the pause a run blocks on while it waits
// for the next message of a subscription. There is at most one.
func subscriptionWaitPauseID(runID string, key string) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription-wait:%s", runID, key))
}

func subscriptionPauseID(runID string, key string, stream int) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription:%s:%d", runID, key, stream))
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

		return answer(map[string]any{"opened": true})

	case state.SubscribeNext:
		if gen.ID != subscriptionStepID(sub.Key, sub.Seq) {
			return fmt.Errorf("subscription step ID does not match message %d", sub.Seq)
		}

		// The run blocks on a pause that the delivery of the next message
		// consumes, like waitForEvent. The run only wakes when it is blocked: a
		// wake while a request is executing would run its steps twice.
		waitIdx := pauses.Index{WorkspaceID: md.ID.Tenant.EnvID, EventName: subscriptionWaitEvent}
		waitID := subscriptionWaitPauseID(runID, sub.Key)
		opcode := gen.Op.String()
		wait := state.Pause{
			ID:           waitID,
			WorkspaceID:  md.ID.Tenant.EnvID,
			Identifier:   sv2.NewPauseIdentifier(md.ID),
			GroupID:      runCtx.GroupID(),
			Outgoing:     gen.ID,
			Incoming:     edge.Edge.Incoming,
			StepName:     gen.UserDefinedName(),
			Opcode:       &opcode,
			Expires:      state.Time(e.now().Add(consts.MaxWaitForEventTimeout)),
			Event:        new(subscriptionWaitEvent),
			DataKey:      gen.ID,
			MaxAttempts:  runCtx.MaxAttempts(),
			ParallelMode: gen.ParallelMode(),
			CreatedAt:    e.now(),
			// The message the run waits for, so a redelivered event is only a wake
			// when that message exists.
			Metadata: map[string]any{"seq": sub.Seq},
		}
		_, err := e.pm.Write(ctx, waitIdx, &wait)
		if errors.Is(err, state.ErrPauseAlreadyExists) {
			// The same wait is written again when the op is sent again. A wait
			// for another step is stale: replace it.
			existing, lookupErr := e.pm.PauseByID(ctx, waitIdx, waitID)
			if lookupErr == nil && existing != nil && existing.DataKey != gen.ID {
				if err := e.pm.Delete(ctx, waitIdx, *existing); err != nil {
					return fmt.Errorf("error replacing subscription wait pause: %w", err)
				}
				_, err = e.pm.Write(ctx, waitIdx, &wait)
			} else {
				err = nil
			}
		}
		if err != nil {
			return fmt.Errorf("error saving subscription wait pause: %w", err)
		}

		// The message may have arrived before the wait existed: then nobody
		// delivers it, so wake the run now.
		ids, err := messages()
		if err != nil {
			return err
		}
		if len(ids) < sub.Seq {
			return nil
		}
		return e.wakeSubscriber(ctx, md.ID, sub.Key, ids[sub.Seq-1], false)

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
		}

		ids, err := messages()
		if err != nil {
			return err
		}
		return answer(map[string]any{"count": len(ids)})
	}

	return fmt.Errorf("unknown subscription mode %q", sub.Mode)
}

// wakeSubscriber consumes the pause a run is blocked on, if there is one, so the
// run sees the message. Its output is the ID of the message step.
func (e *executor) wakeSubscriber(ctx context.Context, id sv2.ID, key string, messageID string, verify bool) error {
	waitIdx := pauses.Index{WorkspaceID: id.Tenant.EnvID, EventName: subscriptionWaitEvent}
	pause, err := e.pm.PauseByID(ctx, waitIdx, subscriptionWaitPauseID(id.RunID.String(), key))
	if errors.Is(err, state.ErrPauseNotFound) || (err == nil && pause == nil) {
		return nil
	}
	if err != nil {
		return err
	}

	if verify {
		awaited, _ := pause.Metadata["seq"].(float64)
		stack, err := e.smv2.LoadStack(ctx, id)
		if err != nil {
			return err
		}
		if float64(len(subscriptionMessageIDs(stack, key))) < awaited {
			return nil
		}
	}

	err = e.Resume(ctx, *pause, execution.ResumeRequest{
		With:     map[string]any{"data": messageID},
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

	// A new message is the one a blocked run waits for. A redelivery may be an
	// old message, so it only wakes a run whose awaited message exists.
	return e.wakeSubscriber(ctx, id, key, messageID, redelivered)
}
