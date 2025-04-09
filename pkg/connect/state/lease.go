package state

import (
	"context"
	"fmt"
	"github.com/oklog/ulid/v2"
	"time"
)

func (r *redisConnectionStateManager) LeaseRequest(ctx context.Context, requestID ulid.ULID, duration time.Time) (leaseID *ulid.ULID, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *redisConnectionStateManager) ExtendRequestLease(ctx context.Context, requestID ulid.ULID, leaseId ulid.ULID, duration time.Time) (newLeaseID *ulid.ULID, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *redisConnectionStateManager) IsRequestLeased(ctx context.Context, requestID ulid.ULID) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
