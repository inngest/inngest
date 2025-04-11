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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	gosync "sync"
	"testing"
	"time"
)

func TestConnectionEstablished(t *testing.T) {
	res := createTestingGateway(t)

	res.lifecycles.Assert(t, testRecorderAssertion{})

	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_HELLO, msg.Kind)

	sendWorkerConnectMessage(t, res)

	msg = awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_CONNECTION_READY, msg.Kind)

	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount: 1,
		onSyncedCount:    1,
		onReadyCount:     1,
	})

	require.Equal(t, res.connID, res.lifecycles.onReady[0].ConnectionId)
	require.Equal(t, *res.workerGroup.AppID, *res.lifecycles.onReady[0].Groups[res.workerGroup.Hash].AppID)
	require.Equal(t, res.workerGroup.FunctionSlugs, res.lifecycles.onReady[0].Groups[res.workerGroup.Hash].FunctionSlugs)
}

func TestExecutorMessageForwarding(t *testing.T) {
	params := testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
	}
	res := createTestingGateway(t, params)

	handshake(t, res)

	expectedPayload := &connect.GatewayExecutorRequestData{
		RequestId:      "test-req",
		AccountId:      res.accountID.String(),
		EnvId:          res.envID.String(),
		AppId:          res.appID.String(),
		AppName:        res.appName,
		FunctionId:     res.fnID.String(),
		FunctionSlug:   res.fnSlug,
		StepId:         ptr.String("step"),
		RequestPayload: []byte("hello world"),
		RunId:          res.runID.String(),
		LeaseId:        "lease-test",
	}

	// Publish message to "PubSub"
	_ = res.testConn.RouteExecutorRequest(context.Background(), res.svc.gatewayId, res.connID, expectedPayload)

	// Expect message to be received by gateway and forwarded over WebSocket
	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST, msg.Kind)

	payload := &connect.GatewayExecutorRequestData{}
	err := proto.Unmarshal(msg.Payload, payload)
	require.NoError(t, err)

	// Expect messages to match
	require.True(t, proto.Equal(expectedPayload, payload))
}

func TestLeaseRenewal(t *testing.T) {
	params := testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
	}
	res := createTestingGateway(t, params)

	handshake(t, res)

	requestID := "test-req"

	leaseID, err := res.svc.stateManager.LeaseRequest(context.Background(), res.envID, requestID, time.Second*5)
	require.NoError(t, err)

	expectedPayload := &connect.GatewayExecutorRequestData{
		RequestId:      requestID,
		AccountId:      res.accountID.String(),
		EnvId:          res.envID.String(),
		AppId:          res.appID.String(),
		AppName:        res.appName,
		FunctionId:     res.fnID.String(),
		FunctionSlug:   res.fnSlug,
		StepId:         ptr.String("step"),
		RequestPayload: []byte("hello world"),
		RunId:          res.runID.String(),
		LeaseId:        leaseID.String(),
	}

	// Publish message to "PubSub"
	_ = res.testConn.RouteExecutorRequest(context.Background(), res.svc.gatewayId, res.connID, expectedPayload)

	// Expect message to be received by gateway and forwarded over WebSocket
	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST, msg.Kind)

	payload := &connect.GatewayExecutorRequestData{}
	err = proto.Unmarshal(msg.Payload, payload)
	require.NoError(t, err)

	// Expect messages to match
	require.True(t, proto.Equal(expectedPayload, payload))

	sendWorkerExtendLeaseMessage(t, res, &connect.WorkerRequestExtendLeaseData{
		RequestId:      payload.RequestId,
		AccountId:      payload.AccountId,
		EnvId:          payload.EnvId,
		AppId:          payload.AppId,
		FunctionSlug:   payload.FunctionSlug,
		StepId:         payload.StepId,
		SystemTraceCtx: payload.SystemTraceCtx,
		UserTraceCtx:   payload.UserTraceCtx,
		RunId:          payload.RunId,
		LeaseId:        payload.LeaseId,
	})

	// Expect lease extension ack
	msg = awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connect.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK, msg.Kind)

	ackPayload := connect.WorkerRequestExtendLeaseAckData{}
	err = proto.Unmarshal(msg.Payload, &ackPayload)
	require.NoError(t, err)

	require.Equal(t, payload.RequestId, ackPayload.RequestId)
	require.Equal(t, payload.AccountId, ackPayload.AccountId)
	require.NotNil(t, ackPayload.NewLeaseId)

	parsed, err := ulid.Parse(*ackPayload.NewLeaseId)
	require.NoError(t, err)

	require.WithinDuration(t, time.Now().Add(consts.ConnectWorkerRequestLeaseDuration), ulid.Time(parsed.Time()), 500*time.Millisecond)
}

