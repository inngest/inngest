package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/singleton"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

// singletonFinalizeCloseID is the step a run's singleton inbox is closed under
// when the run ends without having closed it.
const singletonFinalizeCloseID = "singleton:finalize"

// singletonInboxID is the id of the singleton inbox of a run of mode queue.
func singletonInboxID() string {
	sum := sha1.Sum([]byte(singletonInbox))

	return hex.EncodeToString(sum[:])
}

// handBackSingletonInbox puts the messages of a finished run's singleton inbox
// that the run did not read at the head of the lock's mailbox, in the order they
// arrived. The release of the lock hands the mailbox on, so they start the next
// runs with their original events.
//
// The caller must have closed the inbox first: a message saved before the close
// is in the run's stack, and one that arrives after it is parked in the mailbox
// by queueSingletonEvent. So the messages handed back are all older than
// whatever is already parked, and putting them at the head keeps arrival order.
// The state and the queue share one Redis in the Dev Server, and the push
// happens while the run still holds the lock, so the release cannot take the
// mailbox before them.
//
// taken is how many messages the run read. Nothing is done for a function that
// is not in mode queue.
func (e *executor) handBackSingletonInbox(ctx context.Context, md *sv2.Metadata, taken int) error {
	if e.singletonMgr == nil || e.fl == nil {
		return nil
	}

	ef, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		return fmt.Errorf("error loading function to hand back a singleton inbox: %w", err)
	}
	cfg := ef.Function.Singleton
	if cfg == nil || cfg.Mode != enums.SingletonModeQueue {
		return nil
	}

	events, err := e.smv2.LoadEvents(ctx, md.ID)
	if err != nil {
		return fmt.Errorf("error loading events to hand back a singleton inbox: %w", err)
	}
	if len(events) == 0 {
		return nil
	}
	var trigger event.Event
	if err := json.Unmarshal(events[0], &trigger); err != nil {
		return fmt.Errorf("error decoding the trigger of a singleton run: %w", err)
	}
	key, err := singleton.SingletonKey(ctx, md.ID.FunctionID, *cfg, trigger.Map())
	if err != nil {
		// The run did not take a lock, so nothing is parked for it.
		return nil
	}

	stack, err := e.smv2.LoadStack(ctx, md.ID)
	if err != nil {
		return fmt.Errorf("error loading the stack to hand back a singleton inbox: %w", err)
	}
	steps, err := e.smv2.LoadSteps(ctx, md.ID)
	if err != nil {
		return fmt.Errorf("error loading steps to hand back a singleton inbox: %w", err)
	}

	inbox := singletonInboxID()
	var messages []inboxMessageValue
	for _, id := range stack {
		raw, ok := steps[id]
		if !ok {
			continue
		}
		var v inboxMessageValue
		if json.Unmarshal(raw, &v) != nil || v.Inbox != inbox || v.Data == nil {
			continue
		}
		messages = append(messages, v)
	}
	if taken < 0 {
		taken = 0
	}
	if taken >= len(messages) {
		return nil
	}

	entries := make([]string, 0, len(messages)-taken)
	for _, v := range messages[taken:] {
		data, err := json.Marshal(v.Data)
		if err != nil {
			return err
		}
		var evt event.Event
		if err := json.Unmarshal(data, &evt); err != nil {
			return err
		}
		id, perr := ulid.Parse(v.InternalID)
		if perr != nil {
			id = ulid.Make()
		}
		entry, err := json.Marshal(map[string]any{
			"f": md.ID.FunctionID,
			"e": event.BaseTrackedEvent{
				ID:          id,
				AccountID:   md.ID.Tenant.AccountID,
				WorkspaceID: md.ID.Tenant.EnvID,
				Event:       evt,
			},
		})
		if err != nil {
			return err
		}
		entries = append(entries, string(entry))
	}

	parked, err := e.singletonMgr.ParkFirst(ctx, queue.Scope{
		AccountID:  md.ID.Tenant.AccountID,
		EnvID:      md.ID.Tenant.EnvID,
		FunctionID: md.ID.FunctionID,
	}, key, entries)
	if err != nil {
		return fmt.Errorf("error handing back a singleton inbox: %w", err)
	}
	if !parked {
		logger.StdlibLogger(ctx).Warn("singleton lock gone before its inbox was handed back", "run_id", md.ID.RunID.String(), "events", len(entries))
	}

	return nil
}

// closeSingletonInbox is what a run of mode queue does when it ends: if it
// never closed its singleton inbox (a function that does not read it, or a run
// that failed), the inbox is closed here, so later events park in the mailbox,
// and everything saved in it goes to the next runs. A run that closed the inbox
// itself has already handed back what it did not read. It is best effort: a
// failure is logged.
func (e *executor) closeSingletonInbox(ctx context.Context, md *sv2.Metadata) {
	if e.singletonMgr == nil || e.fl == nil {
		return
	}
	ef, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil || ef == nil || ef.Function.Singleton == nil || ef.Function.Singleton.Mode != enums.SingletonModeQueue {
		return
	}

	l := logger.StdlibLogger(ctx)
	inbox := singletonInboxID()

	waits, err := e.smv2.LoadInboxWaits(ctx, md.ID)
	if err != nil {
		l.Warn("could not load inbox waits to end a singleton run", "error", err)
		return
	}
	for _, closed := range waits.Closed {
		if closed == inbox {
			return
		}
	}

	if _, _, err := e.smv2.CloseInbox(ctx, md.ID, sv2.InboxClose{StepID: singletonFinalizeCloseID, Inbox: inbox}); err != nil {
		l.Warn("could not close the singleton inbox of an ending run", "error", err)
		return
	}
	if err := e.handBackSingletonInbox(ctx, md, 0); err != nil {
		l.Warn("could not hand back the singleton inbox of an ending run", "error", err)
	}
}
