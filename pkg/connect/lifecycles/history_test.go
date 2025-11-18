package lifecycles

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/cqrs"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConnectionHistoryWriter is a mock implementation of cqrs.ConnectionHistoryWriter
type MockConnectionHistoryWriter struct {
	mock.Mock
}

func (m *MockConnectionHistoryWriter) InsertWorkerConnection(ctx context.Context, wc *cqrs.WorkerConnection) error {
	args := m.Called(ctx, wc)
	return args.Error(0)
}

func TestNewHistoryLifecycle(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}

	lifecycle := NewHistoryLifecycle(writer)
	
	require.NotNil(t, lifecycle)
	assert.Implements(t, (*connect.ConnectGatewayLifecycleListener)(nil), lifecycle)
}

func TestGetMaxWorkerConcurrency(t *testing.T) {
	tests := []struct {
		name       string
		connection *state.Connection
		expected   int64
	}{
		{
			name:       "nil connection returns 0",
			connection: nil,
			expected:   0,
		},
		{
			name: "nil data returns 0",
			connection: &state.Connection{
				Data: nil,
			},
			expected: 0,
		},
		{
			name: "nil MaxWorkerConcurrency returns 0",
			connection: &state.Connection{
				Data: &connectpb.WorkerConnectRequestData{
					MaxWorkerConcurrency: nil,
				},
			},
			expected: 0,
		},
		{
			name: "valid MaxWorkerConcurrency returns value",
			connection: &state.Connection{
				Data: &connectpb.WorkerConnectRequestData{
					MaxWorkerConcurrency: ptr.Int64(42),
				},
			},
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMaxWorkerConcurrency(tt.connection)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHistoryLifecycle_OnConnected(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	connectionId := ulid.Make()
	gatewayId := ulid.Make()
	accountId := uuid.New()
	envId := uuid.New()
	appId := uuid.New()
	syncId := uuid.New()

	connection := &state.Connection{
		AccountID:    accountId,
		EnvID:        envId,
		ConnectionId: connectionId,
		GatewayId:    gatewayId,
		WorkerIP:     "192.168.1.100",
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId:           "test-instance",
			MaxWorkerConcurrency: ptr.Int64(10),
			SdkLanguage:          "javascript",
			SdkVersion:           "3.0.0",
			SystemAttributes: &connectpb.SystemAttributes{
				CpuCores: 4,
				MemBytes: 8192,
				Os:       "linux",
							},
		},
		Groups: map[string]*state.WorkerGroup{
			"group1": {
				AppName:       "test-app",
				AppID:         &appId,
				AppVersion:    ptr.String("1.0.0"),
				SyncID:        &syncId,
				FunctionSlugs: []string{"func1", "func2"},
			},
		},
	}

	ctx := context.Background()

	writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
		return wc.AccountID == accountId &&
			wc.WorkspaceID == envId &&
			wc.Id == connectionId &&
			wc.GatewayId == gatewayId &&
			wc.InstanceId == "test-instance" &&
			wc.Status == connectpb.ConnectionStatus_CONNECTED &&
			wc.WorkerIP == "192.168.1.100" &&
			wc.MaxWorkerConcurrency == 10 &&
			wc.AppName == "test-app" &&
			wc.AppID != nil && *wc.AppID == appId &&
			wc.AppVersion != nil && *wc.AppVersion == "1.0.0" &&
			wc.SyncID != nil && *wc.SyncID == syncId &&
			wc.FunctionCount == 2 &&
			wc.GroupHash == "group1" &&
			wc.SDKLang == "javascript" &&
			wc.SDKVersion == "3.0.0" &&
			wc.SDKPlatform == connection.Data.GetPlatform() &&
			wc.CpuCores == 4 &&
			wc.MemBytes == 8192 &&
			wc.Os == "linux" &&
			wc.DisconnectedAt == nil &&
			wc.DisconnectReason == nil &&
			wc.LastHeartbeatAt != nil
	})).Return(nil)

	lifecycle.OnConnected(ctx, connection)

	writer.AssertExpectations(t)
}

func TestHistoryLifecycle_OnReady(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	connection := createTestConnection()
	ctx := context.Background()

	writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
		return wc.Status == connectpb.ConnectionStatus_READY
	})).Return(nil)

	lifecycle.OnReady(ctx, connection)

	writer.AssertExpectations(t)
}

func TestHistoryLifecycle_OnHeartbeat(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	connection := createTestConnection()
	ctx := context.Background()

	writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
		return wc.Status == connectpb.ConnectionStatus_READY
	})).Return(nil)

	lifecycle.OnHeartbeat(ctx, connection)

	writer.AssertExpectations(t)
}

func TestHistoryLifecycle_OnStartDraining(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	connection := createTestConnection()
	ctx := context.Background()

	writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
		return wc.Status == connectpb.ConnectionStatus_DRAINING
	})).Return(nil)

	lifecycle.OnStartDraining(ctx, connection)

	writer.AssertExpectations(t)
}

