package cron

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

var (
	defaultScheduleForwardDur = 10 * time.Second
)

type RedisCronManagerOpt func(c *redisCronManagerOpt)

type redisCronManagerOpt struct {
	// jitter range provides a min and max duration for the jitter to apply to schedules
	// which will move the scheduling time a little earlier than the actual cron schedule.
	//
	// we do this so we can make the actual run start as close as possible to the actual cron schedule.
	jitterMin time.Duration
	jitterMax time.Duration

	// we need to move the time upwards on finding the next time due to how we use jitter
	// to push the time back a little to coordinate the run start timing.
	//
	// considering cron's minimum granularity is a minute, the seconds range should work fine. defaults to 10s
	scheduleForwardDur time.Duration
}

func WithJitterRange(min time.Duration, max time.Duration) RedisCronManagerOpt {
	return func(c *redisCronManagerOpt) {
		if min > max {
			return
		}

		c.jitterMin = min
		c.jitterMax = max
	}
}

func WithScheduleForwardDuration(dur time.Duration) RedisCronManagerOpt {
	return func(c *redisCronManagerOpt) {
		if dur > 0 {
			c.scheduleForwardDur = dur
		}
	}
}

func NewRedisCronManager(
	c *redis_state.CronClient,
	q redis_state.QueueManager,
	log logger.Logger,
	opts ...RedisCronManagerOpt,
) CronManager {
	opt := redisCronManagerOpt{
		jitterMin:          0 * time.Millisecond,
		jitterMax:          20 * time.Millisecond,
		scheduleForwardDur: defaultScheduleForwardDur,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	manager := &redisCronManager{
		c:   c,
		q:   q,
		log: log,
		opt: opt,
	}

	return manager
}

type redisCronManager struct {
	c *redis_state.CronClient
	q redis_state.QueueManager

	log logger.Logger
	opt redisCronManagerOpt
}

func (c *redisCronManager) Sync(ctx context.Context, ci CronItem) error {
	l := c.log.With("action", "redisCronManager.Sync")

	switch ci.Op {
	case enums.CronOpProcess:
		// OpProcess is not meant for syncs
		return nil
	}

	maxAttempts := consts.MaxRetries + 1
	kind := queue.KindCronSync
	at := ulid.Time(ci.ID.Time())
	jobID := ci.SyncID()

	err := c.q.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		GroupID:     uuid.New().String(),
		WorkspaceID: ci.WorkspaceID,
		Kind:        kind,
		Identifier: state.Identifier{
			AccountID:       ci.AccountID,
			WorkspaceID:     ci.WorkspaceID,
			AppID:           ci.AppID,
			WorkflowID:      ci.FunctionID,
			WorkflowVersion: ci.FunctionVersion,
		},
		MaxAttempts: &maxAttempts,
		Payload:     ci,
		QueueName:   &kind,
	}, at, queue.EnqueueOpts{})
	switch err {
	case nil, redis_state.ErrQueueItemExists, redis_state.ErrQueueItemSingletonExists:
		return nil
	default:
		l.ReportError(err, "error enqueueing cron sync job")
		return fmt.Errorf("error enqueueing cron sync job: %w", err)
	}
}

func (c *redisCronManager) ScheduleNext(ctx context.Context, ci CronItem) (*CronItem, error) {
	l := c.log.With("action", "redisCronManager.ScheduleNext", "cron_item", ci)

	from := ci.ID.Timestamp()
	switch ci.Op {
	case enums.CronOpProcess:
		from = from.Add(c.opt.scheduleForwardDur)
	}

	// Parse the cron expression and get the next execution time
	next, err := Next(ci.Expression, from)
	if err != nil {
		// TODO decide on what to do with this because it likely can't be fixed on retries
		return nil, fmt.Errorf("failed to parse cron expression %q: %w", ci.Expression, err)
	}

	// Add jitter to schedule execution slightly earlier
	// This ensures execution starts around the desired time
	jitter := generateJitter(c.opt.jitterMin, c.opt.jitterMax)
	next = next.Add(-jitter)

	nextItem := CronItem{
		ID:              ulid.MustNew(uint64(next.UnixMilli()), rand.Reader),
		AccountID:       ci.AccountID,
		WorkspaceID:     ci.WorkspaceID,
		AppID:           ci.AppID,
		FunctionID:      ci.FunctionID,
		FunctionVersion: ci.FunctionVersion,
		Expression:      ci.Expression,
		// NOTE this simulates how queue item hashes its ID, not a nice implementation detail leak
		// meaning debug tooling will break if that changes somehow even though its unlikely
		JobID: queue.HashID(ctx, ci.ProcessID()),
		Op:    enums.CronOpProcess,
	}

	l = l.With("next_cron_item", nextItem)

	// enqueue new schedule
	jobID := ci.ProcessID()
	kind := queue.KindCron
	maxAttempts := consts.MaxRetries + 1

	err = c.q.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		GroupID:     uuid.New().String(),
		WorkspaceID: ci.WorkspaceID,
		Identifier: state.Identifier{
			AccountID:       ci.AccountID,
			WorkspaceID:     ci.WorkspaceID,
			AppID:           ci.AppID,
			WorkflowID:      ci.FunctionID,
			WorkflowVersion: ci.FunctionVersion,
		},
		Kind:        kind,
		QueueName:   &kind,
		MaxAttempts: &maxAttempts,
		Payload:     nextItem,
	}, next, queue.EnqueueOpts{})
	switch err {
	case nil, redis_state.ErrQueueItemExists, redis_state.ErrQueueItemSingletonExists:
		// no-op
	default:
		l.ReportError(err, "error enqueueing cron for next schedule")
		return nil, fmt.Errorf("error enqueueing cron for next schedule: %w", err)
	}

	return &nextItem, nil
}

