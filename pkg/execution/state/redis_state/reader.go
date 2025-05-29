package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
)

// RunJobs returns a list of jobs that are due to run for a given run ID.
func (q *queue) RunJobs(ctx context.Context, queueShardName string, workspaceID, workflowID uuid.UUID, runID ulid.ULID, limit, offset int64) ([]osqueue.JobResponse, error) {
	if limit > 1000 || limit <= 0 {
		limit = 1000
	}

	shard, ok := q.queueShardClients[queueShardName]
	if !ok {
		return nil, fmt.Errorf("queue shard %s not found", queueShardName)
	}

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for RunJobs: %s", shard.Kind)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "RunJobs"), redis_telemetry.ScopeQueue)

	cmd := shard.RedisClient.unshardedRc.B().Zscan().Key(shard.RedisClient.kg.RunIndex(runID)).Cursor(uint64(offset)).Count(limit).Build()
	jobIDs, err := shard.RedisClient.unshardedRc.Do(ctx, cmd).AsScanEntry()
	if err != nil {
		return nil, fmt.Errorf("error reading index: %w", err)
	}

	if len(jobIDs.Elements) == 0 {
		return []osqueue.JobResponse{}, nil
	}

	// Get all job items.
	jsonItems, err := shard.RedisClient.unshardedRc.Do(ctx, shard.RedisClient.unshardedRc.B().Hmget().Key(shard.RedisClient.kg.QueueItem()).Field(jobIDs.Elements...).Build()).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error reading jobs: %w", err)
	}

	resp := []osqueue.JobResponse{}
	for _, str := range jsonItems {
		if len(str) == 0 {
			continue
		}
		qi := &osqueue.QueueItem{}

		if err := json.Unmarshal([]byte(str), qi); err != nil {
			return nil, fmt.Errorf("error unmarshalling queue item: %w", err)
		}
		if qi.Data.Identifier.WorkspaceID != workspaceID {
			continue
		}
		// TODO Do we need to check backlogs here?
		cmd := shard.RedisClient.unshardedRc.B().Zrank().Key(shard.RedisClient.kg.FnQueueSet(workflowID.String())).Member(qi.ID).Build()
		pos, err := shard.RedisClient.unshardedRc.Do(ctx, cmd).AsInt64()
		if !rueidis.IsRedisNil(err) && err != nil {
			return nil, fmt.Errorf("error reading queue position: %w", err)
		}
		resp = append(resp, osqueue.JobResponse{
			At:       time.UnixMilli(qi.AtMS),
			Position: pos,
			Kind:     qi.Data.Kind,
			Attempt:  qi.Data.Attempt,
			Raw:      qi,
		})
	}

	return resp, nil
}

func (q *queue) OutstandingJobCount(ctx context.Context, workspaceID, workflowID uuid.UUID, runID ulid.ULID) (int, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "OutstandingJobCount"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return 0, fmt.Errorf("unsupported queue shard kind for OutstandingJobCount: %s", q.primaryQueueShard.Kind)
	}

	cmd := q.primaryQueueShard.RedisClient.unshardedRc.B().Zcard().Key(q.primaryQueueShard.RedisClient.kg.RunIndex(runID)).Build()
	count, err := q.primaryQueueShard.RedisClient.unshardedRc.Do(ctx, cmd).AsInt64()
	if err != nil {
		return 0, fmt.Errorf("error counting index cardinality: %w", err)
	}
	return int(count), nil
}

func (q *queue) StatusCount(ctx context.Context, workflowID uuid.UUID, status string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "StatusCount"), redis_telemetry.ScopeQueue)

	iterate := func(client *QueueClient) (int64, error) {
		key := client.kg.Status(status, workflowID)
		cmd := client.unshardedRc.B().Zcount().Key(key).Min("-inf").Max("+inf").Build()
		count, err := client.unshardedRc.Do(ctx, cmd).AsInt64()
		if err != nil {
			return 0, fmt.Errorf("error inspecting function queue status: %w", err)
		}

		return count, nil
	}

	var count int64

	// Map-reduce over shards
	if q.queueShardClients != nil {
		eg := errgroup.Group{}

		for shardName, shard := range q.queueShardClients {
			shard := shard

			if shard.Kind != string(enums.QueueShardKindRedis) {
				// TODO Support other storage backends
				continue
			}

			eg.Go(func() error {
				shardCount, err := iterate(shard.RedisClient)
				if err != nil {
					return fmt.Errorf("could not count status for shard %s: %w", shardName, err)
				}
				atomic.AddInt64(&count, shardCount)
				return nil
			})
		}

		err := eg.Wait()
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

func (q *queue) RunningCount(ctx context.Context, workflowID uuid.UUID) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "RunningCount"), redis_telemetry.ScopeQueue)

	iterate := func(client *QueueClient) (int64, error) {
		// Load the partition for a given queue.  This allows us to generate the concurrency
		// key properly via the given function.
		//
		// TODO: Remove the ability to change keys based off of initialized inputs.  It's more trouble than
		// it's worth, and ends up meaning we have more queries to write (such as this) in order to load
		// relevant data.
		cmd := client.unshardedRc.B().Hget().Key(client.kg.PartitionItem()).Field(workflowID.String()).Build()
		enc, err := client.unshardedRc.Do(ctx, cmd).AsBytes()
		if rueidis.IsRedisNil(err) {
			return 0, nil
		}
		if err != nil {
			return 0, fmt.Errorf("error fetching partition: %w", err)
		}
		item := &QueuePartition{}
		if err = json.Unmarshal(enc, item); err != nil {
			return 0, fmt.Errorf("error reading partition item: %w", err)
		}

		// Fetch the concurrency via the partition concurrency name.
		key := client.kg.Concurrency("p", workflowID.String())
		cmd = client.unshardedRc.B().Zcard().Key(key).Build()
		count, err := client.unshardedRc.Do(ctx, cmd).AsInt64()
		if err != nil {
			return 0, fmt.Errorf("error inspecting running job count: %w", err)
		}
		return count, nil
	}

	var count int64

	// Map-reduce over shards
	if q.queueShardClients != nil {
		eg := errgroup.Group{}

		for _, shard := range q.queueShardClients {
			if shard.Kind != string(enums.QueueShardKindRedis) {
				// TODO Support other storage backends
				continue
			}

			shard := shard
			eg.Go(func() error {
				shardCount, err := iterate(shard.RedisClient)
				if err != nil {
					return fmt.Errorf("could not count running jobs for shard %s: %w", shard.Name, err)
				}
				atomic.AddInt64(&count, shardCount)
				return nil
			})
		}

		err := eg.Wait()
		if err != nil {
			return 0, err
		}
	}

	// TODO Support other storage backends

	return count, nil
}
