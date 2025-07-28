package debugapi

import (
	"context"
	"fmt"
	"net"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc"
)

var (
	errNotImplemented = fmt.Errorf("not implemented")
)

func NewDebugAPI(o Opts) service.Service {
	return &debugAPI{
		Router: chi.NewRouter(),
		rpc:    grpc.NewServer(),
		Opts:   o,
		log:    logger.StdlibLogger(context.Background()),
	}
}

type Opts struct {
	Log   logger.Logger
	Queue redis_state.QueueManager

	ShardSelector redis_state.ShardSelector
}

type debugAPI struct {
	chi.Router
	Opts

	rpc *grpc.Server
	log logger.Logger
}

func (d *debugAPI) Name() string {
	return "debug-api"
}

func (d *debugAPI) Pre(ctx context.Context) error {
	d.Use(middleware.AllowContentType("application/json"))
	d.Use(headers.ContentTypeJsonResponse())

	d.Route("/queue", func(r chi.Router) {
		r.Get("/partitions/{id}", d.partitionByID)
	})

	pb.RegisterDebugServer(d.rpc, d)

	return nil
}

func (d *debugAPI) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", 7777)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		d.log.Error("could not listen on port for debug api", "error", err, "addr", addr)
		return err
	}

	d.log.Info("starting debug api", "addr", addr)
	err = d.rpc.Serve(l)
	if err != nil {
		d.log.Error("error serving debug api", "error", err, "addr", addr)
		return err
	}

	return nil
}

func (d *debugAPI) Stop(ctx context.Context) error {
	d.rpc.GracefulStop() // stop rpc server
	return nil
}
