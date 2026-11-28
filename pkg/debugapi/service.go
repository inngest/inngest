package debugapi

import (
	"context"
	"fmt"
	"net"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc"
)

const DefaultDebugAPIPort = 7777

// NewDebugAPI creates the gRPC server of the debug API.  it listens on
// o.Port, or on DefaultDebugAPIPort when o.Port is 0.
func NewDebugAPI(o Opts) *Server {
	port := DefaultDebugAPIPort
	if o.Port != 0 {
		port = o.Port
	}

	rpc := grpc.NewServer()
	pb.RegisterDebugServer(rpc, &debugAPI{
		log:           o.Log,
		db:            o.DB,
		runReader:     o.RunReader,
		itemReader:    o.ItemReader,
		partReader:    o.PartitionReader,
		backlogReader: o.BacklogReader,
		state:         o.State,
		croner:        o.Cron,
		shards:        o.ShardRegistry,
		pm:            o.PauseManager,
		cm:            o.CapacityManager,
		sm:            o.SemaphoreManager,
		batchManager:  o.BatchManager,
		debouncer:     o.Debouncer,
	})
	return &Server{
		rpc:  rpc,
		port: port,
		log:  o.Log,
	}
}

type Opts struct {
	Log              logger.Logger
	DB               cqrs.Manager
	RunReader        queue.RunQueueReader
	ItemReader       queue.QueueItemReader
	PartitionReader  queue.QueuePartitionReader
	BacklogReader    queue.QueueBacklogReader
	State            state.Manager
	Cron             cron.CronManager
	PauseManager     pauses.Manager
	CapacityManager  constraintapi.CapacityManager
	SemaphoreManager constraintapi.SemaphoreManager

	ShardRegistry queue.ShardRegistry

	// Dependencies for batching and debounce insights
	BatchManager batch.BatchManager
	Debouncer    debounce.Debouncer

	Port int
}

// Server serves the debug API over gRPC.
type Server struct {
	rpc  *grpc.Server
	port int
	log  logger.Logger
}

type debugAPI struct {
	pb.DebugServer

	log    logger.Logger
	shards queue.ShardRegistry

	db            cqrs.Manager
	runReader     queue.RunQueueReader
	itemReader    queue.QueueItemReader
	partReader    queue.QueuePartitionReader
	backlogReader queue.QueueBacklogReader
	state         state.Manager
	croner        cron.CronManager
	pm            pauses.Manager
	cm            constraintapi.CapacityManager
	sm            constraintapi.SemaphoreManager

	// Dependencies for batching and debounce insights
	batchManager batch.BatchManager
	debouncer    debounce.Debouncer
}

// Run serves the debug API until ctx is done.
func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Error("could not listen on port for debug api", "error", err, "addr", addr)
		return err
	}

	// Serve returns when the server stops.  without this, Run does not
	// return when ctx is done, and the stop of the app waits for it.
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			s.rpc.GracefulStop()
		case <-stopped:
		}
	}()

	s.log.Info("starting debug api", "addr", addr)
	if err := s.rpc.Serve(l); err != nil {
		s.log.Error("error serving debug api", "error", err, "addr", addr)
		return err
	}
	return nil
}
