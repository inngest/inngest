package devserver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/inngest/inngest/cmd/internal/envflags"
	"github.com/inngest/inngest/cmd/internal/localconfig"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	cmd := &cli.Command{
		Name:        "dev",
		Usage:       "Run the Inngest Dev Server for local development.",
		UsageText:   "inngest dev [options]",
		Description: "Example: inngest dev -u http://localhost:3000/api/inngest",
		Action:      doDev,

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
			&cli.BoolFlag{
				Name:   "in-memory",
				Value:  true,
				Usage:  "Use in memory sqlite db",
				Hidden: true,
			},
		},
	}

	return cmd
}

func doDev(ctx context.Context, cmd *cli.Command) error {

	go func() {
		ctx, cleanup := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
			syscall.SIGINT,
			syscall.SIGQUIT,
		)
		defer cleanup()
		<-ctx.Done()
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

	config := localconfig.GetConfig()

	portStr := envflags.GetEnvOrFlagWithDefault(cmd, "port", "INNGEST_PORT", "8288")
	// Fallback to config file value if no env var and using default
	if !cmd.IsSet("port") && os.Getenv("INNGEST_PORT") == "" && config.Port != "" {
		portStr = config.Port
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	conf.EventAPI.Port = port
	conf.CoreAPI.Port = port

	host := envflags.GetEnvOrFlag(cmd, "host", "INNGEST_HOST")
	// Fallback to config file value if no CLI flag or env var is set
	if host == "" && config.Host != "" {
		host = config.Host
	}
	if host != "" {
		conf.EventAPI.Addr = host
		conf.CoreAPI.Addr = host
	}

	urls := envflags.GetEnvOrStringSlice(cmd, "sdk-url", "INNGEST_SDK_URL")
	// Fallback to config file values if no CLI flags or env vars are set
	if len(urls) == 0 && len(config.SdkURL) > 0 {
		urls = config.SdkURL
	}

	// Run auto-discovery unless we've explicitly disabled it.
	// Priority: CLI flag (if explicitly set) > koanf config (env vars + config file) > CLI default
	noDiscovery := cmd.Bool("no-discovery")
	if !cmd.IsSet("no-discovery") && config.NoDiscovery != nil {
		noDiscovery = *config.NoDiscovery
	}

	noPoll := cmd.Bool("no-poll")
	if !cmd.IsSet("no-poll") && config.NoPoll != nil {
		noPoll = *config.NoPoll
	}
	pollInterval := cmd.Int("poll-interval")
	retryInterval := cmd.Int("retry-interval")
	queueWorkers := cmd.Int("queue-workers")
	tick := cmd.Int("tick")
	connectGatewayPort := cmd.Int("connect-gateway-port")
	inMemory := cmd.Bool("in-memory")

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
		InMemory:           inMemory,
	}

	err = devserver.New(ctx, opts)
	if err != nil {
		return err
	}
	return nil
}
