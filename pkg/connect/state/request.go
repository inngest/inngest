package state

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"time"
)

func (r *redisConnectionStateManager) LeaseRequest(ctx context.Context, envID uuid.UUID, requestID ulid.ULID, duration time.Duration) (leaseID *ulid.ULID, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *redisConnectionStateManager) ExtendRequestLease(ctx context.Context, envID uuid.UUID, requestID ulid.ULID, leaseId ulid.ULID, duration time.Duration) (newLeaseID *ulid.ULID, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *redisConnectionStateManager) IsRequestLeased(ctx context.Context, envID uuid.UUID, requestID ulid.ULID) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (r *redisConnectionStateManager) SaveResponse(ctx context.Context, envID uuid.UUID, requestID ulid.ULID, resp *connpb.SDKResponse) error {
	return fmt.Errorf("not implemented")
}

func (r *redisConnectionStateManager) GetResponse(ctx context.Context, envID uuid.UUID, requestID ulid.ULID) (*connpb.SDKResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *redisConnectionStateManager) DeleteResponse(ctx context.Context, envID uuid.UUID, requestID ulid.ULID) error {
	return fmt.Errorf("not implemented")
}
