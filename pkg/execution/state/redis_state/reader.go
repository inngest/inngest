package redis_state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
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
		rc := client.unshardedRc

		// Load the partition for a given queue.  This allows us to generate the concurrency
		// key properly via the given function.
		//
		// TODO: Remove the ability to change keys based off of initialized inputs.  It's more trouble than
		// it's worth, and ends up meaning we have more queries to write (such as this) in order to load
		// relevant data.
		cmd := rc.B().Hget().Key(client.kg.PartitionItem()).Field(workflowID.String()).Build()
		enc, err := rc.Do(ctx, cmd).AsBytes()
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

		var count int64
		// Fetch the concurrency via the partition concurrency name.
		key := client.kg.Concurrency("p", workflowID.String())
		cmd = rc.B().Zcard().Key(key).Build()
		cnt, err := rc.Do(ctx, cmd).AsInt64()
		if err != nil {
			return 0, fmt.Errorf("error inspecting job count: %w", err)
		}
		atomic.AddInt64(&count, cnt)
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

func (q *queue) ItemsByPartition(ctx context.Context, shard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*osqueue.QueueItem], error) {
	opt := queueIterOpt{
		batchSize:       1000,
		interval:        500 * time.Millisecond,
		iterateBacklogs: true,
		skipLeased:      true,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	l := q.log.With(
		"method", "ItemsByPartition",
		"partition_id", partitionID,
		"from", from,
		"until", until,
		"queue_shard", shard.Name,
	)

	pt, err := q.PartitionByID(ctx, shard, partitionID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch partition: %w", err)
	}

	l = l.With("account_id", pt.AccountID.String())

	return func(yield func(*osqueue.QueueItem) bool) {
		partitionKey := pt.zsetKey(shard.RedisClient.kg)
		var iterated int

		items := q.iterateSortedSetQueue(ctx, shard, queueSortedSetIterationOptions{
			keySortedSet:  partitionKey,
			partitionID:   partitionID,
			from:          from,
			until:         until,
			pageSize:      int(opt.batchSize),
			includeLeased: !opt.skipLeased,
		})
		for item := range items {
			yield(item)
			iterated++
		}

		l.Debug("iterated items in partition",
			"count", iterated,
		)

		if !opt.iterateBacklogs {
			return
		}

		// NOTE: iterate through backlogs
		backlogFrom := from

		sp, err := q.ShadowPartitionByID(ctx, shard, partitionID)
		if err != nil && !errors.Is(err, ErrShadowPartitionNotFound) {
			l.Warn("error retrieving shadow partition from queue", "error", err)
		}

		if sp == nil {
			return
		}

		l = l.With("shadow_partition", sp)

		for {
			var iterated int

			// TODO: maybe provide a different limit?
			backlogs, _, err := q.ShadowPartitionPeek(ctx, sp, true, until, ShadowPartitionPeekMaxBacklogs,
				WithPeekOptQueueShard(&shard),
			)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					l.ReportError(err, "error peeking backlogs for partition")
				}
				return
			}

			if len(backlogs) == 0 {
				l.Warn("no more backlogs to iterate")
				return
			}

			latestTimes := []time.Time{}
			for _, backlog := range backlogs {
				errTags := map[string]string{
					"backlog_id": backlog.BacklogID,
				}

				var last time.Time
				items, _, err := q.backlogPeek(ctx, backlog, backlogFrom, until, opt.batchSize,
					WithPeekOptQueueShard(&shard),
				)
				if err != nil {
					l.ReportError(err, "error retrieving queue items from backlog",
						logger.WithErrorReportTags(errTags),
					)
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
				l.ReportError(err, "error retrieving queue items from backlog",
					logger.WithErrorReportTags(map[string]string{
						"backlog_id":   backlogID,
						"partition_id": backlog.ShadowPartitionID,
						"queue_shard":  shard.Name,
					}),
				)
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

type QueueIteratorOpts struct {
	// OnPartitionProcessed is called for each partition during instrumentation
	OnPartitionProcessed func(ctx context.Context, partitionKey string, queueKey string, itemCount int64, queueShard QueueShard)
	// OnIterationComplete is called after all partitions are processed with the final totals
	OnIterationComplete func(ctx context.Context, totalPartitions int64, totalQueueItems int64, queueShard QueueShard)
}

func (q *queue) QueueIterator(ctx context.Context, opts QueueIteratorOpts) (partitionCount int64, queueItemCount int64, err error) {
	l := q.log.With("method", "QueueIterator")

	// Check on global partition and queue partition sizes
	var offset, totalPartitions, totalQueueItems int64
	chunkSize := int64(1000)

	r := q.primaryQueueShard.RedisClient.unshardedRc
	// iterate through all the partitions in the global partitions in chunks
	wg := sync.WaitGroup{}
	for {
		// grab the global partition by chunks
		cmd := r.B().Zrange().
			Key(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex()).
			Min("-inf").
			Max("+inf").
			Byscore().
			Limit(offset, chunkSize).
			Build()

		pkeys, err := r.Do(ctx, cmd).AsStrSlice()
		if err != nil {
			return 0, 0, fmt.Errorf("error retrieving partitions for instrumentation: %w", err)
		}

		for _, pk := range pkeys {
			wg.Add(1)

			// check each partition concurrently
			go func(ctx context.Context, pkey string) {
				defer wg.Done()

				log := l.With("partitionKey", pkey)

				// If this is not a fully-qualified key, assume that this is an old (system) partition queue
				queueKey := pkey
				if !isKeyConcurrencyPointerItem(pkey) {
					queueKey = q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeDefault, pkey, "")
				}

				cntCmd := r.B().Zcount().Key(queueKey).Min("-inf").Max("+inf").Build()
				itemCount, err := q.primaryQueueShard.RedisClient.unshardedRc.Do(ctx, cntCmd).AsInt64()
				if err != nil {
					log.Warn("error checking partition count", "pkey", pkey, "context", "instrumentation")
					return
				}

				// Call the callback if provided
				if opts.OnPartitionProcessed != nil {
					opts.OnPartitionProcessed(ctx, pkey, queueKey, itemCount, q.primaryQueueShard)
				}

				atomic.AddInt64(&totalQueueItems, itemCount)
				atomic.AddInt64(&totalPartitions, 1)
				if err := q.tenantInstrumentor(ctx, pk); err != nil {
					log.ReportError(err, "error running tenant instrumentor")
				}
			}(ctx, pk)

		}
		// end of pagination, exit
		if len(pkeys) < int(chunkSize) {
			break
		}

		offset += chunkSize
	}

	wg.Wait()

	// Call the completion callback if provided
	if opts.OnIterationComplete != nil {
		opts.OnIterationComplete(ctx, atomic.LoadInt64(&totalPartitions), atomic.LoadInt64(&totalQueueItems), q.primaryQueueShard)
	}

	return atomic.LoadInt64(&totalPartitions), atomic.LoadInt64(&totalQueueItems), nil
}

func (q *queue) ItemByID(ctx context.Context, jobID string, opts ...QueueOpOpt) (*osqueue.QueueItem, error) {
	opt := newQueueOpOptWithOpts(opts...)

	shard := q.primaryQueueShard
	if opt.shard != nil {
		shard = *opt.shard
	}

	rc := shard.RedisClient.Client()
	kg := shard.RedisClient.kg

	cmd := rc.B().Hget().Key(kg.QueueItem()).Field(jobID).Build()
	byt, err := rc.Do(ctx, cmd).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, ErrQueueItemNotFound
		}
	}

	var item osqueue.QueueItem
	if err := json.Unmarshal(byt, &item); err != nil {
		return nil, fmt.Errorf("error unmarshalling queue item: %w", err)
	}

	return &item, nil
}

func (q *queue) Shard(ctx context.Context, shardName string) (QueueShard, bool) {
	shard, ok := q.queueShardClients[shardName]
	return shard, ok
}

func (q *queue) ItemsByRunID(ctx context.Context, runID ulid.ULID, opts ...QueueOpOpt) ([]*osqueue.QueueItem, error) {
	opt := newQueueOpOptWithOpts(opts...)

	shard := q.primaryQueueShard
	if opt.shard != nil {
		shard = *opt.shard
	}

	rc := shard.RedisClient.Client()
	kg := shard.RedisClient.KeyGenerator()

	itemIDs, err := rc.Do(
		ctx,
		rc.B().
			Zscan().
			Key(kg.RunIndex(runID)).
			Cursor(uint64(0)).
			Count(consts.DefaultMaxStepLimit). // use the default step limit for this
			Build(),
	).AsScanEntry()
	if err != nil {
		return nil, fmt.Errorf("error retrieving queue item IDs: %w", err)
	}

	if len(itemIDs.Elements) == 0 {
		return []*osqueue.QueueItem{}, nil
	}

	items, err := rc.Do(
		ctx,
		rc.B().
			Hmget().
			Key(kg.QueueItem()).
			Field(itemIDs.Elements...).
			Build(),
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error retrieving queue items: %w", err)
	}

	var res []*osqueue.QueueItem
	for _, str := range items {
		var qi osqueue.QueueItem

		if err := json.Unmarshal([]byte(str), &qi); err == nil {
			res = append(res, &qi)
		}
	}

	return res, nil
}
