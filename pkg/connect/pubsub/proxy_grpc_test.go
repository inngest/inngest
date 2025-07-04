package pubsub

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

func useGRPCAlways(ctx context.Context, accountID uuid.UUID) bool {
	return true
}

type mockGatewayGRPCForwarder struct {
	forwardCalls []mockForwardCall
	mu           sync.Mutex
}

type mockForwardCall struct {
	gatewayID    ulid.ULID
	connectionID ulid.ULID
	data         *connectpb.GatewayExecutorRequestData
}

func (m *mockGatewayGRPCForwarder) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forwardCalls = append(m.forwardCalls, mockForwardCall{
		gatewayID:    gatewayID,
		connectionID: connectionID,
		data:         data,
	})
	return nil
}

func (m *mockGatewayGRPCForwarder) ConnectToGateways(ctx context.Context) error {
	return nil
}

func (m *mockGatewayGRPCForwarder) getForwardCalls() []mockForwardCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]mockForwardCall, len(m.forwardCalls))
	copy(calls, m.forwardCalls)
	return calls
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
	mockForwarder := &mockGatewayGRPCForwarder{}

	connector := newRedisPubSubConnector(rc, RedisPubSubConnectorOpts{
		Logger:       l,
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		ShouldUseGRPC:        useGRPCAlways,
		GatewayGRPCForwarder: mockForwarder,
	})

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

	go func() {
		_ = connector.Wait(ctx)
	}()

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

	err = connector.NotifyExecutor(ctx, &connectpb.SDKResponse{
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
	mockForwarder := &mockGatewayGRPCForwarder{}

	connector := newRedisPubSubConnector(rc, RedisPubSubConnectorOpts{
		Logger:       l,
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		ShouldUseGRPC:        useGRPCAlways,
		GatewayGRPCForwarder: mockForwarder,
	})

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

	go func() {
		_ = connector.Wait(ctx)
	}()

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

	calls := mockForwarder.getForwardCalls()
	require.Len(t, calls, 0, "expected no gRPC Forward calls since response was already buffered")

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
	mockForwarder := &mockGatewayGRPCForwarder{}

	connector := newRedisPubSubConnector(rc, RedisPubSubConnectorOpts{
		Logger:       l,
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		ShouldUseGRPC:        useGRPCAlways,
		GatewayGRPCForwarder: mockForwarder,
	})

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

	go func() {
		_ = connector.Wait(ctx)
	}()

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
