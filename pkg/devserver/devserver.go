package devserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config"
	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/service/servicefx"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultTick         = 150
	DefaultTickDuration = time.Millisecond * DefaultTick
	DefaultPollInterval = 5
	DefaultQueueWorkers = 100

	DefaultConnectGatewayPort      = connect.DefaultGatewayPort
	DefaultConnectGatewayGRPCPort  = 50052
	DefaultConnectExecutorGRPCPort = 50053

	DefaultDebugAPIPort = 7777
)

var defaultPartitionConstraintConfig = queue.PartitionConstraintConfig{
	Concurrency: queue.PartitionConcurrency{
		SystemConcurrency:   consts.DefaultConcurrencyLimit,
		AccountConcurrency:  consts.DefaultConcurrencyLimit,
		FunctionConcurrency: consts.DefaultConcurrencyLimit,
	},
}

// StartOpts configures the dev server
type StartOpts struct {
	Config        config.Config `json:"-"`
	RootDir       string        `json:"dir"`
	URLs          []string      `json:"urls"`
	Autodiscover  bool          `json:"autodiscover"`
	Poll          bool          `json:"poll"`
	PollInterval  int           `json:"poll_interval"`
	Tick          time.Duration `json:"tick"`
	RetryInterval int           `json:"retry_interval"`
	QueueWorkers  int           `json:"queue_workers"`

	// SigningKey is used to decide that the server should sign requests and
	// validate responses where applicable, modelling cloud behaviour.
	SigningKey *string `json:"-"`

	// EventKey is used to authorize incoming events, ensuring they match the
	// given key.
	EventKeys []string `json:"-"`

	// CloudSandboxes optionally handles authenticated Cloud compute requests.
	// Local events, functions, runs, and executor state are never forwarded.
	CloudSandboxes http.Handler `json:"-"`

	// RequireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	RequireKeys bool `json:"require_keys"`

	ConnectGatewayPort int                             `json:"connectGatewayPort"`
	ConnectGatewayHost string                          `json:"connectGatewayHost"`
	ConnectGRPCConfig  connectConfig.ConnectGRPCConfig `json:"connectGRPCConfig"`

	NoUI bool

	// Persist controls whether to persist data in between restarts.  If false,
	// the dev server will use in-memory databases.
	Persist bool `json:"persist"`

	// RedisURI allows connecting to external Redis instead of in-memory Redis
	RedisURI string `json:"redis_uri"`

	// PostgresURI allows connecting to external Postgres instead of SQLite
	PostgresURI             string `json:"postgres_uri"`
	PostgresMaxIdleConns    int    `json:"postgres-max-idle-conns"`
	PostgresMaxOpenConns    int    `json:"postgres-max-open-conns"`
	PostgresConnMaxIdleTime int    `json:"postgres-conn-max-idle-time"`
	PostgresConnMaxLifetime int    `json:"postgres-conn-max-lifetime"`

	// SQLiteDir specifies where SQLite files should be stored
	SQLiteDir string `json:"sqlite_dir"`

	// EnableDuckDB opts into the experimental DuckDB dual-write POC, gated
	// behind the `--duckdb` flag (or INNGEST_DUCKDB env var) on both
	// `inngest dev` and `inngest start`. Off by default. Controls writing
	// only -- see EnableDuckDBReads for reading. Enabling this alone always
	// dual-writes to an in-memory catalog; writing an on-disk catalog that
	// outlives the process additionally requires --persist AND the
	// EXPERIMENTAL_DUCKDB_PERSISTENT_DUALWRITE feature flag.
	EnableDuckDB bool `json:"enable_duckdb"`

	// EnableDuckDBReads opts into serving the GQL API and REST trace/run
	// endpoints from DuckDB once dual-write is up, gated behind the
	// `--duckdb-reads` flag (or INNGEST_DUCKDB_READS env var). Has no effect
	// unless EnableDuckDB is also true and dual-write starts successfully --
	// it exists so writing can be validated (throughput, correctness of the
	// dual-written data) before any read traffic depends on it. Off by
	// default, independent of EnableDuckDB's own default.
	EnableDuckDBReads bool `json:"enable_duckdb_reads"`

	// Debug API
	DebugAPIPort int `json:"debugAPIPort"`
}

