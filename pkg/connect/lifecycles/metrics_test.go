package lifecycles

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

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

// MockConnectionMetricsWriter is a mock implementation of cqrs.ConnectionMetricsWriter
type MockConnectionMetricsWriter struct {
	mock.Mock
}

func (m *MockConnectionMetricsWriter) InsertConnectionMetric(ctx context.Context, metric *cqrs.ConnectionMetric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

// MockStateManager is a mock implementation that implements just enough of state.StateManager for testing
type MockStateManager struct {
	mock.Mock
}

// Methods needed by metrics lifecycle
func (m *MockStateManager) GetWorkerCapacities(ctx context.Context, envID uuid.UUID, instanceID string) (*state.WorkerCapacity, error) {
	args := m.Called(ctx, envID, instanceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*state.WorkerCapacity), args.Error(1)
}

// Stub implementations for other StateManager interface methods - these won't be called in the metrics tests
func (m *MockStateManager) GetConnection(ctx context.Context, envID uuid.UUID, connId ulid.ULID) (*connectpb.ConnMetadata, error) {
	panic("not implemented")
}
func (m *MockStateManager) GetConnectionsByEnvID(ctx context.Context, envID uuid.UUID) ([]*connectpb.ConnMetadata, error) {
	panic("not implemented")
}
func (m *MockStateManager) GetConnectionsByAppID(ctx context.Context, envId uuid.UUID, appID uuid.UUID) ([]*connectpb.ConnMetadata, error) {
	panic("not implemented")
}
func (m *MockStateManager) GetConnectionsByGroupID(ctx context.Context, envID uuid.UUID, groupID string) ([]*connectpb.ConnMetadata, error) {
	panic("not implemented")
}
func (m *MockStateManager) UpsertConnection(ctx context.Context, conn *state.Connection, status connectpb.ConnectionStatus, lastHeartbeatAt time.Time) error {
	panic("not implemented")
}
func (m *MockStateManager) DeleteConnection(ctx context.Context, envID uuid.UUID, connId ulid.ULID) error {
	panic("not implemented")
}
func (m *MockStateManager) GarbageCollectConnections(ctx context.Context) (int, error) {
	panic("not implemented")
}
func (m *MockStateManager) GarbageCollectGateways(ctx context.Context) (int, error) {
	panic("not implemented")
}
func (m *MockStateManager) GetWorkerGroupByHash(ctx context.Context, envID uuid.UUID, hash string) (*state.WorkerGroup, error) {
	panic("not implemented")
}
func (m *MockStateManager) UpdateWorkerGroup(ctx context.Context, envID uuid.UUID, group *state.WorkerGroup) error {
	panic("not implemented")
}
func (m *MockStateManager) UpsertGateway(ctx context.Context, gateway *state.Gateway) error {
	panic("not implemented")
}
func (m *MockStateManager) DeleteGateway(ctx context.Context, gatewayId ulid.ULID) error {
	panic("not implemented")
}
func (m *MockStateManager) GetGateway(ctx context.Context, gatewayId ulid.ULID) (*state.Gateway, error) {
	panic("not implemented")
}
func (m *MockStateManager) GetAllGateways(ctx context.Context) ([]*state.Gateway, error) {
	panic("not implemented")
}
func (m *MockStateManager) GetAllGatewayIDs(ctx context.Context) ([]string, error) {
	panic("not implemented")
}
func (m *MockStateManager) LeaseRequest(ctx context.Context, envID uuid.UUID, requestID string, duration time.Duration, executorIP net.IP) (leaseID *ulid.ULID, err error) {
	panic("not implemented")
}
func (m *MockStateManager) ExtendRequestLease(ctx context.Context, envID uuid.UUID, instanceID string, requestID string, leaseID ulid.ULID, duration time.Duration, isWorkerCapacityUnlimited bool) (newLeaseID *ulid.ULID, err error) {
	panic("not implemented")
}
func (m *MockStateManager) IsRequestLeased(ctx context.Context, envID uuid.UUID, requestID string) (bool, error) {
	panic("not implemented")
}
func (m *MockStateManager) AssignRequestToWorker(ctx context.Context, envID uuid.UUID, instanceID string, requestID string) error {
	panic("not implemented")
}
func (m *MockStateManager) DeleteRequestFromWorker(ctx context.Context, envID uuid.UUID, instanceID string, requestID string) error {
	panic("not implemented")
}
func (m *MockStateManager) SetWorkerTotalCapacity(ctx context.Context, envID uuid.UUID, instanceID string, maxConcurrentLeases int64) error {
	panic("not implemented")
}
func (m *MockStateManager) GetWorkerTotalCapacity(ctx context.Context, envID uuid.UUID, instanceID string) (int64, error) {
	panic("not implemented")
}
func (m *MockStateManager) CreateLease(ctx context.Context, envID uuid.UUID, requestID string, executorIP net.IP, instanceID string) error {
	panic("not implemented")
}
func (m *MockStateManager) DeleteLease(ctx context.Context, envID uuid.UUID, requestID string) error {
	panic("not implemented")
}
func (m *MockStateManager) GetExecutorIP(ctx context.Context, envID uuid.UUID, requestID string) (net.IP, error) {
	panic("not implemented")
}
func (m *MockStateManager) GetAssignedWorkerID(ctx context.Context, envID uuid.UUID, requestID string) (string, error) {
	panic("not implemented")
}
func (m *MockStateManager) SaveResponse(ctx context.Context, envID uuid.UUID, requestID string, resp *connectpb.SDKResponse) error {
	panic("not implemented")
}
func (m *MockStateManager) GetResponse(ctx context.Context, envID uuid.UUID, requestID string) (*connectpb.SDKResponse, error) {
	panic("not implemented")
}
func (m *MockStateManager) DeleteResponse(ctx context.Context, envID uuid.UUID, requestID string) error {
	panic("not implemented")
}
func (m *MockStateManager) GetAllActiveWorkerRequests(ctx context.Context, envID uuid.UUID, instanceID string, includeExtended bool) ([]string, error) {
	panic("not implemented")
}
func (m *MockStateManager) WorkerCapacityOnHeartbeat(ctx context.Context, envID uuid.UUID, instanceID string) error {
	panic("not implemented")
}

func TestNewMetricsLifecycle(t *testing.T) {
	writer := &MockConnectionMetricsWriter{}
	stateManager := &MockStateManager{}

	lifecycle := NewMetricsLifecycle(writer, stateManager)
	
	require.NotNil(t, lifecycle)
	assert.Implements(t, (*connect.ConnectGatewayLifecycleListener)(nil), lifecycle)
}

func TestMetricsLifecycle_OnConnected(t *testing.T) {
	tests := []struct {
		name           string
		connection     *state.Connection
		workerCapacity *state.WorkerCapacity
		capacityErr    error
		insertErr      error
		expectInsert   bool
	}{
		{
			name: "successful metric recording",
			connection: &state.Connection{
				AccountID:    uuid.New(),
				EnvID:        uuid.New(),
				ConnectionId: ulid.Make(),
				GatewayId:    ulid.Make(),
				Data: &connectpb.WorkerConnectRequestData{
					InstanceId: "test-instance-1",
				},
			},
			workerCapacity: &state.WorkerCapacity{
				Total:     100,
				Available: 80,
			},
			expectInsert: true,
		},
		{
			name: "worker capacity fetch error",
			connection: &state.Connection{
				AccountID:    uuid.New(),
				EnvID:        uuid.New(),
				ConnectionId: ulid.Make(),
				GatewayId:    ulid.Make(),
				Data: &connectpb.WorkerConnectRequestData{
					InstanceId: "test-instance-2",
				},
			},
			capacityErr:  errors.New("capacity fetch failed"),
			expectInsert: false,
		},
		{
			name: "metric insertion error",
			connection: &state.Connection{
				AccountID:    uuid.New(),
				EnvID:        uuid.New(),
				ConnectionId: ulid.Make(),
				GatewayId:    ulid.Make(),
				Data: &connectpb.WorkerConnectRequestData{
					InstanceId: "test-instance-3",
				},
			},
			workerCapacity: &state.WorkerCapacity{
				Total:     50,
				Available: 25,
			},
			insertErr:    errors.New("insert failed"),
			expectInsert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &MockConnectionMetricsWriter{}
			stateManager := &MockStateManager{}
			
			lifecycle := &metricsLifecycle{
				writer:       writer,
				stateManager: stateManager,
			}

			ctx := context.Background()

			// Setup expectations
			stateManager.On("GetWorkerCapacities", ctx, tt.connection.EnvID, tt.connection.Data.InstanceId).
				Return(tt.workerCapacity, tt.capacityErr)

			if tt.expectInsert {
				writer.On("InsertConnectionMetric", ctx, mock.MatchedBy(func(metric *cqrs.ConnectionMetric) bool {
					return metric.AccountID == tt.connection.AccountID &&
						metric.WorkspaceID == tt.connection.EnvID &&
						metric.InstanceID == tt.connection.Data.InstanceId &&
						metric.ConnectionID == tt.connection.ConnectionId &&
						metric.GatewayID == tt.connection.GatewayId &&
						metric.TotalCapacity == tt.workerCapacity.Total &&
						metric.CurrentCapacity == tt.workerCapacity.Available &&
						!metric.Timestamp.IsZero() &&
						!metric.RecordedAt.IsZero()
				})).Return(tt.insertErr)
			}

			// Execute
			lifecycle.OnConnected(ctx, tt.connection)

			// Verify
			stateManager.AssertExpectations(t)
			writer.AssertExpectations(t)
		})
	}
}

