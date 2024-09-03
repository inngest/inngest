package cqrs

import (
	"context"

	"github.com/oklog/ulid/v2"
)

type SnapshotValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type QueueSnapshot = map[string]SnapshotValue
type SnapshotID = ulid.ULID

// LiteQueueSnapshotManager is a lite-only manager for queue snapshots.
type LiteQueueSnapshotManager interface {
	LiteQueueSnapshotReader
	LiteQueueSnapshotWriter
}

type LiteQueueSnapshotReader interface {
	GetQueueSnapshot(ctx context.Context, snapshotID SnapshotID) (*QueueSnapshot, error)
	GetLatestQueueSnapshot(ctx context.Context) (*QueueSnapshot, error)
}

type LiteQueueSnapshotWriter interface {
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
