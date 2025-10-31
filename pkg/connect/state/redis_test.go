package state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
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

		allGroupIds := map[string]string{"app-1": group1Id}

		attrs := &connect.SystemAttributes{
			CpuCores: 10,
			MemBytes: 1024 * 1024,
			Os:       "testOS",
		}

		expectedConn := &connect.ConnMetadata{
			Id:              connId.String(),
			AllWorkerGroups: allGroupIds,

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

		groupIds := map[string]string{"app-1": group1Id}

		attrs := &connect.SystemAttributes{
			CpuCores: 10,
			MemBytes: 1024 * 1024,
			Os:       "testOS",
		}

		expectedConn := &connect.ConnMetadata{
			Id:              connId.String(),
			AllWorkerGroups: groupIds,

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

			expectedConn.SyncedWorkerGroups = map[string]string{appId1.String(): group1.Hash}
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

		_, err = connManager.GetWorkerGroupByHash(ctx, envId, group1Id)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrWorkerGroupNotFound)

		_, err = connManager.GetWorkerGroupByHash(ctx, envId, group2Id)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrWorkerGroupNotFound)

		_, err = connManager.GetWorkerGroupsByHash(ctx, envId, []string{group1Id, group2Id})
		require.Error(t, err)
		require.ErrorContains(t, err, "could not find group \"app-1-hash\": worker group not found")
		require.ErrorIs(t, err, ErrWorkerGroupNotFound)

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

		allGroupIds := map[string]string{"app-1": group1Id, "app-2": group2Id}
		syncedGroupIds := map[string]string{appId1.String(): group1Id}

		require.Equal(t, -1, strings.Compare(group1Id, group2Id))

		attrs := &connect.SystemAttributes{
			CpuCores: 10,
			MemBytes: 1024 * 1024,
			Os:       "testOS",
		}

		expectedConn := &connect.ConnMetadata{
			Id:                 connId.String(),
			AllWorkerGroups:    allGroupIds,
			SyncedWorkerGroups: syncedGroupIds,

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

			retrievedGroup1, err := connManager.GetWorkerGroupByHash(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, group1, retrievedGroup1)

			retrievedGroup2, err := connManager.GetWorkerGroupByHash(ctx, envId, group2Id)
			require.NoError(t, err)
			require.Equal(t, group2, retrievedGroup2)

			workerGroups, err := connManager.GetWorkerGroupsByHash(ctx, envId, []string{group1Id, group2Id})
			require.NoError(t, err)
			require.Equal(t, []WorkerGroup{*group1, *group2}, workerGroups)
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

			retrievedGroup1, err := connManager.GetWorkerGroupByHash(ctx, envId, group1Id)
			require.NoError(t, err)
			require.Equal(t, group1, retrievedGroup1)

			retrievedGroup2, err := connManager.GetWorkerGroupByHash(ctx, envId, group2Id)
			require.NoError(t, err)
			require.Equal(t, group2, retrievedGroup2)

			workerGroups, err := connManager.GetWorkerGroupsByHash(ctx, envId, []string{group1Id, group2Id})
			require.NoError(t, err)
			require.Equal(t, []WorkerGroup{*group1, *group2}, workerGroups)
		})

		t.Run("delete should drop all data", func(t *testing.T) {
			err := connManager.DeleteConnection(ctx, envId, connId)
			require.NoError(t, err)

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

			_, err = connManager.GetWorkerGroupByHash(ctx, envId, group1Id)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrWorkerGroupNotFound)

			_, err = connManager.GetWorkerGroupByHash(ctx, envId, group2Id)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrWorkerGroupNotFound)

			_, err = connManager.GetWorkerGroupsByHash(ctx, envId, []string{group1Id, group2Id})
			require.Error(t, err)
			require.ErrorContains(t, err, "could not find group \"app-1-hash\": worker group not found")
			require.ErrorIs(t, err, ErrWorkerGroupNotFound)
		})
	})

}

