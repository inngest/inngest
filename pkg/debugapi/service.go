package debugapi

import (
	"context"
	"fmt"
	"net"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc"
)

func NewDebugAPI(o Opts) service.Service {
	return &debugAPI{
		rpc:       grpc.NewServer(),
		log:       o.Log,
		db:        o.DB,
		queue:     o.Queue,
		state:     o.State,
		croner:    o.Cron,
		findShard: o.ShardSelector,
	}
}

type Opts struct {
	Log   logger.Logger
	DB    cqrs.Manager
	Queue redis_state.QueueManager
	State state.Manager
	Cron  cron.CronManager

	ShardSelector redis_state.ShardSelector
}

type debugAPI struct {
	pb.DebugServer

	rpc       *grpc.Server
	log       logger.Logger
	findShard redis_state.ShardSelector

	db     cqrs.Manager
	queue  redis_state.QueueManager
	state  state.Manager
	croner cron.CronManager
}

func (d *debugAPI) Name() string {
	return "debug-api-dev"
}

func (d *debugAPI) Pre(ctx context.Context) error {
	pb.RegisterDebugServer(d.rpc, d)

	return nil
}

func (d *debugAPI) Run(ctx context.Context) error {
	// TODO: make the port overridable
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