func (c *redisCronManager) CanRun(ctx context.Context, ci CronItem) (bool, error) {
	l := c.log.With(
		"action", "redisCronManager.CanRun",
		"cron_item", ci,
	)

	nextItem, err := c.NextScheduledItemForFunction(ctx, ci.FunctionID)
	switch err {
	case nil:
		// no-opt
	case errNextScheduleNotFound:
		// this likely means that the queue item was already dequeued
		// if there are no mapping, we also can't tell what the next schedule of the function is,
		// and we default to nothing if not available.
		return false, nil
	default:
		return false, err
	}

	// it's the same item, it can proceed
	if ci.Equal(*nextItem) {
		return true, nil
	}

	switch nextItem.Op {
	case enums.CronOpProcess:
		// no-op: proceed
	default:
		// wrong types, not used for processing purposes
		return false, nil
	}

	// we need to do some checks if the cron items are different
	//
	// NOTE
	// the checks are for cases where there could be race conditions (shouldn't happen but never know)
	// where somehow the next item scheduled item is an updated version of the cron workload, but this
	// outdated one somehow didn't get dequeued in time.
	// cases where this can happen are on schedule updates
	//
	// generally speaking we shouldn't reach this section of the code, but if we do, make sure to make
	// it known.
	l.Warn("running checks on cron item to make sure it can be ran", "next_item", nextItem)
	metrics.IncrCronProcessingDiffCheck(ctx, metrics.CounterOpt{PkgName: pkgName})

	// this means the item in the mapping has an updated version of the cron, so this one should be discarded
	if nextItem.FunctionVersion > ci.FunctionVersion {
		return false, nil
	}

	// all checks has passed so this means it's okay to run
	return true, nil
}

