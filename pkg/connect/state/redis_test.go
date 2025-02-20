package state

import (
	"context"
	"crypto/rand"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestUpsertConnection(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	connManager := NewRedisConnectionStateManager(rc)

	ctx := context.Background()

	lastHeartbeat := time.Now().Truncate(time.Minute)

	accountId, envId := uuid.New(), uuid.New()
	connId, gatewayId := ulid.MustNew(ulid.Now(), rand.Reader), ulid.MustNew(ulid.Now(), rand.Reader)

	appId1, syncId1 := uuid.New(), uuid.New() // pretend app 1 is synced,
	app1Version, app2Version := "app-1-v.test", "app-2-v.test"

	connectionsByEnvKey := connManager.connIndexByEnv(envId)
	connectionsByApp1Key := connManager.connIndexByApp(envId, appId1)
	group1Id, group2Id := "app-1-hash", "app-2-hash"

	connectionsByGroup1Key := connManager.connIndexByGroup(envId, group1Id)
	connectionsByGroup2Key := connManager.connIndexByGroup(envId, group2Id)

	groupsByEnvKey := connManager.groupIndexByEnv(envId)

	// No groups created
	require.False(t, r.Exists(groupsByEnvKey))

	// No connections upserted
	require.False(t, r.Exists(connectionsByEnvKey))

	// No indexes created
	require.False(t, r.Exists(connectionsByApp1Key))
	require.False(t, r.Exists(connectionsByGroup1Key))
	require.False(t, r.Exists(connectionsByGroup2Key))

	err = connManager.UpsertConnection(ctx, &Connection{
		AccountID:    accountId,
		EnvID:        envId,
		ConnectionId: connId,
		WorkerIP:     "127.0.0.1",
		Data: &connect.WorkerConnectRequestData{
			ConnectionId: connId.String(),
			InstanceId:   "my-worker",
			Apps: []*connect.AppConfiguration{
				{
					AppName:    "app-1",
					AppVersion: &app1Version,
					Functions:  nil,
				},
				{
					AppName:    "app-2",
					AppVersion: &app2Version,
					Functions:  nil,
				},
			},
			WorkerManualReadinessAck: false,
			SystemAttributes: &connect.SystemAttributes{
				CpuCores: 10,
				MemBytes: 1024 * 1024,
				Os:       "testOS",
			},
			SdkVersion:  "v-test",
			SdkLanguage: "go",
		},
		Groups: map[string]*WorkerGroup{
			"app-1": {
				AccountID:     accountId,
				EnvID:         envId,
				AppID:         &appId1,
				SyncID:        &syncId1,
				AppName:       "app-1",
				AppVersion:    &app1Version,
				SDKLang:       "go",
				SDKVersion:    "v-test",
				FunctionSlugs: []string{"fn-1", "fn-2"},
				Hash:          group1Id,
			},
			"app-2": {
				AccountID:     accountId,
				EnvID:         envId,
				AppName:       "app-2",
				AppVersion:    &app2Version,
				SDKLang:       "go",
				SDKVersion:    "v-test",
				FunctionSlugs: []string{"fn-3", "fn-4"},
				Hash:          group2Id,
			},
		},
		GatewayId: gatewayId,
	}, connect.ConnectionStatus_CONNECTED, lastHeartbeat)
	require.NoError(t, err)

	// No groups created
	require.True(t, r.Exists(groupsByEnvKey))

	// No connections upserted
	require.True(t, r.Exists(connectionsByEnvKey))

	// No indexes created
	require.True(t, r.Exists(connectionsByApp1Key))
	require.True(t, r.Exists(connectionsByGroup1Key))
	require.True(t, r.Exists(connectionsByGroup2Key))
}
