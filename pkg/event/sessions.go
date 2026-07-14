package event

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
)

// EventMeta carries event meta shared across runs triggered by this event.
type EventMeta struct {
	// Sessions groups runs triggered by this event.
	Sessions Sessions `json:"sessions,omitempty"`
}

func (m EventMeta) IsZero() bool {
	return len(m.Sessions) == 0
}

// Sessions maps a session key to a session ID.
type Sessions map[string]string

func (s *Sessions) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = nil
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	raw := map[string]any{}
	if err := dec.Decode(&raw); err != nil {
		return err
	}

	out := Sessions{}
	for name, value := range raw {
		switch v := value.(type) {
		case string:
			out[name] = v
		case json.Number:
			out[name] = v.String()
		default:
			// Booleans are intentionally rejected: a boolean is only ever two
			// values, i.e. a low-cardinality label, not a session ID.
			return fmt.Errorf("event session %q must be a string or number", name)
		}
	}

	*s = out
	return nil
}

// Validate checks session keys and IDs against size limits.
func (s Sessions) Validate() error {
	if len(s) > consts.MaxEventSessions {
		return fmt.Errorf("event sessions can include at most %d entries", consts.MaxEventSessions)
	}
	for name, id := range s {
		if name == "" {
			return errors.New("event session keys cannot be empty")
		}
		if len(name) > consts.MaxEventSessionKeyLength {
			return fmt.Errorf("event session key %q exceeds %d bytes", name, consts.MaxEventSessionKeyLength)
		}
		if id == "" {
			return fmt.Errorf("event session %q cannot have an empty ID", name)
		}
		if len(id) > consts.MaxEventSessionIDLength {
			return fmt.Errorf("event session %q exceeds %d bytes", name, consts.MaxEventSessionIDLength)
		}
	}
	return nil
}
