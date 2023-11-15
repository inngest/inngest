package devserver

import (
	"context"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/config"
	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs/sqlitecqrs"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/history_drivers/memory_writer"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/util/awsgateway"
	"github.com/redis/rueidis"
)

// StartOpts configures the dev server
type StartOpts struct {
	Config        config.Config `json:"-"`
	RootDir       string        `json:"dir"`
	URLs          []string      `json:"urls"`
	Autodiscover  bool          `json:"autodiscover"`
	Poll          bool          `json:"poll"`
	RetryInterval int           `json:"retry_interval"`
}

// Create and start a new dev server.  The dev server is used during (surprise surprise)
// development.
//
// It runs all available services from `inngest serve`, plus:
// - Adds development-specific APIs for communicating with the SDK.
func New(ctx context.Context, opts StartOpts) error {
	// The dev server _always_ logs output for development.
	if !opts.Config.Execution.LogOutput {
		opts.Config.Execution.LogOutput = true
	}

	// Before running the development service, ensure that we change the http
	// driver in development to use our AWS Gateway http client, attempting to
	// automatically transform dev requests to lambda invocations.
	httpdriver.DefaultExecutor.Client.Transport = awsgateway.NewTransformTripper(httpdriver.DefaultExecutor.Client.Transport)
	deploy.Client.Transport = awsgateway.NewTransformTripper(deploy.Client.Transport)

	return start(ctx, opts)
}

