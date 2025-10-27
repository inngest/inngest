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
			&cli.IntFlag{
				Name:  "poll-interval",
				Value: devserver.DefaultPollInterval,
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
			&cli.IntFlag{
				Name:  "connect-gateway-grpc-port",
				Value: devserver.DefaultConnectGatewayGRPCPort,
				Usage: "Port to expose connect gateway grpc endpoint",
			},
			&cli.IntFlag{
				Name:  "connect-executor-grpc-port",
				Value: devserver.DefaultConnectExecutorGRPCPort,
				Usage: "Port to expose connect executor grpc endpoint",
			},
			&cli.BoolFlag{
				Name:   "in-memory",
				Value:  true,
				Usage:  "Use in memory sqlite db",
				Hidden: true,
			},
			&cli.IntFlag{
				Name:  "debug-api-port",
				Value: devserver.DefaultDebugAPIPort,
				Usage: "Port to expose the debug api endpoint",
			},
		},
	}

	return cmd
}
