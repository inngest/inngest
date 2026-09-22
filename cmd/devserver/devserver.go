package devserver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/config"
	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/urfave/cli/v3"
)

func action(ctx context.Context, cmd *cli.Command) error {
	// duckDBClosed is closed by pkg/devserver's start() only once the DuckDB
	// dual-write subprocess has been fully torn down -- which, since start()
	// calls service.StartAll synchronously before its shutdown defers run,
	// is necessarily after every other service has already finished its own
	// graceful stop too. With --duckdb, the signal handler below waits for it
	// before exiting rather than exiting immediately on signal, which is what
	// keeps it from reintroducing the dual-write data-loss bug the naive
	// version of this handler caused (unflushed batches dropped on exit). A
	// second signal still forces an immediate exit, so a wedged shutdown can
	// always be interrupted. Without --duckdb it is closed as soon as the
	// flags are read, preserving the exit-on-first-signal behavior.
	duckDBClosed := make(chan struct{})
	go func() {
		sigs := make(chan os.Signal, 2)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
		<-sigs
		select {
		case <-duckDBClosed:
		case <-sigs:
		}
		os.Exit(0)
	}()

	conf, err := config.Dev(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err = localconfig.InitDevConfig(ctx, cmd); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	port := localconfig.GetIntValue(cmd, "port", api.DefaultAPIPort)
	conf.EventAPI.Port = port
	conf.CoreAPI.Port = port

	host := localconfig.GetValue(cmd, "host", "")
	if host != "" {
		conf.EventAPI.Addr = host
		conf.CoreAPI.Addr = host
	}

	urls := localconfig.GetStringSlice(cmd, "sdk-url")

	// Run auto-discovery unless we've explicitly disabled it.
	noDiscovery := localconfig.GetBoolValue(cmd, "no-discovery", false)
	noPoll := localconfig.GetBoolValue(cmd, "no-poll", false)
	pollInterval := localconfig.GetIntValue(cmd, "poll-interval", devserver.DefaultPollInterval)
	retryInterval := localconfig.GetIntValue(cmd, "retry-interval", 0)
	queueWorkers := localconfig.GetIntValue(cmd, "queue-workers", devserver.DefaultQueueWorkers)
	tick := localconfig.GetIntValue(cmd, "tick", devserver.DefaultTick)
	persist := localconfig.GetBoolValue(cmd, "persist", false)
	sqliteDir := localconfig.GetValue(cmd, "sqlite-dir", "")
	enableDuckDB := localconfig.GetBoolValue(cmd, "duckdb", false)
	enableDuckDBReads := localconfig.GetBoolValue(cmd, "duckdb-reads", false)
	// start() only closes DuckDBClosed when it is set; see duckDBClosed above.
	var startDuckDBClosed chan struct{}
	if enableDuckDB {
		startDuckDBClosed = duckDBClosed
	} else {
		close(duckDBClosed)
	}

	debugAPIPort := localconfig.GetIntValue(cmd, "debug-api-port", devserver.DefaultDebugAPIPort)

	connectGatewayPort := localconfig.GetIntValue(cmd, "connect-gateway-port", devserver.DefaultConnectGatewayPort)
	connectGatewayGRPCPort := localconfig.GetIntValue(cmd, "connect-gateway-grpc-port", devserver.DefaultConnectGatewayGRPCPort)
	connectExecutorGRPCPort := localconfig.GetIntValue(cmd, "connect-executor-grpc-port", devserver.DefaultConnectExecutorGRPCPort)
	connectGatewayGRPCIP := localconfig.GetValue(cmd, "connect-gateway-grpc-ip", connectgrpc.DefaultConnectGRPCIP)
	connectExecutorGRPCIP := localconfig.GetValue(cmd, "connect-executor-grpc-ip", connectgrpc.DefaultConnectGRPCIP)
	if err := localconfig.ValidateConnectGRPCIPs(connectGatewayGRPCIP, connectExecutorGRPCIP); err != nil {
		return err
	}

	postgresURI := localconfig.GetValue(cmd, "postgres-uri", "")
	postgresMaxIdleConns := localconfig.GetIntValue(cmd, "postgres-max-idle-conns", 10)
	postgresMaxOpenConns := localconfig.GetIntValue(cmd, "postgres-max-open-conns", 100)
	postgresConnMaxIdleTime := localconfig.GetIntValue(cmd, "postgres-conn-max-idle-time", 5)
	postgresConnMaxLifetime := localconfig.GetIntValue(cmd, "postgres-conn-max-lifetime", 30)

	conf.ServerKind = headers.ServerKindDev

	opts := devserver.StartOpts{
		Autodiscover:       !noDiscovery,
		Config:             *conf,
		Poll:               !noPoll,
		PollInterval:       pollInterval,
		RetryInterval:      retryInterval,
		QueueWorkers:       queueWorkers,
		Tick:               time.Duration(tick) * time.Millisecond,
		URLs:               urls,
		ConnectGatewayPort: connectGatewayPort,
		ConnectGatewayHost: conf.CoreAPI.Addr,
		ConnectGRPCConfig: connectConfig.NewGRPCConfig(
			ctx,
			connectGatewayGRPCIP, connectGatewayGRPCPort,
			connectExecutorGRPCIP, connectExecutorGRPCPort,
		),
		Persist:                 persist,
		SQLiteDir:               sqliteDir,
		EnableDuckDB:            enableDuckDB,
		EnableDuckDBReads:       enableDuckDBReads,
		PostgresURI:             postgresURI,
		PostgresMaxIdleConns:    postgresMaxIdleConns,
		PostgresMaxOpenConns:    postgresMaxOpenConns,
		PostgresConnMaxIdleTime: postgresConnMaxIdleTime,
		PostgresConnMaxLifetime: postgresConnMaxLifetime,
		DebugAPIPort:            debugAPIPort,
		DuckDBClosed:            startDuckDBClosed,
	}

	l := logger.StdlibLogger(ctx)

	opts, changes, err := devserver.ResolvePortConflicts(opts)
	if err != nil {
		return err
	}
	for _, change := range changes {
		l.Info(
			"Port conflict, using new port",
			"name", change.Name,
			"from", change.From,
			"to", change.To,
		)
	}

	traceEndpoint := fmt.Sprintf("localhost:%d", opts.Config.EventAPI.Port)
	if err := itrace.NewUserTracer(ctx, itrace.TracerOpts{
		ServiceName:   "tracing",
		TraceEndpoint: traceEndpoint,
		TraceURLPath:  "/dev/traces",
		Type:          itrace.TracerTypeOTLPHTTP,
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		_ = itrace.CloseUserTracer(ctx)
	}()

	systemTraceEndpoint := localconfig.GetValue(cmd, "system-trace-endpoint", traceEndpoint)
	systemTraceURLPath := localconfig.GetValue(cmd, "system-trace-url-path", "/dev/traces/system")

	if err := itrace.NewSystemTracer(ctx, itrace.TracerOpts{
		ServiceName:   "tracing-system",
		TraceEndpoint: systemTraceEndpoint,
		TraceURLPath:  systemTraceURLPath,
		Type:          itrace.TracerTypeOTLPHTTP,
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		_ = itrace.CloseSystemTracer(ctx)
	}()

	err = devserver.New(ctx, opts)
	if err != nil {
		return err
	}
	return nil
}