func TestGarbageCollectConnections(t *testing.T) {
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

		lastHeartbeat := time.Now().Add(-consts.ConnectGCThreshold)

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

		attrs := &connect.SystemAttributes{
			CpuCores: 10,
			MemBytes: 1024 * 1024,
			Os:       "testOS",
		}

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

		t.Run("garbage collect should delete", func(t *testing.T) {
			err = connManager.UpsertConnection(ctx, conn, connect.ConnectionStatus_READY, lastHeartbeat)
			require.NoError(t, err)

			deleted, err := connManager.GarbageCollectConnections(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, deleted)

			deleted, err = connManager.GarbageCollectConnections(ctx)
			require.NoError(t, err)
			require.Equal(t, 0, deleted)

			connsByEnv, err := connManager.GetConnectionsByEnvID(ctx, envId)
			require.NoError(t, err)
			require.Len(t, connsByEnv, 0)
		})
	})
}

func TestGarbageCollectGateways(t *testing.T) {
	t.Run("should not clean up valid gateway", func(t *testing.T) {
		r := miniredis.RunT(t)

		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		connManager := NewRedisConnectionStateManager(rc)

		ctx := context.Background()

		gwID := ulid.MustNew(ulid.Now(), rand.Reader)

		expectedGw := &Gateway{
			Id:                gwID,
			Status:            GatewayStatusActive,
			LastHeartbeatAtMS: time.Now().Truncate(time.Second).UnixMilli(),
			Hostname:          "gw",
		}

		err = connManager.UpsertGateway(ctx, expectedGw)
		require.NoError(t, err)

		gw, err := connManager.GetGateway(ctx, gwID)
		require.NoError(t, err)
		require.Equal(t, *expectedGw, *gw)

		deleted, err := connManager.GarbageCollectGateways(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, deleted)

		gw, err = connManager.GetGateway(ctx, gwID)
		require.NoError(t, err)
		require.Equal(t, *expectedGw, *gw)
	})

	t.Run("should clean up expired gateway", func(t *testing.T) {
		r := miniredis.RunT(t)

		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		connManager := NewRedisConnectionStateManager(rc)

		ctx := context.Background()

		gwID := ulid.MustNew(ulid.Now(), rand.Reader)

		expectedGw := &Gateway{
			Id:                gwID,
			Status:            GatewayStatusActive,
			LastHeartbeatAtMS: time.Now().Add(-1 * time.Hour).Truncate(time.Second).UnixMilli(),
			Hostname:          "old-gw",
		}

		err = connManager.UpsertGateway(ctx, expectedGw)
		require.NoError(t, err)

		gw, err := connManager.GetGateway(ctx, gwID)
		require.NoError(t, err)
		require.Equal(t, *expectedGw, *gw)

		deleted, err := connManager.GarbageCollectGateways(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, deleted)

		gw, err = connManager.GetGateway(ctx, gwID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrGatewayNotFound)
		require.Nil(t, gw)
	})
}

