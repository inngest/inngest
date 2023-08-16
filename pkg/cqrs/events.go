package cqrs

import (
	"context"

	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"
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
	ID           ulid.ULID      `json:"internal_id"`
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

type EventReader interface {
	GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*Event, error)
}
