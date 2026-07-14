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

type jobQueueReader struct {
	shards                       QueueShardRegistry
	accountShardIterationEnabled AccountShardIterationEnabled
}

func newJobQueueReader(shards QueueShardRegistry, accountShardIterationEnabled AccountShardIterationEnabled) JobQueueReader {
	return &jobQueueReader{
		shards:                       shards,
		accountShardIterationEnabled: accountShardIterationEnabled,
	}
}

// BacklogSize implements JobQueueReader.
func (r *jobQueueReader) BacklogSize(ctx context.Context, shard QueueShard, backlogID string) (int64, error) {
	return shard.BacklogSize(ctx, backlogID)
}

// BacklogByID implements JobQueueReader.
func (r *jobQueueReader) BacklogByID(ctx context.Context, shard QueueShard, backlogID string) (*QueueBacklog, error) {
	return shard.BacklogByID(ctx, backlogID)
}

// BacklogsByPartition implements JobQueueReader.
func (r *jobQueueReader) BacklogsByPartition(ctx context.Context, shard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error) {
	return shard.BacklogsByPartition(ctx, partitionID, from, until, opts...)
}

// ItemExists implements JobQueueReader.
func (r *jobQueueReader) ItemExists(ctx context.Context, shard QueueShard, scope Scope, jobID string) (bool, error) {
	return shard.ItemExists(ctx, scope, jobID)
}

// ItemsByBacklog implements JobQueueReader.
func (r *jobQueueReader) ItemsByBacklog(ctx context.Context, shard QueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	return shard.ItemsByBacklog(ctx, backlogID, from, until, opts...)
}

// ItemsByPartition implements JobQueueReader.
func (r *jobQueueReader) ItemsByPartition(ctx context.Context, shard QueueShard, scope Scope, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	return shard.ItemsByPartition(ctx, scope, partitionID, from, until, opts...)
}

// ItemsByRunID implements JobQueueReader.
func (r *jobQueueReader) ItemsByRunID(ctx context.Context, shard QueueShard, scope Scope, runID ulid.ULID) ([]*QueueItem, error) {
	return shard.ItemsByRunID(ctx, scope, runID)
}

// LoadQueueItem implements JobQueueReader.
func (r *jobQueueReader) LoadQueueItem(ctx context.Context, shardName string, itemID string) (*QueueItem, error) {
	shard, err := r.shards.ByName(shardName)
	if err != nil {
		return nil, err
	}

	return shard.LoadQueueItem(ctx, itemID)
}

func (r *jobQueueReader) forAccountShards(ctx context.Context, accountID uuid.UUID, fn func(context.Context, QueueShard) error) error {
	// Fan-out is feature-flagged because querying every shard increases
	// latency and makes a single shard failure affect the whole read.
	if r.accountShardIterationEnabled != nil && r.accountShardIterationEnabled(ctx, accountID) {
		return r.shards.ForEach(ctx, fn)
	}

	shard, err := r.shards.Resolve(ctx, Scope{AccountID: accountID}, nil)
	if err != nil {
		return fmt.Errorf("could not resolve account shard: %w", err)
	}
	return fn(ctx, shard)
}

// PartitionBacklogSize implements JobQueueReader.
func (r *jobQueueReader) PartitionBacklogSize(ctx context.Context, scope Scope, partitionID string) (int64, error) {
	var totalCount int64

	err := r.forAccountShards(ctx, scope.AccountID, func(ctx context.Context, shard QueueShard) error {
		backlogSize, err := shard.PartitionBacklogSize(ctx, scope, partitionID)
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

// PartitionByID implements JobQueueReader.
func (r *jobQueueReader) PartitionByID(ctx context.Context, shard QueueShard, scope Scope, partitionID string) (*PartitionInspectionResult, error) {
	return shard.PartitionByID(ctx, scope, partitionID)
}

// OutstandingJobCount implements JobQueueReader.
func (r *jobQueueReader) OutstandingJobCount(ctx context.Context, scope Scope, runID ulid.ULID) (int, error) {
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

// RunJobs implements JobQueueReader.
func (r *jobQueueReader) RunJobs(ctx context.Context, shardName string, scope Scope, runID ulid.ULID, limit int64, offset int64) ([]JobResponse, error) {
	shard, err := r.shards.ByName(shardName)
	if err != nil {
		return nil, err
	}

	return shard.RunJobs(ctx, scope, runID, limit, offset)
}

// RunningCount implements JobQueueReader.
func (r *jobQueueReader) RunningCount(ctx context.Context, scope Scope) (int64, error) {
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

// StatusCount implements JobQueueReader.
func (r *jobQueueReader) StatusCount(ctx context.Context, scope Scope, status string) (int64, error) {
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
