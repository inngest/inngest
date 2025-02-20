package state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"testing"
	"time"
)

func TestSortGroups(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	connManager := NewRedisConnectionStateManager(rc)

	accountId, envId := uuid.New(), uuid.New()

	appId1, syncId1 := uuid.New(), uuid.New() // pretend app 1 is synced,
	app1Version, app2Version := "app-1-v.test", "app-2-v.test"

	connectionsByEnvKey := connManager.connectionHash(envId)
	connectionsByApp1Key := connManager.connIndexByApp(envId, appId1)
	group1Id, group2Id := "app-1-hash", "app-2-hash"

	connectionsByGroup1Key := connManager.connIndexByGroup(envId, group1Id)
	connectionsByGroup2Key := connManager.connIndexByGroup(envId, group2Id)

	groupsByEnvKey := connManager.workerGroupHash(envId)

	// No groups created
	require.False(t, r.Exists(groupsByEnvKey))

	// No connections upserted
	require.False(t, r.Exists(connectionsByEnvKey))

	// No indexes created
	require.False(t, r.Exists(connectionsByApp1Key))
	require.False(t, r.Exists(connectionsByGroup1Key))
	require.False(t, r.Exists(connectionsByGroup2Key))

	group1 := &WorkerGroup{
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
	}

	group2 := &WorkerGroup{
		AccountID:     accountId,
		EnvID:         envId,
		AppName:       "app-2",
		AppVersion:    &app2Version,
		SDKLang:       "go",
		SDKVersion:    "v-test",
		FunctionSlugs: []string{"fn-3", "fn-4"},
		Hash:          group2Id,
	}

	t.Run("unsorted", func(t *testing.T) {
		groupsToSort := []*WorkerGroup{group2, group1}
		connManager.sortGroups(groupsToSort)
		require.Equal(t, group1, groupsToSort[0])
		require.Equal(t, group2, groupsToSort[1])
	})

	t.Run("already sorted", func(t *testing.T) {
		groupsToSort := []*WorkerGroup{group1, group2}
		connManager.sortGroups(groupsToSort)
		require.Equal(t, group1, groupsToSort[0])
		require.Equal(t, group2, groupsToSort[1])
	})
}

