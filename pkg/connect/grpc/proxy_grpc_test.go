package grpc

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

type mockGatewayGRPCManager struct {
	forwardCalls  []mockForwardCall
	subscriptions map[string]chan *connectpb.SDKResponse
	mu            sync.Mutex
}

type mockForwardCall struct {
	gatewayID    ulid.ULID
	connectionID ulid.ULID
	data         *connectpb.GatewayExecutorRequestData
}

func (m *mockGatewayGRPCManager) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forwardCalls = append(m.forwardCalls, mockForwardCall{
		gatewayID:    gatewayID,
		connectionID: connectionID,
		data:         data,
	})
	return nil
}

func (m *mockGatewayGRPCManager) ConnectToGateways(ctx context.Context) error {
	return nil
}

func (m *mockGatewayGRPCManager) Subscribe(ctx context.Context, requestID string) chan *connectpb.SDKResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subscriptions == nil {
		m.subscriptions = make(map[string]chan *connectpb.SDKResponse)
	}
	channel := make(chan *connectpb.SDKResponse, 1)
	m.subscriptions[requestID] = channel
	return channel
}

func (m *mockGatewayGRPCManager) SubscribeWorkerAck(ctx context.Context, requestID string) chan *connectpb.AckMessage {
	// Empty mock
	return nil
}

func (m *mockGatewayGRPCManager) Unsubscribe(ctx context.Context, requestID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subscriptions != nil {
		delete(m.subscriptions, requestID)
	}
}

func (m *mockGatewayGRPCManager) UnsubscribeWorkerAck(ctx context.Context, requestID string) {
	// Empty mock
}

func (m *mockGatewayGRPCManager) getForwardCalls() []mockForwardCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]mockForwardCall, len(m.forwardCalls))
	copy(calls, m.forwardCalls)
	return calls
}

func (m *mockGatewayGRPCManager) sendResponse(requestID string, response *connectpb.SDKResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subscriptions != nil {
		if channel, exists := m.subscriptions[requestID]; exists {
			select {
			case channel <- response:
			default:
				// Channel is full or closed, ignore
			}
		}
	}
}

