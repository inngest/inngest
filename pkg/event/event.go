package event

import (
	"encoding/json"
	"sync"
)

const (
	EventReceivedName = "event/event.received"
	FnFailedName      = "inngest/function.failed"
)

type Manager struct {
	events map[string]Event
	l      *sync.RWMutex
}

func NewManager() Manager {
	return Manager{
		events: make(map[string]Event),
		l:      &sync.RWMutex{},
	}
}

// Fetch an individual event by its ID.
func (e Manager) EventById(id string) *Event {
	e.l.RLock()
	defer e.l.RUnlock()

	evt, ok := e.events[id]
	if !ok {
		return nil
	}

	return &evt
}

// Fetch all events with a given name.
func (e Manager) EventsByName(name string) []Event {
	e.l.RLock()
	defer e.l.RUnlock()

	events := []Event{}

	for _, evt := range e.events {
		if evt.Name == name {
			events = append(events, evt)
		}
	}

	return events

}

// Fetch all events.
func (e Manager) Events() []Event {
	e.l.RLock()
	defer e.l.RUnlock()

	events := []Event{}

	for _, evt := range e.events {
		events = append(events, evt)
	}

	return events
}

// Parse and create a new event, adding it to the in-memory map as we go.
func (e Manager) NewEvent(data string) (*Event, error) {
	e.l.Lock()
	defer e.l.Unlock()

	evt, err := NewEvent(data)
	if err != nil {
		return nil, err
	}

	e.events[evt.ID] = *evt

	return evt, err
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
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`

	// User represents user-specific information for the event.
	User map[string]interface{} `json:"user,omitempty"`

	// ID represents the unique ID for this particular event.  If supplied, we should attempt
	// to only ingest this event once.
	ID string `json:"id,omitempty"`

	// Timestamp is the time the event occurred, at millisecond precision.
	// If this is not provided, we will insert the current time upon receipt of the event
	Timestamp int64  `json:"ts,omitempty"`
	Version   string `json:"v,omitempty"`
}

func (evt Event) Map() map[string]interface{} {
	if evt.Data == nil {
		evt.Data = make(map[string]interface{})
	}
	if evt.User == nil {
		evt.User = make(map[string]interface{})
	}

	data := map[string]interface{}{
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
