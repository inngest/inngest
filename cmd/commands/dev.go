package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/inngest/inngest/cmd/commands/internal/localconfig"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func NewCmdDev(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run the Inngest Dev Server for local development.",
		Example: "inngest dev -u http://localhost:3000/api/inngest",
		Run:     doDev,
	}

	groups := []FlagGroup{}

	baseFlags := pflag.NewFlagSet("base", pflag.ExitOnError)
	baseFlags.StringSliceP("sdk-url", "u", []string{}, "App serve URLs to sync (ex. http://localhost:3000/api/inngest)")
	baseFlags.Bool("no-discovery", false, "Disable app auto-discovery")
	baseFlags.Bool("no-poll", false, "Disable polling of apps for updates")
	baseFlags.String("config", "", "Path to an Inngest configuration file")
	baseFlags.String("host", "", "Inngest server host")
	baseFlags.StringP("port", "p", "8288", "Inngest server port")
	baseFlags.BoolP("help", "h", false, "Output this help information")
	cmd.Flags().AddFlagSet(baseFlags)
	groups = append(groups, FlagGroup{name: "Flags:", fs: baseFlags})

	advancedFlags := pflag.NewFlagSet("advanced", pflag.ExitOnError)
	advancedFlags.Int("poll-interval", devserver.DefaultPollInterval, "Interval in seconds between polling for updates to apps")
	advancedFlags.Int("retry-interval", 0, "Retry interval in seconds for linear backoff when retrying functions - must be 1 or above")
	advancedFlags.Int("queue-workers", devserver.DefaultQueueWorkers, "Number of executor workers to execute steps from the queue")
	advancedFlags.Int("tick", devserver.DefaultTick, "The interval (in milliseconds) at which the executor polls the queue")
	advancedFlags.Int("connect-gateway-port", devserver.DefaultConnectGatewayPort, "Port to expose connect gateway endpoint")
	advancedFlags.Bool("in-memory", true, "Use in memory sqlite db")
	err := advancedFlags.MarkHidden("in-memory")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cmd.Flags().AddFlagSet(advancedFlags)
	groups = append(groups, FlagGroup{name: "Advanced Flags:", fs: advancedFlags})

	groups = append(groups, FlagGroup{name: "Global Flags:", fs: rootCmd.PersistentFlags()})

	cmd.SetUsageFunc(func(c *cobra.Command) error {
		fmt.Printf("%s\n  %s\n\n%s\n%s\n\n",
			"Usage:",
			"inngest dev [flags]",
			"Examples:",
			cmd.Example,
		)

		for _, group := range groups {
			usage := group.fs.FlagUsages()

			help := ""
			if group.name != "" {
				help = help + group.name + "\n"
			}
			help = help + usage
			fmt.Println(help)
		}

		return nil
	})

	return cmd
}

func doDev(cmd *cobra.Command, args []string) {

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

	ctx := cmd.Context()
	conf, err := config.Dev(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err = localconfig.InitDevConfig(ctx, cmd); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	port, err := strconv.Atoi(viper.GetString("port"))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	conf.EventAPI.Port = port
	conf.CoreAPI.Port = port

	host := viper.GetString("host")
	if host != "" {
		conf.EventAPI.Addr = host
		conf.CoreAPI.Addr = host
	}

	urls := viper.GetStringSlice("sdk-url")

	// Run auto-discovery unless we've explicitly disabled it.
	noDiscovery := viper.GetBool("no-discovery")
	noPoll := viper.GetBool("no-poll")
	pollInterval := viper.GetInt("poll-interval")
	retryInterval := viper.GetInt("retry-interval")
	queueWorkers := viper.GetInt("queue-workers")
	tick := viper.GetInt("tick")
	connectGatewayPort := viper.GetInt("connect-gateway-port")
	inMemory := viper.GetBool("in-memory")

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
		fmt.Println(err)
		os.Exit(1)
	}
}