func TestProxyGRPCPath(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm := state.NewRedisConnectionStateManager(rc)
	mockForwarder := &mockGatewayGRPCManager{}

	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	}, WithConnectorLogger(l), WithGatewayManager(mockForwarder))

	fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	connID := ulid.MustNew(ulid.Now(), rand.Reader)
	gwID := ulid.MustNew(ulid.Now(), rand.Reader)

	reqID := "reqid-test"
	appName := "test-app"
	fnName := "test-fn"
	fnSlug := "test-app-test-fn"
	appVersion := "v1.1"

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
				"step": {
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

	appConfig := &connectpb.AppConfiguration{
		AppName:    appName,
		AppVersion: &appVersion,
		Functions:  fns,
	}

	caps, err := json.Marshal(sdk.Capabilities{
		InBandSync: sdk.InBandSyncV1,
		TrustProbe: sdk.TrustProbeV1,
		Connect:    sdk.ConnectV1,
	})
	require.NoError(t, err)

	reqData := &connectpb.WorkerConnectRequestData{
		ConnectionId: connID.String(),
		InstanceId:   "test-worker",
		Apps:         []*connectpb.AppConfiguration{appConfig},
		SdkLanguage:  "test-sdk",
		SdkVersion:   "test-version",
		Capabilities: caps,
		AuthData: &connectpb.AuthData{
			SessionToken: "test-session-token",
			SyncToken:    "test-sync-token",
		},
	}

	wg, err := state.NewWorkerGroupFromConnRequest(ctx, reqData, &auth.Response{
		AccountID: accID,
		EnvID:     envID,
		Entitlements: auth.Entitlements{
			ConnectionAllowed: true,
			AppsPerConnection: 10,
		},
	},
		appConfig,
	)
	require.NoError(t, err)

	syncID := uuid.New()

	wg.AppID = &appID
	wg.SyncID = &syncID

	err = sm.UpsertConnection(ctx, &state.Connection{
		AccountID:    accID,
		EnvID:        envID,
		ConnectionId: connID,
		WorkerIP:     "10.0.1.2",
		Data: &connectpb.WorkerConnectRequestData{
			ConnectionId: connID.String(),
			InstanceId:   "test-worker",
			Apps:         []*connectpb.AppConfiguration{appConfig},
		},
		Groups: map[string]*state.WorkerGroup{
			wg.Hash: wg,
		},
		GatewayId: gwID,
	}, connectpb.ConnectionStatus_READY, time.Now())
	require.NoError(t, err)

	err = sm.UpsertGateway(ctx, &state.Gateway{
		Id:                gwID,
		Status:            state.GatewayStatusActive,
		LastHeartbeatAtMS: time.Now().UnixMilli(),
		Hostname:          "gw-host",
		IPAddress:         net.ParseIP("127.0.0.1"),
	})
	require.NoError(t, err)

	conns, err := sm.GetConnectionsByAppID(ctx, envID, appID)
	require.NoError(t, err)
	require.Len(t, conns, 1)

	withTimeout, cancel := context.WithTimeout(ctx, time.Minute*30)
	defer cancel()

	respCh := make(chan *connectpb.SDKResponse)
	go func() {
		resp, err := connector.Proxy(withTimeout, context.Background(), ProxyOpts{
			AccountID: accID,
			EnvID:     envID,
			AppID:     appID,
			Data: &connectpb.GatewayExecutorRequestData{
				RequestId:      reqID,
				AccountId:      accID.String(),
				EnvId:          envID.String(),
				AppId:          appID.String(),
				AppName:        appName,
				FunctionId:     fnID.String(),
				FunctionSlug:   fnSlug,
				StepId:         nil,
				RequestPayload: []byte("request payload"),
				SystemTraceCtx: nil,
				UserTraceCtx:   nil,
				RunId:          runID.String(),
			},
			logger: l,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		respCh <- resp
	}()

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		calls := mockForwarder.getForwardCalls()
		require.Len(t, calls, 1, "expected gRPC Forward to be called once")
		require.Equal(t, gwID, calls[0].gatewayID)
		require.Equal(t, connID, calls[0].connectionID)
		require.Equal(t, reqID, calls[0].data.RequestId)
	}, 3*time.Second, 100*time.Millisecond)

	mockForwarder.sendResponse(reqID, &connectpb.SDKResponse{
		RequestId:      reqID,
		AccountId:      accID.String(),
		EnvId:          envID.String(),
		AppId:          appID.String(),
		Status:         connectpb.SDKResponseStatus_DONE,
		Body:           nil,
		NoRetry:        false,
		RetryAfter:     nil,
		SdkVersion:     "test-version",
		RequestVersion: 1,
		SystemTraceCtx: nil,
		UserTraceCtx:   nil,
		RunId:          runID.String(),
	})

	select {
	case r := <-respCh:
		require.Equal(t, connectpb.SDKResponseStatus_DONE, r.Status)
	case <-time.After(3 * time.Second):
		require.Fail(t, "no response received")
	}
}

