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

const (
	// lead time for scheduling "cron" items.
	defaultJitterMin = 0 * time.Millisecond
	defaultJitterMax = 20 * time.Millisecond

	// interval and lead time for scheduling "cron-health-check" items
	// default: 20 seconds before the top of a minute
	defaultHealthCheckLeadTimeSeconds = 20
	defaultHealthCheckInterval        = time.Minute
)

type RedisCronManagerOpt func(c *redisCronManagerOpt)

type redisCronManagerOpt struct {
	// jitter range provides a min and max duration for the jitter to apply to schedules
	// which will move the scheduling time a little earlier than the actual cron schedule.
	//
	// we do this so we can make the actual run start as close as possible to the actual cron schedule.
	jitterMin time.Duration
	jitterMax time.Duration

	healthCheckLeadTimeSeconds int
	healthCheckInterval        time.Duration
}

func (opts *redisCronManagerOpt) validate() {
	if opts.healthCheckLeadTimeSeconds >= int(opts.healthCheckInterval.Seconds()) {
		l := logger.StdlibLogger(context.Background())

		opts.healthCheckLeadTimeSeconds = defaultHealthCheckLeadTimeSeconds

		l.Warn("Invalid redisCronManagerOpt, health lead time cannot be >= health check interval", "leadTime_seconds", opts.healthCheckLeadTimeSeconds, "interval", opts.healthCheckInterval)
		l.Info("Resetting health check lead time", "leadTime", opts.healthCheckLeadTimeSeconds)
	}
}

func WithJitterRange(min time.Duration, max time.Duration) RedisCronManagerOpt {
	return func(c *redisCronManagerOpt) {
		if min > max {
			logger.StdlibLogger(context.Background()).Warn("rejecting invalid jitter range in redisCronManagerOpt", "min", min, "max", max)
			return
		}

		c.jitterMin = min
		c.jitterMax = max
	}
}

func WithHealthCheckInterval(d time.Duration) RedisCronManagerOpt {
	return func(c *redisCronManagerOpt) {
		if d < time.Minute {
			logger.StdlibLogger(context.Background()).Warn("rejecting invalid health check interval in redisCronManagerOpt", "interval", d)
			return
		}
		c.healthCheckInterval = d
	}
}

func WithHealthCheckLeadTimeSeconds(leadTime int) RedisCronManagerOpt {
	return func(c *redisCronManagerOpt) {
		if leadTime < 0 {
			logger.StdlibLogger(context.Background()).Warn("rejecting invalid health check lead time in redisCronManagerOpt", "leadTime", leadTime)
			return
		}
		c.healthCheckLeadTimeSeconds = leadTime
	}
}

func NewRedisCronManager(
	q redis_state.QueueManager,
	log logger.Logger,
	opts ...RedisCronManagerOpt,
) CronManager {
	opt := redisCronManagerOpt{
		jitterMin:                  defaultJitterMin,
		jitterMax:                  defaultJitterMax,
		healthCheckLeadTimeSeconds: defaultHealthCheckLeadTimeSeconds,
		healthCheckInterval:        defaultHealthCheckInterval,
	}
	for _, apply := range opts {
		apply(&opt)
	}

	opt.validate()

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

func (c *redisCronManager) CronHealthCheckJobID(at time.Time) string {
	return fmt.Sprintf("{%s}:cron:health-check", at)
}

// Sync enqueues a system job of kind "cron-sync" to the system queue.
func (c *redisCronManager) Sync(ctx context.Context, ci CronItem) error {
	l := c.log.With("action", "redisCronManager.Sync", "functionID", ci.FunctionID, "functionVersion", ci.FunctionVersion, "cronExpr", ci.Expression, "operation", ci.Op.String())

	switch ci.Op {
	case enums.CronOpProcess, enums.CronHealthCheck:
		l.Error("CronOpProcess is not meant for syncs or health checks, ignoring CronItem")
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

func (c *redisCronManager) nextHealthCheckTime(now time.Time) time.Time {
	base := now.Truncate(c.opt.healthCheckInterval)
	next := base.Add(c.opt.healthCheckInterval).Add(time.Duration(-1*c.opt.healthCheckLeadTimeSeconds) * time.Second)
	if !next.After(now) {
		next = next.Add(c.opt.healthCheckInterval)
	}
	return next
}

// enqueues a cronItem{op:healthcheck} of {kind:cron-health-check} into system queue.
func (c *redisCronManager) EnqueueNextHealthCheck(ctx context.Context) error {

	now := time.Now()
	nextCheck := c.nextHealthCheckTime(now)

	maxAttempts := consts.MaxRetries + 1
	kind := queue.KindCronHealthCheck
	jobID := c.CronHealthCheckJobID(nextCheck)

	l := c.log.With("action", "redisCronManager.EnqueueNextHealthCheck", "now", now, "nextCheck", nextCheck)

	err := c.q.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		GroupID:     uuid.New().String(),
		Kind:        kind,
		MaxAttempts: &maxAttempts,
		Payload: CronItem{
			Op: enums.CronHealthCheck,
			ID: ulid.MustNew(uint64(nextCheck.UnixMilli()), rand.Reader),
		},
		QueueName: &kind,
	}, nextCheck, queue.EnqueueOpts{})

	switch err {
	case nil:
		l.Trace("cron-health-check enqueued")
		return nil
	case redis_state.ErrQueueItemExists, redis_state.ErrQueueItemSingletonExists:
		l.Trace("cron-health-check already exists")
		return nil
	default:
		l.ReportError(err, "error enqueueing cron health check job")
		return fmt.Errorf("error enqueueing cron health check job: %w", err)
	}
}

// HealthCheck checks if a "cron" queue item exists system queue for the next expected schedule time
func (c *redisCronManager) HealthCheck(ctx context.Context, functionID uuid.UUID, expr string, fnVersion int) (CronHealthCheckStatus, error) {
	from := time.Now()

	// Get the next schedule time based on the cron expression
	next, err := Next(expr, from)
	if err != nil {
		return CronHealthCheckStatus{}, fmt.Errorf("failed to get next schedule time for health check: %w", err)
	}

	// Generate the job ID for this scheduled item
	jobID := queue.HashID(ctx, c.CronProcessJobID(next, expr, functionID, fnVersion))

	// check if the jobID exists in the system queue.
	exists, err := c.q.ItemExists(ctx, jobID)
	if err != nil {
		return CronHealthCheckStatus{}, fmt.Errorf("failed to check if item exits for health check: %w", err)
	}
	return CronHealthCheckStatus{Next: next, JobID: jobID, Scheduled: exists}, nil
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

	// We use robfig cron library to find the next time this cron is supposed to run. If there is no time in the next 5 years this cron will run, robfig.cron.Next returns a zero time.
	// Since we don't allow specifying year as part of the cron expression, these expressions will never run. It is therefore safe to return without scheduling the next run.
	if next.IsZero() {
		l.Warn("next schedule is zero, returning")
		return nil, nil
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
