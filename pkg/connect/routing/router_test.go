package routing

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/util"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// For weighted shuffles generate a new rand.
var rnd = util.NewFrandRNG()

func setupRedis(t *testing.T) (state.StateManager, func()) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	stateMan := state.NewRedisConnectionStateManager(rc)

	return stateMan, func() {
		rc.Close()
	}
}

type setupRes struct {
	acctId    uuid.UUID
	envId     uuid.UUID
	gatewayId ulid.ULID

	appId   uuid.UUID
	syncId  uuid.UUID
	connIds []ulid.ULID
	appName string
	fnSlug  string
}

type testConnection struct {
	status          connectpb.ConnectionStatus
	lastHeartbeatAt time.Time
}

func newTestConn(status connectpb.ConnectionStatus, lastHeartbeatAt time.Time) testConnection {
	return testConnection{
		status:          status,
		lastHeartbeatAt: lastHeartbeatAt,
	}
}

type setupOpts struct {
	acctId    uuid.UUID
	envId     uuid.UUID
	gatewayId *ulid.ULID

	// use fnId if provided, otherwise default to "fn-1"
	fnId string

	// use appName if provided, otherwise default to "app-1"
	appName string

	// use appId if provided, otherwise create new
	appId uuid.UUID

	// use syncId if provided, otherwise create new
	syncId uuid.UUID
}

func setup(t *testing.T, stateMan state.StateManager, opts setupOpts, connsToCreate ...testConnection) setupRes {
	lastHeartbeatAt := time.Now()

	acctId, envId := uuid.New(), uuid.New()
	if opts.acctId != uuid.Nil {
		acctId = opts.acctId
	}

	if opts.envId != uuid.Nil {
		envId = opts.envId
	}

	gatewayId := ulid.MustNew(ulid.Now(), rand.Reader)
	if opts.gatewayId != nil {
		gatewayId = *opts.gatewayId
	}

	err := stateMan.UpsertGateway(context.Background(), &state.Gateway{
		Id:                gatewayId,
		Status:            state.GatewayStatusActive,
		LastHeartbeatAtMS: lastHeartbeatAt.UnixMilli(),
		Hostname:          "host-1",
	})
	require.NoError(t, err)

	fnId := "fn-1"
	if opts.fnId != "" {
		fnId = opts.fnId
	}

	appId, syncId := uuid.New(), uuid.New()
	if opts.appId != uuid.Nil {
		appId = opts.appId
	}

	if opts.syncId != uuid.Nil {
		syncId = opts.syncId
	}

	appName := "app-1"
	if opts.appName != "" {
		appName = opts.appName
	}

	caps, err := json.Marshal(sdk.Capabilities{
		Connect: sdk.ConnectV1,
	})
	require.NoError(t, err)

	fn1 := sdk.SDKFunction{
		Name: "Test Function",
		Slug: fmt.Sprintf("%s-%s", appName, fnId),
	}

	fnBytes, err := json.Marshal([]sdk.SDKFunction{fn1})
	require.NoError(t, err)

	app1Config := &connectpb.AppConfiguration{
		AppName:    appName,
		AppVersion: util.StrPtr("v1"),
		Functions:  fnBytes,
	}

	connIds := make([]ulid.ULID, len(connsToCreate))
	for i, connToCreate := range connsToCreate {
		connId := ulid.MustNew(ulid.Now(), rand.Reader)

		fakeReq := &connectpb.WorkerConnectRequestData{
			ConnectionId: connId.String(),
			InstanceId:   "my-worker",
			Apps: []*connectpb.AppConfiguration{
				app1Config,
			},
			SystemAttributes: &connectpb.SystemAttributes{
				CpuCores: 10,
				MemBytes: 1024 * 1024 * 16,
				Os:       "linux",
			},
			SdkVersion:   "fake-ver",
			SdkLanguage:  "fake-sdk",
			Capabilities: caps,
			AuthData: &connectpb.AuthData{
				SessionToken: "fake-session-token",
				SyncToken:    "fake-sync-token",
			},
		}

		group, err := state.NewWorkerGroupFromConnRequest(context.Background(), fakeReq, &auth.Response{
			AccountID: acctId,
			EnvID:     envId,
		}, app1Config)
		require.NoError(t, err)
		group.AppID = &appId
		group.SyncID = &syncId

		err = stateMan.UpsertConnection(context.Background(), &state.Connection{
			AccountID:    acctId,
			EnvID:        envId,
			ConnectionId: connId,
			WorkerIP:     "1.1.1.1",
			Data:         fakeReq,
			Groups: map[string]*state.WorkerGroup{
				group.Hash: group,
			},
			GatewayId: gatewayId,
		}, connToCreate.status, connToCreate.lastHeartbeatAt)
		require.NoError(t, err)

		connIds[i] = connId
	}

	return setupRes{
		acctId:    acctId,
		envId:     envId,
		gatewayId: gatewayId,
		appId:     appId,
		syncId:    syncId,
		connIds:   connIds,
		appName:   appName,
		fnSlug:    fn1.Slug,
	}
}

