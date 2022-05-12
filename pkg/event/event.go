package event

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