func TestProxyGRPCPolling(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm := state.NewRedisConnectionStateManager(rc)
	mockForwarder := &mockGatewayGRPCManager{}

	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	}, WithConnectorLogger(l), WithGatewayManager(mockForwarder))

	fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	connID := ulid.MustNew(ulid.Now(), rand.Reader)
	gwID := ulid.MustNew(ulid.Now(), rand.Reader)
	reqID := "reqid-test"
	appName := "test-app"
	fnName := "test-fn"
	fnSlug := "test-app-test-fn"
	appVersion := "v1.1"

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
				"step": {
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

	appConfig := &connectpb.AppConfiguration{
		AppName:    appName,
		AppVersion: &appVersion,
		Functions:  fns,
	}

	caps, err := json.Marshal(sdk.Capabilities{
		InBandSync: sdk.InBandSyncV1,
		TrustProbe: sdk.TrustProbeV1,
		Connect:    sdk.ConnectV1,
	})
	require.NoError(t, err)

	reqData := &connectpb.WorkerConnectRequestData{
		ConnectionId: connID.String(),
		InstanceId:   "test-worker",
		Apps:         []*connectpb.AppConfiguration{appConfig},
		SdkLanguage:  "test-sdk",
		SdkVersion:   "test-version",
		Capabilities: caps,
		AuthData: &connectpb.AuthData{
			SessionToken: "test-session-token",
			SyncToken:    "test-sync-token",
		},
	}

	wg, err := state.NewWorkerGroupFromConnRequest(ctx, reqData, &auth.Response{
		AccountID: accID,
		EnvID:     envID,
		Entitlements: auth.Entitlements{
			ConnectionAllowed: true,
			AppsPerConnection: 10,
		},
	},
		appConfig,
	)
	require.NoError(t, err)

	syncID := uuid.New()
	wg.AppID = &appID
	wg.SyncID = &syncID

	err = sm.UpsertConnection(ctx, &state.Connection{
		AccountID:    accID,
		EnvID:        envID,
		ConnectionId: connID,
		WorkerIP:     "10.0.1.2",
		Data:         reqData,
		Groups: map[string]*state.WorkerGroup{
			wg.Hash: wg,
		},
		GatewayId: gwID,
	}, connectpb.ConnectionStatus_READY, time.Now())
	require.NoError(t, err)

	err = sm.UpsertGateway(ctx, &state.Gateway{
		Id:                gwID,
		Status:            state.GatewayStatusActive,
		LastHeartbeatAtMS: time.Now().UnixMilli(),
		Hostname:          "gw-host",
		IPAddress:         net.ParseIP("127.0.0.1"),
	})
	require.NoError(t, err)

	conns, err := sm.GetConnectionsByAppID(ctx, envID, appID)
	require.NoError(t, err)
	require.Len(t, conns, 1)

	withTimeout, cancel := context.WithTimeout(ctx, time.Minute*30)
	defer cancel()

	respCh := make(chan *connectpb.SDKResponse)
	errCh := make(chan error)
	go func() {
		resp, err := connector.Proxy(withTimeout, context.Background(), ProxyOpts{
			AccountID: accID,
			EnvID:     envID,
			AppID:     appID,
			Data: &connectpb.GatewayExecutorRequestData{
				RequestId:      reqID,
				AccountId:      accID.String(),
				EnvId:          envID.String(),
				AppId:          appID.String(),
				AppName:        appName,
				FunctionId:     fnID.String(),
				FunctionSlug:   fnSlug,
				StepId:         nil,
				RequestPayload: []byte("request payload"),
				SystemTraceCtx: nil,
				UserTraceCtx:   nil,
				RunId:          runID.String(),
			},
			logger: l,
		})
		if err != nil {
			errCh <- err
		} else {
			respCh <- resp
		}
	}()

	err = sm.SaveResponse(ctx, envID, reqID, &connectpb.SDKResponse{
		RequestId:      reqID,
		AccountId:      accID.String(),
		EnvId:          envID.String(),
		AppId:          appID.String(),
		Status:         connectpb.SDKResponseStatus_DONE,
		Body:           nil,
		NoRetry:        false,
		RetryAfter:     nil,
		SdkVersion:     "test-version",
		RequestVersion: 1,
		SystemTraceCtx: nil,
		UserTraceCtx:   nil,
		RunId:          runID.String(),
	})
	require.NoError(t, err)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case resp := <-respCh:
		require.NotNil(t, resp)
		require.Equal(t, connectpb.SDKResponseStatus_DONE, resp.Status)
	case <-time.After(5 * time.Second):
		require.Fail(t, "proxy call timed out")
	}

	// No response was sent through gRPC, but we still picked up the buffered response
	buffered, err := sm.GetResponse(ctx, envID, reqID)
	require.NoError(t, err)
	require.Nil(t, buffered, "buffered response should be cleaned up")
}

