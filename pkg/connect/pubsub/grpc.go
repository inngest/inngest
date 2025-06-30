package pubsub

import (
	"context"
	"fmt"
	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
)

type GatewayGrpcForwarder interface {
	ConnectToGateways(ctx context.Context) error
	Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error
}

type gatewayGrpcForwarder struct {
	gatewayManager state.GatewayManager
	mu             sync.RWMutex

	// TODO: Cleanup old gateways
	grpcClients    map[string]connectpb.ConnectGatewayClient
}

func NewGatewayGrpcForwarder(stateManager state.GatewayManager) GatewayGrpcForwarder {
	return &gatewayGrpcForwarder{
		gatewayManager: stateManager,
		grpcClients:    map[string]connectpb.ConnectGatewayClient{},
	}
}

// Connect to all gateways through gRPC
func (i *gatewayGrpcForwarder) ConnectToGateways(ctx context.Context) error {
	gateways, err := i.gatewayManager.GetAllGateways(ctx)
	if err != nil {
		return err
	}

	logger.StdlibLogger(ctx).Debug("got connect gateways to connect to", "len", len(gateways))

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, g := range gateways {
		url := fmt.Sprintf("%s:%d", g.IPAddress, connectConfig.Gateway(ctx).GRPCPort)
		conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not create grpc client", err)
			continue
		}

		rpcClient := connectpb.NewConnectGatewayClient(conn)

		// grpc.NewClient doesn't establish a connection immediately; it connects on the first RPC call.
		// Ping is called to eagerly validate that the connection is working. This can be removed later if not needed.
		result, err := rpcClient.Ping(ctx, &connectpb.PingRequest{})
		message := result.GetMessage()
		if err != nil || message != "ok" {
			logger.StdlibLogger(ctx).Error("could not ping connect gateway at startup", "url", url, "message", message, "err", err)
		} else {
			i.grpcClients[g.Id.String()] = rpcClient
			logger.StdlibLogger(ctx).Info("connected to connect gateway", "message", message, "url", url)
		}
	}

	return nil
}

// connectToGateway attempts to create a new gRPC client for a gateway that wasn't doesn't have a grpc client yet.
func (i *gatewayGrpcForwarder) connectToGateway(ctx context.Context, gatewayID ulid.ULID) (connectpb.ConnectGatewayClient, error) {
	gateway, err := i.gatewayManager.GetGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("could not find gateway %s: %w", gatewayID.String(), err)
	}

	url := fmt.Sprintf("%s:%d", gateway.IPAddress, connectConfig.Gateway(ctx).GRPCPort)
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not create grpc client for %s: %w", url, err)
	}

	rpcClient := connectpb.NewConnectGatewayClient(conn)

	result, err := rpcClient.Ping(ctx, &connectpb.PingRequest{})
	if err != nil || result.GetMessage() != "ok" {
		return nil, fmt.Errorf("could not ping gateway %s: %w", url, err)
	}

	i.mu.Lock()
	i.grpcClients[gatewayID.String()] = rpcClient
	i.mu.Unlock()

	logger.StdlibLogger(ctx).Info("dynamically connected to connect gateway", "message", result, "url", url)

	return rpcClient, nil
}

func (i *gatewayGrpcForwarder) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	i.mu.RLock()
	grpcClient := i.grpcClients[gatewayID.String()]
	i.mu.RUnlock()

	if grpcClient == nil {
		logger.StdlibLogger(ctx).Warn("grpc client not found for gateway, attempting to create dynamically", "gatewayID", gatewayID.String())

		var err error
		grpcClient, err = i.connectToGateway(ctx, gatewayID)
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not create grpc client dynamically", "gatewayID", gatewayID.String(), "err", err)
			return fmt.Errorf("could not find or create grpc client for gateway %s: %w", gatewayID.String(), err)
		}
	}

	reply, err := grpcClient.Forward(ctx, &connectpb.ForwardRequest{
		ConnectionID: connectionID.String(),
		Data:         data,
	})

	logger.StdlibLogger(ctx).Debug("grpc message forwarded to connect gateway", "reply", reply, "err", err)
	return err
}
