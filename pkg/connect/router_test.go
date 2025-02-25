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
	ctx := context.Background()

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	stateMan := state.NewRedisConnectionStateManager(rc)
	rcv, _ := pubsub.NewConnector(ctx, pubsub.WithNoop())

	cond := trace.NewConditionalTracer(noop.Tracer{}, trace.AlwaysTrace)

	svc := NewConnectMessageRouterService(
		stateMan,
		rcv,
		cond,
	)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("healthy connection should receive requests", func(t *testing.T) {
		acctId, envId, appId, syncId := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		connId, gatewayId := ulid.MustNew(ulid.Now(), rand.Reader), ulid.MustNew(ulid.Now(), rand.Reader)

		appName := "app-1"

		fn1 := sdk.SDKFunction{
			Name: "Test Function",
			Slug: fmt.Sprintf("%s-fn-1", appName),
		}

		fnBytes, err := json.Marshal([]sdk.SDKFunction{fn1})
		require.NoError(t, err)

		app1Config := &connect.AppConfiguration{
			AppName:    appName,
			AppVersion: util.StrPtr("v1"),
			Functions:  fnBytes,
		}

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
			SdkVersion:  "fake-ver",
			SdkLanguage: "fake-sdk",
		}

		group, err := NewWorkerGroupFromConnRequest(ctx, fakeReq, &auth.Response{
			AccountID: acctId,
			EnvID:     envId,
		}, app1Config)
		require.NoError(t, err)
		group.AppID = &appId
		group.SyncID = &syncId

		err = stateMan.UpsertConnection(ctx, &state.Connection{
			AccountID:    acctId,
			EnvID:        envId,
			ConnectionId: connId,
			WorkerIP:     "1.1.1.1",
			Data:         fakeReq,
			Groups: map[string]*state.WorkerGroup{
				group.Hash: group,
			},
			GatewayId: gatewayId,
		}, connect.ConnectionStatus_READY, time.Now().Truncate(time.Minute))
		require.NoError(t, err)

		conn, err := svc.getSuitableConnection(ctx, envId, appId, app1Config.AppName, fn1.Slug, log)
		require.NoError(t, err)

		require.Equal(t, connId.String(), conn.Id)
		require.Equal(t, gatewayId.String(), conn.GatewayId)
	})

	t.Run("unhealthy connection should be filtered out", func(t *testing.T) {

	})

	t.Run("no healthy connection should be handled gracefully", func(t *testing.T) {

	})

	t.Run("newer connection should be preferred", func(t *testing.T) {

	})
}
