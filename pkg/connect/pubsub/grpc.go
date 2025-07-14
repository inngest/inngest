package pubsub

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	mu          sync.RWMutex
	grpcClients map[string]connectpb.ConnectGatewayClient
	dialer      GRPCDialer

	// Request receiver
	connectpb.ConnectExecutorServer
	grpcServer       *grpc.Server
	inFlightRequests sync.Map
	inFlightAcks     sync.Map
}

type GRPCDialer func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)

// gatewayURL creates a URL for connecting to a gateway's gRPC port
func gatewayURL(ctx context.Context, gateway *state.Gateway) string {
	return net.JoinHostPort(gateway.IPAddress.String(), fmt.Sprintf("%d", connectConfig.Gateway(ctx).GRPCPort))
}

func NewGatewayGRPCManager(ctx context.Context, stateManager state.GatewayManager, logger logger.Logger) GatewayGRPCManager {
	return NewGatewayGRPCManagerWithDialer(ctx, stateManager, grpc.NewClient, logger)
}

func NewGatewayGRPCManagerWithDialer(ctx context.Context, stateManager state.GatewayManager, dialer GRPCDialer, logger logger.Logger) GatewayGRPCManager {
	forwarder := &gatewayGRPCManager{
		gatewayManager: stateManager,
		logger:         logger,

		grpcClients: map[string]connectpb.ConnectGatewayClient{},
		dialer:      dialer,

		grpcServer: grpc.NewServer(),
	}

	connectpb.RegisterConnectExecutorServer(forwarder.grpcServer, forwarder)

	go forwarder.startGarbageCollectClients(ctx)
	go forwarder.gRPCServerListen(ctx)

	return forwarder
}

func (i *gatewayGRPCManager) gRPCServerListen(ctx context.Context) {
	addr := fmt.Sprintf(":%d", connectConfig.Executor(ctx).GRPCPort)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		i.logger.Error("could not listen for: %w", err)
		return
	}

	i.logger.Info("starting executor grpc server", "addr", addr)
	i.grpcServer.Serve(l)
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
	key := fmt.Sprintf("worker_requests_ack:%s", req.RequestId)

	if ch, ok := i.inFlightAcks.Load(key); ok {
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

func (i *gatewayGRPCManager) Subscribe(ctx context.Context, requestID string) chan *connectpb.SDKResponse {
	channel := make(chan *connectpb.SDKResponse)
	i.inFlightRequests.Store(requestID, channel)
	return channel
}

func (i *gatewayGRPCManager) SubscribeWorkerAck(ctx context.Context, requestID string) chan *connectpb.AckMessage {
	key := fmt.Sprintf("worker_requests_ack:%s", requestID)

	channel := make(chan *connectpb.AckMessage)
	i.inFlightAcks.Store(key, channel)
	return channel
}

func (i *gatewayGRPCManager) Unsubscribe(ctx context.Context, requestID string) {
	ch, loaded := i.inFlightRequests.LoadAndDelete(requestID)
	if loaded {
		replyChan := ch.(chan *connectpb.SDKResponse)
		close(replyChan)
	}
}

func (i *gatewayGRPCManager) UnsubscribeWorkerAck(ctx context.Context, requestID string) {
	key := fmt.Sprintf("worker_requests_ack:%s", requestID)

	ch, loaded := i.inFlightAcks.LoadAndDelete(key)
	if loaded {
		replyChan := ch.(chan *connectpb.AckMessage)
		close(replyChan)
	}
}

// createGRPCClient creates a gRPC client for a gateway and validates the connection
func (i *gatewayGRPCManager) createGRPCClient(ctx context.Context, gateway *state.Gateway) (connectpb.ConnectGatewayClient, error) {
	url := fmt.Sprintf("%s:%d", gateway.IPAddress, connectConfig.Gateway(ctx).GRPCPort)

	var conn *grpc.ClientConn
	var err error

	if i.dialer == nil {
		return nil, fmt.Errorf("gateway dialer is nil")
	}

	conn, err = i.dialer(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not create grpc client for %s: %w", url, err)
	}

	rpcClient := connectpb.NewConnectGatewayClient(conn)

	// grpc.NewClient doesn't establish a connection immediately; it connects on the first RPC call.
	// Ping is called to eagerly validate that the connection is working. This can be removed later if not needed.
	result, err := rpcClient.Ping(ctx, &connectpb.PingRequest{})
	if err != nil {
		return nil, fmt.Errorf("could not ping gateway %s: %w", url, err)
	}

	if result.GetMessage() != "ok" {
		return nil, fmt.Errorf("unexpected connect gateway ping response: %s", result.GetMessage())
	}

	return rpcClient, nil
}

// ConnectToGateways connects to all gateways through gRPC
func (i *gatewayGRPCManager) ConnectToGateways(ctx context.Context) error {
	gateways, err := i.gatewayManager.GetAllGateways(ctx)
	if err != nil {
		return err
	}

	logger.StdlibLogger(ctx).Debug("got connect gateways to connect to", "len", len(gateways))

	i.mu.Lock()
	defer i.mu.Unlock()

	i.grpcClients = map[string]connectpb.ConnectGatewayClient{}

	for _, g := range gateways {
		rpcClient, err := i.createGRPCClient(ctx, g)
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not create grpc client", "error", err)

			metrics.IncrConnectGatewayGRPCClientFailureCounter(ctx, 1, metrics.CounterOpt{})
			continue
		}

		i.grpcClients[g.Id.String()] = rpcClient
		url := gatewayURL(ctx, g)
		logger.StdlibLogger(ctx).Info("connected to connect gateway", "url", url)
	}

	metrics.IncrConnectGatewayGRPCClientCreateCounter(ctx, int64(len(i.grpcClients)), metrics.CounterOpt{
		Tags: map[string]any{"method": "connect-to-all"},
	})

	return nil
}

