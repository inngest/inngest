package devserver

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service/servicefx"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const (
	// startTimeout is the time every service has to finish Pre.  each Pre gets
	// 30 seconds, and they run at the same time.
	startTimeout = time.Minute

	// stopTimeout is the time the services have to stop.  the executor waits
	// for the steps that are running, and a step runs for at most
	// consts.MaxFunctionTimeout.
	stopTimeout = consts.MaxFunctionTimeout
)

// options are the dependencies and services of the server.
func options(ctx context.Context, opts StartOpts, exp experiments) fx.Option {
	options := []fx.Option{
		fx.Supply(
			opts,
			exp,
		),
		fx.Provide(newLogger),
		fx.Module(
			"storage",
			fx.Provide(
				newDatabase,
				newCQRS,
			),
		),
		fx.Module(
			"redis",
			fx.Provide(
				newRedisClients,
				newRedisState,
				newBroadcaster,
			),
		),
		fx.Module(
			"queue",
			fx.Provide(
				newConstraintAPI,
				newQueue,
				newQueueManagers,
			),
		),
		fx.Module(
			"execution",
			fx.Provide(
				newConnect,
				newHTTPClient,
				newPublisher,
				newTracerProvider,
				newDualWrite,
				newLimitOverrides,
				newExecutor,
				newRunner,
				newDevserver,
			),
		),
		servicefx.Provide(
			newLeaseScavenger,
			newExecutorService,
			runnerService,
			devserverService,
			newAPIService,
			newConnectGateway,
		),
	}
	if exp.debugAPI {
		options = append(options, servicefx.Provide(newDebugAPI))
	}
	return fx.Options(append(
		options,
		servicefx.Module(ctx),
		fx.WithLogger(fxLogger),
		fx.StartTimeout(startTimeout),
		fx.StopTimeout(stopTimeout),
	)...)
}

func newLogger(ctx context.Context) logger.Logger {
	return logger.StdlibLogger(ctx)
}

// fxLogger writes Fx events at debug level, so they show with --verbose and
// do not fill the terminal otherwise.
func fxLogger(log logger.Logger) fxevent.Logger {
	l := &fxevent.SlogLogger{
		Logger: log.SLog(),
	}
	l.UseLogLevel(slog.LevelDebug)
	return l
}

// experiments are the EXPERIMENTAL_* and DEBUG environment variables.  New
// reads them once, at startup.
type experiments struct {
	keyQueues bool
	// stepMetadata is on unless EXPERIMENTAL_STEP_METADATA is "false".
	stepMetadata            bool
	asyncDispatchValidation bool
	// splitBatchPartitionByFunction sends schedule-batch jobs to a system
	// partition for each function.  otherwise the jobs share the global
	// schedule-batch partition.
	splitBatchPartitionByFunction bool
	// duckDBPersistence lets --persist with --duckdb write an on-disk DuckLake
	// catalog.
	duckDBPersistence bool
	// debugAPI serves the debug API on StartOpts.DebugAPIPort.
	debugAPI bool
}

func readExperiments() experiments {
	return experiments{
		keyQueues:                     os.Getenv("EXPERIMENTAL_KEY_QUEUES_ENABLE") == "true",
		stepMetadata:                  os.Getenv("EXPERIMENTAL_STEP_METADATA") != "false",
		asyncDispatchValidation:       os.Getenv("EXPERIMENTAL_ASYNC_DISPATCH_VALIDATION") == "true",
		splitBatchPartitionByFunction: os.Getenv("EXPERIMENTAL_SPLIT_BATCH_PARTITION_BY_FUNCTION") == "true",
		duckDBPersistence:             os.Getenv("EXPERIMENTAL_DUCKDB_PERSISTENT_DUALWRITE") == "true",
		debugAPI:                      os.Getenv("DEBUG") != "",
	}
}
