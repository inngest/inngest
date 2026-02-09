package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	grpcLib "google.golang.org/grpc"
)

const (
	DefaultConnectGatewayGRPCPort  = 50052
	DefaultConnectExecutorGRPCPort = 50053
	DefaultConnectGRPCIP           = "127.0.0.1"
)

type GatewayGRPCForwarder interface {
	ConnectToGateways(ctx context.Context) error
	Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error
}

type GatewayGRPCReceiver interface {
	Subscribe(ctx context.Context, requestID string) chan *connectpb.SDKResponse
	SubscribeWorkerAck(ctx context.Context, requestID string) chan *connectpb.AckMessage

	Unsubscribe(ctx context.Context, requestID string)
	UnsubscribeWorkerAck(ctx context.Context, requestID string)
}

type GatewayGRPCManager interface {
	GatewayGRPCForwarder
	GatewayGRPCReceiver
}

type gatewayGRPCManager struct {
	gatewayManager state.GatewayManager
	logger         logger.Logger

	// Request forwarding
	grpcClientManager *GRPCClientManager[connectpb.ConnectGatewayClient]
	dialer            GRPCDialer

	// Request receiver
	connectpb.ConnectExecutorServer
	grpcServer       *grpcLib.Server
	inFlightRequests sync.Map
	inFlightAcks     sync.Map

	gatewayGRPCPort  int
	executorGRPCPort int
}

type GRPCDialer func(target string, opts ...grpcLib.DialOption) (*grpcLib.ClientConn, error)

type GatewayGRPCManagerOption func(*gatewayGRPCManager)

func WithGatewayDialer(dialer GRPCDialer) GatewayGRPCManagerOption {
	return func(m *gatewayGRPCManager) {
		m.dialer = dialer
	}
}

func WithGatewayLogger(logger logger.Logger) GatewayGRPCManagerOption {
	return func(m *gatewayGRPCManager) {
		m.logger = logger
	}
}

func WithGatewayGRPCPort(p int) GatewayGRPCManagerOption {
	return func(m *gatewayGRPCManager) {
		// Keep using the default value if an invalid port is passed
		if p > 0 {
			m.gatewayGRPCPort = p
		}
	}
}

func WithExecutorGRPCPort(p int) GatewayGRPCManagerOption {
	return func(m *gatewayGRPCManager) {
		// Keep using the default value if an invalid port is passed
		if p > 0 {
			m.executorGRPCPort = p
		}
	}
}

// gatewayURL creates a URL for connecting to a gateway's gRPC port
func (i *gatewayGRPCManager) gatewayURL(ctx context.Context, gateway *state.Gateway) string {
	return net.JoinHostPort(gateway.IPAddress.String(), fmt.Sprintf("%d", i.gatewayGRPCPort))
}

func newGatewayGRPCManager(ctx context.Context, stateManager state.GatewayManager, opts ...GatewayGRPCManagerOption) GatewayGRPCManager {
	mgr := &gatewayGRPCManager{
		gatewayManager:   stateManager,
		dialer:           grpcLib.NewClient,
		grpcServer:       grpcLib.NewServer(),
		logger:           logger.StdlibLogger(ctx),
		gatewayGRPCPort:  DefaultConnectGatewayGRPCPort,
		executorGRPCPort: DefaultConnectExecutorGRPCPort,
	}

	for _, opt := range opts {
		opt(mgr)
	}

	var grpcOpts []GRPCClientManagerOption[connectpb.ConnectGatewayClient]
	if mgr.logger != nil {
		grpcOpts = append(grpcOpts, WithLogger[connectpb.ConnectGatewayClient](mgr.logger))
	}
	if mgr.dialer != nil {
		grpcOpts = append(grpcOpts, WithDialer[connectpb.ConnectGatewayClient](mgr.dialer))
	}
	mgr.grpcClientManager = NewGRPCClientManager(connectpb.NewConnectGatewayClient, grpcOpts...)

	connectpb.RegisterConnectExecutorServer(mgr.grpcServer, mgr)

	go mgr.startGarbageCollectClients(ctx)
	go mgr.gRPCServerListen(ctx)

	return mgr
}

func (i *gatewayGRPCManager) gRPCServerListen(ctx context.Context) {
	addr := fmt.Sprintf(":%d", i.executorGRPCPort)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		i.logger.Error("could not listen for grpc", "err", err, "addr", addr)
		return
	}

	i.logger.Info("starting executor grpc server", "addr", addr)
	err = i.grpcServer.Serve(l)
	if err != nil {
		i.logger.Error("could not serve for grpc", "err", err, "addr", addr)
		return
	}
}

func (i *gatewayGRPCManager) Reply(ctx context.Context, req *connectpb.ReplyRequest) (*connectpb.ReplyResponse, error) {
	if ch, ok := i.inFlightRequests.Load(req.Data.RequestId); ok {
		replyChan := ch.(chan *connectpb.SDKResponse)

		select {
		case replyChan <- req.Data:
			return &connectpb.ReplyResponse{Success: true}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			i.logger.Error("reply channel was closed")
			return &connectpb.ReplyResponse{Success: false}, nil
		}
	}

	i.logger.Error("reply channel has likely unsubscribed before getting a reply")
	return &connectpb.ReplyResponse{Success: false}, nil
}

