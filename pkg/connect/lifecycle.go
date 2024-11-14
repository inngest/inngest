package connect

import (
	"context"

	"github.com/inngest/inngest/proto/gen/connect/v1"
)

type ConnectGatewayLifecycleListener interface {
	// OnConnected is called when a new connection is established and authenticated on the gateway
	OnConnected(ctx context.Context, data *connect.WorkerConnectRequestData)

	// OnSynced is called when the gateway successfully synced a worker group configuration
	OnSynced(ctx context.Context)

	// OnDisconnected is called when a connection on the gateway is lost
	OnDisconnected(ctx context.Context)
}
