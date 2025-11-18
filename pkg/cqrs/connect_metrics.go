package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// ConnectionMetric represents a snapshot of connection capacity at a point in time
type ConnectionMetric struct {
	AccountID   uuid.UUID `json:"account_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`

	InstanceID   string    `json:"instance_id"`
	ConnectionID ulid.ULID `json:"connection_id"`
	GatewayID    ulid.ULID `json:"gateway_id"`

	TotalCapacity   int64 `json:"total_capacity"`
	CurrentCapacity int64 `json:"current_capacity"`

	Timestamp  time.Time `json:"timestamp"`
	RecordedAt time.Time `json:"recorded_at"`
}

// ConnectionMetricsWriter defines the interface for writing connection capacity metrics
type ConnectionMetricsWriter interface {
	InsertConnectionMetric(ctx context.Context, metric *ConnectionMetric) error
}
