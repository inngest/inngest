package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/oklog/ulid/v2"
)

// singletonJoinGrace is how long a completed run's result, and its waiter set,
// are kept after the run finalizes. The singleton lock is released shortly
// after finalization (when the last queue item is dequeued), so this only has
// to cover that gap and retried finalizations.
const singletonJoinGrace = 10 * time.Minute

// Singleton "join" mode.
//
// When a function uses singleton mode "join" and a run triggered by step.invoke
// would be skipped because another run holds the singleton lock, the invoking
// run would otherwise wait for its invoke to time out: the skipped run never
// emits inngest/function.finished for the invoker's correlation ID.
//
// Instead, the skipped run's correlation ID is registered as a waiter on the
// active run. When the active run finalizes it emits one extra
// inngest/function.finished per waiter, carrying the waiter's correlation ID
// and the active run's result or error. That event travels the normal invoke
// resume path (HandleInvokeFinish -> Resume), so pause consumption and the
// resume idempotency key give exactly-once resumes, and the invoker's own
// pause timeout still applies.
//
// Registration (Join) and completion (Complete) are single Redis scripts, so a
// waiter is always either returned by Complete or handed the completion
// payload by Join; there is no interleaving in which it is neither.

// joinSingletonRun registers the invoke that triggered req as a waiter on
// activeRunID. It is a no-op unless req was triggered by step.invoke.
func (e *executor) joinSingletonRun(ctx context.Context, req execution.ScheduleRequest, activeRunID ulid.ULID) error {
	evt := req.Events[0].GetEvent()
	corrID := evt.CorrelationID()
	if !evt.IsInvokeEvent() || corrID == "" {
		return nil
	}

	// The correlation ID is "<invoker run ID>.<step ID>". A run cannot wait on
	// itself, so fail the invoke instead of deadlocking until it times out.
	if strings.HasPrefix(corrID, activeRunID.String()+".") {
		return e.failJoin(ctx, req, "singleton join: a run cannot join itself")
	}

	ttl := singletonJoinGrace
	if md, err := evt.InngestMetadata(); err == nil && md.InvokeExpiresAt > 0 {
		if remaining := time.UnixMilli(md.InvokeExpiresAt).Sub(e.now()); remaining > 0 {
			ttl += remaining
		}
	}

	scope := queue.Scope{
		AccountID:  req.AccountID,
		EnvID:      req.WorkspaceID,
		FunctionID: req.Function.ID,
	}

	payload, err := e.singletonMgr.Join(ctx, scope, activeRunID, corrID, ttl)
	if err != nil {
		return fmt.Errorf("error joining singleton run: %w", err)
	}
	if payload == nil {
		// Registered. The active run's finalization resolves this invoke.

		// Only runs scheduled in join mode resolve their waiters when they
		// finalize. A run that predates the mode being enabled never will, so
		// fail fast rather than wait out the invoke timeout. Best effort: a
		// missing or unreadable run is mid-finalize and handled by Complete.
		if md, err := e.smv2.LoadMetadata(ctx, sv2.ID{RunID: activeRunID, FunctionID: req.Function.ID, Tenant: sv2.Tenant{AccountID: req.AccountID, EnvID: req.WorkspaceID}}, sv2.OmitStackAndStepMetrics()); err == nil && !md.Config.SingletonJoin() {
			return e.failJoin(ctx, req, "singleton join: the active run was started before join mode was enabled and cannot be joined")
		}

		return nil
	}

	// The active run finalized before we could register. Resolve from its
	// stored result.
	activeID := sv2.ID{
		RunID:      activeRunID,
		FunctionID: req.Function.ID,
		Tenant: sv2.Tenant{
			AccountID: req.AccountID,
			EnvID:     req.WorkspaceID,
		},
	}
	evts, err := joinEvents(e.now(), payload, []string{corrID})
	if err != nil {
		return err
	}

	return e.publishJoinEvents(ctx, activeID, evts)
}

