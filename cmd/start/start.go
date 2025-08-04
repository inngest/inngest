package start

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/authn"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/urfave/cli/v3"
)

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

	tick := localconfig.GetIntValue(cmd, "tick", devserver.DefaultTick)

	signingKey := localconfig.GetValue(cmd, "signing-key", "")
	if signingKey == "" {
		fmt.Println("Error: signing-key is required")
		os.Exit(1)
	}
	_, err = authn.HashedSigningKey(signingKey)
	if err != nil {
		fmt.Printf("Error: signing-key must be a valid hexadecimal string\n")
		os.Exit(1)
	}

	eventKeys := localconfig.GetStringSlice(cmd, "event-key")
	if len(eventKeys) == 0 {
		fmt.Println("Error: at least one event-key is required")
		os.Exit(1)
	}

	conf.ServerKind = headers.ServerKindCloud

	// Handle configuration options with simplified koanf-based approach
	postgresURI := localconfig.GetValue(cmd, "postgres-uri", "")
	redisURI := localconfig.GetValue(cmd, "redis-uri", "")
	sqliteDir := localconfig.GetValue(cmd, "sqlite-dir", "")
	sdkURLs := localconfig.GetStringSlice(cmd, "sdk-url")

	opts := devserver.StartOpts{
		Config:             *conf,
		ConnectGatewayHost: conf.CoreAPI.Addr,
		ConnectGatewayPort: localconfig.GetIntValue(cmd, "connect-gateway-port", devserver.DefaultConnectGatewayPort),
		EventKeys:          eventKeys,
		InMemory:           false,
		NoUI:               localconfig.GetBoolValue(cmd, "no-ui", false),
		PollInterval:       localconfig.GetIntValue(cmd, "poll-interval", devserver.DefaultPollInterval),
		PostgresURI:        postgresURI,
		QueueWorkers:       localconfig.GetIntValue(cmd, "queue-workers", devserver.DefaultQueueWorkers),
		RedisURI:           redisURI,
		RequireKeys:        true,
		RetryInterval:      localconfig.GetIntValue(cmd, "retry-interval", 0),
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
