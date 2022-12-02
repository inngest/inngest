package state

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
)

var prefixGzip = []byte("1:")

type History struct {
	Type       enums.HistoryType `json:"type"`
	Identifier Identifier        `json:"id"`
	CreatedAt  time.Time         `json:"createdAt"`
	Data       any               `json:"data"`
}

func (h *History) UnmarshalJSON(data []byte) error {
	// We unmarshal into a copy of the struct so that we can
	// correctly unmarshal the Data field into the correct struct
	// type.
	m := struct {
		Type       enums.HistoryType `json:"type"`
		Identifier Identifier        `json:"id"`
		CreatedAt  time.Time         `json:"createdAt"`
	}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	h.Type = m.Type
	h.Identifier = m.Identifier
	h.CreatedAt = m.CreatedAt

	switch h.Type {
	case enums.HistoryTypeStepScheduled,
		enums.HistoryTypeStepStarted,
		enums.HistoryTypeStepCompleted,
		enums.HistoryTypeStepErrored,
		enums.HistoryTypeStepFailed:
		v := struct {
			Data HistoryStep `json:"data"`
		}{}
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		h.Data = v.Data
	}

	return nil
}

func (h History) MarshalBinary() (data []byte, err error) {
	jsonByt, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(jsonByt); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return append([]byte(prefixGzip), b.Bytes()...), nil
}

func (h *History) UnmarshalBinary(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("history must be prefixed; invalid data length")
	}

	prefix, data := data[0:2], data[2:]

	switch string(prefix) {
	case string(prefixGzip):
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("could not un-gzip data: %w", err)
		}
		if err := json.NewDecoder(r).Decode(h); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("history had no recognised prefix")
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
