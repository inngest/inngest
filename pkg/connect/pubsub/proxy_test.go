package pubsub

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

func TestProxyPubSubPath(t *testing.T) {
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

	connector := newRedisPubSubConnector(rc, RedisPubSubConnectorOpts{
		Logger:       l,
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
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
		fmt.Println("proxy done")
		require.NoError(t, err)
		require.NotNil(t, resp)
		respCh <- resp
	}()

	received := make(chan bool)

	onSubscribed := make(chan struct{})
	go func() {
		connector.subscribe(ctx, connector.channelGatewayAppRequests(gwID, connID), func(msg string) {
			fmt.Println("received msg")
			received <- true
		}, true, onSubscribed)
	}()
	<-onSubscribed

	select {
	case <-received:
	case <-time.After(10 * time.Second):
		require.Fail(t, "did not receive message")
	}

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

func TestProxyPolling(t *testing.T) {
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

	connector := newRedisPubSubConnector(rc, RedisPubSubConnectorOpts{
		Logger:       l,
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
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
		fmt.Println("proxy done")
		require.NoError(t, err)
		require.NotNil(t, resp)
		respCh <- resp
	}()

	received := make(chan bool)

	onSubscribed := make(chan struct{})
	go func() {
		connector.subscribe(ctx, connector.channelGatewayAppRequests(gwID, connID), func(msg string) {
			fmt.Println("received msg")
			received <- true
		}, true, onSubscribed)
	}()
	<-onSubscribed

	select {
	case <-received:
	case <-time.After(10 * time.Second):
		require.Fail(t, "did not receive message")
	}

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

	buffered, err := sm.GetResponse(ctx, envID, reqID)
	require.NoError(t, err)
	require.NotNil(t, buffered)

	select {
	case r := <-respCh:
		require.Equal(t, connectpb.SDKResponseStatus_DONE, r.Status)

		// expect buffered response to be deleted
		buffered, err := sm.GetResponse(ctx, envID, reqID)
		require.NoError(t, err)
		require.Nil(t, buffered)

	case <-time.After(10 * time.Second):
		require.Fail(t, "no response received")
	}
}

func TestProxyLeaseExpiry(t *testing.T) {
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

	connector := newRedisPubSubConnector(rc, RedisPubSubConnectorOpts{
		Logger:       l,
		Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
		StateManager: sm,
		EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
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
		var sysErr syscode.Error
		require.ErrorAs(t, err, &sysErr)
		require.Equal(t, syscode.CodeConnectWorkerStoppedResponding, sysErr.Code)
		require.Nil(t, resp)
		respCh <- struct{}{}
	}()

	received := make(chan bool)

	onSubscribed := make(chan struct{})
	go func() {
		connector.subscribe(ctx, connector.channelGatewayAppRequests(gwID, connID), func(msg string) {
			fmt.Println("received msg")

			// simulate the lease being dropped
			require.NoError(t, sm.DeleteLease(ctx, envID, reqID))

			received <- true
		}, true, onSubscribed)
	}()
	<-onSubscribed

	select {
	case <-received:
	case <-time.After(10 * time.Second):
		require.Fail(t, "did not receive message")
	}

	<-time.After(consts.ConnectWorkerRequestExtendLeaseInterval + time.Second)

	select {
	case <-respCh:
	case <-time.After(10 * time.Second):
		require.Fail(t, "no response received")
	}
}
