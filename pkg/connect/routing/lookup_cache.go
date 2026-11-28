package routing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
)

type workerGroupKey struct {
	envID uuid.UUID
	hash  string
}

// routingLookups keeps records for one connection selection attempt.  failed
// reads are repeated so cleanup does not reuse a missing or failed record.
type routingLookups struct {
	state.StateManager
	groups    map[workerGroupKey]*state.WorkerGroup
	gateways  map[ulid.ULID]*state.Gateway
	onCleanup func()
}

func (r *routingLookups) GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*state.WorkerGroup, error) {
	key := workerGroupKey{envID: envID, hash: hash}
	if group, ok := r.groups[key]; ok {
		return group, nil
	}
	group, err := r.StateManager.GetWorkerGroupByHash(ctx, envID, hash)
	if err == nil {
		r.groups[key] = group
	}
	return group, err
}

func (r *routingLookups) GetGateway(ctx context.Context, id ulid.ULID) (*state.Gateway, error) {
	if gateway, ok := r.gateways[id]; ok {
		// the gateway can refresh its heartbeat during selection.  read an
		// expired record again before it can cause gateway cleanup.
		if !time.UnixMilli(gateway.LastHeartbeatAtMS).Before(time.Now().Add(-2 * consts.ConnectGatewayHeartbeatInterval)) {
			return gateway, nil
		}
		delete(r.gateways, id)
	}
	gateway, err := r.StateManager.GetGateway(ctx, id)
	if err == nil {
		r.gateways[id] = gateway
	}
	return gateway, err
}

func (r *routingLookups) DeleteGateway(ctx context.Context, id ulid.ULID) error {
	if r.onCleanup != nil {
		defer r.onCleanup()
	}
	delete(r.gateways, id)
	return r.StateManager.DeleteGateway(ctx, id)
}

func (r *routingLookups) DeleteConnectionIfUnchanged(ctx context.Context, envID uuid.UUID, id ulid.ULID, expected *connectpb.ConnMetadata) (bool, error) {
	if r.onCleanup != nil {
		defer r.onCleanup()
	}
	// deleting the last connection removes its worker group.  later
	// candidates must read the record again after connection cleanup.
	for _, hash := range expected.SyncedWorkerGroups {
		delete(r.groups, workerGroupKey{envID: envID, hash: hash})
	}
	return r.StateManager.DeleteConnectionIfUnchanged(ctx, envID, id, expected)
}
