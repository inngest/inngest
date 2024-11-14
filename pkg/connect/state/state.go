package state

import (
	"context"

	"github.com/google/uuid"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

type ConnectionStateManager interface {
	SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error
	GetConnectionsByEnvID(ctx context.Context, wsID uuid.UUID) ([]*connpb.ConnMetadata, error)
	GetConnectionsByAppID(ctx context.Context, appID uuid.UUID) ([]*connpb.ConnMetadata, error)
	AddConnection(ctx context.Context, data *connpb.SDKConnectRequestData) error
	DeleteConnection(ctx context.Context, connID string) error
}

type AuthContext struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}
