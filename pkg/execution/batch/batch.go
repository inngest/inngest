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
	// Append appends items to a batch.  This may buffer in-memory for functions with
	// the same ID prior to appending.
	//
	// If this happens, the Append function will block until the batch commits.
	Append(ctx context.Context, bi BatchItem, fn inngest.Function) (*BatchAppendResult, error)
	// BulkAppend appends multiple items to a batch in a single call.
	//
	// This method is intended for callers that already have a slice of BatchItem (for example,
	// an in-memory buffer) and want to commit them as one logical operation rather than calling
	// Append for each item. Callers that are appending a single item at a time should prefer
	// Append; BulkAppend is a lower-level API that is typically used by buffering
	// implementations built on top of BatchManager.
	//
	// Like Append, BulkAppend is synchronous: it blocks until the underlying batch changes
	// (such as persisting the items and deciding whether execution should be scheduled or
	// started) have completed, or until an error is returned.
	//
	// Error handling and atomicity semantics:
	//   - The returned BulkAppendResult and error value apply to the entire slice of items
	//     provided in this call.
	//   - Implementations should treat the operation as all-or-nothing with respect to
	//     persistence of the provided items: either all items are appended to the batch, or
	//     none of them are, and a non-nil error is returned.
	//   - Callers must not assume that any particular item has been appended if a non-nil
	//     error is returned.
	//   - If implementations need to surface per-item issues, they should do so via fields
	//     on BulkAppendResult rather than by partially appending items while also returning
	//     an error.
	BulkAppend(ctx context.Context, items []BatchItem, fn inngest.Function) (*BulkAppendResult, error)

	RetrieveItems(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID) ([]BatchItem, error)
	StartExecution(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID, batchPointer string) (string, error)
	ScheduleExecution(ctx context.Context, opts ScheduleBatchOpts) error

	DeleteKeys(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID) error
	// GetBatchInfo retrieves information about the current batch for a function and batch key.
	// This is used for debugging and introspection.
	GetBatchInfo(ctx context.Context, functionID uuid.UUID, batchKey string) (*BatchInfo, error)
	// DeleteBatch deletes the current batch for a function and batch key.
	// Returns information about the deleted batch.
	DeleteBatch(ctx context.Context, functionID uuid.UUID, batchKey string) (*DeleteBatchResult, error)
	// RunBatch schedules immediate execution of a batch by creating a timeout job that runs in one second.
	RunBatch(ctx context.Context, opts RunBatchOpts) (*RunBatchResult, error)
	// Close gracefully shuts down the batch manager, flushing any pending buffers.
	Close() error
}

// BatchInfo contains information about a batch for debugging purposes.
type BatchInfo struct {
	// BatchID is the current batch ULID if one exists.
	BatchID string
	// Items contains the batch items.
	Items []BatchItem
	// Status is the current batch status (pending, started, etc.).
	Status string
}

// DeleteBatchResult contains information about a deleted batch.
type DeleteBatchResult struct {
	// Deleted indicates whether a batch was found and deleted.
	Deleted bool
	// BatchID is the ULID of the deleted batch, if one was deleted.
	BatchID string
	// ItemCount is the number of events that were in the deleted batch.
	ItemCount int
}

// RunBatchOpts contains options for running a batch immediately.
type RunBatchOpts struct {
	FunctionID  uuid.UUID
	BatchKey    string
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
}

// RunBatchResult contains information about a scheduled batch execution.
type RunBatchResult struct {
	// Scheduled indicates whether a batch was found and scheduled.
	Scheduled bool
	// BatchID is the ULID of the batch that was scheduled.
	BatchID string
	// ItemCount is the number of events in the batch.
	ItemCount int
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