func TestGetAllGateways(t *testing.T) {
	t.Run("should return empty slice when no gateways exist", func(t *testing.T) {
		r := miniredis.RunT(t)

		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		connManager := NewRedisConnectionStateManager(rc)

		ctx := context.Background()

		gateways, err := connManager.GetAllGateways(ctx)
		require.NoError(t, err)
		require.Empty(t, gateways)
	})

	t.Run("should return multiple gateways", func(t *testing.T) {
		r := miniredis.RunT(t)

		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		connManager := NewRedisConnectionStateManager(rc)

		ctx := context.Background()

		gwID1 := ulid.MustNew(ulid.Now(), rand.Reader)
		gwID2 := ulid.MustNew(ulid.Now(), rand.Reader)
		gwID3 := ulid.MustNew(ulid.Now(), rand.Reader)

		expectedGw1 := &Gateway{
			Id:                gwID1,
			Status:            GatewayStatusActive,
			LastHeartbeatAtMS: time.Now().Truncate(time.Second).UnixMilli(),
			Hostname:          "gw-1",
			IPAddress:         net.ParseIP("192.168.1.10"),
		}

		expectedGw2 := &Gateway{
			Id:                gwID2,
			Status:            GatewayStatusDraining,
			LastHeartbeatAtMS: time.Now().Add(-1 * time.Minute).Truncate(time.Second).UnixMilli(),
			Hostname:          "gw-2",
			IPAddress:         net.ParseIP("192.168.1.20"),
		}

		expectedGw3 := &Gateway{
			Id:                gwID3,
			Status:            GatewayStatusActive,
			LastHeartbeatAtMS: time.Now().Add(-2 * time.Minute).Truncate(time.Second).UnixMilli(),
			Hostname:          "gw-3",
			IPAddress:         net.ParseIP("10.0.0.5"),
		}

		err = connManager.UpsertGateway(ctx, expectedGw1)
		require.NoError(t, err)

		err = connManager.UpsertGateway(ctx, expectedGw2)
		require.NoError(t, err)

		err = connManager.UpsertGateway(ctx, expectedGw3)
		require.NoError(t, err)

		gateways, err := connManager.GetAllGateways(ctx)
		require.NoError(t, err)
		require.Len(t, gateways, 3)

		gatewayMap := make(map[string]*Gateway)
		for _, gw := range gateways {
			gatewayMap[gw.Id.String()] = gw
		}

		require.Equal(t, *expectedGw1, *gatewayMap[gwID1.String()])
		require.Equal(t, *expectedGw2, *gatewayMap[gwID2.String()])
		require.Equal(t, *expectedGw3, *gatewayMap[gwID3.String()])

		require.True(t, expectedGw1.IPAddress.Equal(gatewayMap[gwID1.String()].IPAddress))
		require.True(t, expectedGw2.IPAddress.Equal(gatewayMap[gwID2.String()].IPAddress))
		require.True(t, expectedGw3.IPAddress.Equal(gatewayMap[gwID3.String()].IPAddress))
	})
}

func TestSetWorkerTotalCapacity(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance-1"

	t.Run("sets capacity with positive value", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		// Verify capacity was set
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(10), capacity)

		// Verify TTL is set
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		ttl := r.TTL(capacityKey)
		require.Greater(t, ttl, time.Duration(0))
		require.LessOrEqual(t, ttl, consts.ConnectWorkerCapacityManagerTTL)
	})

	t.Run("deletes capacity when set to zero", func(t *testing.T) {
		// First set a capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Now set to zero
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 0)
		require.NoError(t, err)

		// Verify capacity is gone
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity)

		// Verify key is deleted
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		require.False(t, r.Exists(capacityKey))
	})

	t.Run("deletes capacity when set to negative", func(t *testing.T) {
		// First set a capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Now set to negative
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, -1)
		require.NoError(t, err)

		// Verify capacity is gone
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity)
	})

	t.Run("updates existing capacity", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Update to different value
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 15)
		require.NoError(t, err)

		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(15), capacity)
	})
}

func TestGetWorkerTotalCapacity(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance-1"

	t.Run("returns zero when no capacity set", func(t *testing.T) {
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity)
	})

	t.Run("returns set capacity", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 25)
		require.NoError(t, err)

		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(25), capacity)
	})
}

func TestGetWorkerCapacities(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-instance-1"

	t.Run("returns ConnectNoWorkerCapacity when no limit set", func(t *testing.T) {
		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(consts.ConnectWorkerNoConcurrencyLimitForRequests), caps.Available)
		require.Equal(t, int64(0), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())
	})

	t.Run("returns full capacity when no active leases", func(t *testing.T) {
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(10), caps.Available)
		require.Equal(t, int64(10), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())
	})

	t.Run("returns reduced capacity after assigning leases", func(t *testing.T) {
		instanceID := "test-instance-2"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Assign 3 leases
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(2), caps.Available)
		require.Equal(t, int64(5), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())
	})

	t.Run("returns zero when at capacity", func(t *testing.T) {
		instanceID := "test-instance-3"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), caps.Available)
		require.Equal(t, int64(2), caps.Total)
		require.True(t, caps.IsAtCapacity())
		require.False(t, caps.IsAvailable())
	})
}

