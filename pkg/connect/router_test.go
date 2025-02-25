package connect

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"log/slog"
	"os"
	"testing"
	"time"
)

func setupRedis(t *testing.T) (*connectRouterSvc, state.StateManager, func()) {
	ctx := context.Background()

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	stateMan := state.NewRedisConnectionStateManager(rc)
	rcv, _ := pubsub.NewConnector(ctx, pubsub.WithNoop())

	cond := trace.NewConditionalTracer(noop.Tracer{}, trace.AlwaysTrace)

	svc := NewConnectMessageRouterService(
		stateMan,
		rcv,
		cond,
	)
	require.NoError(t, svc.Pre(context.Background()))

	return svc, stateMan, func() {
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
	status          connect.ConnectionStatus
	lastHeartbeatAt time.Time
}

func newTestConn(status connect.ConnectionStatus, lastHeartbeatAt time.Time) testConnection {
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
		Id:              gatewayId,
		Status:          state.GatewayStatusActive,
		LastHeartbeatAt: lastHeartbeatAt,
		Hostname:        "host-1",
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

	app1Config := &connect.AppConfiguration{
		AppName:    appName,
		AppVersion: util.StrPtr("v1"),
		Functions:  fnBytes,
	}

	connIds := make([]ulid.ULID, len(connsToCreate))
	for i, connToCreate := range connsToCreate {
		connId := ulid.MustNew(ulid.Now(), rand.Reader)

		fakeReq := &connect.WorkerConnectRequestData{
			ConnectionId: connId.String(),
			InstanceId:   "my-worker",
			Apps: []*connect.AppConfiguration{
				app1Config,
			},
			SystemAttributes: &connect.SystemAttributes{
				CpuCores: 10,
				MemBytes: 1024 * 1024 * 16,
				Os:       "linux",
			},
			SdkVersion:   "fake-ver",
			SdkLanguage:  "fake-sdk",
			Capabilities: caps,
			AuthData: &connect.AuthData{
				SessionToken: "fake-session-token",
				SyncToken:    "fake-sync-token",
			},
		}

		group, err := NewWorkerGroupFromConnRequest(context.Background(), fakeReq, &auth.Response{
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

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("single healthy connection should receive requests", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		conn, err := svc.getSuitableConnection(context.Background(), setupRes.envId, setupRes.appId, setupRes.fnSlug, log)
		require.NoError(t, err)

		require.Equal(t, setupRes.connIds[0].String(), conn.Id)
		require.Equal(t, setupRes.gatewayId.String(), conn.GatewayId)
	})

	t.Run("unhealthy connection should be filtered out", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connect.ConnectionStatus_CONNECTED, time.Now()),
			newTestConn(connect.ConnectionStatus_DISCONNECTING, time.Now()),
			newTestConn(connect.ConnectionStatus_DISCONNECTED, time.Now()),
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		conn, err := svc.getSuitableConnection(context.Background(), setupRes.envId, setupRes.appId, setupRes.fnSlug, log)
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
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connect.ConnectionStatus_DISCONNECTING, time.Now()),
			newTestConn(connect.ConnectionStatus_DISCONNECTED, time.Now()),
		)

		_, err := svc.getSuitableConnection(context.Background(), setupRes.envId, setupRes.appId, setupRes.fnSlug, log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoHealthyConnection)
	})

	t.Run("newer connection should be preferred", func(t *testing.T) {
		// This test is intuitively, but not actually correct. It's good to keep this as a reminder
		// on why the implementation works counter to this test: With connect, we want to prefer newer versions
		// but still send some traffic to connections running older versions. Routing thus doesn't filter out older versions,
		// it merely increases the chances of a connection running a newer version being picked.
		t.Skip("this test serves as an explainer and should not run")

		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupOldVersion := setup(t, stateMan, setupOpts{
			fnId: "fn-1",
		},
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
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
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		conn, err := svc.getSuitableConnection(context.Background(), setupOldVersion.envId, setupOldVersion.appId, setupOldVersion.fnSlug, log)
		require.NoError(t, err)
		require.Equal(t, setupNewVersion.connIds[0].String(), conn.Id)
		require.NotEqual(t, setupOldVersion.connIds[0].String(), conn.Id)
	})

	t.Run("connection without functions should be ignored", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{},
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		// Try to route message for fn-1 (this does not exist)
		_, err := svc.getSuitableConnection(context.Background(), setupRes.envId, setupRes.appId, "fn-2", log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoHealthyConnection)
	})

	t.Run("connection without functions should be ignored, even if newer", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupOldVersion := setup(t, stateMan, setupOpts{
			fnId: "fn-1",
		},
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
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
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		// Try to route message for fn-1 (this does not exist in newer version)
		conn, err := svc.getSuitableConnection(context.Background(), setupOldVersion.envId, setupOldVersion.appId, setupOldVersion.fnSlug, log)
		require.NoError(t, err)
		require.NotEqual(t, setupNewVersion.connIds[0].String(), conn.Id)
		require.Equal(t, setupOldVersion.connIds[0].String(), conn.Id)
	})
}

func TestIsHealthy(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("ready connection should be marked as healthy", func(t *testing.T) {
		_, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(t, stateMan, setupOpts{}, newTestConn(connect.ConnectionStatus_READY, time.Now()))

		conn, err := stateMan.GetConnection(context.Background(), setupRes.envId, setupRes.connIds[0])
		require.NoError(t, err)

		res := isHealthy(context.Background(), stateMan, setupRes.envId, setupRes.appId, setupRes.fnSlug, conn, log)

		require.True(t, res.isHealthy)
		require.False(t, res.shouldDeleteUnhealthyConnection)
		require.False(t, res.shouldDeleteUnhealthyGateway)
	})
}

func TestGetConnectionWeight(t *testing.T) {
	createVersion := func(prefix string, versionTime time.Time, numCandidates int) []connWithGroup {
		conns := make([]connWithGroup, numCandidates)
		group := &state.WorkerGroup{CreatedAt: versionTime}
		for i := range numCandidates {
			conns[i] = connWithGroup{
				conn: &connect.ConnMetadata{
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

func TestPickConnection(t *testing.T) {}
