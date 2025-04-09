package state

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"time"
)

var (
	ErrRequestLeased       = "request already leased"
	ErrRequestLeaseExpired = "request lease expired"
)

// LeaseRequest attempts to lease the given requestID for <duration>. If the request is already leased, this will fail with ErrRequestLeased.
func (r *redisConnectionStateManager) LeaseRequest(ctx context.Context, envID uuid.UUID, requestID ulid.ULID, duration time.Duration) (leaseID *ulid.ULID, err error) {
	return nil, fmt.Errorf("not implemented")
}

// ExtendRequestLease attempts to extend a lease for the given request. This will fail if the lease expired (ErrRequestLeaseExpired) or
// the current lease does not match the passed leaseID (ErrRequestLeased).
func (r *redisConnectionStateManager) ExtendRequestLease(ctx context.Context, envID uuid.UUID, requestID ulid.ULID, leaseID ulid.ULID, duration time.Duration) (newLeaseID *ulid.ULID, err error) {
	return nil, fmt.Errorf("not implemented")
}

// IsRequestLeased checks whether the given request is currently leased and the lease has not expired.
func (r *redisConnectionStateManager) IsRequestLeased(ctx context.Context, envID uuid.UUID, requestID ulid.ULID) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// SaveResponse is an idempotent, atomic write for reliably buffering a response for the executor to pick up
// in case Redis PubSub fails to notify the executor.
func (r *redisConnectionStateManager) SaveResponse(ctx context.Context, envID uuid.UUID, requestID ulid.ULID, resp *connpb.SDKResponse) error {
	return fmt.Errorf("not implemented")
}

// GetResponse retrieves the response for a given request, if exists. Otherwise, the response will be nil.
func (r *redisConnectionStateManager) GetResponse(ctx context.Context, envID uuid.UUID, requestID ulid.ULID) (*connpb.SDKResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// DeleteResponse is an idempotent delete operation for the temporary response buffer.
func (r *redisConnectionStateManager) DeleteResponse(ctx context.Context, envID uuid.UUID, requestID ulid.ULID) error {

	return fmt.Errorf("not implemented")
}