func TestCloseConnectionOnConsecutiveHeartbeatFail(t *testing.T) {
	params := testingParameters{
		consecutiveMissesBeforeClose: 5,
		heartbeatInterval:            250 * time.Millisecond,
	}
	res := createTestingGateway(t, params)

	handshake(t, res)

	// Simulate heartbeat failure
	<-time.After(time.Duration(params.consecutiveMissesBeforeClose)*params.heartbeatInterval + 100*time.Millisecond)

	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount:          1,
		onSyncedCount:             1,
		onReadyCount:              1,
		onHeartbeatCount:          0,
		onStartDrainingCount:      0,
		onStartDisconnectingCount: 1,
		onDisconnectedCount:       1,
	})

	require.Len(t, res.lifecycles.onDisconnected, 1)
	require.Equal(t, res.connID, res.lifecycles.onDisconnected[0].conn.ConnectionId)
	require.Equal(t, connect.WorkerDisconnectReason_CONSECUTIVE_HEARTBEATS_MISSED.String(), res.lifecycles.onDisconnected[0].closeReason)
}

func TestWorkerHeartbeats(t *testing.T) {
	params := testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
	}
	res := createTestingGateway(t, params)

	handshake(t, res)

	// Expect initial heartbeat to be set to now
	conn, err := res.stateManager.GetConnection(context.Background(), res.envID, res.connID)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), conn.LastHeartbeatAt.AsTime(), 500*time.Millisecond)

	// Wait for a bit
	<-time.After(1 * time.Second)

	// Send first real heartbeat
	sendWorkerHeartbeatMessage(t, res)

	// Expect lifecycles to be set
	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount: 1,
		onSyncedCount:    1,
		onReadyCount:     1,
		onHeartbeatCount: 1,
	})

	conn, err = res.stateManager.GetConnection(context.Background(), res.envID, res.connID)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), conn.LastHeartbeatAt.AsTime(), 500*time.Millisecond)
}

type websocketDisconnected struct {
	conn        *state.Connection
	closeReason string
}

type testRecorderLifecycles struct {
	logger *slog.Logger

	lock gosync.Mutex

	onConnected          []*state.Connection
	onReady              []*state.Connection
	onHeartbeat          []*state.Connection
	onStartDraining      []*state.Connection
	onStartDisconnecting []*state.Connection
	onSynced             []*state.Connection
	onDisconnected       []websocketDisconnected
}

type testRecorderAssertion struct {
	onConnectedCount          int
	onSyncedCount             int
	onReadyCount              int
	onHeartbeatCount          int
	onStartDrainingCount      int
	onStartDisconnectingCount int
	onDisconnectedCount       int
}

func (r *testRecorderLifecycles) Assert(t *testing.T, assertion testRecorderAssertion) {
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		r.lock.Lock()
		defer r.lock.Unlock()

		assert.Equal(t, assertion.onConnectedCount, len(r.onConnected), "expected %d connections to be connected", assertion.onConnectedCount)
		assert.Equal(t, assertion.onReadyCount, len(r.onReady), "expected %d connections to be ready", assertion.onReadyCount)
		assert.Equal(t, assertion.onHeartbeatCount, len(r.onHeartbeat), "expected %d connections to be heartbeat", assertion.onHeartbeatCount)
		assert.Equal(t, assertion.onStartDrainingCount, len(r.onStartDraining), "expected %d connections to be draining", assertion.onStartDrainingCount)
		assert.Equal(t, assertion.onStartDisconnectingCount, len(r.onStartDisconnecting), "expected %d connections to be disconnecting", assertion.onStartDisconnectingCount)
		assert.Equal(t, assertion.onSyncedCount, len(r.onSynced), "expected %d connections to be synced", assertion.onSyncedCount)
		assert.Equal(t, assertion.onDisconnectedCount, len(r.onDisconnected), "expected %d connections to be disconnected", assertion.onDisconnectedCount)
	}, 3*time.Second, 200*time.Millisecond)
}