func TestAssignRequestToWorker(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("succeeds when no capacity limit set", func(t *testing.T) {
		instanceID := "test-instance-no-limit"
		err := mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Should not create set when no limit
		setKey := mgr.workerRequestsKey(envID, instanceID)
		require.False(t, r.Exists(setKey))
	})

	t.Run("increments counter when capacity set", func(t *testing.T) {
		instanceID := "test-instance-1"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Check lease was added to set
		setKey := mgr.workerRequestsKey(envID, instanceID)
		require.True(t, r.Exists(setKey))

		// Check set contains the request
		members, err := r.ZMembers(setKey)
		require.NoError(t, err)
		require.Equal(t, []string{"req-1"}, members)
	})

	t.Run("sets TTL on counter", func(t *testing.T) {
		instanceID := "test-instance-ttl"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		setKey := mgr.workerRequestsKey(envID, instanceID)
		ttl := r.TTL(setKey)
		require.Greater(t, ttl, time.Duration(0))
		require.LessOrEqual(t, ttl, consts.ConnectWorkerCapacityManagerTTL)
	})

	t.Run("rejects when at capacity", func(t *testing.T) {
		instanceID := "test-instance-full"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Fill capacity
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Should reject third
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-3")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)
	})

	t.Run("allows multiple workers with different capacities", func(t *testing.T) {
		instance1 := "worker-1"
		instance2 := "worker-2"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, instance1, 1)
		require.NoError(t, err)
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instance2, 10)
		require.NoError(t, err)

		// Worker 1 at capacity
		err = mgr.AssignRequestToWorker(ctx, envID, instance1, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instance1, "req-2")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)

		// Worker 2 still has capacity
		err = mgr.AssignRequestToWorker(ctx, envID, instance2, "req-1")
		require.NoError(t, err)
	})
}

func TestDeleteRequestFromWorker(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("no-op when no capacity set", func(t *testing.T) {
		instanceID := "test-instance-no-cap"
		err := mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err, "should be no-op when no capacity is set")
	})

	t.Run("decrements counter", func(t *testing.T) {
		instanceID := "test-instance-1"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Add some leases
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Remove one
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Check set has remaining lease
		setKey := mgr.workerRequestsKey(envID, instanceID)
		require.True(t, r.Exists(setKey))

		// Check set contains one lease
		members, err := r.ZMembers(setKey)
		require.NoError(t, err)
		require.Equal(t, []string{"req-2"}, members)
	})

	t.Run("deletes counter when reaching zero", func(t *testing.T) {
		instanceID := "test-instance-2"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Set should be deleted
		setKey := mgr.workerRequestsKey(envID, instanceID)
		require.False(t, r.Exists(setKey))
	})

	t.Run("refreshes TTL when counter still positive", func(t *testing.T) {
		instanceID := "test-instance-3a"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Fast forward time a bit in miniredis (use a fraction of the capacity manager TTL)
		r.FastForward(consts.ConnectWorkerCapacityManagerTTL / 4)

		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// TTL should be refreshed
		setKey := mgr.workerRequestsKey(envID, instanceID)
		ttl := r.TTL(setKey)
		require.Greater(t, ttl, consts.ConnectWorkerCapacityManagerTTL/4) // Should have most of the TTL remaining
	})

	t.Run("refresh TTL after counter expires returns unlimited capacity", func(t *testing.T) {
		instanceID := "test-instance-3b"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Fast forward time to exceed ConnectWorkerCapacityManagerTTL (6 * ConnectWorkerRequestLeaseDuration = 120s)
		r.FastForward(consts.ConnectWorkerCapacityManagerTTL + time.Second)

		// Get the Total Capacity, it should have expired, but we still
		// don't return error on expired total capacity
		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), caps.Total)
		require.Equal(t, int64(consts.ConnectWorkerNoConcurrencyLimitForRequests), caps.Available)

		// TTL should be expired
		setKey := mgr.workerRequestsKey(envID, instanceID)
		ttl := r.TTL(setKey)
		require.Equal(t, ttl, 0*time.Second) // Should be 0 since it expired
	})

	t.Run("allows assignment after deletion", func(t *testing.T) {
		instanceID := "test-instance-4"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Fill capacity
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Should reject
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-3")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)

		// Delete one
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Should now succeed
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)
	})

	t.Run("returns error when instance ID doesn't match", func(t *testing.T) {
		instanceID := "test-instance-security"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Instance assigns a lease
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Manually corrupt the lease mapping to point to a different instance
		// This simulates a race condition or data corruption scenario
		requestWorkerKey := fmt.Sprintf("{%s}:request-worker:req-1", envID.String())
		rc, _ := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		rc.Do(ctx, rc.B().Set().Key(requestWorkerKey).Value("different-instance").Build())
		rc.Close()

		// Now when the original instance tries to delete its lease, it should fail
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.ErrorIs(t, err, ErrInstanceIDMismatch)

		// Verify lease still exists in the set
		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(4), caps.Available) // Should still be 4 (5-1)

		// Fix the mapping back to the correct instance
		rc2, _ := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		rc2.Do(ctx, rc2.B().Set().Key(requestWorkerKey).Value(instanceID).Build())
		rc2.Close()

		// Now the deletion should succeed
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Verify lease is now gone
		caps, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(5), caps.Available) // Should be back to 5
	})
}

