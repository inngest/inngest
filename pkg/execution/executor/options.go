package executor

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/loops"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/inngest/inngest/pkg/logger"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/jonboulle/clockwork"
)

// ExecutorOpt modifies the built-in executor on creation.
type ExecutorOpt func(m execution.Executor) error

// set returns an option that applies fn to the concrete executor.
// options that can fail or that need the execution.Executor interface
// keep their own bodies.
func set(fn func(e *executor)) ExecutorOpt {
	return func(e execution.Executor) error {
		fn(e.(*executor))
		return nil
	}
}

func WithHTTPClient(c exechttp.RequestExecutor) ExecutorOpt {
	return set(func(e *executor) { e.httpClient = c })
}

func WithCancellationChecker(c cancellation.Checker) ExecutorOpt {
	return set(func(e *executor) { e.cancellationChecker = c })
}

// WithCancellationCheckDeadline bounds how long checkCancellation waits
// synchronously for the cancellation lookup before proceeding. A zero or
// negative value keeps the default of 100ms.
func WithCancellationCheckDeadline(d time.Duration) ExecutorOpt {
	return set(func(e *executor) { e.cancellationCheckDeadline = d })
}

// WithStateManager sets which state manager to use when creating an executor.
func WithStateManager(sm sv2.RunService) ExecutorOpt {
	return set(func(e *executor) { e.smv2 = sm })
}

// WithLoopStore sets the store that holds loop instances.  scheduling a
// loop function without it fails with ErrNoLoopStore.
func WithLoopStore(s loops.Store) ExecutorOpt {
	return set(func(e *executor) { e.loops = s })
}

// WithQueue sets which state manager to use when creating an executor.
func WithQueue(q queue.Queue) ExecutorOpt {
	return set(func(e *executor) { e.queue = q })
}

// WithPauseManager sets which pause manager to use when creating an executor.
func WithPauseManager(pm pauses.Manager) ExecutorOpt {
	return set(func(e *executor) { e.pm = pm })
}

// WithExpressionAggregator sets the expression aggregator singleton to use
// for matching events using our aggregate evaluator.
func WithExpressionAggregator(agg expragg.Aggregator) ExecutorOpt {
	return set(func(e *executor) { e.exprAggregator = agg })
}

func WithFunctionLoader(l state.FunctionLoader) ExecutorOpt {
	return set(func(e *executor) { e.fl = l })
}

func WithLogger(l logger.Logger) ExecutorOpt {
	return set(func(e *executor) { e.log = l })
}

func WithFinalizer(f execution.FinalizePublisher) ExecutorOpt {
	return set(func(e *executor) { e.SetFinalizer(f) })
}

func WithInvokeFailHandler(f execution.InvokeFailHandler) ExecutorOpt {
	return set(func(e *executor) { e.invokeFailHandler = f })
}

func WithInvokeEventHandler(f execution.HandleInvokeEvent) ExecutorOpt {
	return set(func(e *executor) { e.handleInvokeEvent = f })
}

func WithLifecycleListeners(l ...execution.LifecycleListener) ExecutorOpt {
	return func(e execution.Executor) error {
		for _, item := range l {
			e.AddLifecycleListener(item)
		}
		return nil
	}
}

// WithSyncLifecycleListeners registers listeners invoked synchronously
// (inline, not via a spawned goroutine) alongside the async
// execution.LifecycleListener dispatch. This is the *only* way to register
// them on an executor: there is deliberately no AddSyncLifecycleListener
// counterpart to AddLifecycleListener (see execution.SyncLifecycleListener's
// doc comment on static, construction-time registration). It asserts to the
// concrete *executor type, matching the pattern used by options like
// WithHTTPClient above.
func WithSyncLifecycleListeners(l ...execution.SyncLifecycleListener) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).syncLifecycles = append(e.(*executor).syncLifecycles, l...)
		return nil
	}
}

func WithEventLifecycleListeners(l ...execution.EventLifecycleListener) ExecutorOpt {
	return func(e execution.Executor) error {
		for _, item := range l {
			e.AddEventLifecycleListener(item)
		}
		return nil
	}
}

func WithStepLimits(limit func(id sv2.ID) int) ExecutorOpt {
	return set(func(e *executor) { e.steplimit = limit })
}

func WithStateSizeLimits(limit func(id sv2.ID) int) ExecutorOpt {
	return set(func(e *executor) { e.stateSizeLimit = limit })
}

func WithRateLimiter(rl ratelimit.RateLimiter) ExecutorOpt {
	return set(func(e *executor) { e.rateLimiter = rl })
}

func WithDebouncer(d debounce.Debouncer) ExecutorOpt {
	return set(func(e *executor) { e.debouncer = d })
}

func WithSingletonManager(sn singleton.Singleton) ExecutorOpt {
	return set(func(e *executor) { e.singletonMgr = sn })
}

func WithBatcher(b batch.BatchManager) ExecutorOpt {
	return set(func(e *executor) { e.batcher = b })
}

func WithCapacityManager(cm constraintapi.CapacityManager) ExecutorOpt {
	return set(func(e *executor) { e.capacityManager = cm })
}

// WithSemaphoreManager assigns a semaphore manager to the executor, used to release
// from semaphores when functions end (for fn concurrency)
func WithSemaphoreManager(sm constraintapi.SemaphoreManager) ExecutorOpt {
	return set(func(e *executor) { e.semaphoreManager = sm })
}