func TestMetricsLifecycle_OnReady(t *testing.T) {
	writer := &MockConnectionMetricsWriter{}
	stateManager := &MockStateManager{}
	
	lifecycle := &metricsLifecycle{
		writer:       writer,
		stateManager: stateManager,
	}

	connection := &state.Connection{
		AccountID:    uuid.New(),
		EnvID:        uuid.New(),
		ConnectionId: ulid.Make(),
		GatewayId:    ulid.Make(),
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId: "ready-instance",
		},
	}

	workerCapacity := &state.WorkerCapacity{
		Total:     200,
		Available: 150,
	}

	ctx := context.Background()

	stateManager.On("GetWorkerCapacities", ctx, connection.EnvID, connection.Data.InstanceId).
		Return(workerCapacity, nil)

	writer.On("InsertConnectionMetric", ctx, mock.MatchedBy(func(metric *cqrs.ConnectionMetric) bool {
		return metric.InstanceID == "ready-instance" &&
			metric.TotalCapacity == 200 &&
			metric.CurrentCapacity == 150
	})).Return(nil)

	lifecycle.OnReady(ctx, connection)

	stateManager.AssertExpectations(t)
	writer.AssertExpectations(t)
}

func TestMetricsLifecycle_OnHeartbeat(t *testing.T) {
	writer := &MockConnectionMetricsWriter{}
	stateManager := &MockStateManager{}
	
	lifecycle := &metricsLifecycle{
		writer:       writer,
		stateManager: stateManager,
	}

	connection := &state.Connection{
		AccountID:    uuid.New(),
		EnvID:        uuid.New(),
		ConnectionId: ulid.Make(),
		GatewayId:    ulid.Make(),
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId: "heartbeat-instance",
		},
	}

	workerCapacity := &state.WorkerCapacity{
		Total:     300,
		Available: 275,
	}

	ctx := context.Background()

	stateManager.On("GetWorkerCapacities", ctx, connection.EnvID, connection.Data.InstanceId).
		Return(workerCapacity, nil)

	writer.On("InsertConnectionMetric", ctx, mock.MatchedBy(func(metric *cqrs.ConnectionMetric) bool {
		return metric.InstanceID == "heartbeat-instance" &&
			metric.TotalCapacity == 300 &&
			metric.CurrentCapacity == 275
	})).Return(nil)

	lifecycle.OnHeartbeat(ctx, connection)

	stateManager.AssertExpectations(t)
	writer.AssertExpectations(t)
}

