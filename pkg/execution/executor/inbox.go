package executor

import (
	"crypto/rand"
	"strings"
	"time"

	"context"
	"encoding/json"
	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/inbox"
)

// DeliverInbox hands the next queued message for an inbox key to the
// step.subscribe wait that is pending on it, if any.
func (e *executor) DeliverInbox(ctx context.Context, workspaceID uuid.UUID, key string) (bool, error) {
	return inbox.Deliver(inbox.Default(), workspaceID, key, func(signalID string, data json.RawMessage) (bool, error) {
		res, err := e.ResumeSignal(ctx, workspaceID, signalID, data)
		if err != nil || res == nil {
			return false, err
		}
		return res.MatchedSignal, nil
	})
}

// AgentThreadSession is the event session key that names an agent thread. The
// thread's inbox key is the session value.
const AgentThreadSession = "agent_thread"

// inboxHandoffEvent returns a copy of the trigger event that restarts the
// thread when its inbox still has undelivered messages, or nil.
func (e *executor) inboxHandoffEvent(workspaceID uuid.UUID, trigger event.Event, now time.Time) *event.Event {
	thread := trigger.Meta.Sessions[AgentThreadSession]
	if thread == "" || strings.HasPrefix(trigger.Name, "inngest/") {
		return nil
	}

	store := inbox.Default()
	head := store.Head(workspaceID, thread)
	if head <= store.Acked(workspaceID, thread) || !store.MarkHandoff(workspaceID, thread, head) {
		return nil
	}

	next := trigger
	next.ID = ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String()
	next.Timestamp = now.UnixMilli()
	return &next
}
