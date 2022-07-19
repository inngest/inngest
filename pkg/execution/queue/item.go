package queue

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/state"
)

const (
	KindEdge  = "edge"
	KindPause = "pause"
)

// Item represents an item stored within a queue.
//
// Note that each individual implementation may wrap this to add their own fields,
// such as a job identifier.
type Item struct {
	// Kind represents the job type and payload kind stored within Payload.
	Kind string `json:"kind"`
	// Identifier represents the unique workflow ID and run ID for the current job.
	Identifier state.Identifier `json:"identifier"`
	// ErrorCount stores the total number of errors that this job has currently procesed.
	ErrorCount int `json:"errorCount"`
	// Payload stores item-specific data for use when processing the item.  For example,
	// this may contain the function's edge for running a step.
	Payload any `json:"payload"`
}

func (i *Item) UnmarshalJSON(b []byte) error {
	type kind struct {
		Kind       string           `json:"kind"`
		Identifier state.Identifier `json:"identifier"`
		ErrorCount int              `json:"errorCount"`
		Payload    json.RawMessage  `json:"payload"`
	}
	temp := &kind{}
	err := json.Unmarshal(b, temp)
	if err != nil {
		return err
	}

	i.Kind = temp.Kind
	i.Identifier = temp.Identifier
	i.ErrorCount = temp.ErrorCount

	switch temp.Kind {
	case KindEdge:
		p := &PayloadEdge{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	case KindPause:
		p := &PayloadPauseTimeout{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	default:
		return fmt.Errorf("unknown queue kind: %s", temp.Kind)
	}

	return nil
}

// GetEdge returns the edge from the enqueued item, if the payload is of type PayloadEdge.
func GetEdge(i Item) (*inngest.Edge, error) {
	switch v := i.Payload.(type) {
	case PayloadEdge:
		return &v.Edge, nil
	default:
		return nil, fmt.Errorf("unable to get edge from payload type: %T", v)
	}
}

// PayloadEdge is the payload stored when enqueueing an edge traversal to execute
// the incoming step of the edge.
type PayloadEdge struct {
	Edge inngest.Edge `json:"edge"`
}

// PayloadPauseTimeout is the payload stored when enqueueing a pause timeout, eg.
// a future task to check whether an event has been received yet.
//
// This is always enqueued from any async match;  we must correctly decrement the
// pending count in cases where the event is not received.
type PayloadPauseTimeout struct {
	PauseID   uuid.UUID `json:"pauseID"`
	OnTimeout bool      `json:"onTimeout"`
}
