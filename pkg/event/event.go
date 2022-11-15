package event

var inMemoryEvents map[string]map[string]Event

type EventManager interface {
	// Fetch an individual event by its workspace and ID.
	EventById(workspaceId string, id string) *Event

	// Fetch all events with a given name from a given workspace.
	EventsByName(workspaceId string, name string) []Event

	// Fetch all events from a given workspace.
	Events(workspaceId string) []Event

	// Save an event to a data store.
	SaveEvent(evt Event) error
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
