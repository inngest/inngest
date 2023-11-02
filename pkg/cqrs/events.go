package cqrs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"
)

const MaxEvents = 51

var (
	year = time.Hour * 24 * 365
)

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
	WorkspaceID uuid.UUID  `json:"workspace_id"`
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

type EventManager interface {
	EventWriter
	EventReader
}

type EventWriter interface {
	InsertEvent(ctx context.Context, e Event) error
}

type WorkspaceEventsOpts struct {
	Cursor *ulid.ULID
	Limit  int
	// Name filters events to a given name.
	Name *string
	// Newest represents the newest time to load events from.  Events newer than
	// this cutoff will not be loaded.
	Newest time.Time
	// Oldest represents the oldest events to load.  Events older than this
	// cutoff will not be loaded.
	Oldest time.Time
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
		// 1 year ago, ie all events
		o.Oldest = time.Now().Add(year * -1)
	}
	return nil
}

type EventReader interface {
	GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*Event, error)
	GetEventsTimebound(ctx context.Context, t Timebound, limit int) ([]*Event, error)
	// WorkspaceEvents returns the latest events for a given workspace.
	WorkspaceEvents(ctx context.Context, workspaceID uuid.UUID, opts *WorkspaceEventsOpts) ([]Event, error)
	// Find returns a specific event given an ID.
	FindEvent(ctx context.Context, workspaceID uuid.UUID, id ulid.ULID) (*Event, error)
}