func TestUpsertConnection(t *testing.T) {
	t.Run("single unsynced app", func(t *testing.T) {
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

		app1Version := "app-1-v.test"

		connectionsByEnvKey := connManager.connectionHash(envId)
		group1Id := "app-1-hash"

		connectionsByGroup1Key := connManager.connIndexByGroup(envId, group1Id)

		groupsByEnvKey := connManager.workerGroupHash(envId)

		// No groups created
		require.False(t, r.Exists(groupsByEnvKey))

		// No connections upserted
		require.False(t, r.Exists(connectionsByEnvKey))

		// No indexes created
		require.False(t, r.Exists(connectionsByGroup1Key))

		group1 := &WorkerGroup{
			AccountID:     accountId,
			EnvID:         envId,
			AppName:       "app-1",
			AppVersion:    &app1Version,
			SDKLang:       "go",
			SDKVersion:    "v-test",
			FunctionSlugs: []string{"fn-1", "fn-2"},
			Hash:          group1Id,
		}
		group1Byt, err := json.Marshal(group1)
		require.NoError(t, err)

		groupIds := []string{group1Id}

		attrs := &connect.SystemAttributes{
			CpuCores: 10,
			MemBytes: 1024 * 1024,
			Os:       "testOS",
		}

		expectedConn := &connect.ConnMetadata{
			Id:       connId.String(),
			GroupIds: groupIds,

			InstanceId:      "my-worker",
			Status:          connect.ConnectionStatus_READY,
			SdkLanguage:     "go",
			SdkVersion:      "v-test",
			Attributes:      attrs,
			GatewayId:       gatewayId.String(),
			LastHeartbeatAt: timestamppb.New(lastHeartbeat),
		}
		connByt, err := json.Marshal(expectedConn)
		require.NoError(t, err)

		conn := &Connection{
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
				},
				WorkerManualReadinessAck: false,
				SystemAttributes:         attrs,
				SdkVersion:               "v-test",
				SdkLanguage:              "go",
			},
			Groups: map[string]*WorkerGroup{
				group1Id: group1,
			},
			GatewayId: gatewayId,
		}

		t.Run("initial upsert should create", func(t *testing.T) {
			err = connManager.UpsertConnection(ctx, conn, connect.ConnectionStatus_READY, lastHeartbeat)
			require.NoError(t, err)

			// Groups created
			require.True(t, r.Exists(groupsByEnvKey))
			require.Equal(t, string(group1Byt), r.HGet(groupsByEnvKey, group1Id))

			// Connections upserted
			require.True(t, r.Exists(connectionsByEnvKey))
			hkeysByEnv, err := r.HKeys(connectionsByEnvKey)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, hkeysByEnv)
			require.Equal(t, string(connByt), r.HGet(connectionsByEnvKey, connId.String()))

			// Indexes created
			require.True(t, r.Exists(connectionsByGroup1Key))
			membersByGroup1, err := r.SMembers(connectionsByGroup1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup1)

			retrievedConn, err := connManager.GetConnection(ctx, envId, connId)
			require.NoError(t, err)
			require.Equal(t, expectedConn, retrievedConn)

			connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByEnv)

			connsByGroup1, err := connManager.GetConnectionsByGroupID(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup1)
		})

		t.Run("subsequent upsert should update", func(t *testing.T) {
			err = connManager.UpsertConnection(ctx, conn, connect.ConnectionStatus_DISCONNECTED, lastHeartbeat)
			require.NoError(t, err)

			expectedConn.Status = connect.ConnectionStatus_DISCONNECTED
			connByt, err := json.Marshal(expectedConn)
			require.NoError(t, err)

			// Groups created
			require.True(t, r.Exists(groupsByEnvKey))
			require.Equal(t, string(group1Byt), r.HGet(groupsByEnvKey, group1Id))

			// Connections upserted
			require.True(t, r.Exists(connectionsByEnvKey))
			hkeysByEnv, err := r.HKeys(connectionsByEnvKey)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, hkeysByEnv)
			require.Equal(t, string(connByt), r.HGet(connectionsByEnvKey, connId.String()))

			// Indexes created
			require.True(t, r.Exists(connectionsByGroup1Key))
			membersByGroup1, err := r.SMembers(connectionsByGroup1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup1)

			retrievedConn, err := connManager.GetConnection(ctx, envId, connId)
			require.NoError(t, err)
			require.Equal(t, expectedConn, retrievedConn)

			connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByEnv)

			connsByGroup1, err := connManager.GetConnectionsByGroupID(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup1)
		})
	})

	t.Run("single unsynced app that gets synced", func(t *testing.T) {
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
		app1Version := "app-1-v.test"

		connectionsByEnvKey := connManager.connectionHash(envId)
		connectionsByApp1Key := connManager.connIndexByApp(envId, appId1)
		group1Id := "app-1-hash"

		connectionsByGroup1Key := connManager.connIndexByGroup(envId, group1Id)

		groupsByEnvKey := connManager.workerGroupHash(envId)

		// No groups created
		require.False(t, r.Exists(groupsByEnvKey))

		// No connections upserted
		require.False(t, r.Exists(connectionsByEnvKey))

		// No indexes created
		require.False(t, r.Exists(connectionsByApp1Key))
		require.False(t, r.Exists(connectionsByGroup1Key))

		group1 := &WorkerGroup{
			AccountID:     accountId,
			EnvID:         envId,
			AppName:       "app-1",
			AppVersion:    &app1Version,
			SDKLang:       "go",
			SDKVersion:    "v-test",
			FunctionSlugs: []string{"fn-1", "fn-2"},
			Hash:          group1Id,
		}
		group1Byt, err := json.Marshal(group1)
		require.NoError(t, err)

		groupIds := []string{group1Id}

		attrs := &connect.SystemAttributes{
			CpuCores: 10,
			MemBytes: 1024 * 1024,
			Os:       "testOS",
		}

		expectedConn := &connect.ConnMetadata{
			Id:       connId.String(),
			GroupIds: groupIds,

			InstanceId:      "my-worker",
			Status:          connect.ConnectionStatus_CONNECTED,
			SdkLanguage:     "go",
			SdkVersion:      "v-test",
			Attributes:      attrs,
			GatewayId:       gatewayId.String(),
			LastHeartbeatAt: timestamppb.New(lastHeartbeat),
		}
		connByt, err := json.Marshal(expectedConn)
		require.NoError(t, err)

		conn := &Connection{
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
				},
				WorkerManualReadinessAck: false,
				SystemAttributes:         attrs,
				SdkVersion:               "v-test",
				SdkLanguage:              "go",
			},
			Groups: map[string]*WorkerGroup{
				group1Id: group1,
			},
			GatewayId: gatewayId,
		}

		t.Run("initial upsert should create", func(t *testing.T) {
			err = connManager.UpsertConnection(ctx, conn, connect.ConnectionStatus_CONNECTED, lastHeartbeat)
			require.NoError(t, err)

			// Groups created
			require.True(t, r.Exists(groupsByEnvKey))
			require.Equal(t, string(group1Byt), r.HGet(groupsByEnvKey, group1Id))

			// Connections upserted
			require.True(t, r.Exists(connectionsByEnvKey))
			hkeysByEnv, err := r.HKeys(connectionsByEnvKey)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, hkeysByEnv)
			require.Equal(t, string(connByt), r.HGet(connectionsByEnvKey, connId.String()))

			// Indexes created

			// No app index
			require.False(t, r.Exists(connectionsByApp1Key))

			require.True(t, r.Exists(connectionsByGroup1Key))
			membersByGroup1, err := r.SMembers(connectionsByGroup1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup1)

			retrievedConn, err := connManager.GetConnection(ctx, envId, connId)
			require.NoError(t, err)
			require.Equal(t, expectedConn, retrievedConn)

			connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByEnv)

			connsByApp1, err := connManager.GetConnectionsByAppID(ctx, envId, appId1)
			require.NoError(t, err)
			require.Nil(t, connsByApp1)

			connsByGroup1, err := connManager.GetConnectionsByGroupID(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup1)
		})

		t.Run("subsequent upsert after sync should update", func(t *testing.T) {
			group1.AppID = &appId1
			group1.SyncID = &syncId1
			group1Byt, err := json.Marshal(group1)
			require.NoError(t, err)
			err = connManager.UpdateWorkerGroup(ctx, envId, group1)
			require.NoError(t, err)

			err = connManager.UpsertConnection(ctx, conn, connect.ConnectionStatus_READY, lastHeartbeat)
			require.NoError(t, err)

			expectedConn.Status = connect.ConnectionStatus_READY
			connByt, err := json.Marshal(expectedConn)
			require.NoError(t, err)

			// Groups created
			require.True(t, r.Exists(groupsByEnvKey))
			require.Equal(t, string(group1Byt), r.HGet(groupsByEnvKey, group1Id))

			// Connections upserted
			require.True(t, r.Exists(connectionsByEnvKey))
			hkeysByEnv, err := r.HKeys(connectionsByEnvKey)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, hkeysByEnv)
			require.Equal(t, string(connByt), r.HGet(connectionsByEnvKey, connId.String()))

			// Indexes created
			require.True(t, r.Exists(connectionsByApp1Key))
			membersByApp1, err := r.SMembers(connectionsByApp1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByApp1)

			require.True(t, r.Exists(connectionsByGroup1Key))
			membersByGroup1, err := r.SMembers(connectionsByGroup1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup1)

			retrievedConn, err := connManager.GetConnection(ctx, envId, connId)
			require.NoError(t, err)
			require.Equal(t, expectedConn, retrievedConn)

			connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByEnv)

			connsByApp1, err := connManager.GetConnectionsByAppID(ctx, envId, appId1)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByApp1)

			connsByGroup1, err := connManager.GetConnectionsByGroupID(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup1)
		})
	})

	t.Run("multiple apps", func(t *testing.T) {
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

		connectionsByEnvKey := connManager.connectionHash(envId)
		connectionsByApp1Key := connManager.connIndexByApp(envId, appId1)
		group1Id, group2Id := "app-1-hash", "app-2-hash"

		connectionsByGroup1Key := connManager.connIndexByGroup(envId, group1Id)
		connectionsByGroup2Key := connManager.connIndexByGroup(envId, group2Id)

		groupsByEnvKey := connManager.workerGroupHash(envId)

		// No groups created
		require.False(t, r.Exists(groupsByEnvKey))

		// No connections upserted
		require.False(t, r.Exists(connectionsByEnvKey))

		// No indexes created
		require.False(t, r.Exists(connectionsByApp1Key))
		require.False(t, r.Exists(connectionsByGroup1Key))
		require.False(t, r.Exists(connectionsByGroup2Key))

		retrievedConn, err := connManager.GetConnection(ctx, envId, connId)
		require.NoError(t, err)
		require.Nil(t, retrievedConn)

		connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
		require.NoError(t, err)
		require.Nil(t, connsByEnv)

		connsByApp1, err := connManager.GetConnectionsByAppID(ctx, envId, appId1)
		require.NoError(t, err)
		require.Nil(t, connsByApp1)

		connsByGroup1, err := connManager.GetConnectionsByGroupID(ctx, envId, group1Id)
		require.NoError(t, err)
		require.Nil(t, connsByGroup1)

		connsByGroup2, err := connManager.GetConnectionsByGroupID(ctx, envId, group2Id)
		require.NoError(t, err)
		require.Nil(t, connsByGroup2)

		group1 := &WorkerGroup{
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
		}
		group1Byt, err := json.Marshal(group1)
		require.NoError(t, err)

		group2 := &WorkerGroup{
			AccountID:     accountId,
			EnvID:         envId,
			AppName:       "app-2",
			AppVersion:    &app2Version,
			SDKLang:       "go",
			SDKVersion:    "v-test",
			FunctionSlugs: []string{"fn-3", "fn-4"},
			Hash:          group2Id,
		}
		group2Byt, err := json.Marshal(group2)
		require.NoError(t, err)

		groupIds := []string{group1Id, group2Id}

		require.Equal(t, -1, strings.Compare(group1Id, group2Id))

		attrs := &connect.SystemAttributes{
			CpuCores: 10,
			MemBytes: 1024 * 1024,
			Os:       "testOS",
		}

		expectedConn := &connect.ConnMetadata{
			Id:       connId.String(),
			GroupIds: groupIds,

			InstanceId:      "my-worker",
			Status:          connect.ConnectionStatus_READY,
			SdkLanguage:     "go",
			SdkVersion:      "v-test",
			Attributes:      attrs,
			GatewayId:       gatewayId.String(),
			LastHeartbeatAt: timestamppb.New(lastHeartbeat),
		}
		connByt, err := json.Marshal(expectedConn)
		require.NoError(t, err)

		conn := &Connection{
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
				SystemAttributes:         attrs,
				SdkVersion:               "v-test",
				SdkLanguage:              "go",
			},
			Groups: map[string]*WorkerGroup{
				group1Id: group1,
				group2Id: group2,
			},
			GatewayId: gatewayId,
		}

		t.Run("initial upsert should create", func(t *testing.T) {
			err = connManager.UpsertConnection(ctx, conn, connect.ConnectionStatus_READY, lastHeartbeat)
			require.NoError(t, err)

			// Groups created
			require.True(t, r.Exists(groupsByEnvKey))
			require.Equal(t, string(group1Byt), r.HGet(groupsByEnvKey, group1Id))
			require.Equal(t, string(group2Byt), r.HGet(groupsByEnvKey, group2Id))

			// Connections upserted
			require.True(t, r.Exists(connectionsByEnvKey))
			hkeysByEnv, err := r.HKeys(connectionsByEnvKey)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, hkeysByEnv)
			require.Equal(t, string(connByt), r.HGet(connectionsByEnvKey, connId.String()))

			// Indexes created
			require.True(t, r.Exists(connectionsByApp1Key))
			membersByApp1, err := r.SMembers(connectionsByApp1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByApp1)

			require.True(t, r.Exists(connectionsByGroup1Key))
			membersByGroup1, err := r.SMembers(connectionsByGroup1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup1)

			require.True(t, r.Exists(connectionsByGroup2Key))
			membersByGroup2, err := r.SMembers(connectionsByGroup2Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup2)

			retrievedConn, err := connManager.GetConnection(ctx, envId, connId)
			require.NoError(t, err)
			require.Equal(t, expectedConn, retrievedConn)

			connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByEnv)

			connsByApp1, err := connManager.GetConnectionsByAppID(ctx, envId, appId1)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByApp1)

			connsByGroup1, err := connManager.GetConnectionsByGroupID(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup1)

			connsByGroup2, err := connManager.GetConnectionsByGroupID(ctx, envId, group2Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup2)
		})

		t.Run("subsequent upsert should update", func(t *testing.T) {
			err = connManager.UpsertConnection(ctx, conn, connect.ConnectionStatus_DISCONNECTED, lastHeartbeat)
			require.NoError(t, err)

			expectedConn.Status = connect.ConnectionStatus_DISCONNECTED
			connByt, err := json.Marshal(expectedConn)
			require.NoError(t, err)

			// Groups created
			require.True(t, r.Exists(groupsByEnvKey))
			require.Equal(t, string(group1Byt), r.HGet(groupsByEnvKey, group1Id))
			require.Equal(t, string(group2Byt), r.HGet(groupsByEnvKey, group2Id))

			// Connections upserted
			require.True(t, r.Exists(connectionsByEnvKey))
			hkeysByEnv, err := r.HKeys(connectionsByEnvKey)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, hkeysByEnv)
			require.Equal(t, string(connByt), r.HGet(connectionsByEnvKey, connId.String()))

			// Indexes created
			require.True(t, r.Exists(connectionsByApp1Key))
			membersByApp1, err := r.SMembers(connectionsByApp1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByApp1)

			require.True(t, r.Exists(connectionsByGroup1Key))
			membersByGroup1, err := r.SMembers(connectionsByGroup1Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup1)

			require.True(t, r.Exists(connectionsByGroup2Key))
			membersByGroup2, err := r.SMembers(connectionsByGroup2Key)
			require.NoError(t, err)
			require.Equal(t, []string{connId.String()}, membersByGroup2)

			retrievedConn, err := connManager.GetConnection(ctx, envId, connId)
			require.NoError(t, err)
			require.Equal(t, expectedConn, retrievedConn)

			connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByEnv)

			connsByApp1, err := connManager.GetConnectionsByAppID(ctx, envId, appId1)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByApp1)

			connsByGroup1, err := connManager.GetConnectionsByGroupID(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup1)

			connsByGroup2, err := connManager.GetConnectionsByGroupID(ctx, envId, group2Id)
			require.NoError(t, err)
			require.Equal(t, []*connect.ConnMetadata{expectedConn}, connsByGroup2)
		})
	})

}