func (r *testRecorderLifecycles) OnConnected(ctx context.Context, conn *state.Connection) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.logger.Info("onConnected", "conn", conn)
	r.onConnected = append(r.onConnected, conn)
}

func (r *testRecorderLifecycles) OnReady(ctx context.Context, conn *state.Connection) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.logger.Info("onReady", "conn", conn)
	r.onReady = append(r.onReady, conn)
}

func (r *testRecorderLifecycles) OnHeartbeat(ctx context.Context, conn *state.Connection) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.logger.Info("onHeartbeat", "conn", conn)
	r.onHeartbeat = append(r.onHeartbeat, conn)
}

func (r *testRecorderLifecycles) OnStartDraining(ctx context.Context, conn *state.Connection) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.logger.Info("onStartDraining", "conn", conn)
	r.onStartDraining = append(r.onStartDraining, conn)
}

func (r *testRecorderLifecycles) OnStartDisconnecting(ctx context.Context, conn *state.Connection) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.logger.Info("onStartDisconnecting", "conn", conn)
	r.onStartDisconnecting = append(r.onStartDisconnecting, conn)
}

func (r *testRecorderLifecycles) OnSynced(ctx context.Context, conn *state.Connection) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.logger.Info("onSynced", "conn", conn)
	r.onSynced = append(r.onSynced, conn)
}

func (r *testRecorderLifecycles) OnDisconnected(ctx context.Context, conn *state.Connection, closeReason string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.logger.Info("onDisconnected", "conn", conn)
	r.onDisconnected = append(r.onDisconnected, websocketDisconnected{conn, closeReason})
}

func newRecorderLifecycles(logger *slog.Logger) *testRecorderLifecycles {
	r := &testRecorderLifecycles{
		logger: logger,
	}
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
	svc        *connectGatewaySvc

	envID     uuid.UUID
	accountID uuid.UUID
	syncID    uuid.UUID
	appID     uuid.UUID
	fnID      uuid.UUID

	runID ulid.ULID

	appName string
	fnName  string
	fnSlug  string

	connID ulid.ULID

	reqData     *connect.WorkerConnectRequestData
	workerGroup *state.WorkerGroup
}

type testingParameters struct {
	heartbeatInterval            time.Duration
	leaseDuration                time.Duration
	extendLeaseInterval          time.Duration
	consecutiveMissesBeforeClose int
}