func TestWorkerCapcityOnHeartbeat(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("no-op when no capacity set", func(t *testing.T) {
		instanceID := "test-instance-no-cap"
		err := mgr.WorkerCapcityOnHeartbeat(ctx, envID, instanceID)
		require.NoError(t, err)
	})

	t.Run("refreshes TTL on capacity key", func(t *testing.T) {
		instanceID := "test-instance-1"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Fast forward time (use the request lease duration)
		r.FastForward(consts.ConnectWorkerRequestLeaseDuration)

		// Refresh TTL
		err = mgr.WorkerCapcityOnHeartbeat(ctx, envID, instanceID)
		require.NoError(t, err)

		// Check TTL is reset
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		ttl := r.TTL(capacityKey)
		require.Greater(t, ttl, consts.ConnectWorkerCapacityManagerTTL/4) // Should have most of the TTL remaining
	})

	t.Run("refreshes TTL on both capacity and counter keys", func(t *testing.T) {
		instanceID := "test-instance-2"
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Assign a lease to create the counter key
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Fast forward time (use a fraction of the capacity manager TTL)
		r.FastForward(consts.ConnectWorkerCapacityManagerTTL / 4)

		// Refresh TTL
		err = mgr.WorkerCapcityOnHeartbeat(ctx, envID, instanceID)
		require.NoError(t, err)

		// Check both TTLs are reset
		capacityKey := mgr.workerCapacityKey(envID, instanceID)
		setKey := mgr.workerRequestsKey(envID, instanceID)

		capacityTTL := r.TTL(capacityKey)
		require.Greater(t, capacityTTL, consts.ConnectWorkerCapacityManagerTTL/4) // Should have most of the TTL remaining

		setTTL := r.TTL(setKey)
		require.Greater(t, setTTL, consts.ConnectWorkerCapacityManagerTTL/4) // Should have most of the TTL remaining
	})
}

