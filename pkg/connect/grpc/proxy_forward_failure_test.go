package grpc

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stuckForwarder simulates a Forward stuck on a dead connection: it runs
// duringForward while the call is in flight, then fails with err.
type stuckForwarder struct {
	mu            sync.Mutex
	calls         int
	err           error
	block         time.Duration
	duringForward func(ctx context.Context, data *connectpb.GatewayExecutorRequestData)
	replies       chan *connectpb.SDKResponse
}

func (m *stuckForwarder) Forward(ctx context.Context, gatewayID ulid.ULID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	if m.duringForward != nil {
		m.duringForward(ctx, data)
	}

	select {
	case <-time.After(m.block):
	case <-ctx.Done():
	}
	return m.err
}

func (m *stuckForwarder) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *stuckForwarder) ConnectToGateways(ctx context.Context) error { return nil }

func (m *stuckForwarder) Subscribe(ctx context.Context, requestID string) chan *connectpb.SDKResponse {
	return m.replies
}

func (m *stuckForwarder) Unsubscribe(ctx context.Context, requestID string) {}

func TestProxyGRPCForwardFailsAfterDelivery(t *testing.T) {
	for _, tc := range []struct {
		name string
		// forwarder builds the stuck forwarder. It gets the state manager so it
		// can change the request's lease while Forward is in flight.
		forwarder  func(sm state.StateManager, envID uuid.UUID) *stuckForwarder
		expectResp bool
		expectCode string
	}{
		{
			name: "response arrived during forward is used instead of re-routing",
			forwarder: func(sm state.StateManager, envID uuid.UUID) *stuckForwarder {
				f := &stuckForwarder{
					err:     status.Error(codes.Unavailable, "error reading from server: connection timed out"),
					block:   200 * time.Millisecond,
					replies: make(chan *connectpb.SDKResponse, 1),
				}
				f.duringForward = func(ctx context.Context, data *connectpb.GatewayExecutorRequestData) {
					select {
					case f.replies <- &connectpb.SDKResponse{RequestId: data.RequestId, Status: connectpb.SDKResponseStatus_DONE}:
					default:
					}
				}
				return f
			},
			expectResp: true,
		},
		{
			name: "lease expired during forward fails as worker stopped responding",
			forwarder: func(sm state.StateManager, envID uuid.UUID) *stuckForwarder {
				f := &stuckForwarder{
					err: status.Error(codes.DeadlineExceeded, "context deadline exceeded"),
					// Long enough for the lease watcher to notice and pass its grace period.
					block:   40 * time.Second,
					replies: make(chan *connectpb.SDKResponse),
				}
				f.duringForward = func(ctx context.Context, data *connectpb.GatewayExecutorRequestData) {
					// Nothing extends the lease, as when the request never reached a worker.
					require.NoError(t, sm.DeleteLease(ctx, envID, data.RequestId))
				}
				return f
			},
			expectCode: syscode.CodeConnectWorkerStoppedResponding,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
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

			fnID, accID, envID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			forwarder := tc.forwarder(sm, envID)

			connector := newGRPCConnector(ctx, GRPCConnectorOpts{
				Tracer:       trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace),
				StateManager: sm,
				EnforceLeaseExpiry: func(ctx context.Context, accountID uuid.UUID) bool {
					return true
				},
			}, WithConnectorLogger(l), WithGatewayManager(forwarder))

			runID := ulid.MustNew(ulid.Now(), rand.Reader)
			connID := ulid.MustNew(ulid.Now(), rand.Reader)
			gwID := ulid.MustNew(ulid.Now(), rand.Reader)
			appName := "test-app"
			fnSlug := "test-app-test-fn"
			appVersion := "v1.1"

			fns, err := json.Marshal([]sdk.SDKFunction{
				{
					Name:     "test-fn",
					Slug:     fnSlug,
					Triggers: []inngest.Trigger{{EventTrigger: &inngest.EventTrigger{Event: "hello/world"}}},
					Steps: map[string]sdk.SDKStep{
						"step": {
							ID:      "step",
							Name:    "test-fn",
							Runtime: map[string]any{"url": fmt.Sprintf("ws://connect?fnId=%s&step=step", fnSlug)},
						},
					},
				},
			})
			require.NoError(t, err)

			appConfig := &connectpb.AppConfiguration{AppName: appName, AppVersion: &appVersion, Functions: fns}

			caps, err := json.Marshal(sdk.Capabilities{InBandSync: sdk.InBandSyncV1, TrustProbe: sdk.TrustProbeV1, Connect: sdk.ConnectV1})
			require.NoError(t, err)

			reqData := &connectpb.WorkerConnectRequestData{
				ConnectionId: connID.String(),
				InstanceId:   "test-worker",
				Apps:         []*connectpb.AppConfiguration{appConfig},
				SdkLanguage:  "test-sdk",
				SdkVersion:   "test-version",
				Capabilities: caps,
				AuthData:     &connectpb.AuthData{SessionToken: "test-session-token", SyncToken: "test-sync-token"},
			}

			wg, err := state.NewWorkerGroupFromConnRequest(ctx, reqData, &auth.Response{
				AccountID:    accID,
				EnvID:        envID,
				Entitlements: auth.Entitlements{ConnectionAllowed: true, AppsPerConnection: 10},
			}, appConfig)
			require.NoError(t, err)

			syncID := uuid.New()
			wg.AppID = &appID
			wg.SyncID = &syncID

			require.NoError(t, sm.UpsertConnection(ctx, &state.Connection{
				AccountID:    accID,
				EnvID:        envID,
				ConnectionId: connID,
				WorkerIP:     "10.0.1.2",
				Data:         reqData,
				Groups:       map[string]*state.WorkerGroup{wg.Hash: wg},
				GatewayId:    gwID,
			}, connectpb.ConnectionStatus_READY, time.Now()))

			require.NoError(t, sm.UpsertGateway(ctx, &state.Gateway{
				Id:                gwID,
				Status:            state.GatewayStatusActive,
				LastHeartbeatAtMS: time.Now().UnixMilli(),
				Hostname:          "gw-host",
				IPAddress:         net.ParseIP("127.0.0.1"),
			}))

			resp, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
				AccountID: accID,
				EnvID:     envID,
				AppID:     appID,
				Data: &connectpb.GatewayExecutorRequestData{
					RequestId:      "reqid-test",
					AccountId:      accID.String(),
					EnvId:          envID.String(),
					AppId:          appID.String(),
					AppName:        appName,
					FunctionId:     fnID.String(),
					FunctionSlug:   fnSlug,
					RequestPayload: []byte("request payload"),
					RunId:          runID.String(),
				},
				logger: l,
			})

			// Either way the request must not be sent to another worker.
			require.Equal(t, 1, forwarder.getCalls())

			if tc.expectResp {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, "reqid-test", resp.RequestId)
				return
			}

			require.Nil(t, resp)
			var sysErr syscode.Error
			require.True(t, errors.As(err, &sysErr), "expected syscode error, got %v", err)
			require.Equal(t, tc.expectCode, sysErr.Code)
		})
	}
}