func TestHistoryLifecycle_OnStartDisconnecting(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	connection := createTestConnection()
	ctx := context.Background()

	writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
		return wc.Status == connectpb.ConnectionStatus_DISCONNECTING
	})).Return(nil)

	lifecycle.OnStartDisconnecting(ctx, connection)

	writer.AssertExpectations(t)
}

func TestHistoryLifecycle_OnDisconnected(t *testing.T) {
	tests := []struct {
		name               string
		closeReason        string
		expectedReason     *string
		expectedStatus     connectpb.ConnectionStatus
	}{
		{
			name:               "with close reason",
			closeReason:        "client_disconnect",
			expectedReason:     ptr.String("client_disconnect"),
			expectedStatus:     connectpb.ConnectionStatus_DISCONNECTED,
		},
		{
			name:               "empty close reason",
			closeReason:        "",
			expectedReason:     nil,
			expectedStatus:     connectpb.ConnectionStatus_DISCONNECTED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &MockConnectionHistoryWriter{}
			lifecycle := &historyLifecycles{writer: writer}

			connection := createTestConnection()
			ctx := context.Background()

			writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
				return wc.Status == tt.expectedStatus &&
					wc.DisconnectedAt != nil &&
					((tt.expectedReason == nil && wc.DisconnectReason == nil) ||
						(tt.expectedReason != nil && wc.DisconnectReason != nil && *wc.DisconnectReason == *tt.expectedReason))
			})).Return(nil)

			lifecycle.OnDisconnected(ctx, connection, tt.closeReason)

			writer.AssertExpectations(t)
		})
	}
}

func TestHistoryLifecycle_OnSynced(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	connection := createTestConnection()
	ctx := context.Background()

	// OnSynced is a no-op, so no expectations should be set
	lifecycle.OnSynced(ctx, connection)

	writer.AssertExpectations(t)
}

func TestHistoryLifecycle_ErrorHandling(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	connection := createTestConnection()
	ctx := context.Background()

	// Simulate an error during insertion
	writer.On("InsertWorkerConnection", ctx, mock.AnythingOfType("*cqrs.WorkerConnection")).
		Return(errors.New("database error"))

	// Should not panic despite the error
	lifecycle.OnConnected(ctx, connection)

	writer.AssertExpectations(t)
}

func TestHistoryLifecycle_MultipleGroups(t *testing.T) {
	writer := &MockConnectionHistoryWriter{}
	lifecycle := &historyLifecycles{writer: writer}

	appId1 := uuid.New()
	appId2 := uuid.New()
	syncId1 := uuid.New()
	syncId2 := uuid.New()

	connection := &state.Connection{
		AccountID:    uuid.New(),
		EnvID:        uuid.New(),
		ConnectionId: ulid.Make(),
		GatewayId:    ulid.Make(),
		WorkerIP:     "192.168.1.100",
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId:       "multi-app-instance",
			SdkLanguage:      "go",
			SdkVersion:       "1.0.0",
			SystemAttributes: &connectpb.SystemAttributes{
				CpuCores: 8,
				MemBytes: 16384,
				Os:       "linux",
			},
		},
		Groups: map[string]*state.WorkerGroup{
			"group1": {
				AppName:       "app1",
				AppID:         &appId1,
				SyncID:        &syncId1,
				FunctionSlugs: []string{"func1"},
			},
			"group2": {
				AppName:       "app2",
				AppID:         &appId2,
				SyncID:        &syncId2,
				FunctionSlugs: []string{"func2", "func3"},
			},
		},
	}

	ctx := context.Background()

	// Expect two calls, one for each group
	writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
		return wc.AppName == "app1" && wc.GroupHash == "group1" && wc.FunctionCount == 1
	})).Return(nil)

	writer.On("InsertWorkerConnection", ctx, mock.MatchedBy(func(wc *cqrs.WorkerConnection) bool {
		return wc.AppName == "app2" && wc.GroupHash == "group2" && wc.FunctionCount == 2
	})).Return(nil)

	lifecycle.OnConnected(ctx, connection)

	writer.AssertExpectations(t)
}

// Helper function to create a test connection
func createTestConnection() *state.Connection {
	appId := uuid.New()
	syncId := uuid.New()

	return &state.Connection{
		AccountID:    uuid.New(),
		EnvID:        uuid.New(),
		ConnectionId: ulid.Make(),
		GatewayId:    ulid.Make(),
		WorkerIP:     "192.168.1.100",
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId:           "test-instance",
			MaxWorkerConcurrency: ptr.Int64(5),
			SdkLanguage:          "go",
			SdkVersion:           "1.0.0",
			SystemAttributes: &connectpb.SystemAttributes{
				CpuCores: 2,
				MemBytes: 4096,
				Os:       "darwin",
							},
		},
		Groups: map[string]*state.WorkerGroup{
			"group1": {
				AppName:       "test-app",
				AppID:         &appId,
				AppVersion:    ptr.String("1.0.0"),
				SyncID:        &syncId,
				FunctionSlugs: []string{"func1"},
			},
		},
	}
}