func WithUseConstraintAPI(uca constraintapi.UseConstraintAPIFn) ExecutorOpt {
	return set(func(e *executor) { e.useConstraintAPI = uca })
}

func WithEnableBatchingInstrumentation(ebi func(ctx context.Context, accountID, envID uuid.UUID) (enable bool)) ExecutorOpt {
	return set(func(e *executor) { e.enableBatchingInstrumentation = ebi })
}

// WithEvaluatorFactory allows customizing of the expression evaluator factory function.
func WithEvaluatorFactory(f func(ctx context.Context, expr string) (expressions.Evaluator, error)) ExecutorOpt {
	return set(func(e *executor) { e.evalFactory = f })
}

func WithSigningKeyLoader(f func(ctx context.Context, envID uuid.UUID) ([]byte, error)) ExecutorOpt {
	return set(func(e *executor) { e.signingKeyLoader = f })
}

// WithDriverV1 specifies the drivers available to use when executing steps
// of a function.
//
// When invoking a step in a function, we find the registered driver with the step's URI
// and use that driver to execute the step.
func WithDriverV1(drivers ...driver.DriverV1) ExecutorOpt {
	return func(exec execution.Executor) error {
		e := exec.(*executor)
		for _, d := range drivers {
			if _, ok := e.driverv1[d.Name()]; ok {
				return ErrRuntimeRegistered
			}
			e.driverv1[d.Name()] = d

		}
		return nil
	}
}

func WithDriverV2(drivers ...driver.DriverV2) ExecutorOpt {
	return func(exec execution.Executor) error {
		e := exec.(*executor)
		for _, d := range drivers {
			if _, ok := e.driverv2[d.Name()]; ok {
				return ErrRuntimeRegistered
			}
			e.driverv2[d.Name()] = d

		}
		return nil
	}
}

// WithShardRegistry wires the executor up to the queue's shard registry.
// Replaces the old WithAssignedQueueShard + WithShardSelector pair: the
// registry exposes both the leased primary (Primary) and per-account
// resolution (Resolve) so the executor doesn't need to track them
// independently.
func WithShardRegistry(shards queue.ShardRegistry) ExecutorOpt {
	return set(func(e *executor) { e.shards = shards })
}

func WithTraceReader(m cqrs.TraceReader) ExecutorOpt {
	return set(func(e *executor) { e.traceReader = m })
}

func WithTracerProvider(t tracing.TracerProvider) ExecutorOpt {
	return set(func(e *executor) { e.tracerProvider = t })
}

// WithRealtimePublisher configures a new publisher in the executor.  This publishes
// directly to the backing implementaiton.
func WithRealtimePublisher(b realtime.Publisher) ExecutorOpt {
	return set(func(e *executor) { e.rtpub = b })
}

// WithRealtimeAPIPublisher adds JWT configuration which allows publishing of data to the
// realtime API, without connecting to the backing realtime service directly.
func WithRealtimeConfig(config ExecutorRealtimeConfig) ExecutorOpt {
	return set(func(e *executor) { e.rtconfig = config })
}

func WithClock(clock clockwork.Clock) ExecutorOpt {
	return set(func(e *executor) { e.clock = clock })
}

type ExecutorRealtimeConfig struct {
	Secret     []byte
	PublishURL string
}

// AllowStepMetadata is kept only so existing callers (e.g. the monorepo)
// constructing a WithAllowStepMetadata option still compile.
//
// Deprecated: step metadata is now always enabled; this type and
// WithAllowStepMetadata no longer have any effect and will be removed.
type AllowStepMetadata func(ctx context.Context, acctID uuid.UUID) bool

// WithAllowStepMetadata is a no-op.
//
// Deprecated: step metadata is now always enabled; this option no longer has
// any effect and will be removed. Kept temporarily for compatibility with
// existing callers in the monorepo.
func WithAllowStepMetadata(md AllowStepMetadata) ExecutorOpt {
	return func(e execution.Executor) error {
		return nil
	}
}

func WithFunctionBacklogSizeLimit(fbsl BacklogSizeLimitFn) ExecutorOpt {
	return set(func(e *executor) { e.functionBacklogSizeLimit = fbsl })
}

type BacklogSizeLimit struct {
	Limit   int
	Enforce bool
}

type BacklogSizeLimitFn func(ctx context.Context, accountID, envID, fnID uuid.UUID) BacklogSizeLimit

type AccountPlanMetricTagResolver func(ctx context.Context, accountID uuid.UUID) string

func WithAccountPlanMetricTagResolver(resolver AccountPlanMetricTagResolver) ExecutorOpt {
	return set(func(e *executor) { e.accountPlanMetricTagResolver = resolver })
}

func WithConditionalTracer(tracer itrace.ConditionalTracer) ExecutorOpt {
	return set(func(e *executor) { e.conditionalTracer = tracer })
}

type ExecutionCapLimitDecision struct {
	Exceeded bool
	Enforce  bool
}

type ExecutionCapFn func(ctx context.Context, accountId uuid.UUID) ExecutionCapLimitDecision

func WithAccountExecutionCap(capFn ExecutionCapFn) ExecutorOpt {
	return set(func(e *executor) { e.accountExecutionCap = capFn })
}