func start(ctx context.Context, opts StartOpts) error {
	db, err := sqlitecqrs.New()
	if err != nil {
		return err
	}

	// Initialize the devserver
	dbcqrs := sqlitecqrs.NewCQRS(db)
	hd := sqlitecqrs.NewHistoryDriver(db)
	loader := dbcqrs.(state.FunctionLoader)

	rc, err := createInmemoryRedis(ctx)
	if err != nil {
		return err
	}

	var sm state.Manager
	t := runner.NewTracker()
	sm, err = redis_state.New(
		ctx,
		redis_state.WithFunctionLoader(loader),
		redis_state.WithRedisClient(rc),
		redis_state.WithKeyGenerator(redis_state.DefaultKeyFunc{
			Prefix: "{state}",
		}),
		redis_state.WithFunctionCallbacks(func(ctx context.Context, i state.Identifier, rs enums.RunStatus) {
			switch rs {
			case enums.RunStatusRunning:
				// Add this to the in-memory tracker.
				// XXX: Switch to sqlite.
				s, err := sm.Load(ctx, i.RunID)
				if err != nil {
					return
				}
				t.Add(s.Event()["id"].(string), i)
			}
		}),
	)
	if err != nil {
		return err
	}

	queueKG := &redis_state.DefaultQueueKeyGenerator{
		Prefix: "{queue}",
	}
	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithNumWorkers(100),
		redis_state.WithPollTick(150 * time.Millisecond),
		redis_state.WithQueueKeyGenerator(queueKG),
		redis_state.WithCustomConcurrencyKeyGenerator(func(ctx context.Context, i redis_state.QueueItem) []state.CustomConcurrency {
			fn, err := dbcqrs.GetFunctionByID(ctx, i.Data.Identifier.WorkflowID)
			if err != nil {
				// Use what's stored in the state store.
				return i.Data.Identifier.CustomConcurrencyKeys
			}
			f, err := fn.InngestFunction()
			if err != nil {
				return i.Data.Identifier.CustomConcurrencyKeys
			}

			if f.Concurrency != nil {
				for _, c := range f.Concurrency.Limits {
					if !c.IsCustomLimit() {
						continue
					}
					// If there's a concurrency key with the same hash, use the new function's
					// concurrency limits.
					//
					// NOTE:  This is accidentally quadratic but is okay as we bound concurrency
					// keys to a low value (2-3).
					for _, actual := range i.Data.Identifier.CustomConcurrencyKeys {
						if actual.Hash != "" && actual.Hash == c.Hash {
							actual.Limit = c.Limit
						}
					}
				}
			}
			return i.Data.Identifier.CustomConcurrencyKeys
		}),
		redis_state.WithAccountConcurrencyKeyGenerator(func(ctx context.Context, i redis_state.QueueItem) (string, int) {
			// NOTE: In the dev server there are no account concurrency limits.
			return i.Queue(), consts.DefaultConcurrencyLimit
		}),
		redis_state.WithPartitionConcurrencyKeyGenerator(func(ctx context.Context, p redis_state.QueuePartition) (string, int) {
			// Ensure that we return the correct concurrency values per
			// partition.
			funcs, err := dbcqrs.GetFunctions(ctx)
			if err != nil {
				return p.Queue(), consts.DefaultConcurrencyLimit
			}
			for _, fn := range funcs {
				f, _ := fn.InngestFunction()
				if f.ID == uuid.Nil {
					f.ID = inngest.DeterministicUUID(*f)
				}
				if f.ID == p.WorkflowID && f.Concurrency != nil && f.Concurrency.PartitionConcurrency() > 0 {
					return p.Queue(), f.Concurrency.PartitionConcurrency()
				}
			}
			return p.Queue(), consts.DefaultConcurrencyLimit
		}),
	}
	if opts.RetryInterval > 0 {
		queueOpts = append(queueOpts, redis_state.WithBackoffFunc(
			backoff.GetLinearBackoffFunc(time.Duration(opts.RetryInterval)*time.Second),
		))
	}
	queue := redis_state.NewQueue(rc, queueOpts...)

	rl := ratelimit.New(ctx, rc, "{ratelimit}:")

	debouncer := debounce.NewRedisDebouncer(rc, queueKG, queue)

	var drivers = []driver.Driver{}
	for _, driverConfig := range opts.Config.Execution.Drivers {
		d, err := driverConfig.NewDriver()
		if err != nil {
			return err
		}
		drivers = append(drivers, d)
	}
	exec, err := executor.NewExecutor(
		executor.WithStateManager(sm),
		executor.WithRuntimeDrivers(
			drivers...,
		),
		executor.WithQueue(queue),
		executor.WithLogger(logger.From(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithLifecycleListeners(
			history.NewLifecycleListener(
				nil,
				hd,
				memory_writer.NewWriter(),
			),
			lifecycle{
				sm:   sm,
				cqrs: dbcqrs,
			},
		),
		executor.WithStepLimits(consts.DefaultMaxStepLimit),
		executor.WithDebouncer(debouncer),
	)
	if err != nil {
		return err
	}

	// Create an executor.
	executorSvc := executor.NewService(
		opts.Config,
		executor.WithExecutionLoader(dbcqrs),
		executor.WithState(sm),
		executor.WithServiceQueue(queue),
		executor.WithServiceExecutor(exec),
		executor.WithServiceDebouncer(debouncer),
	)

	runner := runner.NewService(
		opts.Config,
		runner.WithCQRS(dbcqrs),
		runner.WithExecutor(exec),
		runner.WithExecutionLoader(dbcqrs),
		runner.WithEventManager(event.NewManager()),
		runner.WithStateManager(sm),
		runner.WithRunnerQueue(queue),
		runner.WithTracker(t),
		runner.WithRateLimiter(rl),
	)

	// The devserver embeds the event API.
	ds := newService(opts, runner, dbcqrs)
	// embed the tracker
	ds.tracker = t
	ds.state = sm
	ds.queue = queue
	ds.executor = exec

	return service.StartAll(ctx, ds, runner, executorSvc)
}

func createInmemoryRedis(ctx context.Context) (rueidis.Client, error) {
	r := miniredis.NewMiniRedis()
	_ = r.Start()
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	if err != nil {
		return nil, err
	}
	go func() {
		for range time.Tick(time.Second) {
			r.FastForward(time.Second)
		}
	}()
	return rc, nil
}
