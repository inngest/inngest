package cron

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
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

// TODO(kasinath) comments
func (c *redisCronManager) Sync(ctx context.Context, ci CronItem) error {
	l := c.log.With("action", "redisCronManager.Sync", "functionID", ci.FunctionID, "functionVersion", ci.FunctionVersion, "cronExpr", ci.Expression, "operation", ci.Op.String())

	switch ci.Op {
	case enums.CronOpProcess:
		l.Error("CronOpProcess is not meant for syncs, ignoring CronItem")
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

// TODO(kasinath) comments
func (c *redisCronManager) ScheduleNext(ctx context.Context, ci CronItem) (*CronItem, error) {
	l := c.log.With("action", "redisCronManager.ScheduleNext", "fnID", ci.FunctionID, "fnVersion", ci.FunctionVersion, "cronExpr", ci.Expression)

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

	jobID := c.c.KeyGenerator().CronProcessJobID(next, ci.Expression, ci.FunctionID, ci.FunctionVersion)
	// Add jitter to schedule execution slightly earlier
	// This ensures execution starts around the desired time
	jitter := generateJitter(c.opt.jitterMin, c.opt.jitterMax)
	enqueueAt := next.Add(-jitter)

	nextItem := CronItem{
		ID:              ulid.MustNew(uint64(next.UnixMilli()), rand.Reader),
		AccountID:       ci.AccountID,
		WorkspaceID:     ci.WorkspaceID,
		AppID:           ci.AppID,
		FunctionID:      ci.FunctionID,
		FunctionVersion: ci.FunctionVersion,
		Expression:      ci.Expression,
		JobID:           jobID,
		Op:              enums.CronOpProcess,
	}

	l = l.With("next_cron_item", nextItem)

	// enqueue new schedule
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
	}, enqueueAt, queue.EnqueueOpts{})
	switch err {
	case nil, redis_state.ErrQueueItemExists, redis_state.ErrQueueItemSingletonExists:
		// no-op
	default:
		l.ReportError(err, "error enqueueing cron for next schedule")
		return nil, fmt.Errorf("error enqueueing cron for next schedule: %w", err)
	}

	return &nextItem, nil
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
