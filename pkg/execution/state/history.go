package state

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

var (
	// DefaultHistoryEncoding represents the encoding used when storing data within Redis.
	// This can be changed on init to change how we globally store history.
	DefaultHistoryEncoding = HistoryEncodingGZIP
)

const (
	HistoryEncodingJSON = "0:"
	HistoryEncodingGZIP = "1:"
)

type History struct {
	ID         ulid.ULID         `json:"id"`
	Type       enums.HistoryType `json:"type"`
	Identifier Identifier        `json:"run"`
	CreatedAt  time.Time         `json:"createdAt"`
	Data       any               `json:"data"`
}

func (h *History) UnmarshalJSON(data []byte) error {
	// We unmarshal into a copy of the struct so that we can
	// correctly unmarshal the Data field into the correct struct
	// type.
	m := struct {
		ID         ulid.ULID         `json:"id"`
		Type       enums.HistoryType `json:"type"`
		Identifier Identifier        `json:"run"`
		CreatedAt  time.Time         `json:"createdAt"`
		Data       json.RawMessage   `json:"data"`
	}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	h.ID = m.ID
	h.Type = m.Type
	h.Identifier = m.Identifier
	h.CreatedAt = m.CreatedAt

	switch h.Type {
	case enums.HistoryTypeStepScheduled,
		enums.HistoryTypeStepStarted,
		enums.HistoryTypeStepCompleted,
		enums.HistoryTypeStepErrored,
		enums.HistoryTypeStepFailed:
		// Assume that for step history items we must have a HistoryStep
		// struct within data.
		v := HistoryStep{}
		if err := json.Unmarshal(m.Data, &v); err != nil {
			return err
		}
		h.Data = v
	default:
		// For other items, unmarshal as JSON
		v := map[string]any{}
		if err := json.Unmarshal(m.Data, &v); err != nil {
			return err
		}
		h.Data = v
	}

	return nil
}

func (h History) MarshalBinary() (data []byte, err error) {
	jsonByt, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	if DefaultHistoryEncoding == HistoryEncodingJSON {
		return jsonByt, nil
	}

	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(jsonByt); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return append([]byte(HistoryEncodingGZIP), b.Bytes()...), nil
}

func (h *History) UnmarshalBinary(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("history must be prefixed; invalid data length")
	}

	prefix, suffix := data[0:2], data[2:]

	switch string(prefix) {
	case string(HistoryEncodingGZIP):
		r, err := gzip.NewReader(bytes.NewReader(suffix))
		if err != nil {
			return fmt.Errorf("could not un-gzip data: %w", err)
		}
		if err := json.NewDecoder(r).Decode(h); err != nil {
			return err
		}
		return nil
	case string(HistoryEncodingJSON):
		if err := json.Unmarshal(suffix, h); err != nil {
			return err
		}
		return nil
	default:
		// The default is JSON-encoded, assuming the entire
		// data is JSON
		if err := json.Unmarshal(data, h); err != nil {
			return err
		}
		return nil
	}
}

type HistoryFunctionCancelled struct {
	Type enums.CancellationType `json:"type"`
	Data any                    `json:"data"`
}

// TODO Add tracking of the parent steps so that we can create a visual DAG
type HistoryStep struct {
	// ID stores the step ID.  This is the key used within the state
	// store to represent the step's data.
	ID string `json:"id"`
	// Name represents the human step name.  This is included as generator
	// based steps do not have names included in function config and is
	// needed for the UI.
	Name    string `json:"name"`
	Attempt int    `json:"attempt"`
	// Opcode stores the generator opcode for this step, if any.
	Opcode enums.Opcode `json:"opcode"`
	// Data stores data for this event, dependent on the history type.
	Data any `json:"data"`
}

// HistoryStepWaiting is stored within HistoryStep when we create a pause to wait
// for an event
type HistoryStepWaiting struct {
	// EventName is the name of the event we're waiting for.
	EventName  *string   `json:"eventName"`
	Expression *string   `json:"expression"`
	ExpiryTime time.Time `json:"expiry"`
}
