package state

import (
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
	// ErrInboxClosed is returned when a message was not saved because its inbox
	// was closed.
	ErrInboxClosed = errors.New("inbox message not saved: the inbox is closed")
)

// InboxMessage is one event an inbox received, saved as its own step.
type InboxMessage struct {
	// StepID is the id of the message step.
	StepID string
	// Inbox is the id of the op that opened the inbox.
	Inbox string
	// Value is the step's saved value: the event plus the inbox.
	Value []byte
	// MaxSteps and MaxStateSize are the run's existing step and state size
	// bounds. Zero means unchecked. A message that would pass either is not
	// saved.
	MaxSteps     int
	MaxStateSize int
}

// InboxWait is a read of an inbox that had nothing unread. An inbox has one
// reader, so it has at most one wait, recorded under the inbox. It stays
// recorded in run state until a message of the inbox satisfies it.
type InboxWait struct {
	// ID is the id of the read's op.
	ID string `json:"id"`
	// Inbox is the id of the op that opened the inbox.
	Inbox string `json:"inbox"`
	// Seen is the last step id of the stack the SDK had loaded.
	Seen string `json:"seen"`
	// Timer is whether a timer is queued to time the read out.
	Timer bool `json:"timer,omitempty"`
	// Batch is set for a batch read, whose result is a cut at the last message
	// the executor found for it.
	Batch *InboxBatch `json:"batch,omitempty"`

	// The rest is what waking the read needs.
	Incoming            string             `json:"incoming"`
	GroupID             string             `json:"groupID"`
	MaxAttempts         *int               `json:"maxAtts,omitempty"`
	ParallelMode        enums.ParallelMode `json:"pm,omitempty"`
	ParallelCoalesceKey string             `json:"pck,omitempty"`
}

// InboxWaitSet is what a run holds for its waiting reads.
type InboxWaitSet struct {
	// Waits are the recorded waits, at most one per inbox.
	Waits []InboxWait
	// Closed are the ids of the inboxes that were closed, so a read recorded
	// later under one of them is ended and a message for one is dropped.
	Closed []string
}

// InboxBatch is what makes a read a batch read.
type InboxBatch struct {
	// Drain is whether the read never waits.
	Drain bool `json:"drain,omitempty"`
	// Unread is whether the SDK holds unread messages already.
	Unread bool `json:"unread,omitempty"`
	// Max is how many messages the answer may cover. Missing means all.
	Max *int `json:"max,omitempty"`
}

// InboxClose closes an inbox.
type InboxClose struct {
	// StepID is the id of the close op.
	StepID string
	// Inbox is the id of the op that opened the inbox.
	Inbox string
}

// InboxWake is a read that a message satisfied.
type InboxWake struct {
	Wait InboxWait
	// MessageStep is the step id of the message that satisfied the read.
	MessageStep string
	// TimedOut is whether the read's timer satisfied it, so there is no message.
	TimedOut bool
	// Closed is whether closing the read's inbox satisfied it, so there is no
	// message.
	Closed bool
	// HasPending is whether the run has other pending steps once the read has
	// left the pending set.
	HasPending bool
}
