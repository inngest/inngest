package connect

import (
	"context"

	"github.com/inngest/inngest/pkg/connect/state"
)

type ConnectGatewayLifecycleListener interface {
	// OnConnected is called when a new connection is established and authenticated on the gateway
	OnConnected(ctx context.Context, conn *state.Connection)
	OnReady(ctx context.Context, conn *state.Connection)

	OnHeartbeat(ctx context.Context, conn *state.Connection)

	OnStartDraining(ctx context.Context, conn *state.Connection)
	OnStartDisconnecting(ctx context.Context, conn *state.Connection)

	// OnSynced is called when the gateway successfully synced a worker group configuration
	OnSynced(ctx context.Context, conn *state.Connection)

	// OnDisconnected is called when a connection on the gateway is lost
	OnDisconnected(ctx context.Context, conn *state.Connection, closeReason string)
}