// joinAfterSingletonConflict handles losing the atomic race for the singleton
// lock at enqueue time: it finds the run that won and joins it.
func (e *executor) joinAfterSingletonConflict(ctx context.Context, req execution.ScheduleRequest, key string) error {
	cfg := *req.Function.Singleton
	cfg.Mode = enums.SingletonModeSkip // look up only

	activeRunID, err := e.singletonMgr.HandleSingleton(ctx, queue.Scope{
		AccountID:  req.AccountID,
		EnvID:      req.WorkspaceID,
		FunctionID: req.Function.ID,
	}, key, cfg)
	if err != nil {
		return err
	}

	if activeRunID == nil {
		// The lock was released between the conflict and this lookup. Never
		// leave an invoker hanging: fail it so it can fall back.
		return e.failJoin(ctx, req, "singleton join: the active run ended before it could be joined")
	}

	return e.joinSingletonRun(ctx, req, *activeRunID)
}

// failJoin resolves the invoke that triggered req with an error.
func (e *executor) failJoin(ctx context.Context, req execution.ScheduleRequest, message string) error {
	if !req.Events[0].GetEvent().IsInvokeEvent() {
		return nil
	}

	return e.InvokeFailHandler(ctx, execution.InvokeFailHandlerOpts{
		OriginalEvent: req.Events[0],
		Err: map[string]any{
			"name":    "Error",
			"message": message,
		},
	})
}

// completeSingletonJoin is called when a join-mode run finalizes. It stores the
// run's result for late joiners and returns one inngest/function.finished event
// per registered waiter.
func (e *executor) completeSingletonJoin(ctx context.Context, opts execution.FinalizeOpts, base functionFinishedData) ([]event.Event, error) {
	if e.singletonMgr == nil {
		return nil, nil
	}

	// Waiters only need the outcome, not the triggering events.
	base.Events = nil
	data := base.Map()
	data[consts.InngestEventDataPrefix] = map[string]any{"status": opts.Status()}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshalling singleton join result: %w", err)
	}

	id := opts.Metadata.ID
	waiters, err := e.singletonMgr.Complete(ctx, queue.Scope{
		AccountID:  id.Tenant.AccountID,
		EnvID:      id.Tenant.EnvID,
		FunctionID: id.FunctionID,
	}, id.RunID, payload, singletonJoinGrace)
	if err != nil {
		return nil, fmt.Errorf("error completing singleton join: %w", err)
	}

	return joinEvents(e.now(), payload, waiters)
}

// joinEvents builds an inngest/function.finished event for each waiter from
// the active run's result payload.
func joinEvents(now time.Time, payload []byte, waiters []string) ([]event.Event, error) {
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("error reading singleton join result: %w", err)
	}

	evts := make([]event.Event, 0, len(waiters))
	for _, corrID := range waiters {
		data := maps.Clone(result)
		data[consts.InvokeCorrelationId] = corrID

		evts = append(evts, event.Event{
			ID:        ulid.MustNew(uint64(now.UnixMilli()), ulid.DefaultEntropy()).String(),
			Name:      event.FnFinishedName,
			Timestamp: now.UnixMilli(),
			Data:      data,
		})
	}

	return evts, nil
}

// publishJoinEvents publishes joined finished events and fast-resumes their
// invokers, mirroring finalizeEvents. Publishing is at-least-once; resuming is
// idempotent.
func (e *executor) publishJoinEvents(ctx context.Context, id sv2.ID, evts []event.Event) error {
	if len(evts) == 0 {
		return nil
	}

	for _, evt := range evts {
		tracked := event.BaseTrackedEvent{
			ID:          ulid.MustParse(evt.ID),
			Event:       evt,
			AccountID:   id.Tenant.AccountID,
			WorkspaceID: id.Tenant.EnvID,
		}
		service.Go(func() {
			err := e.HandleInvokeFinish(context.WithoutCancel(ctx), tracked)
			if err != nil && !errors.Is(err, ErrNoCorrelationID) {
				logger.From(ctx).Error("error fast resuming joined invoke", "error", err)
			}
		})
	}

	if e.finishHandler == nil {
		return nil
	}

	return e.finishHandler(ctx, id, evts)
}