// New runs the server until ctx is cancelled, the process receives an
// interrupt, terminate, or quit signal, or a service returns.  it returns
// after every service stops and every dependency closes, so the DuckDB
// dual-write has flushed when it returns.
func New(ctx context.Context, opts StartOpts) error {
	// The dev server _always_ logs output for development.
	opts.Config.Execution.LogOutput = true
	if opts.Tick == 0 {
		opts.Tick = DefaultTickDuration
	}

	exp := readExperiments()
	if err := validateDuckDBPersistenceFlags(opts.Persist, opts.EnableDuckDB, exp.duckDBPersistence); err != nil {
		return err
	}

	// cancelling ctx when New returns ends the goroutines that the
	// constructors start, also when the app fails to build or start.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var services *servicefx.Runner
	app := fx.New(
		options(ctx, opts, exp),
		fx.Populate(&services),
	)
	if err := app.Err(); err != nil {
		return err
	}
	shutdown := app.Wait()

	startCtx, cancelStart := context.WithTimeout(ctx, app.StartTimeout())
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		return err
	}

	signals, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stopSignals()
	select {
	case <-signals.Done():
	case <-shutdown:
	}

	stopCtx, cancelStop := context.WithTimeout(context.WithoutCancel(ctx), app.StopTimeout())
	defer cancelStop()
	return errors.Join(services.Err(), app.Stop(stopCtx))
}

func enforceConnectLeaseExpiry(ctx context.Context, accountID uuid.UUID) bool {
	return os.Getenv("INNGEST_CONNECT_DISABLE_ENFORCE_LEASE_EXPIRY") != "true"
}

func getInvokeEventHandler(ctx context.Context, pb pubsub.Publisher, topic string) execution.HandleInvokeEvent {
	return func(ctx context.Context, evt event.TrackedEvent) error {
		byt, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("error marshalling invocation event: %w", err)
		}

		carrier := itrace.NewTraceCarrier()
		itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

		err = pb.Publish(
			ctx,
			topic,
			pubsub.Message{
				Name:      event.EventReceivedName,
				Data:      string(byt),
				Timestamp: time.Now(),
				Metadata: map[string]any{
					consts.OtelPropagationKey: carrier,
				},
			},
		)
		if err != nil {
			return fmt.Errorf("error publishing invocation event: %w", err)
		}

		return nil
	}
}

func getInvokeFailHandler(ctx context.Context, pb pubsub.Publisher, topic string) execution.InvokeFailHandler {
	return func(ctx context.Context, opts execution.InvokeFailHandlerOpts, evts []event.Event) error {
		eg := errgroup.Group{}

		for _, e := range evts {
			evt := e
			eg.Go(func() error {
				trackedEvent := event.NewBaseTrackedEvent(evt, nil)
				byt, err := json.Marshal(trackedEvent)
				if err != nil {
					return fmt.Errorf("error marshalling function finished event: %w", err)
				}

				err = pb.Publish(
					ctx,
					topic,
					pubsub.Message{
						Name:      event.EventReceivedName,
						Data:      string(byt),
						Timestamp: trackedEvent.GetEvent().Time(),
					},
				)
				if err != nil {
					return fmt.Errorf("error publishing function finished event: %w", err)
				}

				return nil
			})
		}

		return eg.Wait()
	}
}

func NormalizeConcurrencyKeys(smv2 sv2.StateLoader, dbcqrs cqrs.Manager) queue.NormalizeRefreshItemCustomConcurrencyKeysFn {
	return func(ctx context.Context, item *queue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *queue.QueueShadowPartition) ([]state.CustomConcurrency, error) {
		id := sv2.IDFromV1(item.Data.Identifier)

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, id.FunctionID)
		if err != nil {
			return nil, fmt.Errorf("could not find workflow: %w", err)
		}
		fn, err := workflow.InngestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not convert workflow to inngest function: %w", err)
		}

		if fn.Concurrency == nil || len(fn.Concurrency.Limits) == 0 {
			return nil, nil
		}

		events, err := smv2.LoadEvents(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not load events: %w", err)
		}

		if len(events) == 0 {
			return nil, nil
		}

		var evt0 event.Event

		if err := json.Unmarshal(events[0], &evt0); err != nil {
			return nil, fmt.Errorf("could not unmarshal event: %w", err)
		}

		evtMap := evt0.Map()

		return queue.GetCustomConcurrencyKeys(ctx, id, fn.Concurrency.Limits, evtMap), nil
	}
}