func TestFullConnectRouting(t *testing.T) {

	log := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	t.Run("single healthy connection should receive requests", func(t *testing.T) {
		stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
		)

		conn, err := getSuitableConnection(context.Background(), rnd, stateMan, setupRes.envId, setupRes.appId, setupRes.fnSlug, log)
		require.NoError(t, err)

		require.Equal(t, setupRes.connIds[0].String(), conn.Id)
		require.Equal(t, setupRes.gatewayId.String(), conn.GatewayId)
	})

	t.Run("unhealthy connection should be filtered out", func(t *testing.T) {
		stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connectpb.ConnectionStatus_CONNECTED, time.Now()),
			newTestConn(connectpb.ConnectionStatus_DISCONNECTING, time.Now()),
			newTestConn(connectpb.ConnectionStatus_DISCONNECTED, time.Now()),
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
			newTestConn(connectpb.ConnectionStatus_DRAINING, time.Now().Add(-time.Minute)),
		)

		conn, err := getSuitableConnection(context.Background(), rnd, stateMan, setupRes.envId, setupRes.appId, setupRes.fnSlug, log)
		require.NoError(t, err)

		require.Equal(t, setupRes.connIds[3].String(), conn.Id)
		require.Equal(t, setupRes.gatewayId.String(), conn.GatewayId)

		conns, err := stateMan.GetConnectionsByEnvID(context.Background(), setupRes.envId)
		require.NoError(t, err)
		require.Len(t, conns, 3) // disconnected conn should not be in there

		connIds := make([]string, len(conns))
		for i, metadata := range conns {
			connIds[i] = metadata.Id
		}

		require.Contains(t, connIds, setupRes.connIds[0].String())
		require.Contains(t, connIds, setupRes.connIds[1].String())
		require.Contains(t, connIds, setupRes.connIds[3].String())
	})

	t.Run("no healthy connection should be handled gracefully", func(t *testing.T) {
		stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connectpb.ConnectionStatus_DISCONNECTING, time.Now()),
			newTestConn(connectpb.ConnectionStatus_DISCONNECTED, time.Now()),
		)

		_, err := getSuitableConnection(context.Background(), rnd, stateMan, setupRes.envId, setupRes.appId, setupRes.fnSlug, log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoHealthyConnection)
	})

	t.Run("newer connection should be preferred", func(t *testing.T) {
		// This test is intuitively, but not actually correct. It's good to keep this as a reminder
		// on why the implementation works counter to this test: With connect, we want to prefer newer versions
		// but still send some traffic to connections running older versions. Routing thus doesn't filter out older versions,
		// it merely increases the chances of a connection running a newer version being picked.
		t.Skip("this test serves as an explainer and should not run")

		stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupOldVersion := setup(t, stateMan, setupOpts{
			fnId: "fn-1",
		},
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
		)

		<-time.After(1 * time.Second)

		setupNewVersion := setup(t, stateMan, setupOpts{
			acctId:    setupOldVersion.acctId,
			envId:     setupOldVersion.envId,
			gatewayId: &setupOldVersion.gatewayId,
			fnId:      "fn-1",
			appName:   setupOldVersion.appName,
			appId:     setupOldVersion.appId,
			syncId:    setupOldVersion.syncId,
		},
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
		)

		conn, err := getSuitableConnection(context.Background(), rnd, stateMan, setupOldVersion.envId, setupOldVersion.appId, setupOldVersion.fnSlug, log)
		require.NoError(t, err)
		require.Equal(t, setupNewVersion.connIds[0].String(), conn.Id)
		require.NotEqual(t, setupOldVersion.connIds[0].String(), conn.Id)
	})

	t.Run("connection without functions should be ignored", func(t *testing.T) {
		stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
		)

		// Try to route message for fn-1 (this does not exist)
		_, err := getSuitableConnection(context.Background(), rnd, stateMan, setupRes.envId, setupRes.appId, "fn-2", log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoHealthyConnection)
	})

	t.Run("connection without functions should be ignored, even if newer", func(t *testing.T) {
		stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupOldVersion := setup(t, stateMan, setupOpts{
			fnId: "fn-1",
		},
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
		)

		<-time.After(1 * time.Second)

		setupNewVersion := setup(t, stateMan, setupOpts{
			acctId:    setupOldVersion.acctId,
			envId:     setupOldVersion.envId,
			gatewayId: &setupOldVersion.gatewayId,
			fnId:      "fn-2", // note: different fn slug
			appName:   setupOldVersion.appName,
			appId:     setupOldVersion.appId,
			syncId:    setupOldVersion.syncId,
		},
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
			newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
		)

		// Try to route message for fn-1 (this does not exist in newer version)
		conn, err := getSuitableConnection(context.Background(), rnd, stateMan, setupOldVersion.envId, setupOldVersion.appId, setupOldVersion.fnSlug, log)
		require.NoError(t, err)
		require.NotEqual(t, setupNewVersion.connIds[0].String(), conn.Id)
		require.Equal(t, setupOldVersion.connIds[0].String(), conn.Id)
	})
}

