package connect

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/aws/smithy-go/ptr"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs/sync"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
	"net"
	"net/http"
	"os"
	sync2 "sync"
	"testing"
	"time"
)

func TestCloseConnectionOnConsecutiveHeartbeatFail(t *testing.T) {
	res := createTestingGateway(t)

	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_HELLO, msg.Kind)

	sendWorkerConnectMessage(t, res)

	msg = awaitNextMessage(t, res.ws, 5*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_CONNECTION_READY, msg.Kind)
}

type websocketDisconnected struct {
	conn        *state.Connection
	closeReason string
}

type testRecorderLifecycles struct {
	onConnected          []*state.Connection
	onReady              []*state.Connection
	onHeartbeat          []*state.Connection
	onStartDraining      []*state.Connection
	onStartDisconnecting []*state.Connection
	onSynced             []*state.Connection
	onDisconnected       []websocketDisconnected
}

func (r *testRecorderLifecycles) OnConnected(ctx context.Context, conn *state.Connection) {
	r.onConnected = append(r.onConnected, conn)
}

func (r *testRecorderLifecycles) OnReady(ctx context.Context, conn *state.Connection) {
	r.onReady = append(r.onReady, conn)
}

func (r *testRecorderLifecycles) OnHeartbeat(ctx context.Context, conn *state.Connection) {
	r.onHeartbeat = append(r.onHeartbeat, conn)
}

func (r *testRecorderLifecycles) OnStartDraining(ctx context.Context, conn *state.Connection) {
	r.onStartDraining = append(r.onStartDraining, conn)
}

func (r *testRecorderLifecycles) OnStartDisconnecting(ctx context.Context, conn *state.Connection) {
	r.onStartDisconnecting = append(r.onStartDisconnecting, conn)
}

func (r *testRecorderLifecycles) OnSynced(ctx context.Context, conn *state.Connection) {
	r.onSynced = append(r.onSynced, conn)
}

func (r *testRecorderLifecycles) OnDisconnected(ctx context.Context, conn *state.Connection, closeReason string) {
	r.onDisconnected = append(r.onDisconnected, websocketDisconnected{conn, closeReason})
}

func newRecorderLifecycles() *testRecorderLifecycles {
	r := &testRecorderLifecycles{}
	r.reset()
	return r
}

func (r *testRecorderLifecycles) reset() {
	r.onDisconnected = make([]websocketDisconnected, 0)
	r.onSynced = make([]*state.Connection, 0)
	r.onStartDisconnecting = make([]*state.Connection, 0)
	r.onStartDraining = make([]*state.Connection, 0)
	r.onHeartbeat = make([]*state.Connection, 0)
	r.onReady = make([]*state.Connection, 0)
	r.onConnected = make([]*state.Connection, 0)
}

type testingResources struct {
	redis        *miniredis.Miniredis
	rc           rueidis.Client
	stateManager state.StateManager
	testConn     *testingConnector

	ws         *websocket.Conn
	lifecycles *testRecorderLifecycles

	envID     uuid.UUID
	accountID uuid.UUID
	syncID    uuid.UUID
	appID     uuid.UUID

	connID ulid.ULID
}

func createTestingGateway(t *testing.T) testingResources {
	envID, accountID := uuid.New(), uuid.New()
	syncID, appID := uuid.New(), uuid.New()

	ctx := context.Background()
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		rc.Close()
	})

	connManager := state.NewRedisConnectionStateManager(rc)

	testConn := &testingConnector{}

	conn, err := pubsub.NewConnector(ctx, withTestingConnector(testConn))
	require.NoError(t, err)

	var fakeApiBaseUrl string
	{
		// 1. Create a listener on a random port by specifying port 0
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Cannot create listener: %v", err)
		}

		// Get the assigned port
		port := listener.Addr().(*net.TCPAddr).Port
		fakeApiBaseUrl = fmt.Sprintf("http://127.0.0.1:%d", port)

		mux := http.NewServeMux()

		srv := http.Server{
			Handler: mux,
		}

		go func() {
			_ = srv.Serve(listener)
		}()
		t.Cleanup(func() {
			_ = srv.Shutdown(ctx)
		})

		reply, err := json.Marshal(sync.Reply{
			OK:     true,
			SyncID: &syncID,
			AppID:  &appID,
		})
		require.NoError(t, err)

		// Emulate sync endpoint
		mux.HandleFunc("POST /fn/register", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(reply)
		})
	}

	gwPort := freePort()

	websocketUrl := fmt.Sprintf("ws://127.0.0.1:%d/v0/connect", gwPort)

	lifecycles := newRecorderLifecycles()

	svc := NewConnectGatewayService(
		WithGatewayAuthHandler(func(ctx context.Context, data *connect.WorkerConnectRequestData) (*auth.Response, error) {
			return &auth.Response{
				AccountID: accountID,
				EnvID:     envID,
				Entitlements: auth.Entitlements{
					ConnectionAllowed: true,
					AppsPerConnection: 10,
				},
			}, nil
		}),
		WithConnectionStateManager(connManager),
		WithGroupName("gw-1"),
		WithRequestReceiver(conn),
		WithLifeCycles([]ConnectGatewayLifecycleListener{lifecycles}),
		WithApiBaseUrl(fakeApiBaseUrl),
		WithGatewayPublicPort(gwPort),

		WithWorkerHeartbeatInterval(consts.ConnectGatewayHeartbeatInterval),
		WithWorkerRequestLeaseDuration(consts.ConnectWorkerRequestLeaseDuration),
		WithWorkerExtendLeaseInterval(consts.ConnectWorkerRequestExtendLeaseInterval),
	)

	require.NoError(t, svc.Pre(ctx))

	svc.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	go func() {
		err := svc.Run(ctx)
		if err != nil {
			require.ErrorIs(t, err, context.Canceled)
		}
	}()
	t.Cleanup(func() {
		_ = svc.Stop(context.Background())
	})

	ws, _, err := websocket.Dial(ctx, websocketUrl, &websocket.DialOptions{
		Subprotocols: []string{types.GatewaySubProtocol},
	})
	require.NoError(t, err)

	return testingResources{
		redis:        r,
		rc:           rc,
		stateManager: connManager,
		testConn:     testConn,
		ws:           ws,
		lifecycles:   lifecycles,
		envID:        envID,
		accountID:    accountID,
		syncID:       syncID,
		appID:        appID,
		connID:       ulid.MustNew(ulid.Now(), rand.Reader),
	}
}

