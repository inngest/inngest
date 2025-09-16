package capacity

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

type redisCapacityManager struct {
	client rueidis.Client
}

// ExtendCapacityLease implements CapacityManager.
func (r *redisCapacityManager) ExtendCapacityLease(ctx context.Context, leaseID ulid.ULID, duration time.Duration) (ulid.ULID, error) {
	panic("unimplemented")
}

// LeaseCapacity implements CapacityManager.
func (r *redisCapacityManager) LeaseCapacity(ctx context.Context, req CapacityLeaseRequest) (*LeaseCapacityResponse, error) {
	panic("unimplemented")
}

// ReleaseCapacity implements CapacityManager.
func (r *redisCapacityManager) ReleaseCapacity(ctx context.Context, leaseID ulid.ULID) error {
	panic("unimplemented")
}

func NewRedisCapacityManager(client rueidis.Client) CapacityManager {
	return &redisCapacityManager{
		client: client,
	}
}
