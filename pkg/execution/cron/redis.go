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

type RedisCronManagerOpt func(c *redisCronManagerOpt)

type redisCronManagerOpt struct {
	// jitter range provides a min and max duration for the jitter to apply to schedules
	// which will move the scheduling time a little earlier than the actual cron schedule.
	//
	// we do this so we can make the actual run start as close as possible to the actual cron schedule.
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
	q redis_state.QueueManager,
	log logger.Logger,
	opts ...RedisCronManagerOpt,
) CronManager {
	opt := redisCronManagerOpt{
		jitterMin: 0 * time.Millisecond,
		jitterMax: 20 * time.Millisecond,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	manager := &redisCronManager{
		q:   q,
		log: log,
		opt: opt,
	}

	return manager
}

type redisCronManager struct {
	q redis_state.QueueManager

	log logger.Logger
	opt redisCronManagerOpt
}

func (c *redisCronManager) CronProcessJobID(schedule time.Time, expr string, fnID uuid.UUID, fnVersion int) string {
	return fmt.Sprintf("{%s}:{%s}:{%s}:{%d}:cron:schedule", schedule, expr, fnID, fnVersion)
}

// Sync enqueues a system job of kind "cron-sync" to the system queue.
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
		l.Trace("cron-sync enqueued", "jobID", jobID)
		return nil
	default:
		l.ReportError(err, "error enqueueing cron sync job")
		return fmt.Errorf("error enqueueing cron sync job: %w", err)
	}
}

// NextScheduledItemIDForFunction returns the expected identifier (ID, JobID) information for the next scheduled system "cron" job.
// Note that this reconstructs the identifier based on exact logic used by the system job handler and does not verify that the item for the next schedule is actually scheduled.
func (c *redisCronManager) NextScheduledItemIDForFunction(ctx context.Context, functionID uuid.UUID, expr string, fnVersion int) (*CronItem, error) {
	// Get current time as the starting point
	from := time.Now()

	// Get the next schedule time based on the cron expression
	next, err := Next(expr, from)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron expression %q: %w", expr, err)
	}

	// Generate the job ID for this scheduled item
	jobID := queue.HashID(ctx, c.CronProcessJobID(next, expr, functionID, fnVersion))

	// Construct the cron item with ID and JobID populated
	item := &CronItem{
		ID:              ulid.MustNew(uint64(next.UnixMilli()), rand.Reader),
		FunctionID:      functionID,
		FunctionVersion: fnVersion,
		Expression:      expr,
		JobID:           jobID,
	}

	return item, nil
}

// ScheduleNext schedules the next "cron" job w.r.t the CronItem provided.
// While CronItem.ID and CronItem.JobID encode the _actual_ timestamp of the next schedule, the CronItem is scheduled for a few milliseconds (jitterOpts) before the schedule to allow for some processing time to create the function run.
func (c *redisCronManager) ScheduleNext(ctx context.Context, ci CronItem) (*CronItem, error) {
	l := c.log.With("action", "redisCronManager.ScheduleNext", "fnID", ci.FunctionID, "fnVersion", ci.FunctionVersion, "cronExpr", ci.Expression)

	from := ci.ID.Timestamp()

	// Parse the cron expression and get the next execution time
	next, err := Next(ci.Expression, from)
	if err != nil {
		// TODO decide on what to do with this because it likely can't be fixed on retries
		// This also should never happen as long as this expression is set from an actual function config, which should be validated on function registration.
		return nil, fmt.Errorf("failed to parse cron expression %q: %w", ci.Expression, err)
	}

	// We want only one cron loop to exist for a {FnID, FnVersion, CronExpr} combination. This jobID helps achieve that idempotency.
	jobID := queue.HashID(ctx, c.CronProcessJobID(next, ci.Expression, ci.FunctionID, ci.FunctionVersion))

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
	}, enqueueAt, queue.EnqueueOpts{PassthroughJobId: true})

	l = l.With("next", next, "JobID", jobID, "enqueueAt", enqueueAt, "jitter", jitter)

	switch err {
	case nil:
		l.Trace("ScheduleNext success")
	case redis_state.ErrQueueItemExists, redis_state.ErrQueueItemSingletonExists:
		l.Trace("ScheduleNext already exists")
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
