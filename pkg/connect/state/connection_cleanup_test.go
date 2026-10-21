package state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestConnectionCleanupSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name       string
		refreshAt  string
		gc         bool
		wantDelete bool
	}{
		{name: "unchanged connection", wantDelete: true},
		{name: "refreshed before cleanup", refreshAt: "before"},
		{name: "refreshed after cleanup read", refreshAt: "HGET"},
		{name: "refreshed after garbage collection scan", refreshAt: "HSCAN", gc: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			defer rc.Close()
			hooked := &connectionCleanupClient{Client: rc}
			manager := NewRedisConnectionStateManager(hooked)
			writer := NewRedisConnectionStateManager(rc)
			envID, appID := uuid.New(), uuid.New()
			group := &WorkerGroup{Hash: "group", AppID: &appID, AppName: "app"}
			conn := &Connection{
				EnvID: envID, ConnectionId: ulid.Make(), GatewayId: ulid.Make(),
				Data:   &connectpb.WorkerConnectRequestData{InstanceId: "worker"},
				Groups: map[string]*WorkerGroup{group.Hash: group},
			}
			require.NoError(t, writer.UpsertConnection(ctx, conn, connectpb.ConnectionStatus_READY, time.Now().Add(-time.Hour)))
			snapshot, err := manager.GetConnection(ctx, envID, conn.ConnectionId)
			require.NoError(t, err)
			refresh := func() {
				require.NoError(t, writer.UpsertConnection(ctx, conn, connectpb.ConnectionStatus_READY, time.Now()))
			}
			if tc.refreshAt == "before" {
				refresh()
			} else if tc.refreshAt != "" {
				hooked.afterRead = func(command string) {
					if command == tc.refreshAt {
						hooked.afterRead = nil
						refresh()
					}
				}
			}
			var deleted bool
			if tc.gc {
				count, err := manager.GarbageCollectConnections(ctx)
				require.NoError(t, err)
				require.Zero(t, count, "a refreshed connection must not count as deleted")
			} else {
				deleted, err = manager.DeleteConnectionIfUnchanged(ctx, envID, conn.ConnectionId, snapshot)
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantDelete, deleted)
			require.Nil(t, hooked.afterRead, "the intended interleaving must have run")
			current, err := manager.GetConnection(ctx, envID, conn.ConnectionId)
			require.NoError(t, err)
			_, groupErr := manager.GetWorkerGroupByHash(ctx, envID, group.Hash)
			byApp, err := manager.GetConnectionsByAppID(ctx, envID, appID)
			require.NoError(t, err)
			byGroup, err := manager.GetConnectionsByGroupID(ctx, envID, group.Hash)
			require.NoError(t, err)
			if tc.wantDelete {
				require.Nil(t, current)
				require.ErrorIs(t, groupErr, ErrWorkerGroupNotFound)
				require.Empty(t, byApp)
				require.Empty(t, byGroup)
			} else {
				require.NotNil(t, current)
				require.NoError(t, groupErr)
				require.Len(t, byApp, 1)
				require.Len(t, byGroup, 1)
				// Gateway-driven disconnect remains unconditional.
				require.NoError(t, manager.DeleteConnection(ctx, envID, conn.ConnectionId))
			}
			deleted, err = manager.DeleteConnectionIfUnchanged(ctx, envID, conn.ConnectionId, snapshot)
			require.NoError(t, err)
			require.False(t, deleted, "repeated cleanup of a missing connection is a no-op")
		})
	}
}

// Refresh after returning an old read result, without sleeps.
type connectionCleanupClient struct {
	rueidis.Client
	afterRead func(string)
}

func (c *connectionCleanupClient) Do(ctx context.Context, cmd rueidis.Completed) rueidis.RedisResult {
	command := cmd.Commands()[0]
	result := c.Client.Do(ctx, cmd)
	if c.afterRead != nil {
		c.afterRead(command)
	}
	return result
}
