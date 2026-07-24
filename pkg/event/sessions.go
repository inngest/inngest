package event

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sort"

	"github.com/inngest/inngest/pkg/consts"
)

// EventMeta carries event meta shared across runs triggered by this event.
type EventMeta struct {
	// Sessions groups runs triggered by this event. These are the manual
	// (user-set) sessions and always win over propagated sessions on merge.
	Sessions Sessions `json:"sessions,omitempty"`

	// PropagatedSessions carries sessions inherited from the parent run that
	// emitted this event (SDK-stamped during a run). It is an un-merged layer:
	// ResolveSessions folds it into Sessions at ingest and clears it, so it is
	// never persisted or forwarded downstream.
	PropagatedSessions Sessions `json:"propagatedSessions,omitempty"`
}

func (m EventMeta) IsZero() bool {
	return len(m.Sessions) == 0 && len(m.PropagatedSessions) == 0
}

// ResolveSessions folds the propagated (inherited) session layer into the
// manual layer, producing the single Sessions map carried downstream. Manual
// keys always win and are never evicted; remaining slots up to
// consts.MaxEventSessions are filled from propagated sessions in
// lexicographic key order (matching run-level session truncation) for
// deterministic output. PropagatedSessions is cleared so it never persists.
//
// Called at API ingest before Event.Validate: each layer may independently be
// up to MaxEventSessions, so the pre-merge union can exceed the cap —
// validating the raw fields would falsely reject; validating the merged result
// does not.
func (m *EventMeta) ResolveSessions() {
	if len(m.PropagatedSessions) == 0 {
		// Nothing to merge; drop the (empty) propagated layer defensively.
		m.PropagatedSessions = nil
		return
	}

	m.Sessions = mergeSessions(m.PropagatedSessions, m.Sessions)
	m.PropagatedSessions = nil
}

// mergeSessions folds the propagated (inherited) session candidates into the
// manual (user-set) map, producing the final per-event session set. Manual
// keys always win and are never evicted; remaining slots up to
// consts.MaxEventSessions are filled from propagatedCandidates deterministically
// (lexicographic key order) so output is stable across retries and map
// iteration order. Returns nil when the merged result is empty.
func mergeSessions(propagatedCandidates, manualMap Sessions) Sessions {
	out := Sessions{}

	// Manual keys always win and are never evicted — even when they alone
	// exceed consts.MaxEventSessions. An oversized manual layer is rejected by
	// the post-merge Validate rather than silently truncated here, so user-set
	// keys are never dropped in favour of inherited ones.
	maps.Copy(out, manualMap)

	// Fill any remaining slots from the propagated candidates in lexicographic
	// key order. Go's native string ordering is byte-wise over UTF-8, matching
	// run-level truncation (normalizeRunSessions) and the SDK's aggregate — so
	// the chosen subset is deterministic and identical across implementations,
	// independent of map iteration order.
	if remaining := consts.MaxEventSessions - len(out); remaining > 0 && len(propagatedCandidates) > 0 {
		keys := make([]string, 0, len(propagatedCandidates))
		for k := range propagatedCandidates {
			if _, manual := out[k]; manual {
				continue // manual wins; a shadowed propagated key consumes no slot
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if remaining <= 0 {
				break
			}
			out[k] = propagatedCandidates[k]
			remaining--
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
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
