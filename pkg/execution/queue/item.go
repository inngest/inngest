package queue

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
)

const (
	KindEdge  = "edge"
	KindPause = "pause"
)

// Item represents an item stored within a queue.
//
// Note that each individual implementation may wrap this to add their own fields,
// such as a job identifier.
//
// TODO: Refactor this with the QueueItem in redis state to remove duplicates.
type Item struct {
	// JobID is an internal ID used to deduplicate queue items.
	JobID *string `json:"-"`
	// Workspace is the ID that this workspace job belongs to
	WorkspaceID uuid.UUID `json:"wsID"`
	// Kind represents the job type and payload kind stored within Payload.
	Kind string `json:"kind"`
	// Identifier represents the unique workflow ID and run ID for the current job.
	Identifier state.Identifier `json:"identifier"`
	// Attempt stores the zero index attempt counter
	Attempt int `json:"atts"`
	// MaxAttempts is the maximum number of attempts we can retry.  When attempts == this,
	// do not schedule another try.  If nil, use queue.DefaultRetryCount.
	MaxAttempts *int `json:"maxAtts,omitempty"`
	// Payload stores item-specific data for use when processing the item.  For example,
	// this may contain the function's edge for running a step.
	Payload any `json:"payload,omitempty"`
}

func (i Item) GetMaxAttempts() int {
	if i.MaxAttempts == nil {
		return consts.DefaultRetryCount
	}
	return *i.MaxAttempts
}

func (i *Item) UnmarshalJSON(b []byte) error {
	type kind struct {
		Kind        string           `json:"kind"`
		Identifier  state.Identifier `json:"identifier"`
		Attempt     int              `json:"atts"`
		MaxAttempts *int             `json:"maxAtts,omitempty"`
		Payload     json.RawMessage  `json:"payload"`
		WorkspaceID uuid.UUID        `json:"wsID"`
	}
	temp := &kind{}
	err := json.Unmarshal(b, temp)
	if err != nil {
		return fmt.Errorf("error unmarshalling queue item: %w", err)
	}

	i.Kind = temp.Kind
	i.Identifier = temp.Identifier
	i.Attempt = temp.Attempt
	i.MaxAttempts = temp.MaxAttempts
	i.WorkspaceID = temp.WorkspaceID
	// Save this for custom unmarshalling of other jobs.  This is overwritten
	// for known queue kinds.
	if len(temp.Payload) > 0 {
		i.Payload = temp.Payload
	}

	switch temp.Kind {
	case KindEdge:
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadEdge{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	case KindPause:
		if len(temp.Payload) == 0 {
			return nil
		}
		p := &PayloadPauseTimeout{}
		if err := json.Unmarshal(temp.Payload, p); err != nil {
			return err
		}
		i.Payload = *p
	}
	return nil
}

// GetEdge returns the edge from the enqueued item, if the payload is of type PayloadEdge.
func GetEdge(i Item) (*PayloadEdge, error) {
	switch v := i.Payload.(type) {
	case PayloadEdge:
		return &v, nil
	default:
		return nil, fmt.Errorf("unable to get edge from payload type: %T", v)
	}
}

// PayloadEdge is the payload stored when enqueueing an edge traversal to execute
// the incoming step of the edge.
type PayloadEdge struct {
	Edge inngest.Edge `json:"edge"`
	// StackIndex represents the current index within the run state
	// when enqueueing the next step.  This is necessary to calculate
	// steps with parallelism within SDKs.
	StackIndex           int                   `json:"idx"`
	ResponseSaveOnHandle *state.DriverResponse `json:"respSaveOnHandle,omitempty"`
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
