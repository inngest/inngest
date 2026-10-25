package routing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

// A route may hold an old snapshot while the gateway refreshes the heartbeat.
// Cleanup must preserve the refreshed connection and its group for the next route.
func TestCleanupPreservesRefreshedConnection(t *testing.T) {
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	wrapped := &heartbeatAfterSnapshot{StateManager: mgr, t: t}
	s := setup(t, wrapped, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now().Add(-time.Minute)))
	ctx := context.Background()
	_, err := getSuitableConnection(ctx, rnd, wrapped, s.envId, s.appId, s.fnSlug, logger.VoidLogger())
	require.ErrorIs(t, err, ErrNoHealthyConnection)
	require.True(t, wrapped.refreshed)
	conn, err := mgr.GetConnection(ctx, s.envId, s.connIds[0])
	require.NoError(t, err)
	require.NotNil(t, conn, "routing cleanup must preserve a worker whose heartbeat refreshed")
	_, err = mgr.GetWorkerGroupByHash(ctx, s.envId, wrapped.groupHash)
	require.NoError(t, err)
	route, err := getSuitableConnection(ctx, rnd, wrapped, s.envId, s.appId, s.fnSlug, logger.VoidLogger())
	require.NoError(t, err)
	require.Equal(t, s.connIds[0].String(), route.Id)
}

type heartbeatAfterSnapshot struct {
	state.StateManager
	t         *testing.T
	conn      *state.Connection
	groupHash string
	refreshed bool
}

func (s *heartbeatAfterSnapshot) UpsertConnection(ctx context.Context, conn *state.Connection, status connectpb.ConnectionStatus, heartbeat time.Time) error {
	s.conn = conn
	return s.StateManager.UpsertConnection(ctx, conn, status, heartbeat)
}

func (s *heartbeatAfterSnapshot) GetConnectionsByAppID(ctx context.Context, env, app uuid.UUID) ([]*connectpb.ConnMetadata, error) {
	snapshot, err := s.StateManager.GetConnectionsByAppID(ctx, env, app)
	require.NoError(s.t, err)
	require.Len(s.t, snapshot, 1)
	s.groupHash = snapshot[0].SyncedWorkerGroups[app.String()]
	require.NoError(s.t, s.StateManager.UpsertConnection(ctx, s.conn, connectpb.ConnectionStatus_READY, time.Now()))
	fresh, err := s.StateManager.GetConnection(ctx, env, s.conn.ConnectionId)
	require.NoError(s.t, err)
	require.WithinDuration(s.t, time.Now(), fresh.LastHeartbeatAt.AsTime(), time.Second)
	_, err = s.StateManager.GetWorkerGroupByHash(ctx, env, s.groupHash)
	require.NoError(s.t, err)
	s.refreshed = true
	return snapshot, nil
}
