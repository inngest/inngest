package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// subscriptionHints remembers the next free message slot per subscription
// pause, so saving a message does not probe from slot 1 every time. It is only
// a hint: probing is the source of truth, so a stale or missing hint is safe.
var subscriptionHints sync.Map

func subscriptionPauseID(runID string, key string) uuid.UUID {
	return inngest.DeterministicSha1UUID(runID + ":subscription:" + key)
}

// handleGeneratorSubscribe handles the WaitForEvent ops of a step.subscribe
// subscription: open, next, poll and close.
func (e *executor) handleGeneratorSubscribe(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, group OpcodeGroup, opts *state.WaitForEventOpts) error {
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
			Expression:  opts.If,
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
			// Not here yet. The delivery of message Seq enqueues discovery.
			return nil
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

	data, err := json.Marshal(map[string]any{"data": eventMap})
	if err != nil {
		return err
	}

	seq := 1
	if hint, ok := subscriptionHints.Load(pause.ID); ok {
		seq = hint.(int)
	}

	var stepID string
	for ; ; seq++ {
		stepID = subscriptionStepID(key, seq)
		_, err := e.smv2.SaveStep(ctx, id, stepID, data)
		if errors.Is(err, state.ErrDuplicateResponse) {
			// Another message has this slot.
			continue
		}
		if err != nil && !errors.Is(err, state.ErrIdempotentResponse) {
			// A finished run (state deleted) cannot take the message. The
			// finalizer re-emits unread messages, so this one is handled there
			// or has been dropped with the run.
			return err
		}
		break
	}
	subscriptionHints.Store(pause.ID, seq+1)

	l.Debug("subscription message saved", "seq", seq, "step_id", stepID)

	return e.resumeSubscriber(ctx, pause, stepID, data)
}

var subscriptionReemitDelays = []time.Duration{0, time.Second, 3 * time.Second, 8 * time.Second}

// unreadSubscriptionEvents returns the events of the run's subscriptions that
// were saved as messages but never acked, in order. Call it before the run
// state is deleted.
func (e *executor) unreadSubscriptionEvents(ctx context.Context, id sv2.ID) ([]event.Event, error) {
	pauseIDs, err := e.pm.PauseIDsForRun(ctx, id.RunID)
	if err != nil {
		return nil, err
	}

	var unread []event.Event
	for _, pauseID := range pauseIDs {
		pause, err := e.pm.PauseByID(ctx, pauses.Index{WorkspaceID: id.Tenant.EnvID}, pauseID)
		if err != nil || pause == nil || pause.Subscribe == nil {
			continue
		}
		key := pause.Subscribe.Key

		saved, err := e.countSubscriptionMessages(ctx, id, key)
		if err != nil {
			return nil, err
		}
		acked, err := e.countSubscriptionAcks(ctx, id, key)
		if err != nil {
			return nil, err
		}
		if acked >= saved {
			continue
		}

		ids := make([]string, 0, saved-acked)
		for seq := acked + 1; seq <= saved; seq++ {
			ids = append(ids, subscriptionStepID(key, seq))
		}
		steps, err := e.smv2.LoadStepsWithIDs(ctx, id, ids)
		if err != nil {
			return nil, err
		}
		for _, stepID := range ids {
			var stored struct {
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(steps[stepID], &stored); err != nil || stored.Data == nil {
				continue
			}
			byt, err := json.Marshal(stored.Data)
			if err != nil {
				continue
			}
			evt, err := event.NewEvent(byt)
			if err != nil {
				continue
			}
			unread = append(unread, *evt)
		}
	}

	return unread, nil
}

// countSubscriptionAcks is how many consecutive messages the run acked.
func (e *executor) countSubscriptionAcks(ctx context.Context, id sv2.ID, key string) (int, error) {
	count := 0
	for {
		ids := make([]string, 16)
		for i := range ids {
			ids[i] = subscriptionAckID(key, count+i+1)
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
		count += len(ids)
	}
}

// republishSubscriptionEvents publishes events that a finished run did not
// read, again after each delay. Their internal IDs are derived from their
// event IDs, so scheduling is idempotent: a start that the thread singleton
// skipped while the old run was still finishing is retried until a run takes
// it, and an event that already started a run is skipped.
func (e *executor) republishSubscriptionEvents(ctx context.Context, id sv2.ID, events []event.Event) {
	if len(events) == 0 || e.finishHandler == nil {
		return
	}

	ctx = context.WithoutCancel(ctx)
	// The first publish rides with the finalize events; these are the retries.
	for _, delay := range subscriptionReemitDelays[1:] {
		time.AfterFunc(delay, func() {
			if err := e.finishHandler(ctx, id, events); err != nil {
				logger.StdlibLogger(ctx).Warn("error republishing unread subscription events", "error", err, "run_id", id.RunID)
			}
		})
	}
}

// reemitSubscriptionEvent publishes an event that matched the subscription of
// a run that has already finished, now and again after each delay.
func (e *executor) reemitSubscriptionEvent(ctx context.Context, id sv2.ID, evt event.Event) {
	if e.finishHandler == nil {
		return
	}

	events := []event.Event{evt}
	if err := e.finishHandler(context.WithoutCancel(ctx), id, events); err != nil {
		logger.StdlibLogger(ctx).Warn("error re-emitting subscription event", "error", err, "run_id", id.RunID)
	}
	e.republishSubscriptionEvents(ctx, id, events)
}
