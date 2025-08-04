package start

import (
	"github.com/inngest/inngest/pkg/devserver"
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
