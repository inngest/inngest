package cqrs

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

type SnapshotValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type (
	QueueSnapshot = map[string]SnapshotValue
	SnapshotID    = ulid.ULID
)

// QueueSnapshotManager is a manager for queue snapshots.
type QueueSnapshotManager interface {
	QueueSnapshotReader
	QueueSnapshotWriter
}

type QueueSnapshotReader interface {
	GetQueueSnapshot(ctx context.Context, snapshotID SnapshotID) (*QueueSnapshot, error)
	GetLatestQueueSnapshot(ctx context.Context) (*QueueSnapshot, error)
}

type QueueSnapshotWriter interface {
	InsertQueueSnapshot(ctx context.Context, params InsertQueueSnapshotParams) (SnapshotID, error)
	InsertQueueSnapshotChunk(ctx context.Context, params InsertQueueSnapshotChunkParams) error
}

type InsertQueueSnapshotParams struct {
	Snapshot QueueSnapshot
}

type InsertQueueSnapshotChunkParams struct {
	SnapshotID SnapshotID
	ChunkID    int
	Chunk      []byte
}

// Queue operations

type QueuePartition struct {
	ID string `json:"id"`

	// Identifiers
	AccountID  uuid.UUID  `json:"acct_id"`
	EnvID      *uuid.UUID `json:"env_id,omitempty"`
	FunctionID *uuid.UUID `json:"fn_id,omitempty"`

	// Config represents the configuration related to the partition
	Config *inngest.Function `json:"config.omitempty"`

	// Paused shows if the partition is paused or not
	Paused bool `json:"paused"`

	// AccountActive shows the active count value for the account, this is used in key queues
	AccountActive int `json:"acct_active"`
	// AccountInProgress shows the count for the items currently being leased and running,
	// this is part of the original indices
	AccountInProgress int `json:"acct_in_progress"`

	// Ready shows the count of items in the queue ready for processing
	// which is also the function partition size when key queues are not enabled
	Ready int `json:"ready"`
	// InProgress shows the count of items currently leased and is being processed
	InProgress int `json:"in_progress"`
	// Active shows the count of items either marked as ready or in-progress
	// NOTE: this is used for key queues
	Active int `json:"active"`
	// Future shows the count of items scheduled to be run in the future in the fn partition
	Future int `json:"future"`

	// Backlogs shows the number of backlogs this partition has
	Backlogs int `json:"backlogs"`
}
