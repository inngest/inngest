package event

import (
	"crypto/rand"
	"encoding/json"

	"github.com/oklog/ulid/v2"
)

const (
	EventReceivedName = "event/event.received"
	FnFailedName      = "inngest/function.failed"
)

type TrackedEvent interface {
	InternalID() ulid.ULID
	Event() Event
}

func NewEvent(data string) (*Event, error) {
	evt := &Event{}
	if err := json.Unmarshal([]byte(data), evt); err != nil {
		return nil, err
	}

	return evt, nil
}

// Event represents an event sent to Inngest.
type Event struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`

	// User represents user-specific information for the event.
	User map[string]any `json:"user,omitempty"`

	// ID represents the unique ID for this particular event.  If supplied, we should attempt
	// to only ingest this event once.
	ID string `json:"id,omitempty"`

	// Timestamp is the time the event occurred, at millisecond precision.
	// If this is not provided, we will insert the current time upon receipt of the event
	Timestamp int64  `json:"ts,omitempty"`
	Version   string `json:"v,omitempty"`
}

func (evt Event) Map() map[string]any {
	if evt.Data == nil {
		evt.Data = make(map[string]any)
	}
	if evt.User == nil {
		evt.User = make(map[string]any)
	}

	data := map[string]any{
		"name": evt.Name,
		"data": evt.Data,
		"user": evt.User,
		"id":   evt.ID,
		// We cast to float64 because marshalling and unmarshalling from
		// JSON automatically uses float64 as its type;  JS has no notion
		// of ints.
		"ts": float64(evt.Timestamp),
	}

	if evt.Version != "" {
		data["v"] = evt.Version
	}

	return data
}

func NewOSSTrackedEvent(e Event) TrackedEvent {
	id, err := ulid.Parse(e.ID)
	if err != nil {
		id = ulid.MustNew(ulid.Now(), rand.Reader)
	}
	if e.ID == "" {
		e.ID = id.String()
	}
	return ossTrackedEvent{
		id:    id,
		event: e,
	}
}

type ossTrackedEvent struct {
	id    ulid.ULID
	event Event
}

func (o ossTrackedEvent) Event() Event {
	return o.event
}

func (o ossTrackedEvent) InternalID() ulid.ULID {
	return o.id
}
