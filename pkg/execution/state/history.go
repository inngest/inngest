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

		return json.NewDecoder(r).Decode(h)
	}

	return fmt.Errorf("history had no recognised prefix")
}

type HistoryFunctionCancelled struct {
	Type enums.CancellationType `json:"type"`
	Data any                    `json:"data"`
}

// TODO Add tracking of the parent steps so that we can create a visual DAG
type HistoryStep struct {
	Name    string `json:"name"`
	Attempt int    `json:"attempt"`
	Data    any    `json:"data"`
}

type HistoryStepWaitingForEvent struct {
	Name       string    `json:"name"`
	EventName  string    `json:"eventName"`
	Expression string    `json:"expression"`
	ExpiryTime time.Time `json:"expiry"`
}

type HistoryStepSleepingUntil struct {
	Name   string    `json:"name"`
	WakeAt time.Time `json:"wakeAt"`
}