func TestProxyGRPCLeaseExpiry(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm := state.NewRedisConnectionStateManager(rc)
	mockForwarder := &mockGatewayGRPCManager{}

	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	}, WithConnectorLogger(l), WithGatewayManager(mockForwarder))

	fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	connID := ulid.MustNew(ulid.Now(), rand.Reader)
	gwID := ulid.MustNew(ulid.Now(), rand.Reader)
	reqID := "reqid-test"
	appName := "test-app"
	fnName := "test-fn"
	fnSlug := "test-app-test-fn"
	appVersion := "v1.1"

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
				"step": {
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

	appConfig := &connectpb.AppConfiguration{
		AppName:    appName,
		AppVersion: &appVersion,
		Functions:  fns,
	}

	caps, err := json.Marshal(sdk.Capabilities{
		InBandSync: sdk.InBandSyncV1,
		TrustProbe: sdk.TrustProbeV1,
		Connect:    sdk.ConnectV1,
	})
	require.NoError(t, err)

	reqData := &connectpb.WorkerConnectRequestData{
		ConnectionId: connID.String(),
		InstanceId:   "test-worker",
		Apps:         []*connectpb.AppConfiguration{appConfig},
		SdkLanguage:  "test-sdk",
		SdkVersion:   "test-version",
		Capabilities: caps,
		AuthData: &connectpb.AuthData{
			SessionToken: "test-session-token",
			SyncToken:    "test-sync-token",
		},
	}

	wg, err := state.NewWorkerGroupFromConnRequest(ctx, reqData, &auth.Response{
		AccountID: accID,
		EnvID:     envID,
		Entitlements: auth.Entitlements{
			ConnectionAllowed: true,
			AppsPerConnection: 10,
		},
	},
		appConfig,
	)
	require.NoError(t, err)

	syncID := uuid.New()
	wg.AppID = &appID
	wg.SyncID = &syncID

	err = sm.UpsertConnection(ctx, &state.Connection{
		AccountID:    accID,
		EnvID:        envID,
		ConnectionId: connID,
		WorkerIP:     "10.0.1.2",
		Data:         reqData,
		Groups: map[string]*state.WorkerGroup{
			wg.Hash: wg,
		},
		GatewayId: gwID,
	}, connectpb.ConnectionStatus_READY, time.Now())
	require.NoError(t, err)

	err = sm.UpsertGateway(ctx, &state.Gateway{
		Id:                gwID,
		Status:            state.GatewayStatusActive,
		LastHeartbeatAtMS: time.Now().UnixMilli(),
		Hostname:          "gw-host",
		IPAddress:         net.ParseIP("127.0.0.1"),
	})
	require.NoError(t, err)

	withTimeout, cancel := context.WithTimeout(ctx, time.Minute*30)
	defer cancel()

	respCh := make(chan struct{})
	go func() {
		resp, err := connector.Proxy(withTimeout, context.Background(), ProxyOpts{
			AccountID: accID,
			EnvID:     envID,
			AppID:     appID,
			Data: &connectpb.GatewayExecutorRequestData{
				RequestId:      reqID,
				AccountId:      accID.String(),
				EnvId:          envID.String(),
				AppId:          appID.String(),
				AppName:        "test-app",
				FunctionId:     fnID.String(),
				FunctionSlug:   "test-app-test-fn",
				StepId:         nil,
				RequestPayload: []byte("request payload"),
				SystemTraceCtx: nil,
				UserTraceCtx:   nil,
				RunId:          runID.String(),
			},
			logger: l,
		})
		var sysErr syscode.Error
		require.ErrorAs(t, err, &sysErr)
		require.Equal(t, syscode.CodeConnectWorkerStoppedResponding, sysErr.Code)
		require.Nil(t, resp)
		respCh <- struct{}{}
	}()

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		calls := mockForwarder.getForwardCalls()
		require.Len(t, calls, 1, "expected gRPC Forward to be called once")
		require.Equal(t, gwID, calls[0].gatewayID)
		require.Equal(t, connID, calls[0].connectionID)
		require.Equal(t, reqID, calls[0].data.RequestId)
	}, 3*time.Second, 100*time.Millisecond)

	require.NoError(t, sm.DeleteLease(ctx, envID, reqID))

	<-time.After(consts.ConnectWorkerRequestExtendLeaseInterval + time.Second)

	select {
	case <-respCh:
	case <-time.After(10 * time.Second):
		require.Fail(t, "no response received")
	}
}