func TestWorkerCapacityEndToEnd(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()
	instanceID := "test-worker"

	t.Run("complete lifecycle", func(t *testing.T) {
		// Worker connects with capacity 3
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 3)
		require.NoError(t, err)

		// Check available capacity
		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(3), caps.Available)
		require.Equal(t, int64(3), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())

		// Assign 3 requests
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		// At capacity
		caps, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), caps.Available)
		require.Equal(t, int64(3), caps.Total)
		require.True(t, caps.IsAtCapacity())
		require.False(t, caps.IsAvailable())

		// Reject new request
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-4")
		require.ErrorIs(t, err, ErrWorkerCapacityExceeded)

		// Complete one request
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Verify key deleted
		requestWorkerKey := mgr.requestWorkerKey(envID, "req-1")
		require.False(t, r.Exists(requestWorkerKey))

		// Verify the other keys still exist
		requestWorkerKey = mgr.requestWorkerKey(envID, "req-2")
		require.True(t, r.Exists(requestWorkerKey))
		requestWorkerKey = mgr.requestWorkerKey(envID, "req-3")
		require.True(t, r.Exists(requestWorkerKey))

		// Now has capacity again
		caps, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(1), caps.Available)
		require.Equal(t, int64(3), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())

		// Can assign new request
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-4")
		require.NoError(t, err)

		// Complete all requests
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)
		err = mgr.DeleteRequestFromWorker(ctx, envID, instanceID, "req-4")
		require.NoError(t, err)

		// Back to full capacity
		caps, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(3), caps.Available)
		require.Equal(t, int64(3), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())

		// Set should be deleted when all leases are removed
		setKey := mgr.workerRequestsKey(envID, instanceID)
		require.False(t, r.Exists(setKey))

		// All lease mappings should be deleted
		requestWorkerKey = mgr.requestWorkerKey(envID, "req-2")
		require.False(t, r.Exists(requestWorkerKey))

		for i := 0; i < 6; i++ {
			err = mgr.WorkerCapcityOnHeartbeat(ctx, envID, instanceID)
			// TODO: extend lease for req-2
			require.NoError(t, err)
			r.FastForward(consts.ConnectWorkerRequestLeaseDuration / 2)
		}

		// All leases have been deleted, so capacity should be back to full
		caps, err = mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(3), caps.Available)
		require.Equal(t, int64(3), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())

	})

	t.Run("worker reconnects with different capacity", func(t *testing.T) {
		instanceID := "test-worker-2"

		// Initial capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Assign some leases
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Worker reconnects with lower capacity
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Capacity updated
		capacity, err := mgr.GetWorkerTotalCapacity(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(2), capacity)
	})

	t.Run("worker removes capacity limit", func(t *testing.T) {
		instanceID := "test-worker-3"

		// Set capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 3)
		require.NoError(t, err)

		// Assign lease
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Remove capacity limit
		err = mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 0)
		require.NoError(t, err)

		// Should return unlimited
		caps, err := mgr.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(consts.ConnectWorkerNoConcurrencyLimitForRequests), caps.Available)
		require.Equal(t, int64(0), caps.Total)
		require.False(t, caps.IsAtCapacity())
		require.True(t, caps.IsAvailable())

		// Can assign without limit
		for i := 0; i < 100; i++ {
			err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "req")
			require.NoError(t, err)
		}
	})
}

func TestGetRequestWorkerInstanceID(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("returns empty when no mapping exists", func(t *testing.T) {
		instanceID, err := mgr.GetRequestWorkerInstanceID(ctx, envID, "non-existent-request")
		require.NoError(t, err)
		require.Equal(t, "", instanceID)
	})

	t.Run("returns worker instance ID after assignment", func(t *testing.T) {
		workerInstance := "test-worker-1"
		requestID := "test-request-1"

		// Set capacity
		err := mgr.SetWorkerTotalCapacity(ctx, envID, workerInstance, 5)
		require.NoError(t, err)

		// Assign request
		err = mgr.AssignRequestToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Get worker instance ID
		retrievedInstance, err := mgr.GetRequestWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, workerInstance, retrievedInstance)
	})

	t.Run("mapping is deleted after request completion", func(t *testing.T) {
		workerInstance := "test-worker-2"
		requestID := "test-request-2"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, workerInstance, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Verify mapping exists
		retrievedInstance, err := mgr.GetRequestWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, workerInstance, retrievedInstance)

		// Delete lease
		err = mgr.DeleteRequestFromWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Mapping should be deleted
		retrievedInstance, err = mgr.GetRequestWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, "", retrievedInstance)
	})

	t.Run("different requests map to different workers", func(t *testing.T) {
		worker1 := "test-worker-3"
		worker2 := "test-worker-4"
		request1 := "test-request-3"
		request2 := "test-request-4"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, worker1, 5)
		require.NoError(t, err)
		err = mgr.SetWorkerTotalCapacity(ctx, envID, worker2, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, worker1, request1)
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, worker2, request2)
		require.NoError(t, err)

		// Check mappings
		retrieved1, err := mgr.GetRequestWorkerInstanceID(ctx, envID, request1)
		require.NoError(t, err)
		require.Equal(t, worker1, retrieved1)

		retrieved2, err := mgr.GetRequestWorkerInstanceID(ctx, envID, request2)
		require.NoError(t, err)
		require.Equal(t, worker2, retrieved2)
	})

	t.Run("mapping has TTL set", func(t *testing.T) {
		workerInstance := "test-worker-5"
		requestID := "test-request-5"

		err := mgr.SetWorkerTotalCapacity(ctx, envID, workerInstance, 5)
		require.NoError(t, err)

		err = mgr.AssignRequestToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// Check TTL is set
		requestWorkerKey := mgr.requestWorkerKey(envID, requestID)
		ttl := r.TTL(requestWorkerKey)
		require.Greater(t, ttl, time.Duration(0))
		require.LessOrEqual(t, ttl, consts.ConnectWorkerCapacityManagerTTL)
	})

	t.Run("no mapping created when no capacity limit", func(t *testing.T) {
		workerInstance := "test-worker-no-limit"
		requestID := "test-request-no-limit"

		// Don't set capacity - worker is unlimited

		err := mgr.AssignRequestToWorker(ctx, envID, workerInstance, requestID)
		require.NoError(t, err)

		// No mapping should exist
		retrievedInstance, err := mgr.GetRequestWorkerInstanceID(ctx, envID, requestID)
		require.NoError(t, err)
		require.Equal(t, "", retrievedInstance)
	})
}

