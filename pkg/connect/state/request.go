package state

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"google.golang.org/protobuf/proto"
	"time"
)

var (
	ErrRequestLeased        = fmt.Errorf("request already leased")
	ErrRequestLeaseExpired  = fmt.Errorf("request lease expired")
	ErrRequestLeaseNotFound = fmt.Errorf("request not leased")

	ErrResponseAlreadyBuffered = fmt.Errorf("response already buffered")
)

// keyRequestLease points to the key storing the request lease
func (r *redisConnectionStateManager) keyRequestLease(envID uuid.UUID, requestID string) string {
	return fmt.Sprintf("{%s}:request-lease:%s", envID, requestID)
}

// keyBufferedResponse points to the key storing the buffered SDK response
func (r *redisConnectionStateManager) keyBufferedResponse(envID uuid.UUID, requestID string) string {
	return fmt.Sprintf("{%s}:buffered-response:%s", envID, requestID)
}

// LeaseRequest attempts to lease the given requestID for <duration>. If the request is already leased, this will fail with ErrRequestLeased.
func (r *redisConnectionStateManager) LeaseRequest(ctx context.Context, envID uuid.UUID, requestID string, duration time.Duration) (*ulid.ULID, error) {
	keys := []string{
		r.keyRequestLease(envID, requestID),
	}

	now := r.c.Now()

	leaseExpiry := now.Add(duration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not create lease ID: %w", err)
	}

	// Expire request lease key after the max request duration with a tiny fudge factor
	keyExpiry := consts.MaxFunctionTimeout + duration

	args := []string{
		leaseID.String(),
		fmt.Sprintf("%d", int(keyExpiry.Seconds())),
		fmt.Sprintf("%d", now.UnixMilli()),
	}

	status, err := scripts["lease"].Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("could not execute lease script: %w", err)
	}

	switch status {
	case 1:
		return &leaseID, nil
	case -1:
		return nil, ErrRequestLeased
	default:
		return nil, fmt.Errorf("unexpected status %d", status)
	}
}

// ExtendRequestLease attempts to extend a lease for the given request. This will fail if the lease expired (ErrRequestLeaseExpired) or
// the current lease does not match the passed leaseID (ErrRequestLeased).
func (r *redisConnectionStateManager) ExtendRequestLease(ctx context.Context, envID uuid.UUID, requestID string, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	keys := []string{
		r.keyRequestLease(envID, requestID),
	}

	now := r.c.Now()

	leaseExpiry := now.Add(duration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not create lease ID: %w", err)
	}

	// Expire request lease key after the max request duration with a tiny fudge factor
	keyExpiry := consts.MaxFunctionTimeout + duration

	args := []string{
		leaseID.String(),
		newLeaseID.String(),
		fmt.Sprintf("%d", int(keyExpiry.Seconds())),
		fmt.Sprintf("%d", now.UnixMilli()),
	}

	status, err := scripts["extend_lease"].Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("could not execute lease script: %w", err)
	}

	switch status {
	case -2:
		return nil, ErrRequestLeased
	case -1:
		return nil, ErrRequestLeaseNotFound
	case 1:
		// Lease extended
		return &newLeaseID, nil
	case 2:
		// Lease deleted (duration <= 0)
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected status %d", status)
	}
}

// IsRequestLeased checks whether the given request is currently leased and the lease has not expired.
func (r *redisConnectionStateManager) IsRequestLeased(ctx context.Context, envID uuid.UUID, requestID string) (bool, error) {
	keys := []string{
		r.keyRequestLease(envID, requestID),
	}

	now := r.c.Now()

	args := []string{
		fmt.Sprintf("%d", now.UnixMilli()),
	}

	status, err := scripts["is_leased"].Exec(
		ctx,
		r.client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return false, fmt.Errorf("could not execute lease script: %w", err)
	}

	switch status {
	case 0, 1:
		return false, nil
	case 2:
		return true, nil
	default:
		return false, fmt.Errorf("unexpected status %d", status)
	}
}

// DeleteLease allows the executor to clean up the lease once the request is done processing.
func (r *redisConnectionStateManager) DeleteLease(ctx context.Context, envID uuid.UUID, requestID string) error {
	cmd := r.client.B().Del().Key(r.keyRequestLease(envID, requestID)).Build()

	err := r.client.Do(ctx, cmd).Error()
	if err != nil && !rueidis.IsRedisNil(err) {
		return fmt.Errorf("could not delete lease: %w", err)
	}

	return nil
}

// SaveResponse is an idempotent, atomic write for reliably buffering a response for the executor to pick up
// in case Redis PubSub fails to notify the executor.
func (r *redisConnectionStateManager) SaveResponse(ctx context.Context, envID uuid.UUID, requestID string, resp *connpb.SDKResponse) error {
	marshaled, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("could not marshal response: %w", err)
	}

	responseExpiry := 30 * time.Second

	cmd := r.client.
		B().
		Set().
		Key(r.keyBufferedResponse(envID, requestID)).
		Value(string(marshaled)).
		Nx().
		Ex(responseExpiry).
		Build()

	set, err := r.client.Do(ctx, cmd).AsBool()
	if err != nil && !rueidis.IsRedisNil(err) {
		return fmt.Errorf("could not buffer response: %w", err)
	}

	if !set {
		return ErrResponseAlreadyBuffered
	}

	return nil
}

// GetResponse retrieves the response for a given request, if exists. Otherwise, the response will be nil.
func (r *redisConnectionStateManager) GetResponse(ctx context.Context, envID uuid.UUID, requestID string) (*connpb.SDKResponse, error) {

	cmd := r.client.
		B().
		Get().
		Key(r.keyBufferedResponse(envID, requestID)).
		Build()

	res, err := r.client.Do(ctx, cmd).ToString()
	if err != nil && !rueidis.IsRedisNil(err) {
		return nil, fmt.Errorf("could not buffer response: %w", err)
	}

	if rueidis.IsRedisNil(err) {
		return nil, nil
	}

	reply := &connpb.SDKResponse{}
	if err := proto.Unmarshal([]byte(res), reply); err != nil {
		return nil, fmt.Errorf("could not unmarshal sdk response: %w", err)
	}

	return reply, nil
}

// DeleteResponse is an idempotent delete operation for the temporary response buffer.
func (r *redisConnectionStateManager) DeleteResponse(ctx context.Context, envID uuid.UUID, requestID string) error {
	cmd := r.client.B().Del().Key(r.keyBufferedResponse(envID, requestID)).Build()

	err := r.client.Do(ctx, cmd).Error()
	if err != nil && !rueidis.IsRedisNil(err) {
		return fmt.Errorf("could not delete buffered response: %w", err)
	}

	return nil
}