// TestProxyGRPCNoHealthyConnection tests routing when no healthy connections are available
func TestProxyGRPCNoHealthyConnection(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm := state.NewRedisConnectionStateManager(rc)
	mockForwarder := &mockGatewayGRPCManager{}

	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	}, WithConnectorLogger(l), WithGatewayManager(mockForwarder))

	fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	reqID := "reqid-test"

	// Don't create any connections - this will cause routing to fail
	resp, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
		AccountID: accID,
		EnvID:     envID,
		AppID:     appID,
		Data: &connectpb.GatewayExecutorRequestData{
			RequestId:      reqID,
			AccountId:      accID.String(),
			EnvId:          envID.String(),
			AppId:          appID.String(),
			AppName:        "test-app",
			FunctionId:     fnID.String(),
			FunctionSlug:   "test-app-test-fn",
			StepId:         nil,
			RequestPayload: []byte("request payload"),
			SystemTraceCtx: nil,
			UserTraceCtx:   nil,
			RunId:          runID.String(),
		},
		logger: l,
	})

	require.Nil(t, resp)
	var sysErr syscode.Error
	require.ErrorAs(t, err, &sysErr)
	require.Equal(t, syscode.CodeConnectNoHealthyConnection, sysErr.Code)
}

// TestProxyGRPCForwardError tests error handling when forwarding fails
func TestProxyGRPCForwardError(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm := state.NewRedisConnectionStateManager(rc)
	
	// Mock forwarder that always returns an error
	mockForwarder := &mockFailingGatewayGRPCManager{}

	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	}, WithConnectorLogger(l), WithGatewayManager(mockForwarder))

	fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	connID := ulid.MustNew(ulid.Now(), rand.Reader)
	gwID := ulid.MustNew(ulid.Now(), rand.Reader)
	reqID := "reqid-test"
	appName := "test-app"
	fnName := "test-fn"
	fnSlug := "test-app-test-fn"
	appVersion := "v1.1"

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
				"step": {
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

	appConfig := &connectpb.AppConfiguration{
		AppName:    appName,
		AppVersion: &appVersion,
		Functions:  fns,
	}

	caps, err := json.Marshal(sdk.Capabilities{
		InBandSync: sdk.InBandSyncV1,
		TrustProbe: sdk.TrustProbeV1,
		Connect:    sdk.ConnectV1,
	})
	require.NoError(t, err)

	reqData := &connectpb.WorkerConnectRequestData{
		ConnectionId: connID.String(),
		InstanceId:   "test-worker",
		Apps:         []*connectpb.AppConfiguration{appConfig},
		SdkLanguage:  "test-sdk",
		SdkVersion:   "test-version",
		Capabilities: caps,
		AuthData: &connectpb.AuthData{
			SessionToken: "test-session-token",
			SyncToken:    "test-sync-token",
		},
	}

	wg, err := state.NewWorkerGroupFromConnRequest(ctx, reqData, &auth.Response{
		AccountID: accID,
		EnvID:     envID,
		Entitlements: auth.Entitlements{
			ConnectionAllowed: true,
			AppsPerConnection: 10,
		},
	}, appConfig)
	require.NoError(t, err)

	syncID := uuid.New()
	wg.AppID = &appID
	wg.SyncID = &syncID

	err = sm.UpsertConnection(ctx, &state.Connection{
		AccountID:    accID,
		EnvID:        envID,
		ConnectionId: connID,
		WorkerIP:     "10.0.1.2",
		Data:         reqData,
		Groups: map[string]*state.WorkerGroup{
			wg.Hash: wg,
		},
		GatewayId: gwID,
	}, connectpb.ConnectionStatus_READY, time.Now())
	require.NoError(t, err)

	err = sm.UpsertGateway(ctx, &state.Gateway{
		Id:                gwID,
		Status:            state.GatewayStatusActive,
		LastHeartbeatAtMS: time.Now().UnixMilli(),
		Hostname:          "gw-host",
		IPAddress:         net.ParseIP("127.0.0.1"),
	})
	require.NoError(t, err)

	resp, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
		AccountID: accID,
		EnvID:     envID,
		AppID:     appID,
		Data: &connectpb.GatewayExecutorRequestData{
			RequestId:      reqID,
			AccountId:      accID.String(),
			EnvId:          envID.String(),
			AppId:          appID.String(),
			AppName:        appName,
			FunctionId:     fnID.String(),
			FunctionSlug:   fnSlug,
			StepId:         nil,
			RequestPayload: []byte("request payload"),
			SystemTraceCtx: nil,
			UserTraceCtx:   nil,
			RunId:          runID.String(),
		},
		logger: l,
	})

	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to route request to gateway")
}

