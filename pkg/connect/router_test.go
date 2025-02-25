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

func TestFullConnectRouting(t *testing.T) {
	setupRedis := func(t *testing.T) (*connectRouterSvc, state.StateManager, func()) {
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

	acctId, envId := uuid.New(), uuid.New()
	gatewayId := ulid.MustNew(ulid.Now(), rand.Reader)

	type setupRes struct {
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

	newTestConn := func(status connect.ConnectionStatus, lastHeartbeatAt time.Time) testConnection {
		return testConnection{
			status:          status,
			lastHeartbeatAt: lastHeartbeatAt,
		}
	}

	setup := func(stateMan state.StateManager, fnId string, connsToCreate ...testConnection) setupRes {
		lastHeartbeatAt := time.Now()

		err := stateMan.UpsertGateway(context.Background(), &state.Gateway{
			Id:              gatewayId,
			Status:          state.GatewayStatusActive,
			LastHeartbeatAt: lastHeartbeatAt,
			Hostname:        "host-1",
		})
		require.NoError(t, err)

		appId, syncId := uuid.New(), uuid.New()

		appName := "app-1"

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
			appId:   appId,
			syncId:  syncId,
			connIds: connIds,
			appName: appName,
			fnSlug:  fn1.Slug,
		}
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("single healthy connection should receive requests", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(stateMan, "fn-1",
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		conn, err := svc.getSuitableConnection(context.Background(), envId, setupRes.appId, setupRes.fnSlug, log)
		require.NoError(t, err)

		require.Equal(t, setupRes.connIds[0].String(), conn.Id)
		require.Equal(t, gatewayId.String(), conn.GatewayId)
	})

	t.Run("unhealthy connection should be filtered out", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(stateMan, "fn-1",
			newTestConn(connect.ConnectionStatus_CONNECTED, time.Now()),
			newTestConn(connect.ConnectionStatus_DISCONNECTING, time.Now()),
			newTestConn(connect.ConnectionStatus_DISCONNECTED, time.Now()),
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		conn, err := svc.getSuitableConnection(context.Background(), envId, setupRes.appId, setupRes.fnSlug, log)
		require.NoError(t, err)

		require.Equal(t, setupRes.connIds[3].String(), conn.Id)
		require.Equal(t, gatewayId.String(), conn.GatewayId)

		conns, err := stateMan.GetConnectionsByEnvID(context.Background(), envId)
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

		setupRes := setup(stateMan, "fn-1",
			newTestConn(connect.ConnectionStatus_DISCONNECTING, time.Now()),
			newTestConn(connect.ConnectionStatus_DISCONNECTED, time.Now()),
		)

		_, err := svc.getSuitableConnection(context.Background(), envId, setupRes.appId, setupRes.fnSlug, log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoHealthyConnection)
	})

	t.Run("newer connection should be preferred", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(stateMan, "fn-1",
			newTestConn(connect.ConnectionStatus_DISCONNECTING, time.Now()),
			newTestConn(connect.ConnectionStatus_DISCONNECTED, time.Now()),
		)

		_, err := svc.getSuitableConnection(context.Background(), envId, setupRes.appId, setupRes.fnSlug, log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoHealthyConnection)
	})

	t.Run("connection without functions should be ignored, even if newer", func(t *testing.T) {
		svc, stateMan, cleanup := setupRedis(t)
		defer cleanup()

		setupRes := setup(stateMan, "fn-1",
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
			newTestConn(connect.ConnectionStatus_READY, time.Now()),
		)

		_, err := svc.getSuitableConnection(context.Background(), envId, setupRes.appId, "fn-2", log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoHealthyConnection)
	})
}