// connectToGateway attempts to create a new gRPC client for a gateway that wasn't doesn't have a grpc client yet.
func (i *gatewayGRPCManager) connectToGateway(ctx context.Context, gatewayID ulid.ULID) (connectpb.ConnectGatewayClient, error) {
	gateway, err := i.gatewayManager.GetGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("could not find gateway %s: %w", gatewayID.String(), err)
	}

	rpcClient, err := i.createGRPCClient(ctx, gateway)
	if err != nil {
		return nil, fmt.Errorf("could not create grpc client for gateway %s: %w", gatewayID.String(), err)
	}

	// Mutex lock should have been acquired before calling the current function
	i.grpcClients[gatewayID.String()] = rpcClient

	url := gatewayURL(ctx, gateway)
	logger.StdlibLogger(ctx).Info("just-in-time connected to connect gateway", "url", url)

	metrics.IncrConnectGatewayGRPCClientCreateCounter(ctx, int64(1), metrics.CounterOpt{
		Tags: map[string]any{"method": "just-in-time"},
	})

	return rpcClient, nil
}

func (i *gatewayGRPCManager) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	var grpcClient connectpb.ConnectGatewayClient

	i.mu.RLock()
	grpcClient = i.grpcClients[gatewayID.String()]
	i.mu.RUnlock()

	if grpcClient == nil {
		// Upgrade lock to make sure that only one instance is creating a grpc client
		i.mu.Lock()
		grpcClient = i.grpcClients[gatewayID.String()]

		if grpcClient == nil {

			logger.StdlibLogger(ctx).Warn("grpc client not found for gateway, attempting to create dynamically", "gatewayID", gatewayID.String())

			var err error
			grpcClient, err = i.connectToGateway(ctx, gatewayID)
			if err != nil {
				logger.StdlibLogger(ctx).Error("could not create just-in-time grpc client", "gatewayID", gatewayID.String(), "err", err)

				metrics.IncrConnectGatewayGRPCClientFailureCounter(ctx, 1, metrics.CounterOpt{})
				i.mu.Unlock()
				return fmt.Errorf("could not find or create grpc client for gateway %s: %w", gatewayID.String(), err)
			}
		}
		i.mu.Unlock()
	}
	reply, err := grpcClient.Forward(ctx, &connectpb.ForwardRequest{
		ConnectionID: connectionID.String(),
		Data:         data,
	})

	logger.StdlibLogger(ctx).Debug("grpc message forwarded to connect gateway", "reply", reply, "err", err)

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

	i.mu.Lock()
	defer i.mu.Unlock()

	var deletedCount int

	for gatewayId := range i.grpcClients {
		if !existingSet[gatewayId] {
			delete(i.grpcClients, gatewayId)
			deletedCount++
		}
	}

	i.logger.Debug("cleaned up gRPC clients of dead connect gateways", "deleted", deletedCount)
	return deletedCount, nil
}