// TestProxyGRPCContextCancellation tests proxy behavior when context is cancelled
func TestProxyGRPCContextCancellation(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm := state.NewRedisConnectionStateManager(rc)
	mockForwarder := &mockGatewayGRPCManager{}

	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	}, WithConnectorLogger(l), WithGatewayManager(mockForwarder))

	fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	connID := ulid.MustNew(ulid.Now(), rand.Reader)
	gwID := ulid.MustNew(ulid.Now(), rand.Reader)
	reqID := "reqid-test"
	appName := "test-app"
	fnName := "test-fn"
	fnSlug := "test-app-test-fn"
	appVersion := "v1.1"

	// Set up connections so we get past the routing stage
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
				"step": {
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

	appConfig := &connectpb.AppConfiguration{
		AppName:    appName,
		AppVersion: &appVersion,
		Functions:  fns,
	}

	caps, err := json.Marshal(sdk.Capabilities{
		InBandSync: sdk.InBandSyncV1,
		TrustProbe: sdk.TrustProbeV1,
		Connect:    sdk.ConnectV1,
	})
	require.NoError(t, err)

	reqData := &connectpb.WorkerConnectRequestData{
		ConnectionId: connID.String(),
		InstanceId:   "test-worker",
		Apps:         []*connectpb.AppConfiguration{appConfig},
		SdkLanguage:  "test-sdk",
		SdkVersion:   "test-version",
		Capabilities: caps,
		AuthData: &connectpb.AuthData{
			SessionToken: "test-session-token",
			SyncToken:    "test-sync-token",
		},
	}

	wg, err := state.NewWorkerGroupFromConnRequest(ctx, reqData, &auth.Response{
		AccountID: accID,
		EnvID:     envID,
		Entitlements: auth.Entitlements{
			ConnectionAllowed: true,
			AppsPerConnection: 10,
		},
	}, appConfig)
	require.NoError(t, err)

	syncID := uuid.New()
	wg.AppID = &appID
	wg.SyncID = &syncID

	err = sm.UpsertConnection(ctx, &state.Connection{
		AccountID:    accID,
		EnvID:        envID,
		ConnectionId: connID,
		WorkerIP:     "10.0.1.2",
		Data:         reqData,
		Groups: map[string]*state.WorkerGroup{
			wg.Hash: wg,
		},
		GatewayId: gwID,
	}, connectpb.ConnectionStatus_READY, time.Now())
	require.NoError(t, err)

	err = sm.UpsertGateway(ctx, &state.Gateway{
		Id:                gwID,
		Status:            state.GatewayStatusActive,
		LastHeartbeatAtMS: time.Now().UnixMilli(),
		Hostname:          "gw-host",
		IPAddress:         net.ParseIP("127.0.0.1"),
	})
	require.NoError(t, err)

	// Create a short-lived context that will be cancelled quickly
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer shortCancel()

	resp, err := connector.Proxy(shortCtx, context.Background(), ProxyOpts{
		AccountID: accID,
		EnvID:     envID,
		AppID:     appID,
		Data: &connectpb.GatewayExecutorRequestData{
			RequestId:      reqID,
			AccountId:      accID.String(),
			EnvId:          envID.String(),
			AppId:          appID.String(),
			AppName:        "test-app",
			FunctionId:     fnID.String(),
			FunctionSlug:   "test-app-test-fn",
			StepId:         nil,
			RequestPayload: []byte("request payload"),
			SystemTraceCtx: nil,
			UserTraceCtx:   nil,
			RunId:          runID.String(),
		},
		logger: l,
	})

	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parent context was closed unexpectedly")
}