func TestIsHealthy(t *testing.T) {
	log := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	stateMan, cleanup := setupRedis(t)
	defer cleanup()

	type testCase struct {
		name           string
		status         connectpb.ConnectionStatus
		heartbeatDelay time.Duration
		expected       isHealthyRes
	}

	cases := []testCase{
		{
			name:   "ready connection should be marked as healthy",
			status: connectpb.ConnectionStatus_READY,
			expected: isHealthyRes{
				isHealthy:                       true,
				shouldDeleteUnhealthyConnection: false,
				shouldDeleteUnhealthyGateway:    false,
			},
		},
		{
			name:           "ready but timed out connection should not be marked as healthy",
			status:         connectpb.ConnectionStatus_READY,
			heartbeatDelay: 3 * consts.ConnectWorkerHeartbeatInterval,
			expected: isHealthyRes{
				isHealthy:                       false,
				shouldDeleteUnhealthyConnection: true,
				shouldDeleteUnhealthyGateway:    false,
			},
		},
		{
			name:   "non-ready connection should not be marked as healthy",
			status: connectpb.ConnectionStatus_DISCONNECTING,
			expected: isHealthyRes{
				isHealthy:                       false,
				shouldDeleteUnhealthyConnection: false,
				shouldDeleteUnhealthyGateway:    false,
			},
		},
		{
			name:   "disconnected connection should be cleaned up",
			status: connectpb.ConnectionStatus_DISCONNECTED,
			expected: isHealthyRes{
				isHealthy:                       false,
				shouldDeleteUnhealthyConnection: true,
				shouldDeleteUnhealthyGateway:    false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupRes := setup(t, stateMan, setupOpts{},
				newTestConn(tc.status, time.Now().Add(-tc.heartbeatDelay)),
			)

			conn, err := stateMan.GetConnection(context.Background(), setupRes.envId, setupRes.connIds[0])
			require.NoError(t, err)
			res := isHealthy(context.Background(), stateMan, setupRes.envId, setupRes.appId, setupRes.fnSlug, conn, log)

			require.Equal(t, tc.expected.isHealthy, res.isHealthy, "expected isHealthy to match")
			require.Equal(t, tc.expected.shouldDeleteUnhealthyGateway, res.shouldDeleteUnhealthyGateway, "expected shouldDeleteUnhealthyGateway to match")
			require.Equal(t, tc.expected.shouldDeleteUnhealthyConnection, res.shouldDeleteUnhealthyConnection, "expected shouldDeleteUnhealthyConnection to match")
		})
	}

}

