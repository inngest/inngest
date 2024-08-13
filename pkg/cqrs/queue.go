package cqrs

import (
	"context"
)

type SnapshotValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type QueueSnapshot = map[string]SnapshotValue

// LiteQueueSnapshotManager is a development-only function manager
type LiteQueueSnapshotManager interface {
	LiteQueueSnapshotReader
	LiteQueueSnapshotWriter
}

type LiteQueueSnapshotReader interface {
	GetQueueSnapshot(ctx context.Context, snapshotID int64) (QueueSnapshot, error)
	GetLatestQueueSnapshot(ctx context.Context) (QueueSnapshot, error)
	GetLatestQueueSnapshotID(ctx context.Context) (int64, error)
}

type LiteQueueSnapshotWriter interface {
	InsertQueueSnapshot(ctx context.Context, params InsertQueueSnapshotParams) (int64, error)
	InsertQueueSnapshotChunk(ctx context.Context, params InsertQueueSnapshotChunkParams) error
}

type InsertQueueSnapshotParams struct {
	Snapshot QueueSnapshot
}

type InsertQueueSnapshotChunkParams struct {
	SnapshotID int64
	ChunkID    int
	Chunk      []byte
}
