package routing

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

// Router keeps app connection snapshots for one state manager.  separate
// routers do not share records from different stores.
type Router struct {
	stateManager state.StateManager
	snapshots    *connectionSnapshotCache
}

// NewRouter caches app connection snapshots for 250 milliseconds so repeated
// invocations do not reload every worker connection.
func NewRouter(manager state.StateManager) *Router {
	return &Router{stateManager: manager, snapshots: newConnectionSnapshotCache()}
}

// GetRoute refreshes an exhausted connection snapshot before reporting no
// route.  worker capacity and request leases remain outside the snapshot.
func (r *Router) GetRoute(ctx context.Context, rnd *util.FrandRNG, tracer trace.ConditionalTracer, log logger.Logger, data *connectpb.GatewayExecutorRequestData) (*RouteResult, error) {
	return getRoute(ctx, r.stateManager, rnd, tracer, log, data, r.snapshots)
}

// InvalidateConnections removes the app snapshot after failed forwarding so
// the next attempt can select a replacement connection.  it does not cancel a
// read that is in progress.  if it did, many failed forwards for one app could
// stop every route for that app until each request times out.
func (r *Router) InvalidateConnections(envID, appID uuid.UUID) {
	r.snapshots.drop(connectionSnapshotKey{envID: envID, appID: appID})
}