func TestGetConnectionWeight(t *testing.T) {
	createVersion := func(prefix string, versionTime time.Time, numCandidates int) []connWithGroup {
		conns := make([]connWithGroup, numCandidates)
		group := &state.WorkerGroup{CreatedAt: versionTime}
		for i := range numCandidates {
			conns[i] = connWithGroup{
				conn: &connectpb.ConnMetadata{
					Id: fmt.Sprintf("%s-%d", prefix, i),
				},
				group: group,
			}
		}
		return conns
	}

	t.Run("newer connections should receive higher weights", func(t *testing.T) {
		candidates := make([]connWithGroup, 0)

		t1 := time.Date(2025, 02, 25, 0, 0, 0, 0, time.Local)
		oldConns := createVersion("old", t1, 3)
		candidates = append(candidates, oldConns...)

		diff := 10 * time.Minute

		t2 := t1.Add(diff)
		newConns := createVersion("new", t2, 3)
		candidates = append(candidates, newConns...)

		sortByGroupCreatedAt(candidates)

		distr := getVersionTimeDistribution(candidates)
		require.Equal(t, distr.timeRange, diff.Seconds())
		require.Equal(t, t1, distr.oldestVersionCreatedAt)
		require.Equal(t, t2, distr.newestVersionCreatedAt)

		oldConnWeight := getConnectionWeight(distr.timeRange, distr.oldestVersionCreatedAt, oldConns[0])
		require.NotZero(t, oldConnWeight)

		newConnWeight := getConnectionWeight(distr.timeRange, distr.oldestVersionCreatedAt, newConns[0])
		require.NotZero(t, newConnWeight)

		require.Greater(t, newConnWeight, oldConnWeight)
	})

	t.Run("scoring should work with a single group", func(t *testing.T) {
		candidates := make([]connWithGroup, 0)

		t1 := time.Date(2025, 02, 25, 0, 0, 0, 0, time.Local)
		oldConns := createVersion("old", t1, 3)
		candidates = append(candidates, oldConns...)

		diff := time.Duration(0)

		sortByGroupCreatedAt(candidates)

		distr := getVersionTimeDistribution(candidates)
		require.Equal(t, distr.timeRange, diff.Seconds())
		require.Equal(t, t1, distr.oldestVersionCreatedAt)
		require.Equal(t, t1, distr.newestVersionCreatedAt)

		oldConnWeight := getConnectionWeight(distr.timeRange, distr.oldestVersionCreatedAt, oldConns[0])
		require.NotZero(t, oldConnWeight)
	})

	t.Run("scoring should work with multiple groups", func(t *testing.T) {
		candidates := make([]connWithGroup, 0)

		t0 := time.Date(2025, 02, 25, 0, 0, 0, 0, time.Local)
		numGroups := 10
		diff := 10 * time.Minute
		for i := range numGroups {
			tn := t0.Add(time.Duration(i) * diff)
			conns := createVersion(fmt.Sprintf("v-%d", i), tn, 1)
			candidates = append(candidates, conns...)
		}

		sortByGroupCreatedAt(candidates)

		distr := getVersionTimeDistribution(candidates)
		require.Equal(t, distr.timeRange, (time.Duration(numGroups-1) * diff).Seconds())
		require.Equal(t, t0, distr.oldestVersionCreatedAt)
		require.Equal(t, t0.Add(time.Duration(numGroups-1)*diff), distr.newestVersionCreatedAt)

		for i := range candidates {
			if i == 0 || i == len(candidates)-1 {
				continue
			}

			oldConnWeight := getConnectionWeight(distr.timeRange, distr.oldestVersionCreatedAt, candidates[i-1])
			newConnWeight := getConnectionWeight(distr.timeRange, distr.oldestVersionCreatedAt, candidates[i])

			require.Greater(t, newConnWeight, oldConnWeight)
		}
	})
}