// TestProxyGRPCBufferedResponse tests when response is already buffered
func TestProxyGRPCBufferedResponse(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm := state.NewRedisConnectionStateManager(rc)
	mockForwarder := &mockGatewayGRPCManager{}

	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	}, WithConnectorLogger(l), WithGatewayManager(mockForwarder))

	fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	reqID := "reqid-test"

	// Pre-buffer a response
	expectedResponse := &connectpb.SDKResponse{
		RequestId:      reqID,
		AccountId:      accID.String(),
		EnvId:          envID.String(),
		AppId:          appID.String(),
		Status:         connectpb.SDKResponseStatus_DONE,
		Body:           []byte("buffered response"),
		NoRetry:        false,
		RetryAfter:     nil,
		SdkVersion:     "test-version",
		RequestVersion: 1,
		SystemTraceCtx: nil,
		UserTraceCtx:   nil,
		RunId:          runID.String(),
	}

	err = sm.SaveResponse(ctx, envID, reqID, expectedResponse)
	require.NoError(t, err)

	resp, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
		AccountID: accID,
		EnvID:     envID,
		AppID:     appID,
		Data: &connectpb.GatewayExecutorRequestData{
			RequestId:      reqID,
			AccountId:      accID.String(),
			EnvId:          envID.String(),
			AppId:          appID.String(),
			AppName:        "test-app",
			FunctionId:     fnID.String(),
			FunctionSlug:   "test-app-test-fn",
			StepId:         nil,
			RequestPayload: []byte("request payload"),
			SystemTraceCtx: nil,
			UserTraceCtx:   nil,
			RunId:          runID.String(),
		},
		logger: l,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, expectedResponse.RequestId, resp.RequestId)
	require.Equal(t, expectedResponse.Status, resp.Status)
	require.Equal(t, expectedResponse.Body, resp.Body)
}

// mockFailingGatewayGRPCManager is a mock that always fails forwards
type mockFailingGatewayGRPCManager struct {
	mu sync.Mutex
}

func (m *mockFailingGatewayGRPCManager) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	return fmt.Errorf("mock forward error")
}

func (m *mockFailingGatewayGRPCManager) ConnectToGateways(ctx context.Context) error {
	return nil
}

func (m *mockFailingGatewayGRPCManager) Subscribe(ctx context.Context, requestID string) chan *connectpb.SDKResponse {
	return make(chan *connectpb.SDKResponse)
}

func (m *mockFailingGatewayGRPCManager) SubscribeWorkerAck(ctx context.Context, requestID string) chan *connectpb.AckMessage {
	return make(chan *connectpb.AckMessage)
}

func (m *mockFailingGatewayGRPCManager) Unsubscribe(ctx context.Context, requestID string) {}

func (m *mockFailingGatewayGRPCManager) UnsubscribeWorkerAck(ctx context.Context, requestID string) {}

// TestCleanupWorkerRequestOrLogError tests the cleanup function
func TestCleanupWorkerRequestOrLogError(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	l := logger.StdlibLogger(context.Background(),
		logger.WithHandler(logger.TextHandler),
		logger.WithLoggerWriter(os.Stdout),
		logger.WithLoggerLevel(logger.LevelDebug),
	)

	ctx := context.Background()
	sm := state.NewRedisConnectionStateManager(rc)
	envID := uuid.New()
	instanceID := "test-instance"
	requestID := "test-request"

	// Test with empty instance ID - should log error
	cleanupWorkerRequestOrLogError(ctx, sm, envID, "", requestID, l, "test message")

	// Test with valid instance ID - should work without error
	cleanupWorkerRequestOrLogError(ctx, sm, envID, instanceID, requestID, l, "test message")
}
