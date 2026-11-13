package devserver

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/jonboulle/clockwork"
	"go.uber.org/fx"
)

// capacityManager is the capacity manager of the Constraint API.  the lease
// scavenger uses it to release expired leases.
type capacityManager interface {
	constraintapi.CapacityManager
	constraintapi.CapacityLeaseScavenger
}

type constraintAPI struct {
	fx.Out

	Semaphores constraintapi.SemaphoreManager
	Capacity   capacityManager
}

func newConstraintAPI(clients *redisClients) (constraintAPI, error) {
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clockwork.NewRealClock()),
		constraintapi.WithShardName("default"),
		constraintapi.WithClient(clients.unsharded),
		constraintapi.WithEnableHighCardinalityInstrumentation(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
			return false
		}),
	)
	if err != nil {
		return constraintAPI{}, fmt.Errorf("could not create constraint API: %w", err)
	}
	return constraintAPI{
		Semaphores: constraintapi.NewRedisSemaphoreManager(clients.unsharded),
		Capacity:   cm,
	}, nil
}

// queueProcessor is the queue of the server.  the executor consumes it, and
// the APIs read from it.
type queueProcessor interface {
	queue.Queue
	queue.QueueProcessor
}

// retryPolicy is the backoff between function retries.  backoff is nil
// unless StartOpts.RetryInterval is set, and the queue then uses its default
// backoff.
type retryPolicy struct {
	backoff backoff.BackoffFunc
}

type queueParams struct {
	fx.In

	Context     context.Context
	Opts        StartOpts
	Experiments experiments
	Log         logger.Logger
	CQRS        cqrs.Manager
	Unsharded   *redis_state.UnshardedClient
	RunService  sv2.RunService
	Capacity    capacityManager
}

type queueResult struct {
	fx.Out

	Shard     redis_state.RedisQueueShard
	Registry  queue.ShardRegistryController
	Queue     queueProcessor
	Retry     retryPolicy
	Functions *cqrsFunctionProvider
}

// newQueue creates the single queue shard of the server and the queue that
// runs on it.
func newQueue(p queueParams) (queueResult, error) {
	runMode := queue.QueueRunMode{
		Sequential:    true,
		Scavenger:     true,
		Partition:     true,
		Continuations: true,
	}
	if p.Experiments.keyQueues {
		runMode.ShadowPartition = true
		runMode.AccountShadowPartition = true
		runMode.AccountShadowPartitionWeight = 80
		runMode.ShadowContinuations = true
		runMode.ShadowContinuationSkipProbability = consts.QueueContinuationSkipProbability
		runMode.NormalizePartition = true
	}

	pauses := &functionPauses{}
	queueOpts := []queue.QueueOpt{
		queue.WithRunMode(runMode),
		queue.WithIdempotencyTTL(time.Hour),
		queue.WithNumWorkers(int32(p.Opts.QueueWorkers)),
		queue.WithPollTick(p.Opts.Tick),
		queue.WithShadowPollTick(2 * p.Opts.Tick),
		queue.WithBacklogNormalizePollTick(5 * p.Opts.Tick),

		queue.WithLogger(p.Log),

		// Key queues
		queue.WithNormalizeRefreshItemCustomConcurrencyKeys(NormalizeConcurrencyKeys(p.RunService, p.CQRS)),
		queue.WithRefreshItemThrottle(NormalizeThrottle(p.RunService, p.CQRS)),
		queue.WithPartitionConstraintConfigGetter(PartitionConstraintConfigGetter(p.CQRS)),

		queue.WithAllowKeyQueues(func(ctx context.Context, acctID, envID, functionID uuid.UUID) bool {
			return p.Experiments.keyQueues
		}),
		queue.WithBacklogRefillLimit(10),
		queue.WithPartitionPausedGetter(pauses.partitionPaused),
		queue.WithConditionalTracer(itrace.NewConditionalTracer(itrace.QueueTracer(), itrace.AlwaysTrace)),
		queue.WithCapacityManager(p.Capacity),
		queue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		queue.WithCapacityReleaseWakeEnabled(func(context.Context, uuid.UUID) bool {
			return true
		}),
	}

	var retry retryPolicy
	if p.Opts.RetryInterval > 0 {
		retry.backoff = backoff.GetLinearBackoffFunc(time.Duration(p.Opts.RetryInterval) * time.Second)
		queueOpts = append(queueOpts, queue.WithBackoffFunc(retry.backoff))
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, p.Unsharded.Queue(), queueOpts...)
	registry, err := queue.NewSingleShardRegistry(shard)
	if err != nil {
		return queueResult{}, fmt.Errorf("could not create shard registry: %w", err)
	}

	rq, err := queue.New(p.Context, "queue", registry, queueOpts...)
	if err != nil {
		return queueResult{}, fmt.Errorf("could not create queue: %w", err)
	}

	return queueResult{
		Shard:     shard,
		Registry:  registry,
		Queue:     rq,
		Retry:     retry,
		Functions: NewFunctionProvider(p.CQRS, pauses, rq),
	}, nil
}

type queueManagersParams struct {
	fx.In

	Context     context.Context
	Experiments experiments
	Log         logger.Logger
	Clients     *redisClients
	Sharded     *redis_state.ShardedClient
	Shard       redis_state.RedisQueueShard
	Registry    queue.ShardRegistryController
	Queue       queueProcessor
}

type queueManagers struct {
	fx.Out

	RateLimiter ratelimit.RateLimiter
	Batcher     batch.BatchManager
	Debouncer   debounce.Debouncer
	Croner      cron.CronManager
	Singleton   singleton.Singleton
}

// newQueueManagers creates the rate limiter, and the batch, debounce, cron,
// and singleton managers.  each one enqueues jobs on the queue.
func newQueueManagers(p queueManagersParams) (queueManagers, error) {
	// Create the batch manager. In production, a second BatchClient can be provided
	// to enable zero-downtime migration between Redis clusters via
	// batch.NewMigratingBatchManager.
	batchOpts := []batch.RedisBatchManagerOpt{batch.WithLogger(p.Log)}
	if p.Experiments.splitBatchPartitionByFunction {
		batchOpts = append(batchOpts, batch.WithSplitBatchPartitionByFunction(func(_ context.Context, _ uuid.UUID) bool {
			return true
		}))
	}

	debouncer, err := debounce.NewDebouncer(p.Registry, p.Shard.Name(), p.Queue)
	if err != nil {
		return queueManagers{}, fmt.Errorf("could not create debounce manager: %w", err)
	}

	return queueManagers{
		RateLimiter: ratelimit.New(p.Context, p.Clients.unsharded, "{ratelimit}:"),
		Batcher:     batch.NewRedisBatchManager(p.Sharded.Batch(), p.Queue, batchOpts...),
		Debouncer:   debouncer,
		Croner: cron.NewManager(p.Shard, p.Queue, p.Log, cron.WithSplitCronPartitionByWorkspace(func(_ context.Context, _ uuid.UUID) bool {
			return true
		})),
		Singleton: singleton.New(p.Context, p.Registry),
	}, nil
}