func createTestingGateway(t *testing.T, params ...testingParameters) testingResources {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	envID, accountID := uuid.New(), uuid.New()
	syncID, appID, fnID := uuid.New(), uuid.New(), uuid.New()

	connID := ulid.MustNew(ulid.Now(), rand.Reader)
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	appName := "test-app"
	fnName := "test-fn"
	fnSlug := "test-app-test-fn"

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
		fakeApiPort := freePort()

		fakeApiBaseUrl = fmt.Sprintf("http://127.0.0.1:%d", fakeApiPort)

		mux := http.NewServeMux()

		srv := http.Server{
			Handler: mux,
			Addr:    fmt.Sprintf("127.0.0.1:%d", fakeApiPort),
		}

		go func() {
			_ = srv.ListenAndServe()
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
			body, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			l.Info("got register request", "headers", request.Header, "body", string(body))

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(reply)
		})

		mux.HandleFunc("GET /ready", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("ok"))
		})
	}

	gwPort := freePort()

	websocketUrl := fmt.Sprintf("ws://127.0.0.1:%d/v0/connect", gwPort)

	lifecycles := newRecorderLifecycles(l)

	authResp := &auth.Response{
		AccountID: accountID,
		EnvID:     envID,
		Entitlements: auth.Entitlements{
			ConnectionAllowed: true,
			AppsPerConnection: 10,
		},
	}

	opts := []gatewayOpt{
		WithGatewayAuthHandler(func(ctx context.Context, data *connect.WorkerConnectRequestData) (*auth.Response, error) {
			l.Info("got auth request", "data", data)

			return authResp, nil
		}),
		WithConnectionStateManager(connManager),
		WithGroupName("gw-1"),
		WithRequestReceiver(conn),
		WithLifeCycles([]ConnectGatewayLifecycleListener{lifecycles}),
		WithApiBaseUrl(fakeApiBaseUrl),
		WithGatewayPublicPort(gwPort),
	}

	if len(params) > 0 {
		if params[0].heartbeatInterval > 0 {
			opts = append(opts, WithWorkerHeartbeatInterval(params[0].heartbeatInterval))
		}

		if params[0].leaseDuration > 0 {
			opts = append(opts, WithWorkerRequestLeaseDuration(params[0].leaseDuration))
		}

		if params[0].extendLeaseInterval > 0 {
			opts = append(opts, WithWorkerExtendLeaseInterval(params[0].extendLeaseInterval))
		}

		if params[0].consecutiveMissesBeforeClose > 0 {
			opts = append(opts, WithConsecutiveWorkerHeartbeatMissesBeforeConnectionClose(params[0].consecutiveMissesBeforeClose))
		}
	}

	svc := NewConnectGatewayService(
		opts...,
	)

	require.NoError(t, svc.Pre(ctx))

	svc.logger = l

	go func() {
		err := svc.Run(ctx)
		if err != nil {
			require.ErrorIs(t, err, context.Canceled)
		}
	}()
	t.Cleanup(func() {
		_ = svc.Stop(context.Background())
	})

	// Wait until fake API is up
	maxAttempts := 10
	for i := 0; i <= maxAttempts; i++ {
		if i == maxAttempts {
			require.Fail(t, "failed to connect to fake api")
		}

		resp, err := http.Get(fakeApiBaseUrl + "/ready")
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Wait until gateway is up
	maxAttempts = 10
	for i := 0; i <= maxAttempts; i++ {
		if i == maxAttempts {
			require.Fail(t, "failed to connect to gateway")
		}

		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ready", gwPort))
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	ws, _, err := websocket.Dial(ctx, websocketUrl, &websocket.DialOptions{
		Subprotocols: []string{types.GatewaySubProtocol},
	})
	require.NoError(t, err)

	caps, err := json.Marshal(sdk.Capabilities{
		InBandSync: sdk.InBandSyncV1,
		TrustProbe: sdk.TrustProbeV1,
		Connect:    sdk.ConnectV1,
	})
	require.NoError(t, err)

	fns, err := json.Marshal([]sdk.SDKFunction{
		{
			Name: fnName,
			Slug: fnSlug,
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
					Name: fnName,
					Runtime: map[string]any{
						"url": fmt.Sprintf("ws://connect?fnId=%s&step=step", fnSlug),
					},
				},
			},
		},
	})
	require.NoError(t, err)

	testApp := &connect.AppConfiguration{
		AppName:    appName,
		AppVersion: ptr.String("v1"),
		Functions:  fns,
	}

	reqData := &connect.WorkerConnectRequestData{
		ConnectionId: connID.String(),
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
	}

	// Worker group to compare against (this is what we expect the synced worker group to look like)
	workerGroup, err := state.NewWorkerGroupFromConnRequest(ctx, reqData, authResp, testApp)
	require.NoError(t, err)
	workerGroup.AppID = &appID
	workerGroup.SyncID = &syncID

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
		fnID:         fnID,
		appID:        appID,
		connID:       connID,
		svc:          svc,
		appName:      appName,
		fnName:       fnName,
		fnSlug:       fnSlug,
		reqData:      reqData,
		workerGroup:  workerGroup,
		runID:        runID,
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

func handshake(t *testing.T, res testingResources) {
	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_HELLO, msg.Kind)

	sendWorkerConnectMessage(t, res)

	msg = awaitNextMessage(t, res.ws, 5*time.Second)
	require.Equal(t, connect.GatewayMessageType_GATEWAY_CONNECTION_READY, msg.Kind)

	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount: 1,
		onSyncedCount:    1,
		onReadyCount:     1,
	})
}