func (c *redisCronManager) UpdateSchedule(ctx context.Context, ci CronItem) error {
	l := c.log.With("action", "redisCronManager.UpdateSchedule", "cron_item", ci)

	switch ci.Op {
	case
		// these are net new, so there should be no existing cron workloads running
		enums.CronOpNew, enums.CronOpUnpause,
		// pure processing
		enums.CronOpProcess:

		next, err := c.ScheduleNext(ctx, ci)
		if err != nil {
			return fmt.Errorf("error scheduling next item for Op: %s: %w", ci.Op, err)
		}
		l.Trace("scheduled next cron job", "next", next, "op", ci.Op, "job_id", next.JobID)

		// - update mapping
		return c.setFunctionScheduleMap(ctx, *next)

	case enums.CronOpUpdate:
		// NOTE
		// This logic will have race conditions where retrieving of the item and dequeue happens
		// separately.
		// For the sake of simplicity, we will not be dealing with it in this logic, and it's up
		// to the caller to handle things accordingly with the provided interfaces available (e.g. CanRun)
		// and additional context on the caller side that's out of scope for this module
		existing, err := c.NextScheduledItemForFunction(ctx, ci.FunctionID)
		switch err {
		case nil, errNextScheduleNotFound:
			// no-op
		default:
			return fmt.Errorf("error finding existing cron item for function: %w", err)
		}

		// no action should be taken if the existing one has a higher function version
		// this means there's a race condition and we don't want to update the schedule to
		// an older version
		if existing != nil {
			if existing.FunctionVersion > ci.FunctionVersion {
				return nil
			}

			// dequeue the item
			if err := c.q.DequeueByJobID(ctx, existing.JobID); err != nil {
				if !errors.Is(err, redis_state.ErrQueueItemNotFound) {
					return fmt.Errorf("error dequeueing item: %w", err)
				}
			}
		}

		next, err := c.ScheduleNext(ctx, ci)
		if err != nil {
			return fmt.Errorf("error scheduling next item for Op: %s: %w", ci.Op, err)
		}
		l.Trace("scheduled next cron job after update", "next", next, "op", ci.Op, "job_id", next.JobID)

		// - update mapping
		return c.setFunctionScheduleMap(ctx, *next)

	case enums.CronOpArchive, enums.CronOpPause:
		// NOTE
		// This logic will have race conditions where retrieving of the item and dequeue happens
		// separately.
		// For the sake of simplicity, we will not be dealing with it in this logic, and it's up
		// to the caller to handle things accordingly with the provided interfaces available (e.g. CanRun)
		// and additional context on the caller side that's out of scope for this module
		existing, err := c.NextScheduledItemForFunction(ctx, ci.FunctionID)
		switch err {
		case nil, errNextScheduleNotFound:
			// no-op
		default:
			return fmt.Errorf("error finding existing cron item for function: %w", err)
		}

		// dequeue the item if it exists
		if existing != nil {
			if err := c.q.DequeueByJobID(ctx, existing.JobID); err != nil {
				if !errors.Is(err, redis_state.ErrQueueItemNotFound) {
					return fmt.Errorf("error dequeueing item: %w", err)
				}
			}

			l.Trace("deleted schedule")
		}

		return c.removeScheduleMap(ctx, ci.FunctionID)

	case enums.CronInit:
		// Check if there's already an item scheduled
		existing, err := c.NextScheduledItemForFunction(ctx, ci.FunctionID)
		switch err {
		case nil, errNextScheduleNotFound:
			// no-op
		default:
			return fmt.Errorf("error finding existing cron item for function: %w", err)
		}

		// if something already exist don't do anything
		if existing != nil {
			return nil
		}

		next, err := c.ScheduleNext(ctx, ci)
		if err != nil {
			return fmt.Errorf("error scheduling next item for Op: %s: %w", ci.Op, err)
		}
		l.Trace("initialized cron job", "next", next, "op", ci.Op, "job_id", next.JobID)

		// - update mapping
		return c.setFunctionScheduleMap(ctx, *next)

	default:
		return fmt.Errorf("unknown cron operation provided: %s", ci.Op)
	}
}

func (c *redisCronManager) NextScheduledItemForFunction(ctx context.Context, fnID uuid.UUID) (*CronItem, error) {
	rc := c.c.Client()
	kg := c.c.KeyGenerator()

	cmd := rc.B().
		Hget().
		Key(kg.Schedule()).
		Field(fnID.String()).
		Build()

	byt, err := rc.Do(ctx, cmd).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, errNextScheduleNotFound
		}
		return nil, fmt.Errorf("error retrieving cron item: %w", err)
	}

	var res CronItem
	if err := json.Unmarshal(byt, &res); err != nil {
		return nil, fmt.Errorf("error unmarshalling cron item for function: %w", err)
	}

	return &res, nil
}

func (c *redisCronManager) setFunctionScheduleMap(ctx context.Context, ci CronItem) error {
	byt, err := json.Marshal(ci)
	if err != nil {
		return fmt.Errorf("error marshalling cron item: %w", err)
	}

	rc := c.c.Client()
	kg := c.c.KeyGenerator()

	cmd := rc.B().
		Hset().
		Key(kg.Schedule()).
		FieldValue().
		FieldValue(ci.FunctionID.String(), string(byt)).
		Build()

	_, err = rc.Do(ctx, cmd).AsInt64()
	if err != nil {
		return fmt.Errorf("error adding job to schedule map: %w", err)
	}

	return nil
}

func (c *redisCronManager) removeScheduleMap(ctx context.Context, fnID uuid.UUID) error {
	rc := c.c.Client()
	kg := c.c.KeyGenerator()

	err := rc.Do(
		ctx,
		rc.B().
			Hdel().
			Key(kg.Schedule()).
			Field(fnID.String()).
			Build(),
	).Error()
	if err != nil {
		return fmt.Errorf("error clearing out schedule map: %w", err)
	}

	return nil
}

// generateJitter generates a random jitter duration between min and max (inclusive)
func generateJitter(min, max time.Duration) time.Duration {
	if min > max {
		return 0
	}

	rangeNs := int64(max - min + time.Nanosecond)
	randomBytes := make([]byte, 8)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return 0
	}

	// Convert bytes to uint64 and get value in range
	randomValue := uint64(randomBytes[0])<<56 | uint64(randomBytes[1])<<48 | uint64(randomBytes[2])<<40 | uint64(randomBytes[3])<<32 |
		uint64(randomBytes[4])<<24 | uint64(randomBytes[5])<<16 | uint64(randomBytes[6])<<8 | uint64(randomBytes[7])
	jitterNs := int64(randomValue%uint64(rangeNs)) + int64(min)

	return time.Duration(jitterNs)
}
