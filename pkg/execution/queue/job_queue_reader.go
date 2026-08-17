package queue

import (
	"context"
	"fmt"
	"iter"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

type shardBackedReaders struct {
	shards                       ShardRegistry
	accountShardIterationEnabled AccountShardIterationEnabled
}

func backlogOperations(shard QueueShard) (BacklogOperations, error) {
	reader, ok := shard.(BacklogOperations)
	if !ok {
		return nil, fmt.Errorf("queue shard %q does not support backlog reads", shard.Name())
	}
	return reader, nil
}

func newShardBackedReaders(shards ShardRegistry, accountShardIterationEnabled AccountShardIterationEnabled) *shardBackedReaders {
	return &shardBackedReaders{
		shards:                       shards,
		accountShardIterationEnabled: accountShardIterationEnabled,
	}
}

// NewRunQueueReader returns a run reader backed by the provided shard registry.
func NewRunQueueReader(shards ShardRegistry, accountShardIterationEnabled AccountShardIterationEnabled) RunQueueReader {
	return newShardBackedReaders(shards, accountShardIterationEnabled)
}

// NewQueueStatusReader returns a status reader backed by the provided shard registry.
func NewQueueStatusReader(shards ShardRegistry, accountShardIterationEnabled AccountShardIterationEnabled) QueueStatusReader {
	return newShardBackedReaders(shards, accountShardIterationEnabled)
}

// NewQueuePartitionReader returns a partition reader backed by the provided shard registry.
func NewQueuePartitionReader(shards ShardRegistry, accountShardIterationEnabled AccountShardIterationEnabled) QueuePartitionReader {
	return newShardBackedReaders(shards, accountShardIterationEnabled)
}

// NewQueueBacklogReader returns a backlog reader backed by the provided shard registry.
func NewQueueBacklogReader(shards ShardRegistry, accountShardIterationEnabled AccountShardIterationEnabled) QueueBacklogReader {
	return newShardBackedReaders(shards, accountShardIterationEnabled)
}

// NewQueueItemReader returns an item reader backed by the provided shard registry.
func NewQueueItemReader(shards ShardRegistry, accountShardIterationEnabled AccountShardIterationEnabled) QueueItemReader {
	return newShardBackedReaders(shards, accountShardIterationEnabled)
}

// BacklogSize implements QueueBacklogReader.
func (r *shardBackedReaders) BacklogSize(ctx context.Context, shard QueueShard, backlogID string) (int64, error) {
	reader, err := backlogOperations(shard)
	if err != nil {
		return 0, err
	}
	return reader.BacklogSize(ctx, backlogID)
}

// BacklogByID implements QueueBacklogReader.
func (r *shardBackedReaders) BacklogByID(ctx context.Context, shard QueueShard, backlogID string) (*QueueBacklog, error) {
	reader, err := backlogOperations(shard)
	if err != nil {
		return nil, err
	}
	return reader.BacklogByID(ctx, backlogID)
}

// BacklogsByPartition implements QueueBacklogReader.
func (r *shardBackedReaders) BacklogsByPartition(ctx context.Context, shard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error) {
	reader, err := backlogOperations(shard)
	if err != nil {
		return nil, err
	}
	return reader.BacklogsByPartition(ctx, partitionID, from, until, opts...)
}

// ItemExists implements QueueItemReader.
func (r *shardBackedReaders) ItemExists(ctx context.Context, shard QueueShard, scope Scope, jobID string) (bool, error) {
	return shard.ItemExists(ctx, scope, jobID)
}

// ItemsByBacklog implements QueueBacklogReader.
func (r *shardBackedReaders) ItemsByBacklog(ctx context.Context, shard QueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	reader, err := backlogOperations(shard)
	if err != nil {
		return nil, err
	}
	return reader.ItemsByBacklog(ctx, backlogID, from, until, opts...)
}

// ItemsByPartition implements QueuePartitionReader.
func (r *shardBackedReaders) ItemsByPartition(ctx context.Context, shard QueueShard, scope Scope, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	return shard.ItemsByPartition(ctx, scope, partitionID, from, until, opts...)
}

// ItemsByRunID implements RunQueueReader.
func (r *shardBackedReaders) ItemsByRunID(ctx context.Context, shard QueueShard, scope Scope, runID ulid.ULID) ([]*QueueItem, error) {
	return shard.ItemsByRunID(ctx, scope, runID)
}

// LoadQueueItem implements QueueItemReader.
func (r *shardBackedReaders) LoadQueueItem(ctx context.Context, shardName string, itemID string) (*QueueItem, error) {
	shard, err := r.shards.ByName(shardName)
	if err != nil {
		return nil, err
	}

	return shard.LoadQueueItem(ctx, itemID)
}

func (r *shardBackedReaders) forAccountShards(ctx context.Context, accountID uuid.UUID, fn func(context.Context, QueueShard) error) error {
	// Fan-out is feature-flagged because querying every shard increases latency.
	// Shard failures are logged and suppressed by ForEach so healthy shard
	// results remain available.
	if r.accountShardIterationEnabled != nil && r.accountShardIterationEnabled(ctx, accountID) {
		return r.shards.ForEach(ctx, fn)
	}

	shard, err := r.shards.Resolve(ctx, Scope{AccountID: accountID}, nil)
	if err != nil {
		return fmt.Errorf("could not resolve account shard: %w", err)
	}
	return fn(ctx, shard)
}

// PartitionBacklogSize implements QueueBacklogReader.
func (r *shardBackedReaders) PartitionBacklogSize(ctx context.Context, scope Scope, partitionID string) (int64, error) {
	var totalCount int64

	err := r.forAccountShards(ctx, scope.AccountID, func(ctx context.Context, shard QueueShard) error {
		reader, err := backlogOperations(shard)
		if err != nil {
			return err
		}
		backlogSize, err := reader.PartitionBacklogSize(ctx, scope, partitionID)
		if err != nil {
			return fmt.Errorf("could not load partition backlog size: %w", err)
		}
		l := logger.StdlibLogger(ctx)
		l.Trace("retrieved backlog size", "size", backlogSize)
		atomic.AddInt64(&totalCount, int64(backlogSize))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load partition backlog size: %w", err)
	}
	return totalCount, nil
}

// PartitionSize implements QueuePartitionReader.
func (r *shardBackedReaders) PartitionSize(ctx context.Context, scope Scope, partitionID string, until time.Time) (int64, error) {
	var totalCount int64

	err := r.forAccountShards(ctx, scope.AccountID, func(ctx context.Context, shard QueueShard) error {
		partitionSize, err := shard.PartitionSize(ctx, scope, partitionID, until)
		if err != nil {
			return fmt.Errorf("could not load partition size: %w", err)
		}
		atomic.AddInt64(&totalCount, partitionSize)
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load partition size: %w", err)
	}
	return totalCount, nil
}

// PartitionByID implements QueuePartitionReader.
func (r *shardBackedReaders) PartitionByID(ctx context.Context, shard QueueShard, scope Scope, partitionID string) (*PartitionInspectionResult, error) {
	return shard.PartitionByID(ctx, scope, partitionID)
}

// OutstandingJobCount implements RunQueueReader.
func (r *shardBackedReaders) OutstandingJobCount(ctx context.Context, scope Scope, runID ulid.ULID) (int, error) {
	var totalCount int64

	err := r.forAccountShards(ctx, scope.AccountID, func(ctx context.Context, shard QueueShard) error {
		outstanding, err := shard.OutstandingJobCount(ctx, scope, runID)
		if err != nil {
			return fmt.Errorf("could not load outstanding job count: %w", err)
		}
		atomic.AddInt64(&totalCount, int64(outstanding))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load outstanding count: %w", err)
	}
	return int(totalCount), nil
}

// RunJobs implements RunQueueReader.
func (r *shardBackedReaders) RunJobs(ctx context.Context, shardName string, scope Scope, runID ulid.ULID, limit int64, offset int64) ([]JobResponse, error) {
	shard, err := r.shards.ByName(shardName)
	if err != nil {
		return nil, err
	}

	return shard.RunJobs(ctx, scope, runID, limit, offset)
}

// RunningCount implements QueueStatusReader.
func (r *shardBackedReaders) RunningCount(ctx context.Context, scope Scope) (int64, error) {
	var totalCount int64

	err := r.forAccountShards(ctx, scope.AccountID, func(ctx context.Context, shard QueueShard) error {
		running, err := shard.RunningCount(ctx, scope)
		if err != nil {
			return fmt.Errorf("could not load running count: %w", err)
		}
		atomic.AddInt64(&totalCount, int64(running))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load running count: %w", err)
	}
	return totalCount, nil
}

// StatusCount implements QueueStatusReader.
func (r *shardBackedReaders) StatusCount(ctx context.Context, scope Scope, status string) (int64, error) {
	var totalCount int64

	err := r.forAccountShards(ctx, scope.AccountID, func(ctx context.Context, shard QueueShard) error {
		running, err := shard.StatusCount(ctx, scope, status)
		if err != nil {
			return fmt.Errorf("could not load status count: %w", err)
		}
		atomic.AddInt64(&totalCount, int64(running))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load status count: %w", err)
	}
	return totalCount, nil
}

var (
	_ RunQueueReader       = (*shardBackedReaders)(nil)
	_ QueueStatusReader    = (*shardBackedReaders)(nil)
	_ QueuePartitionReader = (*shardBackedReaders)(nil)
	_ QueueBacklogReader   = (*shardBackedReaders)(nil)
	_ QueueItemReader      = (*shardBackedReaders)(nil)
)
