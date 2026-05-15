package apiv2

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"google.golang.org/grpc"
)

const DefaultGRPCPort = 8290

const gracefulStopTimeout = 10 * time.Second

// NewGRPCService returns a long-running service that exposes the V2 API over
// gRPC on the given port. Auth middleware is adapted to gRPC interceptors via
// apiv2base.
func NewGRPCService(port int, serviceOpts ServiceOptions, httpOpts HTTPHandlerOptions) service.Service {
	if port == 0 {
		port = DefaultGRPCPort
	}
	base := apiv2base.NewBase()
	return &grpcService{
		port:   port,
		server: NewGRPCServerFromHTTPOptions(serviceOpts, httpOpts, base),
	}
}

type grpcService struct {
	port   int
	server *grpc.Server
}

func (g *grpcService) Name() string { return "api-grpc" }

func (g *grpcService) Pre(ctx context.Context) error { return nil }

func (g *grpcService) Run(ctx context.Context) error {
	log := logger.StdlibLogger(ctx)
	addr := fmt.Sprintf(":%d", g.port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("could not listen for api grpc", "error", err, "addr", addr)
		return err
	}

	log.Info("starting api grpc", "addr", addr)

	errCh := make(chan error, 1)
	go func() { errCh <- g.server.Serve(l) }()

	select {
	case <-ctx.Done():
		g.stopWithTimeout()
		return nil
	case err := <-errCh:
		return err
	}
}

func (g *grpcService) Stop(ctx context.Context) error {
	g.stopWithTimeout()
	return nil
}

func (g *grpcService) stopWithTimeout() {
	done := make(chan struct{})
	go func() {
		g.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(gracefulStopTimeout):
		g.server.Stop()
	}
}
