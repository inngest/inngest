package batch

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"
)

// BatchManager represents an implementation-agnostic event batching, running functions
// only when either the specified buffer is full or the specified time it up.
//
// The order of operation for batching is,
//  1. Find an existing batch key, or create one if there are none.
//  2. Append the batch item to the key
//     2a. If this is the first item in the batch, schedule a job to run after the provided timeout.
//     2b. If this is the last item and the batch is full, start execution immediately, and mark the batch as started
//  3. When time is up for 2a, check if the batch has already started or not
//     3a. If batch has already started, do nothing and exit immediately
//  4. If batch has not started,
//     4a. Mark the batch as started
//     4b. Create a new batch key
//     4c. Update the batch pointer to the newly created key
//
// NOTE:
//
//	#4 needs to happen in one transaction in order to make sure there will not be any race conditions.
type BatchManager interface {
	Append(context.Context, BatchItem) (*BatchAppendResult, error)
	RetrieveItems(context.Context, ulid.ULID) ([]BatchItem, error)
	ScheduleExecution(context.Context, ScheduleBatchOpts) error
	ExpireKeys(context.Context, ulid.ULID) error
}

// BatchItem represents the item that are being batched.
type BatchItem struct {
	AccountID       uuid.UUID   `json:"acctID"`
	WorkspaceID     uuid.UUID   `json:"wsID"`
	FunctionID      uuid.UUID   `json:"fnID"`
	FunctionVersion int         `json:"fnV"`
	EventID         ulid.ULID   `json:"evtID"`
	Event           event.Event `json:"evt"`
	Version         int         `json:"v"`
}

func (b BatchItem) GetInternalID() ulid.ULID {
	return b.EventID
}

func (b BatchItem) GetEvent() event.Event {
	return b.Event
}

// BatchAppendResult represents the status of attempting to append to a batch
type BatchAppendResult struct {
	// Status represents the result of the operation
	//   0: Appended to Batch
	//   1: A new batch is created and appended to it
	//   2: Appened to batch, and the batch is now full
	Status  int    `json:"status"` // TODO: change this to use enums.Batch instead
	BatchID string `json:"batchID,omitempty"`
}

type ScheduleBatchOpts struct {
	BatchID         ulid.ULID `json:"batchID"`
	AccountID       uuid.UUID `json:"acctID"`
	WorkspaceID     uuid.UUID `json:"wsID"`
	FunctionID      uuid.UUID `json:"fnID"`
	FunctionVersion int       `json:"fnV"`
	At              time.Time `json:"at"`
}

type ScheduleBatchPayload struct {
	BatchID         ulid.ULID `json:"batchID"`
	AccountID       uuid.UUID `json:"acctID"`
	WorkspaceID     uuid.UUID `json:"wsID"`
	FunctionID      uuid.UUID `json:"fnID"`
	FunctionVersion int       `json:"fnV"`
}
