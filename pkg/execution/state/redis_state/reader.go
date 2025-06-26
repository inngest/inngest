package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
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
			JobID:    qi.ID,
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

func (q *queue) ItemsByPartition(ctx context.Context, shard QueueShard, partitionID uuid.UUID, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*osqueue.QueueItem], error) {
	opt := queueIterOpt{
		batchSize: 1000,
		interval:  500 * time.Millisecond,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	l := q.log.With(
		"method", "ItemsByPartition",
		"partitionID", partitionID.String(),
		"from", from,
		"until", until,
	)

	// retrieve partition by ID
	hash := shard.RedisClient.kg.PartitionItem()
	rc := shard.RedisClient.Client()

	cmd := rc.B().Hget().Key(hash).Field(partitionID.String()).Build()
	byt, err := rc.Do(ctx, cmd).AsBytes()
	if err != nil {
		return nil, fmt.Errorf("error retrieving partition '%s': %w", partitionID.String(), err)
	}

	var pt QueuePartition
	if err := json.Unmarshal(byt, &pt); err != nil {
		return nil, fmt.Errorf("error unmarshalling queue partition '%s': %w", partitionID.String(), err)
	}

	return func(yield func(*osqueue.QueueItem) bool) {
		ptFrom := from
		for {
			var iterated int

			// peek function partition
			items, err := q.peek(ctx, shard, peekOpts{
				From:         &ptFrom,
				Until:        until,
				Limit:        opt.batchSize,
				PartitionID:  partitionID.String(),
				PartitionKey: pt.zsetKey(shard.RedisClient.kg),
			})
			if err != nil {
				l.Error("error peeking items for iterator", "error", err)
				return
			}

			var start, end time.Time
			for _, qi := range items {
				if qi == nil {
					continue
				}

				if !yield(qi) {
					return
				}

				at := time.UnixMilli(qi.AtMS)
				if start.IsZero() {
					start = at
				}
				end = at
				ptFrom = at
				iterated++
			}

			l.Debug("iterated items in partition",
				"count", iterated,
				"start", start.Format(time.StampMilli),
				"end", end.Format(time.StampMilli),
			)

			// didn't process anything, exit loop
			if iterated == 0 {
				break
			}

			// shift the starting point 1ms so it doesn't try to grab the same stuff again
			// NOTE: this could result skipping items if the previous batch of items are all on
			// the same millisecond
			ptFrom = ptFrom.Add(time.Millisecond)

			// wait a little before proceeding
			<-time.After(opt.interval)
		}

		// NOTE: iterate through backlogs
		backlogFrom := from

		hash := shard.RedisClient.kg.ShadowPartitionMeta()
		cmd := rc.B().Hget().Key(hash).Field(partitionID.String()).Build()
		byt, err := rc.Do(ctx, cmd).AsBytes()
		if err != nil {
			l.Warn("error retrieving shadow partition from queue", "error", err)
			return
		}

		var spt QueueShadowPartition
		if err := json.Unmarshal(byt, &spt); err != nil {
			l.Error("error unmarshalling shadow partition", "error", err)
			return
		}

		l = l.With("shadow_partition", spt)

		for {
			var iterated int

			// TODO: maybe provide a different limit?
			backlogs, _, err := q.ShadowPartitionPeek(ctx, &spt, true, until, ShadowPartitionPeekMaxBacklogs,
				WithPeekOptQueueShard(&shard),
			)
			if err != nil {
				l.Error("error peeking backlogs for partition", "error", err)
				return
			}

			if len(backlogs) == 0 {
				l.Warn("no more backlogs to iterate")
				return
			}

			latestTimes := []time.Time{}
			for _, backlog := range backlogs {
				var last time.Time
				items, _, err := q.backlogPeek(ctx, backlog, backlogFrom, until, opt.batchSize,
					WithPeekOptQueueShard(&shard),
				)
				if err != nil {
					l.Error("error retrieving queue items from backlog", "error", err)
					return
				}

				var start, end time.Time
				for _, qi := range items {
					if qi == nil {
						continue
					}

					if !yield(qi) {
						return
					}
					iterated++

					at := time.UnixMilli(qi.AtMS)
					if start.IsZero() {
						start = at
					}
					end = at
					last = at
				}

				l.Debug("iterated items in backlog",
					"count", iterated,
					"start", start.Format(time.StampMilli),
					"end", end.Format(time.StampMilli),
				)
				latestTimes = append(latestTimes, last)

				// didn't process anything, meaning there's nothing left to do
				// exit loop
				if iterated == 0 {
					return
				}
			}

			// find the earliest time within the last item timestamp of the previously processed backlogs
			var earliest time.Time
			for _, t := range latestTimes {
				if earliest.IsZero() || t.Before(earliest) {
					earliest = t
				}
			}
			// shift the starting point 1ms so it doesn't try to grab the same stuff again
			// NOTE: this could result skipping items if the previous batch of items are all on
			// the same millisecond
			backlogFrom = earliest.Add(time.Millisecond)

			// wait a little before proceeding
			<-time.After(opt.interval)
		}
	}, nil
}

func (q *queue) ItemsByBacklog(ctx context.Context, shard QueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*osqueue.QueueItem], error) {
	opt := queueIterOpt{
		batchSize: 1000,
		interval:  500 * time.Millisecond,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	l := q.log.With(
		"method", "ItemsByBacklog",
		"backlogID", backlogID,
		"from", from,
		"until", until,
	)

	hash := shard.RedisClient.kg.BacklogMeta()
	rc := shard.RedisClient.Client()

	cmd := rc.B().Hget().Key(hash).Field(backlogID).Build()
	byt, err := rc.Do(ctx, cmd).AsBytes()
	if err != nil {
		return nil, fmt.Errorf("error retrieving backlog: %w", err)
	}

	var backlog QueueBacklog
	if err := json.Unmarshal(byt, &backlog); err != nil {
		return nil, fmt.Errorf("error unmarshalling backlog: %w", err)
	}

	backlogFrom := from
	return func(yield func(*osqueue.QueueItem) bool) {
		for {
			var iterated int

			// peek items for backlog
			items, _, err := q.backlogPeek(ctx, &backlog, backlogFrom, until, opt.batchSize,
				WithPeekOptQueueShard(&shard),
			)
			if err != nil {
				l.Error("error retrieving queue items from backlog", "error", err)
				return
			}

			var start, end time.Time
			for _, qi := range items {
				if !yield(qi) {
					return
				}
				iterated++

				at := time.UnixMilli(qi.AtMS)
				if start.IsZero() {
					start = at
				}
				end = at
				backlogFrom = at
			}

			l.Debug("iterated items in backlog",
				"count", iterated,
				"start", start.Format(time.StampMilli),
				"end", end.Format(time.StampMilli),
			)

			// didn't process anything, meaning there's nothing left to do
			// exit loop
			if iterated == 0 {
				return
			}

			// shift the starting point 1ms so it doesn't try to grab the same stuff again
			// NOTE: this could result skipping items if the previous batch of items are all on
			// the same millisecond
			backlogFrom = backlogFrom.Add(time.Millisecond)

			<-time.After(opt.interval)
		}
	}, nil
}