func NormalizeThrottle(smv2 sv2.StateLoader, dbcqrs cqrs.Manager) queue.RefreshItemThrottleFn {
	return func(ctx context.Context, item *queue.QueueItem) (*queue.Throttle, error) {
		id := sv2.IDFromV1(item.Data.Identifier)

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, id.FunctionID)
		if err != nil {
			return nil, fmt.Errorf("could not find workflow: %w", err)
		}
		fn, err := workflow.InngestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not convert workflow to inngest function: %w", err)
		}

		if fn.Throttle == nil {
			return nil, nil
		}

		events, err := smv2.LoadEvents(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not load events: %w", err)
		}

		if len(events) == 0 {
			return nil, nil
		}

		var evt0 event.Event

		if err := json.Unmarshal(events[0], &evt0); err != nil {
			return nil, fmt.Errorf("could not unmarshal event: %w", err)
		}

		evtMap := evt0.Map()

		return queue.GetThrottleConfig(ctx, id.FunctionID, fn.Throttle, evtMap), nil
	}
}

func PartitionConstraintConfigGetter(dbcqrs cqrs.Manager) queue.PartitionConstraintConfigGetter {
	return func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
		if p.SystemQueueName != nil {
			return defaultPartitionConstraintConfig
		}

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, p.FunctionID)
		if err != nil {
			return defaultPartitionConstraintConfig
		}
		fn, err := workflow.InngestFunction()
		if err != nil {
			return defaultPartitionConstraintConfig
		}

		// TODO Make this reusable in cloud, it's the same operation with different data sources
		accountLimit := consts.DefaultConcurrencyLimit

		fnLimit := fn.ConcurrencyLimit()
		if fnLimit <= 0 {
			fnLimit = accountLimit
		}

		fnVersion := fn.FunctionVersion
		if fnVersion <= 0 {
			fnVersion = 1
		}

		constraints := queue.PartitionConstraintConfig{
			FunctionVersion: fnVersion,

			Concurrency: queue.PartitionConcurrency{
				SystemConcurrency:     consts.DefaultConcurrencyLimit,
				AccountConcurrency:    accountLimit,
				FunctionConcurrency:   fnLimit,
				CustomConcurrencyKeys: nil,
			},
		}

		if fn.Concurrency != nil && len(fn.Concurrency.Limits) > 0 {
			for _, limit := range fn.Concurrency.Limits {
				if !limit.IsCustomLimit() {
					continue
				}

				constraints.Concurrency.CustomConcurrencyKeys = append(constraints.Concurrency.CustomConcurrencyKeys,
					queue.CustomConcurrencyLimit{
						Mode:                enums.ConcurrencyModeStep,
						Scope:               limit.Scope,
						HashedKeyExpression: limit.Hash,
						Limit:               limit.Limit,
					})
			}
		}

		if fn.Throttle != nil {
			var keyExpr string
			if fn.Throttle.Key != nil {
				keyExpr = *fn.Throttle.Key
			}

			constraints.Throttle = &queue.PartitionThrottle{
				ThrottleKeyExpressionHash: util.XXHash(keyExpr),
				Limit:                     int(fn.Throttle.Limit),
				Burst:                     int(fn.Throttle.Burst),
				Period:                    int(fn.Throttle.Period.Seconds()),
			}
		}

		// NOTE: Manual-release semaphores (fn concurrency) are NOT added to the partition config.
		// They are only embedded on the start job's queue item. Subsequent steps don't need them —
		// the semaphore is already held for the run and released on finalization.
		// Worker concurrency (auto-release) would go here when implemented.

		return constraints
	}
}