func sendWorkerConnectMessage(t *testing.T, res testingResources) {
	ctx := context.Background()

	connectMsg, err := proto.Marshal(res.reqData)
	require.NoError(t, err)

	err = wsproto.Write(ctx, res.ws, &connect.ConnectMessage{
		Kind:    connect.GatewayMessageType_WORKER_CONNECT,
		Payload: connectMsg,
	})
	require.NoError(t, err)
}

func sendWorkerHeartbeatMessage(t *testing.T, res testingResources) {
	ctx := context.Background()

	err := wsproto.Write(ctx, res.ws, &connect.ConnectMessage{
		Kind: connect.GatewayMessageType_WORKER_HEARTBEAT,
	})
	require.NoError(t, err)
}

func sendWorkerExtendLeaseMessage(t *testing.T, res testingResources, payload *connect.WorkerRequestExtendLeaseData) {
	ctx := context.Background()

	marshaled, err := proto.Marshal(payload)
	require.NoError(t, err)

	err = wsproto.Write(ctx, res.ws, &connect.ConnectMessage{
		Kind:    connect.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
		Payload: marshaled,
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
	subsLock            gosync.Mutex
	executorRequestSubs map[string][]chan *connect.GatewayExecutorRequestData
	ackSubs             map[string][]chan pubsub.AckSource
}

func (t *testingConnector) Proxy(ctx, traceCtx context.Context, opts pubsub.ProxyOpts) (*connect.SDKResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (t *testingConnector) RouteExecutorRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, data *connect.GatewayExecutorRequestData) error {
	t.subsLock.Lock()
	defer t.subsLock.Unlock()

	subKey := fmt.Sprintf("%s-%s", gatewayId.String(), connId.String())

	sub, ok := t.executorRequestSubs[subKey]
	if !ok {
		return nil
	}

	for _, ch := range sub {
		ch <- data
	}
	return nil
}

func (t *testingConnector) ReceiveRoutedRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, onMessage func(rawBytes []byte, data *connect.GatewayExecutorRequestData), onSubscribed chan struct{}) error {
	logger.StdlibLogger(ctx).Error("using no-op connector receive routed request", "gateway_id", gatewayId, "conn_id", connId)

	t.subsLock.Lock()
	if t.executorRequestSubs == nil {
		t.executorRequestSubs = make(map[string][]chan *connect.GatewayExecutorRequestData)
	}
	subKey := fmt.Sprintf("%s-%s", gatewayId.String(), connId.String())

	sub := make(chan *connect.GatewayExecutorRequestData)
	t.executorRequestSubs[subKey] = append(t.executorRequestSubs[subKey], sub)

	t.subsLock.Unlock()

	onSubscribed <- struct{}{} // Notify that subscription is ready
	for {
		select {
		case <-ctx.Done():
			t.subsLock.Lock()
			defer t.subsLock.Unlock()

			// Remove the subscription
			if subs, ok := t.executorRequestSubs[subKey]; ok {
				for i, s := range subs {
					if s == sub {
						t.executorRequestSubs[subKey] = append(subs[:i], subs[i+1:]...)
						break
					}
				}
				if len(t.executorRequestSubs[subKey]) == 0 {
					delete(t.executorRequestSubs, subKey)
				}
			}

			return nil
		case msg := <-sub:
			marshaled, err := proto.Marshal(msg)
			if err != nil {
				return err
			}
			onMessage(marshaled, msg)
		}
	}
}

func (t *testingConnector) AckMessage(ctx context.Context, requestId string, source pubsub.AckSource) error {
	ackKey := fmt.Sprintf("ack:%s", requestId)

	t.subsLock.Lock()
	defer t.subsLock.Unlock()

	subs, ok := t.ackSubs[ackKey]
	if !ok {
		return nil
	}

	for _, ch := range subs {
		ch <- source
	}

	return nil
}

func (t *testingConnector) NotifyExecutor(ctx context.Context, resp *connect.SDKResponse) error {
	return fmt.Errorf("not implemented")
}

func (t *testingConnector) Wait(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
