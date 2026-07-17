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

	// sessionTombstones holds manual session keys explicitly set to JSON null — RFC
	// 7386 per-key tombstones. Each cuts the matching key from the propagated
	// layer at ResolveSessions; tombstones are consumed there (never merged into
	// Sessions, never counted against MaxEventSessions, never persisted). They
	// are captured transiently in UnmarshalJSON so Sessions stays a plain
	// map[string]string for every downstream reader — see [null-tombstone
	// typing] in the session-propagation design.
	sessionTombstones []string

	// clearPropagated is set when the manual `sessions` field is JSON null
	// (RFC 7386 whole-document null): "clear all inherited sessions". It is
	// distinct from an absent field (keep propagated) and from an empty object
	// (keep propagated), which is why the manual layer needs a custom
	// UnmarshalJSON. Consumed and reset at ResolveSessions.
	clearPropagated bool
}

func (m EventMeta) IsZero() bool {
	return len(m.Sessions) == 0 && len(m.PropagatedSessions) == 0 &&
		len(m.sessionTombstones) == 0 && !m.clearPropagated
}

// UnmarshalJSON parses the two session layers, capturing null tombstones on the
// manual layer separately so Sessions itself stays a plain map[string]string.
//
// The manual `sessions` field distinguishes three JSON states that a
// map[string]string cannot: absent (keep propagated), null (clear all
// propagated), and an object whose values may include per-key null tombstones.
// The propagated layer is machine-stamped and admits no tombstones, so it uses
// the ordinary Sessions decoding (which rejects null values).
func (m *EventMeta) UnmarshalJSON(data []byte) error {
	var raw struct {
		Sessions           json.RawMessage `json:"sessions"`
		PropagatedSessions Sessions        `json:"propagatedSessions"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	sessions, tombstones, clearAll, err := parseManualSessions(raw.Sessions)
	if err != nil {
		return err
	}

	m.Sessions = sessions
	m.PropagatedSessions = raw.PropagatedSessions
	m.sessionTombstones = tombstones
	m.clearPropagated = clearAll
	return nil
}

// parseManualSessions decodes the raw manual `sessions` field into its real
// (non-null) entries, the set of per-key null tombstones, and whether the whole
// field was JSON null. Numeric ids are stringified, matching Sessions
// decoding; booleans (and other non-string/number/null values) are rejected.
func parseManualSessions(raw json.RawMessage) (sessions Sessions, tombstones []string, clearAll bool, err error) {
	// Absent field: nothing set, keep any propagated layer untouched.
	if len(raw) == 0 {
		return nil, nil, false, nil
	}
	// Whole-field null (RFC 7386): clear all inherited sessions.
	if string(raw) == "null" {
		return nil, nil, true, nil
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	obj := map[string]any{}
	if err := dec.Decode(&obj); err != nil {
		return nil, nil, false, err
	}

	for name, value := range obj {
		switch v := value.(type) {
		case nil:
			// Per-key null tombstone (RFC 7386): cut this inherited key.
			tombstones = append(tombstones, name)
		case string:
			if sessions == nil {
				sessions = Sessions{}
			}
			sessions[name] = v
		case json.Number:
			if sessions == nil {
				sessions = Sessions{}
			}
			sessions[name] = v.String()
		default:
			// Booleans etc. are rejected — a session id is a string or number,
			// and null is the only accepted non-value (a tombstone).
			return nil, nil, false, fmt.Errorf("event session %q must be a string, number, or null", name)
		}
	}

	return sessions, tombstones, false, nil
}

// ResolveSessions folds the propagated (inherited) session layer into the
// manual layer, producing the single Sessions map carried downstream. Manual
// keys always win and are never evicted; remaining slots up to
// consts.MaxEventSessions are filled from propagated sessions in
// lexicographic key order (matching run-level session truncation) for
// deterministic output. PropagatedSessions is cleared so it never persists.
//
// Callers that want adoption metrics read len(Sessions)/len(PropagatedSessions)
// before calling this (the propagated layer is nil afterwards).
//
// Called at API ingest before Event.Validate: each layer may independently be
// up to MaxEventSessions, so the pre-merge union can exceed the cap —
// validating the raw fields would falsely reject; validating the merged result
// does not.
//
// Null tombstones (RFC 7386) are applied against the propagated layer first and
// are always consumed here: a whole-field-null manual layer clears every
// inherited session, and per-key nulls cut the matching inherited keys.
// Tombstones never survive into Sessions and so never count against the cap.
func (m *EventMeta) ResolveSessions() {
	// Apply the manual layer's tombstones to the inherited candidates before
	// merging. Whole-field null wins over individual tombstones (there is nothing
	// left to cut once everything is cleared).
	if m.clearPropagated {
		m.PropagatedSessions = nil
	} else {
		for _, key := range m.sessionTombstones {
			delete(m.PropagatedSessions, key)
		}
	}
	m.sessionTombstones = nil
	m.clearPropagated = false

	if len(m.PropagatedSessions) == 0 {
		// Nothing to merge; drop the (empty) propagated layer defensively and
		// normalize an empty manual layer to nil.
		m.PropagatedSessions = nil
		if len(m.Sessions) == 0 {
			m.Sessions = nil
		}
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