func awaitNextMessage(t *testing.T, ws *websocket.Conn, timeout time.Duration) *connect.ConnectMessage {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	parsed := connect.ConnectMessage{}
	err := wsproto.Read(ctx, ws, &parsed)
	require.NoError(t, err)

	return &parsed
}

func sendWorkerConnectMessage(t *testing.T, res testingResources) {
	ctx := context.Background()

	caps, err := json.Marshal(sdk.Capabilities{
		InBandSync: sdk.InBandSyncV1,
		TrustProbe: sdk.TrustProbeV1,
		Connect:    sdk.ConnectV1,
	})
	require.NoError(t, err)

	fns, err := json.Marshal([]sdk.SDKFunction{
		{
			Name: "test-fn",
			Slug: "test-app-test-fn",
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: "hello/world",
					},
				},
			},
			Steps: map[string]sdk.SDKStep{
				"step": sdk.SDKStep{
					ID:   "step",
					Name: "test-fn",
					Runtime: map[string]any{
						"url": "ws://connect?fnId=test-app-test-fn&step=step",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	testApp := &connect.AppConfiguration{
		AppName:    "test-app",
		AppVersion: ptr.String("v1"),
		Functions:  fns,
	}

	connectMsg, err := proto.Marshal(&connect.WorkerConnectRequestData{
		ConnectionId: res.connID.String(),
		InstanceId:   "test-worker",
		AuthData: &connect.AuthData{
			SessionToken: "test-session-token",
			SyncToken:    "test-sync-token",
		},
		Capabilities:             caps,
		Apps:                     []*connect.AppConfiguration{testApp},
		WorkerManualReadinessAck: false,
		SystemAttributes: &connect.SystemAttributes{
			CpuCores: 4,
			MemBytes: 1024 * 1024 * 1024,
			Os:       "linux-test",
		},
		Environment: nil,
		Framework:   "",
		Platform:    nil,
		SdkVersion:  "test-sdk",
		SdkLanguage: "test-lang",
		StartedAt:   timestamppb.Now(),
	})
	require.NoError(t, err)

	err = wsproto.Write(ctx, res.ws, &connect.ConnectMessage{
		Kind:    connect.GatewayMessageType_WORKER_CONNECT,
		Payload: connectMsg,
	})
	require.NoError(t, err)
}

func freePort() int {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func withTestingConnector(t *testingConnector) pubsub.ConnectorOpt {
	return func(ctx context.Context) (pubsub.Connector, error) {
		return t, nil
	}
}

// testingConnector is a blank implementation of the Connector interface
type testingConnector struct {
	subsLock sync2.Mutex
	subs     map[string]chan struct{}
}

func (t *testingConnector) Proxy(ctx, traceCtx context.Context, opts pubsub.ProxyOpts) (*connect.SDKResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (t *testingConnector) RouteExecutorRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, data *connect.GatewayExecutorRequestData) error {
	return fmt.Errorf("not implemented")
}

func (t *testingConnector) ReceiveRoutedRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, onMessage func(rawBytes []byte, data *connect.GatewayExecutorRequestData), onSubscribed chan struct{}) error {
	logger.StdlibLogger(ctx).Error("using no-op connector receive routed request", "gateway_id", gatewayId, "conn_id", connId)

	// Simulate setting up subscription and waiting for ctx to be done
	onSubscribed <- struct{}{}
	<-ctx.Done()

	return nil
}

func (t *testingConnector) AckMessage(ctx context.Context, requestId string, source pubsub.AckSource) error {
	return fmt.Errorf("not implemented")
}

func (t *testingConnector) NotifyExecutor(ctx context.Context, resp *connect.SDKResponse) error {
	return fmt.Errorf("not implemented")
}

func (t *testingConnector) Wait(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
