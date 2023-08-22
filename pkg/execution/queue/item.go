package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
)

const (
	KindEdge  = "edge"
	KindSleep = "sleep"
	KindPause = "pause"
)

type jobIDValType struct{}

var (
	jobCtxVal = jobIDValType{}
)

// WithJobID returns a context that stores the given job ID inside.
func WithJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobCtxVal, jobID)
}

// JobIDFromContext returns the job ID given the current context, or an
// empty string if there's no job ID.
func JobIDFromContext(ctx context.Context) string {
	str, _ := ctx.Value(jobCtxVal).(string)
	return str
}

// Item represents an item stored within a queue.
//
// Note that each individual implementation may wrap this to add their own fields,
// such as a job identifier.
//
// TODO: Refactor this with the QueueItem in redis state to remove duplicates.
type Item struct {
	// JobID is an internal ID used to deduplicate queue items.
	JobID *string `json:"-"`
	// GroupID allows tracking step history across many jobs;  if a step is scheduled,
	// then runs and fails, it's rescheduled.  We want the same group ID to be stored
	// across the lifetime of a step so that we can correlate all history entries across
	// a specific step.
	GroupID string `json:"groupID,omitempty"`
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

	// Metadata is used for storing additional metadata related to the queue item.
	// e.g. tracing data
	Metadata map[string]string `json:"metadata,omitempty"`

	// QueueName allows control over the queue name.  If not provided, this falls
	// back to the queue mapping defined on the queue or the workflow ID of the fn.
	QueueName *string `json:"qn,omitempty"`
	// DisableImmediateExecution is used to store whether steps should be allowed
	// to execute immediately upon being found, or whether they should be scheduled
	// to ensure idempotency.
	DisableImmediateExection bool `json:"disableImmExec,omitempty"`
}

func (i Item) GetMaxAttempts() int {
	if i.MaxAttempts == nil {
		return consts.DefaultRetryCount
	}
	return *i.MaxAttempts
}

func (i *Item) UnmarshalJSON(b []byte) error {
	type kind struct {
		GroupID     string            `json:"groupID"`
		WorkspaceID uuid.UUID         `json:"wsID"`
		Kind        string            `json:"kind"`
		Identifier  state.Identifier  `json:"identifier"`
		Attempt     int               `json:"atts"`
		MaxAttempts *int              `json:"maxAtts,omitempty"`
		Payload     json.RawMessage   `json:"payload"`
		Metadata    map[string]string `json:"metadata"`
	}
	temp := &kind{}
	err := json.Unmarshal(b, temp)
	if err != nil {
		return fmt.Errorf("error unmarshalling queue item: %w", err)
	}

	i.GroupID = temp.GroupID
	i.WorkspaceID = temp.WorkspaceID
	i.Kind = temp.Kind
	i.Identifier = temp.Identifier
	i.Attempt = temp.Attempt
	i.MaxAttempts = temp.MaxAttempts
	i.Metadata = temp.Metadata
	// Save this for custom unmarshalling of other jobs.  This is overwritten
	// for known queue kinds.
	if len(temp.Payload) > 0 {
		i.Payload = temp.Payload
	}

	switch temp.Kind {
	case KindEdge, KindSleep:
		// Edge and Sleep are the same;  the only difference is that the executor
		// runner should always save nil to the state store using the outgoing edge's
		// ID when processing a sleep so that the state + stack are updated properly.
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