func (i *gatewayGRPCManager) Ack(ctx context.Context, req *connectpb.AckMessage) (*connectpb.AckResponse, error) {
	if ch, ok := i.inFlightAcks.Load(req.RequestId); ok {
		ackChan := ch.(chan *connectpb.AckMessage)

		select {
		case ackChan <- req:
			return &connectpb.AckResponse{Success: true}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			i.logger.Error("ack channel was closed")
			return &connectpb.AckResponse{Success: false}, nil
		}
	}

	i.logger.Error("ack channel has likely unsubscribed before getting an ack")
	return &connectpb.AckResponse{Success: false}, nil
}

func (i *gatewayGRPCManager) Ping(ctx context.Context, req *connectpb.PingRequest) (*connectpb.PingResponse, error) {
	return &connectpb.PingResponse{Message: "ok"}, nil
}

func (i *gatewayGRPCManager) Subscribe(ctx context.Context, requestID string) chan *connectpb.SDKResponse {
	channel := make(chan *connectpb.SDKResponse)
	i.inFlightRequests.Store(requestID, channel)
	return channel
}

func (i *gatewayGRPCManager) SubscribeWorkerAck(ctx context.Context, requestID string) chan *connectpb.AckMessage {
	channel := make(chan *connectpb.AckMessage)
	i.inFlightAcks.Store(requestID, channel)
	return channel
}

func (i *gatewayGRPCManager) Unsubscribe(ctx context.Context, requestID string) {
	i.inFlightRequests.Delete(requestID)

	// NOTE: To avoid panics due to sending on a closed channel, we do not close the message channel
	// and instead let the gc reclaim it once no more goroutine is sending to it
}

func (i *gatewayGRPCManager) UnsubscribeWorkerAck(ctx context.Context, requestID string) {
	i.inFlightAcks.Delete(requestID)

	// NOTE: To avoid panics due to sending on a closed channel, we do not close the message channel
	// and instead let the gc reclaim it once no more goroutine is sending to it
}

// ConnectToGateways connects to all gateways through gRPC
func (i *gatewayGRPCManager) ConnectToGateways(ctx context.Context) error {
	gateways, err := i.gatewayManager.GetAllGateways(ctx)
	if err != nil {
		return err
	}

	logger.StdlibLogger(ctx).Debug("got connect gateways to connect to", "len", len(gateways))

	var successCount int64
	for _, g := range gateways {
		url := i.gatewayURL(ctx, g)
		_, err := i.grpcClientManager.GetOrCreateClient(ctx, g.Id.String(), url)
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not create grpc client", "error", err)

			metrics.IncrConnectGatewayGRPCClientFailureCounter(ctx, 1, metrics.CounterOpt{})
			continue
		}

		successCount++
		logger.StdlibLogger(ctx).Info("connected to connect gateway", "url", url)
	}

	metrics.IncrConnectGatewayGRPCClientCreateCounter(ctx, successCount, metrics.CounterOpt{
		Tags: map[string]any{"method": "connect-to-all"},
	})

	return nil
}

func (i *gatewayGRPCManager) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	grpcClient, err := i.grpcClientManager.GetClient(ctx, gatewayID.String())
	if err != nil && err != ErrGatewayNotFound {
		logger.StdlibLogger(ctx).Error("could not get grpc client", "gatewayID", gatewayID.String(), "err", err)
	}

	if grpcClient == nil {
		gateway, err := i.gatewayManager.GetGateway(ctx, gatewayID)
		if err != nil {
			return fmt.Errorf("could not find gateway %s: %w", gatewayID.String(), err)
		}

		url := i.gatewayURL(ctx, gateway)

		grpcClient, err = i.grpcClientManager.GetOrCreateClient(ctx, gatewayID.String(), url)
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not create grpc client", "gatewayID", gatewayID.String(), "err", err)

			metrics.IncrConnectGatewayGRPCClientFailureCounter(ctx, 1, metrics.CounterOpt{})
			return fmt.Errorf("could not find or create grpc client for gateway %s: %w", gatewayID.String(), err)
		}
	}

	reply, err := grpcClient.Forward(ctx, &connectpb.ForwardRequest{
		ConnectionID: connectionID.String(),
		Data:         data,
	})

	logger.StdlibLogger(ctx).Trace("grpc message forwarded to connect gateway", "reply", reply, "err", err)

	success := err == nil
	metrics.IncrConnectGatewayGRPCForwardCounter(ctx, 1, metrics.CounterOpt{
		Tags: map[string]any{"success": success},
	})

	return err
}

func (i *gatewayGRPCManager) startGarbageCollectClients(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _ = i.GarbageCollectClients()
		case <-ctx.Done():
			return
		}
	}
}

func (i *gatewayGRPCManager) GarbageCollectClients() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	existingGatewayIDs, err := i.gatewayManager.GetAllGatewayIDs(ctx)
	if err != nil {
		i.logger.Error("could not get connect gateways IDs")
		return 0, err
	}

	existingSet := make(map[string]bool, len(existingGatewayIDs))
	for _, id := range existingGatewayIDs {
		existingSet[id] = true
	}

	currentClientIDs := i.grpcClientManager.GetClientKeys()

	var deletedCount int
	for _, gatewayID := range currentClientIDs {
		if !existingSet[gatewayID] {
			i.grpcClientManager.RemoveClient(gatewayID)
			deletedCount++
		}
	}

	i.logger.Debug("cleaned up gRPC clients of dead connect gateways", "deleted", deletedCount)
	return deletedCount, nil
}