func TestMetricsLifecycle_NoOpMethods(t *testing.T) {
	writer := &MockConnectionMetricsWriter{}
	stateManager := &MockStateManager{}
	
	lifecycle := &metricsLifecycle{
		writer:       writer,
		stateManager: stateManager,
	}

	connection := &state.Connection{
		AccountID:    uuid.New(),
		EnvID:        uuid.New(),
		ConnectionId: ulid.Make(),
		GatewayId:    ulid.Make(),
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId: "noop-instance",
		},
	}

	ctx := context.Background()

	// These methods should not call any dependencies
	lifecycle.OnSynced(ctx, connection)
	lifecycle.OnStartDraining(ctx, connection)
	lifecycle.OnStartDisconnecting(ctx, connection)
	lifecycle.OnDisconnected(ctx, connection, "test reason")

	// No expectations should be set and no calls should be made
	stateManager.AssertExpectations(t)
	writer.AssertExpectations(t)
}

func TestMetricsLifecycle_TimestampTruncation(t *testing.T) {
	writer := &MockConnectionMetricsWriter{}
	stateManager := &MockStateManager{}
	
	lifecycle := &metricsLifecycle{
		writer:       writer,
		stateManager: stateManager,
	}

	connection := &state.Connection{
		AccountID:    uuid.New(),
		EnvID:        uuid.New(),
		ConnectionId: ulid.Make(),
		GatewayId:    ulid.Make(),
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId: "timestamp-test",
		},
	}

	workerCapacity := &state.WorkerCapacity{
		Total:     100,
		Available: 80,
	}

	ctx := context.Background()

	stateManager.On("GetWorkerCapacities", ctx, connection.EnvID, connection.Data.InstanceId).
		Return(workerCapacity, nil)

	var capturedMetric *cqrs.ConnectionMetric
	writer.On("InsertConnectionMetric", ctx, mock.MatchedBy(func(metric *cqrs.ConnectionMetric) bool {
		capturedMetric = metric
		return true
	})).Return(nil)

	lifecycle.OnConnected(ctx, connection)

	require.NotNil(t, capturedMetric)
	
	// Verify timestamp is truncated to minute
	expectedTruncated := capturedMetric.RecordedAt.Truncate(time.Minute)
	assert.Equal(t, expectedTruncated, capturedMetric.Timestamp)
	
	// Verify RecordedAt is more precise
	assert.True(t, capturedMetric.RecordedAt.After(capturedMetric.Timestamp) || 
		capturedMetric.RecordedAt.Equal(capturedMetric.Timestamp))

	stateManager.AssertExpectations(t)
	writer.AssertExpectations(t)
}