package cqrs

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"
)

const MaxEvents = 51

func ConvertFromEvent(internalID ulid.ULID, e event.Event) Event {
	return Event{
		ID:           internalID,
		EventID:      e.ID,
		EventName:    e.Name,
		EventData:    e.Data,
		EventUser:    e.User,
		EventTS:      e.Timestamp,
		EventVersion: e.Version,
	}
}

type Event struct {
	ID          ulid.ULID  `json:"internal_id"`
	AccountID   uuid.UUID  `json:"account_id"`
	WorkspaceID uuid.UUID  `json:"environment_id"`
	Source      string     `json:"source"`
	SourceID    *uuid.UUID `json:"source_id"`
	ReceivedAt  time.Time  `json:"received_at"`

	EventID      string         `json:"id,omitempty"`
	EventName    string         `json:"name"`
	EventData    map[string]any `json:"data"`
	EventUser    map[string]any `json:"user,omitempty"`
	EventTS      int64          `json:"ts,omitempty"`
	EventVersion string         `json:"v,omitempty"`
}

func (e Event) InternalID() ulid.ULID {
	return e.ID
}

func (e Event) Event() event.Event {
	return event.Event{
		ID:        e.EventID,
		Name:      e.EventName,
		Data:      e.EventData,
		User:      e.EventUser,
		Timestamp: e.EventTS,
		Version:   e.EventVersion,
	}
}

// -- event.TrackedEvent interfaces
func (e Event) GetInternalID() ulid.ULID {
	return e.InternalID()
}

func (e Event) GetAccountID() uuid.UUID {
	return e.AccountID
}

func (e Event) GetWorkspaceID() uuid.UUID {
	return e.WorkspaceID
}

func (e Event) GetEvent() event.Event {
	return e.Event()
}

type EventManager interface {
	EventWriter
	EventReader
}

type EventWriter interface {
	InsertEvent(ctx context.Context, e Event) error
	InsertEventBatch(ctx context.Context, eb EventBatch) error
}

type WorkspaceEventsOpts struct {
	Cursor *ulid.ULID
	Limit  int
	// Name filters events to given names.
	Names []string
	// Newest represents the newest time to load events from.  Events newer than
	// this cutoff will not be loaded.
	Newest time.Time
	// Oldest represents the oldest events to load.  Events older than this
	// cutoff will not be loaded.
	Oldest                time.Time
	IncludeInternalEvents bool
}

func (o *WorkspaceEventsOpts) Validate() error {
	if o.Limit < 1 {
		return fmt.Errorf("limit must be positive")
	}
	if o.Limit > MaxEvents {
		return fmt.Errorf("limit must be less than %d", MaxEvents)
	}
	if o.Newest.IsZero() {
		// Now
		o.Newest = time.Now()
	}
	if o.Oldest.IsZero() {
		// Default to one hour ago.
		o.Oldest = time.Now().Add(time.Hour * -1)
	}
	return nil
}

type EventReader interface {
	GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*Event, error)
	GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*Event, error)
	// GetEventsByExpressions retrieves the events that match all the CEL expressions provided.
	GetEventsByExpressions(ctx context.Context, cel []string) ([]*Event, error)
	GetEventBatchesByEventID(ctx context.Context, eventID ulid.ULID) ([]*EventBatch, error)
	GetEventBatchByRunID(ctx context.Context, runID ulid.ULID) (*EventBatch, error)
	GetEventsIDbound(
		ctx context.Context,
		ids IDBound,
		limit int,
		includeInternal bool,
	) ([]*Event, error)
	// GetEvents returns the latest events for a given workspace.
	GetEvents(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, opts *WorkspaceEventsOpts) ([]*Event, error)
	// GetEventsCount returns the total count of events filtered by event name and from/until time range
	GetEventsCount(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, opts WorkspaceEventsOpts) (int64, error)
	// GetEvent returns a specific event given an ID.
	GetEvent(ctx context.Context, id ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (*Event, error)
}

type EventBatchOpt func(eb *EventBatch)

// EventBatch represents a event batch execution
type EventBatch struct {
	ID          ulid.ULID            `json:"id"`
	AccountID   uuid.UUID            `json:"account_id"`
	WorkspaceID uuid.UUID            `json:"workspace_id"`
	AppID       uuid.UUID            `json:"app_id"`
	FunctionID  uuid.UUID            `json:"workflow_id"`
	RunID       ulid.ULID            `json:"run_id"`
	Events      []event.TrackedEvent `json:"events"`
	Time        time.Time            `json:"ts"`
}

func NewEventBatch(opts ...EventBatchOpt) *EventBatch {
	eb := &EventBatch{
		ID:   ulid.MustNew(ulid.Now(), rand.Reader),
		Time: time.Now(),
	}

	for _, opt := range opts {
		opt(eb)
	}

	return eb
}

// WithEventBatchID sets the new EventBatch ID with the provided ID
func WithEventBatchID(id ulid.ULID) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.ID = id
	}
}

func WithEventBatchAccountID(acctID uuid.UUID) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.AccountID = acctID
	}
}

func WithEventBatchWorkspaceID(wsID uuid.UUID) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.WorkspaceID = wsID
	}
}

func WithEventBatchAppID(appID uuid.UUID) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.AppID = appID
	}
}

func WithEventBatchFunctionID(fnID uuid.UUID) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.FunctionID = fnID
	}
}

func WithEventBatchEvent(evt event.TrackedEvent) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.Events = []event.TrackedEvent{evt}
	}
}

func WithEventBatchRunID(runID ulid.ULID) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.RunID = runID
	}
}

func WithEventBatchEvents(evts []event.TrackedEvent) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.Events = evts
	}
}

func WithEventBatchEventIDs(evtIDs []ulid.ULID) EventBatchOpt {
	return func(eb *EventBatch) {
		evts := make([]event.TrackedEvent, len(evtIDs))
		for i, id := range evtIDs {
			evts[i] = Event{ID: id}
		}
		eb.Events = evts
	}
}

func WithEventBatchExecutedTime(t time.Time) EventBatchOpt {
	return func(eb *EventBatch) {
		eb.Time = t
	}
}

func (eb *EventBatch) StartedAt() time.Time {
	return ulid.Time(eb.ID.Time())
}

func (eb *EventBatch) ExecutedAt() time.Time {
	return eb.Time
}

func (eb *EventBatch) EventID() *ulid.ULID {
	if len(eb.Events) < 1 {
		return nil
	}
	id := eb.Events[0].GetInternalID()
	return &id
}

func (eb *EventBatch) EventIDs() []ulid.ULID {
	ids := make([]ulid.ULID, len(eb.Events))
	for i, evt := range eb.Events {
		ids[i] = evt.GetInternalID()
	}
	return ids
}

func (eb *EventBatch) IsMulti() bool {
	return len(eb.Events) > 1
}
