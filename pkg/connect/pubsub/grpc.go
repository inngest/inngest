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

type gatewayGRPCForwarder struct {
	gatewayManager state.GatewayManager
	mu             sync.RWMutex

	grpcClients map[string]connectpb.ConnectGatewayClient
	dialer      GRPCDialer
}

type GRPCDialer func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)

// gatewayURL creates a URL for connecting to a gateway's gRPC port
func gatewayURL(ctx context.Context, gateway *state.Gateway) string {
	return net.JoinHostPort(gateway.IPAddress.String(), fmt.Sprintf("%d", connectConfig.Gateway(ctx).GRPCPort))
}

func NewGatewayGRPCForwarder(ctx context.Context, stateManager state.GatewayManager) GatewayGRPCForwarder {
	return NewGatewayGRPCForwarderWithDialer(ctx, stateManager, grpc.NewClient)
}

func NewGatewayGRPCForwarderWithDialer(ctx context.Context, stateManager state.GatewayManager, dialer GRPCDialer) GatewayGRPCForwarder {
	forwarder := &gatewayGRPCForwarder{
		gatewayManager: stateManager,
		grpcClients:    map[string]connectpb.ConnectGatewayClient{},
		dialer:         dialer,
	}

	go forwarder.startGarbageCollectClients(ctx)

	return forwarder
}

// createGRPCClient creates a gRPC client for a gateway and validates the connection
func (i *gatewayGRPCForwarder) createGRPCClient(ctx context.Context, gateway *state.Gateway) (connectpb.ConnectGatewayClient, error) {
	url := gatewayURL(ctx, gateway)

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
func (i *gatewayGRPCForwarder) ConnectToGateways(ctx context.Context) error {
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
func (i *gatewayGRPCForwarder) connectToGateway(ctx context.Context, gatewayID ulid.ULID) (connectpb.ConnectGatewayClient, error) {
	gateway, err := i.gatewayManager.GetGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("could not find gateway %s: %w", gatewayID.String(), err)
	}

	rpcClient, err := i.createGRPCClient(ctx, gateway)
	if err != nil {
		return nil, fmt.Errorf("could not create grpc client for gateway %s: %w", gatewayID.String(), err)
	}

	i.mu.Lock()
	i.grpcClients[gatewayID.String()] = rpcClient
	i.mu.Unlock()

	url := gatewayURL(ctx, gateway)
	logger.StdlibLogger(ctx).Info("just-in-time connected to connect gateway", "url", url)

	metrics.IncrConnectGatewayGRPCClientCreateCounter(ctx, int64(1), metrics.CounterOpt{
		Tags: map[string]any{"method": "just-in-time"},
	})

	return rpcClient, nil
}

func (i *gatewayGRPCForwarder) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	i.mu.RLock()
	grpcClient := i.grpcClients[gatewayID.String()]
	i.mu.RUnlock()

	if grpcClient == nil {
		logger.StdlibLogger(ctx).Warn("grpc client not found for gateway, attempting to create dynamically", "gatewayID", gatewayID.String())

		var err error
		grpcClient, err = i.connectToGateway(ctx, gatewayID)
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not create just-in-time grpc client", "gatewayID", gatewayID.String(), "err", err)

			metrics.IncrConnectGatewayGRPCClientFailureCounter(ctx, 1, metrics.CounterOpt{})
			return fmt.Errorf("could not find or create grpc client for gateway %s: %w", gatewayID.String(), err)
		}
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

func (i *gatewayGRPCForwarder) startGarbageCollectClients(ctx context.Context) {
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

func (i *gatewayGRPCForwarder) GarbageCollectClients() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	existingGatewayIDs, err := i.gatewayManager.GetAllGatewayIDs(ctx)
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not get connect gateways IDs")
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

	logger.StdlibLogger(ctx).Debug("cleaned up gRPC clients of dead connect gateways", "deleted", deletedCount)
	return deletedCount, nil
}
