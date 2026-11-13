package executor

import (
	"github.com/inngest/inngest/pkg/event"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
)

// ScheduleQueueRoutingError is returned by Schedule when the queue cannot
// route the first item of a run, or when the shard of the run is not
// connected.  Schedule does not delete the run state.
// a delete sets a tombstone on the idempotency key, and every later retry of
// the same event skips the run.  only a caller that drops the event for good
// can delete the state.
type ScheduleQueueRoutingError struct {
	// ID identifies the run state that Schedule created or found.
	ID sv2.ID
	// StateExisted is true when the run state was in the state store before
	// this Schedule call.  it is false when this call created the state.
	StateExisted bool
	// EventInternalID is the internal ID of the first event of the run.  it
	// is zero when the run has no events.
	EventInternalID ulid.ULID
	// Err is the enqueue error.  it wraps queue.ErrQueueShardNotFound or
	// queue.ErrQueueShardUnavailable.
	Err error
}

func (e *ScheduleQueueRoutingError) Error() string {
	return e.Err.Error()
}

func (e *ScheduleQueueRoutingError) Unwrap() error {
	return e.Err
}

func firstEventInternalID(evts []event.TrackedEvent) ulid.ULID {
	if len(evts) == 0 || evts[0] == nil {
		return ulid.ULID{}
	}
	return evts[0].GetInternalID()
}
