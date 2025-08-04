package start

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/inngest/inngest/cmd/internal/localconfig"
	"github.com/inngest/inngest/pkg/authn"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	cmd := &cli.Command{
		Name:        "start",
		Usage:       "[Beta] Run Inngest as a single-node service.",
		UsageText:   "inngest start [options]",
		Description: "Example: inngest start",
		Action:      action,

		Flags: []cli.Flag{
			// Base flags
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to an Inngest configuration file",
			},
			&cli.StringFlag{
				Name:  "host",
				Usage: "Inngest server hostname",
			},
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   "8288",
				Usage:   "Inngest server port",
			},
			&cli.StringSliceFlag{
				Name:    "sdk-url",
				Aliases: []string{"u"},
				Usage:   "App serve URLs to sync (ex. http://localhost:3000/api/inngest)",
			},
			&cli.StringFlag{
				Name:  "signing-key",
				Usage: "Signing key used to sign and validate data between the server and apps.",
			},
			&cli.StringSliceFlag{
				Name:  "event-key",
				Usage: "Event key(s) that will be used by apps to send events to the server.",
			},

			// Persistence flags
			&cli.StringFlag{
				Name:  "sqlite-dir",
				Usage: "Directory for where to write SQLite database.",
			},
			&cli.StringFlag{
				Name:  "redis-uri",
				Usage: "Redis server URI for external queue and run state. Defaults to self-contained, in-memory Redis server with periodic snapshot backups.",
			},
			&cli.StringFlag{
				Name:  "postgres-uri",
				Usage: "PostgreSQL database URI for configuration and history persistence. Defaults to SQLite database.",
			},

			// Advanced flags
			&cli.IntFlag{
				Name:  "poll-interval",
				Value: 0,
				Usage: "Interval in seconds between polling for updates to apps",
			},
			&cli.IntFlag{
				Name:  "retry-interval",
				Value: 0,
				Usage: "Retry interval in seconds for linear backoff when retrying functions - must be 1 or above",
			},
			&cli.IntFlag{
				Name:  "queue-workers",
				Value: devserver.DefaultQueueWorkers,
				Usage: "Number of executor workers to execute steps from the queue",
			},
			&cli.IntFlag{
				Name:  "tick",
				Value: devserver.DefaultTick,
				Usage: "The interval (in milliseconds) at which the executor polls the queue",
			},
			&cli.IntFlag{
				Name:  "connect-gateway-port",
				Value: devserver.DefaultConnectGatewayPort,
				Usage: "Port to expose connect gateway endpoint",
			},
			&cli.BoolFlag{
				Name:  "no-ui",
				Usage: "Disable the web UI and GraphQL API endpoint",
			},
		},
	}

	return cmd
}

func action(ctx context.Context, cmd *cli.Command) error {
	// TODO Likely need a `Start()`
	conf, err := config.Dev(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err = localconfig.InitStartConfig(ctx, cmd); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	portStr := localconfig.GetValue(cmd, "port", "port", "8288")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	conf.EventAPI.Port = port
	conf.CoreAPI.Port = port

	host := localconfig.GetValue(cmd, "host", "host", "")
	if host != "" {
		conf.EventAPI.Addr = host
		conf.CoreAPI.Addr = host
	}

	traceEndpoint := fmt.Sprintf("localhost:%d", port)
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

	if err := itrace.NewSystemTracer(ctx, itrace.TracerOpts{
		ServiceName:   "tracing-system",
		TraceEndpoint: traceEndpoint,
		TraceURLPath:  "/dev/traces/system",
		Type:          itrace.TracerTypeOTLPHTTP,
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		_ = itrace.CloseSystemTracer(ctx)
	}()

	tick := cmd.Int("tick")
	if tick < 1 {
		tick = devserver.DefaultTick
	}

	signingKey := localconfig.GetValue(cmd, "signing-key", "signing-key", "")
	if signingKey == "" {
		fmt.Println("Error: signing-key is required")
		os.Exit(1)
	}
	_, err = authn.HashedSigningKey(signingKey)
	if err != nil {
		fmt.Printf("Error: signing-key must be a valid hexadecimal string\n")
		os.Exit(1)
	}

	eventKeys := localconfig.GetStringSlice(cmd, "event-key", "event-key")
	if len(eventKeys) == 0 {
		fmt.Println("Error: at least one event-key is required")
		os.Exit(1)
	}

	conf.ServerKind = headers.ServerKindCloud

	// Handle configuration options with simplified koanf-based approach
	postgresURI := localconfig.GetValue(cmd, "postgres-uri", "postgres-uri", "")
	redisURI := localconfig.GetValue(cmd, "redis-uri", "redis-uri", "")
	sqliteDir := localconfig.GetValue(cmd, "sqlite-dir", "sqlite-dir", "")
	sdkURLs := localconfig.GetStringSlice(cmd, "sdk-url", "sdk-url")

	opts := devserver.StartOpts{
		Config:             *conf,
		ConnectGatewayHost: conf.CoreAPI.Addr,
		ConnectGatewayPort: cmd.Int("connect-gateway-port"),
		EventKeys:          eventKeys,
		InMemory:           false,
		NoUI:               cmd.Bool("no-ui"),
		PollInterval:       cmd.Int("poll-interval"),
		PostgresURI:        postgresURI,
		QueueWorkers:       cmd.Int("queue-workers"),
		RedisURI:           redisURI,
		RequireKeys:        true,
		RetryInterval:      cmd.Int("retry-interval"),
		SigningKey:         &signingKey,
		SQLiteDir:          sqliteDir,
		Tick:               time.Duration(tick) * time.Millisecond,
		URLs:               sdkURLs,
	}

	err = devserver.New(ctx, opts)
	if err != nil {
		return err
	}
	return nil
}