func TestCheckCapacity(t *testing.T) {
	ctx := context.Background()
	stateMan, cleanup := setupRedis(t)
	defer cleanup()

	log := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	envID := uuid.New()

	t.Run("worker without capacity limit should have capacity", func(t *testing.T) {
		instanceID := "worker-no-limit"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-1",
			InstanceId: instanceID,
		}

		// No capacity limit set for this worker
		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.True(t, res.hasWorkerCapacity)
	})

	t.Run("worker with capacity limit and no leases should have capacity", func(t *testing.T) {
		instanceID := "worker-with-limit"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-2",
			InstanceId: instanceID,
		}

		// Set capacity limit
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.True(t, res.hasWorkerCapacity)
	})

	t.Run("worker with partial capacity usage should have capacity", func(t *testing.T) {
		instanceID := "worker-partial-usage"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-3",
			InstanceId: instanceID,
		}

		// Set capacity limit
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 10)
		require.NoError(t, err)

		// Assign some leases (partial usage)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.True(t, res.hasWorkerCapacity)

		// Verify available capacity
		capacity, err := stateMan.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(7), capacity.Available) // 10 - 3 = 7
	})

	t.Run("worker at full capacity should not have capacity", func(t *testing.T) {
		instanceID := "worker-at-capacity"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-4",
			InstanceId: instanceID,
		}

		// Set capacity limit
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 3)
		require.NoError(t, err)

		// Fill capacity completely
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.False(t, res.hasWorkerCapacity)

		// Verify capacity is at zero
		capacity, err := stateMan.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity.Available)
		require.True(t, capacity.IsAtCapacity())
	})

	t.Run("capacity changes with lease assignments and releases", func(t *testing.T) {
		instanceID := "worker-dynamic-capacity"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-5",
			InstanceId: instanceID,
		}

		// Set capacity limit
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 5)
		require.NoError(t, err)

		// Initially should have capacity
		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.True(t, res.hasWorkerCapacity)

		// Assign leases one by one and verify capacity decreases
		for i := 1; i <= 5; i++ {
			err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, fmt.Sprintf("req-%d", i))
			require.NoError(t, err)

			capacity, err := stateMan.GetWorkerCapacities(ctx, envID, instanceID)
			require.NoError(t, err)
			require.Equal(t, int64(5-i), capacity.Available)

			res = checkCapacity(ctx, stateMan, envID, conn, log)
			if i < 5 {
				require.True(t, res.hasWorkerCapacity, "should have capacity after %d assignments", i)
			} else {
				require.False(t, res.hasWorkerCapacity, "should not have capacity after %d assignments", i)
			}
		}

		// Release leases one by one and verify capacity increases
		for i := 1; i <= 5; i++ {
			err = stateMan.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, fmt.Sprintf("req-%d", i))
			require.NoError(t, err)

			capacity, err := stateMan.GetWorkerCapacities(ctx, envID, instanceID)
			require.NoError(t, err)
			require.Equal(t, int64(i), capacity.Available)

			res = checkCapacity(ctx, stateMan, envID, conn, log)
			require.True(t, res.hasWorkerCapacity, "should have capacity after releasing %d leases", i)
		}
	})

	t.Run("capacity limit can be updated", func(t *testing.T) {
		instanceID := "worker-capacity-update"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-6",
			InstanceId: instanceID,
		}

		// Set initial capacity
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 3)
		require.NoError(t, err)

		// Assign leases up to capacity
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		// Should be at capacity
		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.False(t, res.hasWorkerCapacity)

		// Increase capacity
		err = stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 6)
		require.NoError(t, err)

		// Should now have capacity
		res = checkCapacity(ctx, stateMan, envID, conn, log)
		require.True(t, res.hasWorkerCapacity)

		capacity, err := stateMan.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		require.Equal(t, int64(3), capacity.Available) // 6 - 3 = 3

		// Decrease capacity below current usage
		err = stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Should not have capacity (over limit)
		capacity, err = stateMan.GetWorkerCapacities(ctx, envID, instanceID)
		require.NoError(t, err)
		// 2 - 3 = -1 (over capacity), but we take a max with 0
		require.Equal(t, int64(0), capacity.Available)
		require.True(t, capacity.IsAtCapacity())

		res = checkCapacity(ctx, stateMan, envID, conn, log)
		require.False(t, res.hasWorkerCapacity)
	})

	t.Run("capacity can be removed by setting to zero", func(t *testing.T) {
		instanceID := "worker-remove-capacity"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-7",
			InstanceId: instanceID,
		}

		// Set capacity and assign leases
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Should be at capacity
		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.False(t, res.hasWorkerCapacity)

		// Remove capacity limit (set to 0)
		err = stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 0)
		require.NoError(t, err)

		// Should now have unlimited capacity
		res = checkCapacity(ctx, stateMan, envID, conn, log)
		require.True(t, res.hasWorkerCapacity)
	})

	t.Run("connection without instance ID should not have capacity", func(t *testing.T) {
		conn := &connectpb.ConnMetadata{
			Id:         "conn-no-instance",
			InstanceId: "", // Empty instance ID
		}

		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.False(t, res.hasWorkerCapacity)
	})

	t.Run("concurrent lease assignments respect capacity limits", func(t *testing.T) {
		instanceID := "worker-concurrent"
		conn := &connectpb.ConnMetadata{
			Id:         "conn-8",
			InstanceId: instanceID,
		}

		// Set small capacity
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID, 2)
		require.NoError(t, err)

		// Assign leases
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-2")
		require.NoError(t, err)

		// Should be at capacity
		res := checkCapacity(ctx, stateMan, envID, conn, log)
		require.False(t, res.hasWorkerCapacity)

		// Release one lease
		err = stateMan.DeleteRequestLeaseFromWorker(ctx, envID, instanceID, "req-1")
		require.NoError(t, err)

		// Should have capacity again
		res = checkCapacity(ctx, stateMan, envID, conn, log)
		require.True(t, res.hasWorkerCapacity)

		// Assign another lease
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID, "req-3")
		require.NoError(t, err)

		// Should be at capacity again
		res = checkCapacity(ctx, stateMan, envID, conn, log)
		require.False(t, res.hasWorkerCapacity)
	})

	t.Run("multiple workers can have independent capacity tracking", func(t *testing.T) {
		instanceID1 := "worker-independent-1"
		instanceID2 := "worker-independent-2"
		conn1 := &connectpb.ConnMetadata{
			Id:         "conn-9",
			InstanceId: instanceID1,
		}
		conn2 := &connectpb.ConnMetadata{
			Id:         "conn-10",
			InstanceId: instanceID2,
		}

		// Set different capacities
		err := stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID1, 3)
		require.NoError(t, err)
		err = stateMan.SetWorkerTotalCapacity(ctx, envID, instanceID2, 5)
		require.NoError(t, err)

		// Fill first worker to capacity
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID1, "req-1-1")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID1, "req-1-2")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID1, "req-1-3")
		require.NoError(t, err)

		// Partially fill second worker
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID2, "req-2-1")
		require.NoError(t, err)
		err = stateMan.AssignRequestLeaseToWorker(ctx, envID, instanceID2, "req-2-2")
		require.NoError(t, err)

		// Check capacities independently
		res1 := checkCapacity(ctx, stateMan, envID, conn1, log)
		require.False(t, res1.hasWorkerCapacity) // First worker at capacity

		res2 := checkCapacity(ctx, stateMan, envID, conn2, log)
		require.True(t, res2.hasWorkerCapacity) // Second worker has capacity

		// Verify actual capacities
		capacity1, err := stateMan.GetWorkerCapacities(ctx, envID, instanceID1)
		require.NoError(t, err)
		require.Equal(t, int64(0), capacity1.Available)

		capacity2, err := stateMan.GetWorkerCapacities(ctx, envID, instanceID2)
		require.NoError(t, err)
		require.Equal(t, int64(3), capacity2.Available) // 5 - 2 = 3
	})
}
