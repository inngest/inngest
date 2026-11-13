// Package server has the flag handling, tracer setup, and signal handling
// that `inngest dev` and `inngest start` share.
package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/config"
	connectconfig "github.com/inngest/inngest/pkg/config/connect"
	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/devserver"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/urfave/cli/v3"
)

// SetAddress sets the address of the event API and the core API from --host
// and --port.
func SetAddress(cmd *cli.Command, conf *config.Config) {
	port := localconfig.GetIntValue(cmd, "port", api.DefaultAPIPort)
	conf.EventAPI.Port = port
	conf.CoreAPI.Port = port

	if host := localconfig.GetValue(cmd, "host", ""); host != "" {
		conf.EventAPI.Addr = host
		conf.CoreAPI.Addr = host
	}
}

// ConnectGRPCConfig returns the gRPC config of the connect gateway and the
// connect executor from the connect-*-grpc-ip and connect-*-grpc-port flags.
func ConnectGRPCConfig(ctx context.Context, cmd *cli.Command) (connectconfig.ConnectGRPCConfig, error) {
	gatewayIP := localconfig.GetValue(cmd, "connect-gateway-grpc-ip", connectgrpc.DefaultConnectGRPCIP)
	executorIP := localconfig.GetValue(cmd, "connect-executor-grpc-ip", connectgrpc.DefaultConnectGRPCIP)
	if err := localconfig.ValidateConnectGRPCIPs(gatewayIP, executorIP); err != nil {
		return connectconfig.ConnectGRPCConfig{}, err
	}

	return connectconfig.NewGRPCConfig(
		ctx,
		gatewayIP,
		localconfig.GetIntValue(cmd, "connect-gateway-grpc-port", devserver.DefaultConnectGatewayGRPCPort),
		executorIP,
		localconfig.GetIntValue(cmd, "connect-executor-grpc-port", devserver.DefaultConnectExecutorGRPCPort),
	), nil
}

// SetupTracers sends the user spans and the system spans of the process to
// the server on port over OTLP HTTP.  systemEndpoint and systemURLPath send
// the system spans elsewhere when they are not empty.  the tracers are
// process-wide and can be set once, so a tracer read before SetupTracers stays
// a no-op.  the returned function closes both tracers.
func SetupTracers(ctx context.Context, port int, systemEndpoint, systemURLPath string) (func(), error) {
	endpoint := fmt.Sprintf("localhost:%d", port)
	if err := itrace.NewUserTracer(ctx, itrace.TracerOpts{
		ServiceName:   "tracing",
		TraceEndpoint: endpoint,
		TraceURLPath:  "/dev/traces",
		Type:          itrace.TracerTypeOTLPHTTP,
	}); err != nil {
		return nil, fmt.Errorf("could not set up the user tracer: %w", err)
	}

	if systemEndpoint == "" {
		systemEndpoint = endpoint
	}
	if systemURLPath == "" {
		systemURLPath = "/dev/traces/system"
	}
	if err := itrace.NewSystemTracer(ctx, itrace.TracerOpts{
		ServiceName:   "tracing-system",
		TraceEndpoint: systemEndpoint,
		TraceURLPath:  systemURLPath,
		Type:          itrace.TracerTypeOTLPHTTP,
	}); err != nil {
		_ = itrace.CloseUserTracer(ctx)
		return nil, fmt.Errorf("could not set up the system tracer: %w", err)
	}

	return func() {
		_ = itrace.CloseSystemTracer(ctx)
		_ = itrace.CloseUserTracer(ctx)
	}, nil
}

// ExitAfterSignals exits the process with code 0 when it receives the
// count-th interrupt, terminate, or quit signal.  devserver.New also receives
// each signal, and stops the server gracefully on the first one.  so a count
// of 2 lets a second signal end a slow stop.
func ExitAfterSignals(count int) {
	signals := make(chan os.Signal, count)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for range count {
			<-signals
		}
		os.Exit(0)
	}()
}
