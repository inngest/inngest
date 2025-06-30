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
)

type GatewayGrpcForwarder interface {
	ConnectToGateways(ctx context.Context) error
	Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error
}

type gatewayGrpcForwarder struct {
	gatewayManager state.GatewayManager

	// TODO: Synchronization
	grpcClients map[string]connectpb.ConnectGatewayClient
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

	for _, g := range gateways {
		url := fmt.Sprintf("%s:%d", g.IPAddress, connectConfig.Gateway(ctx).GRPCPort)
		conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.StdlibLogger(ctx).Error("could not create grpc client", err)
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
			logger.StdlibLogger(ctx).Info("connect gateway successful", "message", message, "url", url)
		}
	}
	return nil
}
func (i *gatewayGrpcForwarder) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	grpcClient := i.grpcClients[gatewayID.String()]
	if grpcClient == nil {
		// TODO: Switch to a warning or info and try to create a new grpc client dynamically
		logger.StdlibLogger(ctx).Error("could not find grpc client for connect gateway")
		return fmt.Errorf("could not find grpc client for connect gateway")
	}
	// TODO: Call forward
	reply, err := grpcClient.Forward(ctx, &connectpb.ForwardRequest{
		ConnectionID: connectionID.String(),
		Data:         data,
	})
	logger.StdlibLogger(ctx).Debug("grpc message forwarded to connect gateway", "reply", reply, "err", err)

	return nil
}
