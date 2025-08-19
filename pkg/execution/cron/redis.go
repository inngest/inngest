package cron

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

type RedisCronManagerOpt func(c *redisCronManagerOpt)

type redisCronManagerOpt struct {
	jitterMin time.Duration
	jitterMax time.Duration
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

func NewRedisCronManager(
	c *redis_state.CronClient,
	q redis_state.QueueManager,
	log logger.Logger,
	opts ...RedisCronManagerOpt,
) CronManager {
	opt := redisCronManagerOpt{
		jitterMin: 10 * time.Millisecond,
		jitterMax: 100 * time.Millisecond,
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

func (c *redisCronManager) Next(ctx context.Context, ci CronItem) (time.Time, error) {
	// Parse the cron expression and get the next execution time
	schedule, err := Parse(ci.Expression)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron expression %q: %w", ci.Expression, err)
	}
	next := schedule.Next(ci.ID.Timestamp())

	// Add jitter to schedule execution slightly earlier
	// This ensures execution starts around the desired time
	jitter := generateJitter(c.opt.jitterMin, c.opt.jitterMax)
	next = next.Add(-jitter)

	return next, nil
}

func (c *redisCronManager) UpdateSchedule(ctx context.Context, ci CronItem) error {
	l := c.log.With("action", "redisCronManager.UpdateSchedule", "cron_item", ci)

	switch ci.Op {
	case CronOpNew, CronOpUnpause:
		jobID := ci.ID.String()
		queueName := queue.KindCron
		maxAttempts := consts.MaxRetries + 1

		next, err := c.Next(ctx, ci)
		if err != nil {
			return queue.NeverRetryError(err)
		}

		// enqueue new schedule
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
			Kind:        queue.KindCron,
			QueueName:   &queueName,
			MaxAttempts: &maxAttempts,
			Payload: CronItem{
				ID:              ulid.MustNew(uint64(next.UnixMilli()), rand.Reader),
				AccountID:       ci.AccountID,
				WorkspaceID:     ci.WorkspaceID,
				AppID:           ci.AppID,
				FunctionID:      ci.FunctionID,
				FunctionVersion: ci.FunctionVersion,
				Expression:      ci.Expression,
				Op:              CronOpProcess,
			},
		}, next, queue.EnqueueOpts{})
		switch err {
		case nil, redis_state.ErrQueueItemExists, redis_state.ErrQueueItemSingletonExists:
			// no-op
		default:
			l.ReportError(err, "error enqueueing cron for next schedule")
			return fmt.Errorf("error enqueueing cron for next schedule: %w", err)
		}

		// TODO
		// - update mapping
		return c.setFunctionScheduleMap(ctx, ci, jobID)

	case CronOpUpdate:
		// TODO
		// - delete and dequeue existing queue item
		// - enqueue new schedule
		// - update mapping

	case CronOpPause:
		// TODO
		// - delete and dequeue existing queue item

	default:
		return fmt.Errorf("unknow cron operation: %d", ci.Op)
	}

	return fmt.Errorf("not implemented")
}

func (c *redisCronManager) setFunctionScheduleMap(ctx context.Context, ci CronItem, jobID string) error {
	// compute the hashID from provided jobID
	//
	// NOTE ideally, we should just be able to reference the newly enqueued item but Enqueue doesn't return any objects
	hashedID := queue.HashID(ctx, jobID)

	rc := c.c.Client()
	kg := c.c.KeyGenerator()

	cmd := rc.B().
		Hset().
		Key(kg.Schedule()).
		FieldValue().
		FieldValue(ci.FunctionID.String(), hashedID).
		Build()

	added, err := rc.Do(ctx, cmd).AsInt64()
	if err != nil {
		return fmt.Errorf("error adding job to schedule map: %w", err)
	}
	if added != 1 {
		return fmt.Errorf("expected 1 change made to schedule map, actual: %d", added)
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
