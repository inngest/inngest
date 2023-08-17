package event

import "sync"

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
