package devserver

import (
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	cmd := &cli.Command{
		Name:        "dev",
		Usage:       "Run the Inngest Dev Server for local development.",
		UsageText:   "inngest dev [options]",
		Description: "Example: inngest dev -u http://localhost:3000/api/inngest",
		Action:      action,

		Flags: []cli.Flag{
			// Base flags
			&cli.StringSliceFlag{
				Name:    "sdk-url",
				Aliases: []string{"u"},
				Usage:   "App serve URLs to sync (ex. http://localhost:3000/api/inngest)",
			},
			&cli.BoolFlag{
				Name:  "no-discovery",
				Usage: "Disable app auto-discovery",
			},
			&cli.BoolFlag{
				Name:  "no-poll",
				Usage: "Disable polling of apps for updates",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to an Inngest configuration file",
			},
			&cli.StringFlag{
				Name:  "host",
				Usage: "Inngest server host",
			},
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   "8288",
				Usage:   "Inngest server port",
			},

			// Advanced flags
			&cli.BoolFlag{
				Category: "Advanced",
				Name:     "persist",
				Value:    false,
				Usage:    "Persist data in between restarts",
			},
			&cli.IntFlag{
				Category: "Advanced",
				Name:     "poll-interval",
				Value:    devserver.DefaultPollInterval,
				Usage:    "Interval in seconds between polling for updates to apps",
			},
			&cli.IntFlag{
				Category: "Advanced",
				Name:     "retry-interval",
				Value:    0,
				Usage:    "Retry interval in seconds for linear backoff when retrying functions - must be 1 or above",
			},
			&cli.IntFlag{
				Category: "Advanced",
				Name:     "queue-workers",
				Value:    devserver.DefaultQueueWorkers,
				Usage:    "Number of executor workers to execute steps from the queue",
			},
			&cli.IntFlag{
				Category: "Advanced",
				Name:     "tick",
				Value:    devserver.DefaultTick,
				Usage:    "The interval (in milliseconds) at which the executor polls the queue",
			},
			&cli.IntFlag{
				Category: "Advanced",
				Name:     "connect-gateway-port",
				Value:    devserver.DefaultConnectGatewayPort,
				Usage:    "Port to expose connect gateway endpoint",
			},
			&cli.IntFlag{
				Category: "Advanced",
				Name:     "connect-gateway-grpc-port",
				Value:    devserver.DefaultConnectGatewayGRPCPort,
				Usage:    "Port to expose connect gateway grpc endpoint",
			},
			&cli.IntFlag{
				Category: "Advanced",
				Name:     "connect-executor-grpc-port",
				Value:    devserver.DefaultConnectExecutorGRPCPort,
				Usage:    "Port to expose connect executor grpc endpoint",
			},
			&cli.StringFlag{
				Category: "Advanced",
				Name:     "sqlite-dir",
				Usage:    "Directory for where to write SQLite database.",
				Hidden:   true,
			},
			&cli.StringFlag{
				Category: "Persistence",
				Name:     "postgres-uri",
				Usage:    "PostgreSQL database URI for configuration and history persistence. Defaults to SQLite database.",
				Hidden:   true,
			},
			&cli.IntFlag{
				Category: "Persistence",
				Name:     "postgres-max-idle-conns",
				Usage:    "Sets the maximum number of idle database connections in the PostgreSQL connection pool.",
				Value:    10,
				Hidden:   true,
			},
			&cli.IntFlag{
				Category: "Persistence",
				Name:     "postgres-max-open-conns",
				Usage:    "Sets the maximum number of open database connections allowed in the PostgreSQL connection pool.",
				Value:    100,
				Hidden:   true,
			},
			&cli.IntFlag{
				Category: "Persistence",
				Name:     "postgres-conn-max-idle-time",
				Usage:    "Sets the maximum amount of time, in minutes, a PostgreSQL connection may be idle.",
				Value:    5,
				Hidden:   true,
			},
			&cli.IntFlag{
				Category: "Persistence",
				Name:     "postgres-conn-max-lifetime",
				Usage:    "Sets the maximum amount of time, in minutes, a PostgreSQL connection may be reused.",
				Value:    30,
				Hidden:   true,
			},

			// Internal
			&cli.IntFlag{
				Name:   "debug-api-port",
				Value:  devserver.DefaultDebugAPIPort,
				Usage:  "Port to expose the debug api endpoint",
				Hidden: true,
			},

			&cli.StringFlag{
				Name:   "system-trace-endpoint",
				Usage:  "Endpoint for sending OTLP HTTP traces",
				Hidden: true,
			},
			&cli.StringFlag{
				Name:   "system-trace-url-path",
				Usage:  "URL path for sending OTLP HTTP traces",
				Hidden: true,
			},
		},
	}

	return cmd
}
