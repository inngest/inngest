package state

import (
	"encoding/json"
	"errors"

	"github.com/inngest/inngest/pkg/enums"
)

var (
	// ErrInboxStepsFull is returned when a message was not saved because the
	// run already has as many steps as it is allowed.
	ErrInboxStepsFull = errors.New("inbox message not saved: the run is at its step bound")
	// ErrInboxStateFull is returned when a message was not saved because it
	// would take the run's state past its size bound.
	ErrInboxStateFull = errors.New("inbox message not saved: the run is at its state size bound")
)

// InboxMessage is one event an inbox received, saved as its own step.
type InboxMessage struct {
	// StepID is the id of the message step.
	StepID string
	// Inbox is the id of the op that opened the inbox.
	Inbox string
	// Value is the step's saved value: the event plus the inbox and sources.
	Value []byte
	// MaxSteps and MaxStateSize are the run's existing step and state size
	// bounds. Zero means unchecked. A message that would pass either is not
	// saved.
	MaxSteps     int
	MaxStateSize int
}

// InboxWait is a read of an inbox that had nothing unread. It stays recorded
// in run state until a message it wants satisfies it.
type InboxWait struct {
	// ID is the id of the read's op.
	ID string `json:"id"`
	// Inbox is the id of the op that opened the inbox.
	Inbox string `json:"inbox"`
	// Want is the filter of the read. Empty means every message of the inbox.
	Want json.RawMessage `json:"want,omitempty"`
	// Seen is the last step id of the stack the SDK had loaded.
	Seen string `json:"seen"`
	// Timer is whether a timer is queued to time the read out.
	Timer bool `json:"timer,omitempty"`

	// The rest is what waking the read needs.
	Incoming            string             `json:"incoming"`
	GroupID             string             `json:"groupID"`
	MaxAttempts         *int               `json:"maxAtts,omitempty"`
	ParallelMode        enums.ParallelMode `json:"pm,omitempty"`
	ParallelCoalesceKey string             `json:"pck,omitempty"`
}

// InboxWake is a read that a message satisfied.
type InboxWake struct {
	Wait InboxWait
	// MessageStep is the step id of the message that satisfied the read.
	MessageStep string
	// TimedOut is whether the read's timer satisfied it, so there is no message.
	TimedOut bool
	// HasPending is whether the run has other pending steps once the read has
	// left the pending set.
	HasPending bool
}
