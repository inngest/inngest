package devserver

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/cmd/internal/cloudsandboxes"
	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/cmd/internal/server"
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/urfave/cli/v3"
)

func action(ctx context.Context, cmd *cli.Command) error {
	conf, err := config.Dev(ctx)
	if err != nil {
		return err
	}

	if err = localconfig.InitDevConfig(ctx, cmd); err != nil {
		return err
	}

	// Sent here instead of the root Before hook so subcommand flags and the
	// config file/env have been resolved.
	tel.SendCommandExecutedEvent(ctx, "inngest dev", map[string]any{
		"flags": localconfig.NonDefaultFlags(cmd),
	})

	server.SetAddress(cmd, conf)
	conf.ServerKind = headers.ServerKindDev

	connectGRPCConfig, err := server.ConnectGRPCConfig(ctx, cmd)
	if err != nil {
		return err
	}

	tick := localconfig.GetIntValue(cmd, "tick", devserver.DefaultTick)
	opts := devserver.StartOpts{
		Autodiscover:            !localconfig.GetBoolValue(cmd, "no-discovery", false),
		Config:                  *conf,
		Poll:                    !localconfig.GetBoolValue(cmd, "no-poll", false),
		PollInterval:            localconfig.GetIntValue(cmd, "poll-interval", devserver.DefaultPollInterval),
		RetryInterval:           localconfig.GetIntValue(cmd, "retry-interval", 0),
		QueueWorkers:            localconfig.GetIntValue(cmd, "queue-workers", devserver.DefaultQueueWorkers),
		Tick:                    time.Duration(tick) * time.Millisecond,
		URLs:                    localconfig.GetStringSlice(cmd, "sdk-url"),
		ConnectGatewayPort:      localconfig.GetIntValue(cmd, "connect-gateway-port", devserver.DefaultConnectGatewayPort),
		ConnectGatewayHost:      conf.CoreAPI.Addr,
		ConnectGRPCConfig:       connectGRPCConfig,
		Persist:                 localconfig.GetBoolValue(cmd, "persist", false),
		SQLiteDir:               localconfig.GetValue(cmd, "sqlite-dir", ""),
		EnableDuckDB:            localconfig.GetBoolValue(cmd, "duckdb", false),
		EnableDuckDBReads:       localconfig.GetBoolValue(cmd, "duckdb-reads", false),
		PostgresURI:             localconfig.GetValue(cmd, "postgres-uri", ""),
		PostgresMaxIdleConns:    localconfig.GetIntValue(cmd, "postgres-max-idle-conns", 10),
		PostgresMaxOpenConns:    localconfig.GetIntValue(cmd, "postgres-max-open-conns", 100),
		PostgresConnMaxIdleTime: localconfig.GetIntValue(cmd, "postgres-conn-max-idle-time", 5),
		PostgresConnMaxLifetime: localconfig.GetIntValue(cmd, "postgres-conn-max-lifetime", 30),
		DebugAPIPort:            localconfig.GetIntValue(cmd, "debug-api-port", devserver.DefaultDebugAPIPort),
	}

	// a second signal always exits at once.  without --duckdb the first
	// signal exits at once too, because nothing needs to flush.  with
	// --duckdb the first signal stops the server gracefully, so dual-write
	// flushes its batches.
	signals := 1
	if opts.EnableDuckDB {
		signals = 2
	}
	server.ExitAfterSignals(signals)

	opts, changes, err := devserver.ResolvePortConflicts(opts)
	if err != nil {
		return err
	}
	l := logger.StdlibLogger(ctx)
	for _, change := range changes {
		l.Info(
			"Port conflict, using new port",
			"name", change.Name,
			"from", change.From,
			"to", change.To,
		)
	}

	bridge, err := cloudsandboxes.New(ctx, opts.Config.EventAPI.Port)
	if err != nil {
		return fmt.Errorf("could not create the cloud sandboxes bridge: %w", err)
	}
	opts.CloudSandboxes = bridge

	closeTracers, err := server.SetupTracers(
		ctx,
		opts.Config.EventAPI.Port,
		localconfig.GetValue(cmd, "system-trace-endpoint", ""),
		localconfig.GetValue(cmd, "system-trace-url-path", ""),
	)
	if err != nil {
		return err
	}
	defer closeTracers()

	return devserver.New(ctx, opts)
}
