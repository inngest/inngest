package debugapi

import (
	"context"
	"fmt"
	"net"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc"
)

const DefaultDebugAPIPort = 7777

func NewDebugAPI(o Opts) service.Service {
	port := DefaultDebugAPIPort
	if o.Port != 0 {
		port = o.Port
	}

	return &debugAPI{
		rpc:            grpc.NewServer(),
		port:           port,
		log:            o.Log,
		db:             o.DB,
		queue:          o.Queue,
		state:          o.State,
		croner:         o.Cron,
		findShard:      o.ShardSelector,
		pm:             o.PauseManager,
		cm:             o.CapacityManager,
		batchClient:    o.BatchClient,
		singletonStore: o.SingletonStore,
		debounceClient: o.DebounceClient,
	}
}

type Opts struct {
	Log             logger.Logger
	DB              cqrs.Manager
	Queue           queue.QueueManager
	State           state.Manager
	Cron            cron.CronManager
	PauseManager    pauses.Manager
	CapacityManager constraintapi.CapacityManager

	ShardSelector queue.ShardSelector

	// New dependencies for batching, singleton, and debounce insights
	BatchClient    *redis_state.BatchClient
	SingletonStore singleton.Singleton
	DebounceClient *redis_state.DebounceClient

	Port int
}

type debugAPI struct {
	pb.DebugServer
	port int

	rpc       *grpc.Server
	log       logger.Logger
	findShard queue.ShardSelector

	shards map[string]queue.QueueShard

	db     cqrs.Manager
	queue  queue.QueueManager
	state  state.Manager
	croner cron.CronManager
	pm     pauses.Manager
	cm     constraintapi.CapacityManager

	// New dependencies for batching, singleton, and debounce insights
	batchClient    *redis_state.BatchClient
	singletonStore singleton.Singleton
	debounceClient *redis_state.DebounceClient
}

func (d *debugAPI) Name() string {
	return "debug-api-dev"
}

func (d *debugAPI) Pre(ctx context.Context) error {
	pb.RegisterDebugServer(d.rpc, d)

	return nil
}

func (d *debugAPI) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", d.port)

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
