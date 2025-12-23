package queue

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

// QueuePartition represents an individual queue for a workflow.  It stores the
// time of the earliest job within the workflow.
type QueuePartition struct {
	// ID represents the key used within the global Partition hash and global pointer set
	// which represents this QueuePartition.  This is the function ID for enums.PartitionTypeDefault,
	// or the entire key returned from the key generator for other types.
	ID string `json:"id,omitempty"`
	// QueueName is used for manually overriding queue items to be enqueued for
	// system jobs like pause events and timeouts, batch timeouts, and replays.
	//
	// NOTE: This field is required for backwards compatibility, as old system partitions
	// simply set the queue name.
	//
	// This should almost always be nil.
	QueueName *string `json:"queue,omitempty"`
	// FunctionID represents the function ID that this partition manages.
	// NOTE:  If this partition represents many fns (eg. acct or env), this may be nil
	FunctionID *uuid.UUID `json:"wid,omitempty"`
	// EnvID represents the environment ID for the partition, either from the
	// function ID or the environment scope itself.
	EnvID *uuid.UUID `json:"wsID,omitempty"`
	// AccountID represents the account ID for the partition
	AccountID uuid.UUID `json:"aID,omitempty"`
	// LeaseID represents a lease on this partition.  If the LeaseID is not nil,
	// this partition can be claimed by a shared-nothing worker to work on the
	// queue items within this partition.
	//
	// A lease is shortly held (eg seconds).  It should last long enough for
	// workers to claim QueueItems only.
	LeaseID *ulid.ULID `json:"leaseID,omitempty"`
	// Last represents the time that this partition was last leased, as a millisecond
	// unix epoch.  In essence, we need this to track how frequently we're leasing and
	// attempting to run items in the partition's queue.
	// Without this, we cannot track sojourn latency.
	Last int64 `json:"last"`
	// ForcedAtMS records the time that the partition is forced to, in milliseconds, if
	// the partition has been forced into the future via concurrency issues. This means
	// that it was requeued due to concurrency issues and should not be brought forward
	// when a new step is enqueued, if now < ForcedAtMS.
	ForceAtMS int64 `json:"forceAtMS"`
}

func (qp QueuePartition) IsSystem() bool {
	return qp.QueueName != nil && *qp.QueueName != ""
}

// ItemPartitions returns the partition for a given item.
func ItemPartition(ctx context.Context, i QueueItem) QueuePartition {
	l := logger.StdlibLogger(ctx)

	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		l.Warn("encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set",
			"item", i,
		)
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		l.Warn("encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName",
			"item", i,
		)
	}

	// The only case when we manually set a queueName is for system partitions
	if queueName != nil {
		systemPartition := QueuePartition{
			// NOTE: Never remove this. The ID is required to enqueue items to the
			// partition, as it is used for conditional checks in Lua
			ID:        *queueName,
			QueueName: queueName,
		}
		return systemPartition
	}

	if i.FunctionID == uuid.Nil {
		l.Error("unexpected missing functionID in ItemPartitions()", "item", i)
	}

	fnPartition := QueuePartition{
		ID:         i.FunctionID.String(),
		FunctionID: &i.FunctionID,
		AccountID:  i.Data.Identifier.AccountID,
	}

	return fnPartition
}

func (qp QueuePartition) Queue() string {
	// This is redundant but acts as a safeguard, so that
	// we always return the ID (queueName) for system partitions
	if qp.IsSystem() {
		return *qp.QueueName
	}

	if qp.ID == "" && qp.FunctionID != nil {
		return qp.FunctionID.String()
	}

	return qp.ID
}

func (qp QueuePartition) MarshalBinary() ([]byte, error) {
	return json.Marshal(qp)
}
