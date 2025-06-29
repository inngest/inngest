package pubsub

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GatewayGrpcForwarder interface {
	ConnectToGateways(ctx context.Context) error
}

type gatewayGrpcForwarder struct {
	gatewayManager state.GatewayManager
}

func NewGatewayGrpcForwarder(stateManager state.GatewayManager) GatewayGrpcForwarder {
	return &gatewayGrpcForwarder{gatewayManager: stateManager}
}

// Connect to all gateways through gRPC
func (i *gatewayGrpcForwarder) ConnectToGateways(ctx context.Context) error {
	gateways, err := i.gatewayManager.GetAllGateways(ctx)
	if err != nil {
		return err
	}
	logger.StdlibLogger(ctx).Debug("got connect gateways to connect to", "len", len(gateways))

	for _, g := range gateways {
		url := fmt.Sprintf("%s:%d", g.IPAddress, 50051)
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
			logger.StdlibLogger(ctx).Info("connect gateway successful", "message", message, "url", url)
		}
	}
	return nil
}
