package devserver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/debugapi"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/urfave/cli/v3"
)

func action(ctx context.Context, cmd *cli.Command) error {
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

	portStr := localconfig.GetValue(cmd, "port", "8288")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
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
	connectGatewayPort := localconfig.GetIntValue(cmd, "connect-gateway-port", devserver.DefaultConnectGatewayPort)
	inMemory := localconfig.GetBoolValue(cmd, "in-memory", true)
	debugAPIPort := localconfig.GetIntValue(cmd, "debug-api-port", debugapi.DefaultDebugAPIPort)

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
		DebugAPIPort:       debugAPIPort,
	}

	err = devserver.New(ctx, opts)
	if err != nil {
		return err
	}
	return nil
}
