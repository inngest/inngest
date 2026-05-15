package lifecycles

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
)

type metricsLifecycle struct {
	writer       cqrs.ConnectionMetricsWriter
	stateManager state.StateManager
}

// NewMetricsLifecycle creates a new lifecycle listener for connection metrics
func NewMetricsLifecycle(
	writer cqrs.ConnectionMetricsWriter,
	stateManager state.StateManager,
) connect.ConnectGatewayLifecycleListener {
	return &metricsLifecycle{
		writer:       writer,
		stateManager: stateManager,
	}
}

func (m *metricsLifecycle) OnConnected(ctx context.Context, conn *state.Connection) {
	m.recordConnectionMetric(ctx, conn)
}

func (m *metricsLifecycle) OnReady(ctx context.Context, conn *state.Connection) {
	m.recordConnectionMetric(ctx, conn)
}

func (m *metricsLifecycle) OnHeartbeat(ctx context.Context, conn *state.Connection) {
	m.recordConnectionMetric(ctx, conn)
}

func (m *metricsLifecycle) OnSynced(ctx context.Context, conn *state.Connection) {
	// No-op for now
}

func (m *metricsLifecycle) OnStartDraining(ctx context.Context, conn *state.Connection) {
	// No-op for now
}

func (m *metricsLifecycle) OnStartDisconnecting(ctx context.Context, conn *state.Connection) {
	// No-op for now
}

func (m *metricsLifecycle) OnDisconnected(ctx context.Context, conn *state.Connection, closeReason string) {
	// No-op for now
}

func (m *metricsLifecycle) recordConnectionMetric(ctx context.Context, conn *state.Connection) {
	workerCap, err := m.stateManager.GetWorkerCapacities(ctx, conn.EnvID, conn.Data.InstanceId)
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to get worker capacities for connection metrics recording",
			"error", err,
			"connection_id", conn.ConnectionId.String(),
			"instance_id", conn.Data.InstanceId,
		)
		return
	}

	// Record a single metric per connection
	metric := &cqrs.ConnectionMetric{
		AccountID:   conn.AccountID,
		WorkspaceID: conn.EnvID,

		InstanceID:   conn.Data.InstanceId,
		ConnectionID: conn.ConnectionId,
		GatewayID:    conn.GatewayId,

		TotalCapacity:     workerCap.Total,
		AvailableCapacity: workerCap.Available,

		Timestamp:  time.Now().Truncate(time.Minute), // track minute level metrics
		RecordedAt: time.Now(),                       // always records the latest value for the minute
	}

	if err := m.writer.InsertConnectionMetric(ctx, metric); err != nil {
		logger.StdlibLogger(ctx).Error("failed to insert connection metric",
			"error", err,
			"connection_id", conn.ConnectionId.String(),
			"instance_id", conn.Data.InstanceId,
		)
	}
}
