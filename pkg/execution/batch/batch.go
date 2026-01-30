package batch

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

// HashBatchKey hashes a batch key using SHA256 and encodes it as base64.
// This is used to create a consistent key for batch pointers.
func HashBatchKey(batchKey string) string {
	hashedBatchKey := sha256.Sum256([]byte(batchKey))
	return base64.StdEncoding.EncodeToString(hashedBatchKey[:])
}

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
	Append(ctx context.Context, bi BatchItem, fn inngest.Function) (*BatchAppendResult, error)
	RetrieveItems(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) ([]BatchItem, error)
	StartExecution(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID, batchPointer string) (string, error)
	ScheduleExecution(ctx context.Context, opts ScheduleBatchOpts) error
	DeleteKeys(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) error
}

// BatchItem represents the item that are being batched.
type BatchItem struct {
	AccountID       uuid.UUID   `json:"acctID"`
	WorkspaceID     uuid.UUID   `json:"wsID"`
	AppID           uuid.UUID   `json:"appID"`
	FunctionID      uuid.UUID   `json:"fnID"`
	FunctionVersion int         `json:"fnV"`
	EventID         ulid.ULID   `json:"evtID"`
	Event           event.Event `json:"evt"`
	Version         int         `json:"v"`
}

func (b BatchItem) GetAccountID() uuid.UUID {
	return b.AccountID
}

func (b BatchItem) GetWorkspaceID() uuid.UUID {
	return b.WorkspaceID
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
	//   append: An event successfully appended to an existing batch
	//   new: A new batch was created with the passed in event
	//   full: The batch is full and ready for execution
	Status          enums.Batch `json:"status"`
	BatchID         string      `json:"batchID,omitempty"`
	BatchPointerKey string      `json:"batchPointerKey"`
}

type ScheduleBatchOpts struct {
	ScheduleBatchPayload

	At time.Time `json:"at"`
}

func (o *ScheduleBatchOpts) JobID() string {
	return fmt.Sprintf("%s:%s", o.WorkspaceID, o.BatchID)
}

type ScheduleBatchPayload struct {
	BatchID                    ulid.ULID  `json:"batchID"`
	BatchPointer               string     `json:"batchPointer"`
	AccountID                  uuid.UUID  `json:"acctID"`
	WorkspaceID                uuid.UUID  `json:"wsID"`
	AppID                      uuid.UUID  `json:"appID"`
	FunctionID                 uuid.UUID  `json:"fnID"`
	FunctionVersion            int        `json:"fnV"`
	DeprecatedFunctionPausedAt *time.Time `json:"fpAt,omitempty"` // deprecated
}