func TestGetAllActiveWorkerRequests(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	mgr := NewRedisConnectionStateManager(rc)
	ctx := context.Background()
	envID := uuid.New()

	t.Run("returns error for nil envID", func(t *testing.T) {
		instanceID := "test-instance"
		leases, err := mgr.getAllActiveWorkerRequests(ctx, uuid.Nil, instanceID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "envID cannot be nil")
		require.Nil(t, leases)
	})

	t.Run("returns error for empty instanceID", func(t *testing.T) {
		leases, err := mgr.getAllActiveWorkerRequests(ctx, envID, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "instanceID cannot be empty")
		require.Nil(t, leases)

		// Test with whitespace-only instanceID
		leases, err = mgr.getAllActiveWorkerRequests(ctx, envID, "   ")
		require.Error(t, err)
		require.Contains(t, err.Error(), "instanceID cannot be empty")
		require.Nil(t, leases)
	})

	t.Run("returns empty slice when no leases exist", func(t *testing.T) {
		instanceID := "non-existent-instance"
		leases, err := mgr.getAllActiveWorkerRequests(ctx, envID, instanceID)
		require.NoError(t, err)
		require.NotNil(t, leases)
		require.Equal(t, []string{}, leases)
	})

	t.Run("returns active leases correctly", func(t *testing.T) {
		instanceID := "test-instance-active"

		// Set capacity to enable lease tracking
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		// Add some leases
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "lease-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "lease-2")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "lease-3")
		require.NoError(t, err)

		leases, err := mgr.getAllActiveWorkerRequests(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Len(t, leases, 3)
		require.Contains(t, leases, "lease-1")
		require.Contains(t, leases, "lease-2")
		require.Contains(t, leases, "lease-3")
	})

	t.Run("filters out expired leases", func(t *testing.T) {
		instanceID := "test-instance-expired"

		// Set capacity to enable lease tracking
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		// Add lease that should be active
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "active-lease")
		require.NoError(t, err)

		// Manually add an expired lease to the sorted set
		setKey := mgr.workerRequestsKey(envID, instanceID)
		pastTime := time.Now().Add(-90 * time.Second).Unix()
		_, _ = r.ZAdd(setKey, float64(pastTime), "expired-lease")

		leases, err := mgr.getAllActiveWorkerRequests(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Len(t, leases, 1)
		require.Contains(t, leases, "active-lease")
		require.NotContains(t, leases, "expired-lease")
	})

	t.Run("handles mixed active and expired leases", func(t *testing.T) {
		instanceID := "test-instance-mixed"

		// Set capacity to enable lease tracking
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		// Add active leases
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "active-1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "active-2")
		require.NoError(t, err)

		// Manually add expired leases
		setKey := mgr.workerRequestsKey(envID, instanceID)
		pastTime1 := time.Now().Add(-2 * time.Hour).Unix()
		pastTime2 := time.Now().Add(-1 * time.Hour).Unix()
		_, _ = r.ZAdd(setKey, float64(pastTime1), "expired-1")
		_, _ = r.ZAdd(setKey, float64(pastTime2), "expired-2")

		leases, err := mgr.getAllActiveWorkerRequests(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Len(t, leases, 2)
		require.Contains(t, leases, "active-1")
		require.Contains(t, leases, "active-2")
		require.NotContains(t, leases, "expired-1")
		require.NotContains(t, leases, "expired-2")
	})

	t.Run("filters out empty lease values", func(t *testing.T) {
		instanceID := "test-instance-empty"

		// Set capacity to enable lease tracking
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		// Add valid lease
		err = mgr.AssignRequestToWorker(ctx, envID, instanceID, "valid-lease")
		require.NoError(t, err)

		// Manually add empty entries to the sorted set
		setKey := mgr.workerRequestsKey(envID, instanceID)
		futureTime := time.Now().Add(1 * time.Hour).Unix()
		_, _ = r.ZAdd(setKey, float64(futureTime), "")    // empty string
		_, _ = r.ZAdd(setKey, float64(futureTime), "   ") // whitespace only

		leases, err := mgr.getAllActiveWorkerRequests(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Len(t, leases, 1)
		require.Contains(t, leases, "valid-lease")
	})

	t.Run("handles large number of leases", func(t *testing.T) {
		instanceID := "test-instance-large"

		// Set capacity to enable lease tracking
		err := mgr.SetWorkerTotalCapacity(ctx, envID, instanceID, 1000)
		require.NoError(t, err)

		// Add many leases
		expectedLeases := make([]string, 100)
		for i := 0; i < 100; i++ {
			leaseID := fmt.Sprintf("lease-%d", i)
			expectedLeases[i] = leaseID
			err = mgr.AssignRequestToWorker(ctx, envID, instanceID, leaseID)
			require.NoError(t, err)
		}

		leases, err := mgr.getAllActiveWorkerRequests(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Len(t, leases, 100)

		// Check all expected leases are present
		for _, expectedLease := range expectedLeases {
			require.Contains(t, leases, expectedLease)
		}
	})

	t.Run("works with different envID and instanceID combinations", func(t *testing.T) {
		envID1 := uuid.New()
		envID2 := uuid.New()
		instanceID1 := "instance-1"
		instanceID2 := "instance-2"

		// Set capacity for both instances in both environments
		err := mgr.SetWorkerTotalCapacity(ctx, envID1, instanceID1, 5)
		require.NoError(t, err)
		err = mgr.SetWorkerTotalCapacity(ctx, envID1, instanceID2, 5)
		require.NoError(t, err)
		err = mgr.SetWorkerTotalCapacity(ctx, envID2, instanceID1, 5)
		require.NoError(t, err)

		// Add leases to different environments and instances
		err = mgr.AssignRequestToWorker(ctx, envID1, instanceID1, "env1-inst1-lease1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID1, instanceID2, "env1-inst2-lease1")
		require.NoError(t, err)
		err = mgr.AssignRequestToWorker(ctx, envID2, instanceID1, "env2-inst1-lease1")
		require.NoError(t, err)

		// Verify isolation
		leases1, err := mgr.getAllActiveWorkerRequests(ctx, envID1, instanceID1)
		require.NoError(t, err)
		require.Len(t, leases1, 1)
		require.Contains(t, leases1, "env1-inst1-lease1")

		leases2, err := mgr.getAllActiveWorkerRequests(ctx, envID1, instanceID2)
		require.NoError(t, err)
		require.Len(t, leases2, 1)
		require.Contains(t, leases2, "env1-inst2-lease1")

		leases3, err := mgr.getAllActiveWorkerRequests(ctx, envID2, instanceID1)
		require.NoError(t, err)
		require.Len(t, leases3, 1)
		require.Contains(t, leases3, "env2-inst1-lease1")
	})
}
