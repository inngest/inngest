package capacity

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type CapacityKind string

const (
	// Pre-scheduling capacity is enforced before a run is scheduled and controls run scheduling.
	CapacityKindRateLimit CapacityKind = "rate-limit"

	// Post-scheduling capacity is enforced after a run is scheduled and controls step execution.
	CapacityKindConcurrency CapacityKind = "concurrency"
	CapacityKindThrottle    CapacityKind = "throttle"
)

type ConcurrencyKey struct {
	Mode       string    `json:"mode,omitempty"`
	Scope      string    `json:"scope,omitempty"`
	AccountID  uuid.UUID `json:"aID,omitempty"`
	EnvID      uuid.UUID `json:"eID,omitempty"`
	FunctionID uuid.UUID `json:"fnID,omitempty"`
	// Key represents the evaluated key.
	Key            string `json:"key,omitempty"`
	ExpressionHash string `json:"keh,omitempty"`
}

type ThrottleKey struct {
	Key            string `json:"key,omitempty"`
	ExpressionHash string `json:"keh,omitempty"`
}

type CapacityIdentifier struct {
	AccountID  uuid.UUID `json:"aID,omitempty"`
	FunctionID uuid.UUID `json:"fnID,omitempty"`
}

type CapacityLeaseRequest struct {
	// IdempotencyKey provides an idempotency identifier. If a request for this identifier was
	// recently processed, the cached response will be returned.
	IdempotencyKey string `json:"ik,omitempty"`

	// Duration specifies the period the claimed lease should be valid for.
	Duration time.Duration

	// Capacity specifies one or more capacity requests associated with this lease.
	Capacity []CapacityResource
}

type CapacityResource struct {
	// Kind specifies the constraint represented by this resource.
	Kind CapacityKind `json:"kind"`

	Identifier            CapacityIdentifier `json:"ci,omitzero"`
	CustomConcurrencyKeys []ConcurrencyKey   `json:"ck,omitempty"`
	Throttle              *ThrottleKey       `json:"tk,omitempty"`

	Amount int `json:"amount,omitempty"`
}

type LeaseCapacityResponse struct {
	LeaseID ulid.ULID `json:"lID,omitempty"`
	Allowed int       `json:"allowed,omitempty"`
}

type CapacityManager interface {
	// LeaseCapacity attempts to lease capacity.
	LeaseCapacity(ctx context.Context, req CapacityLeaseRequest) (*LeaseCapacityResponse, error)

	// ExtendCapacityLease renews a capacity lease, extending the time capacity remains claimed.
	ExtendCapacityLease(ctx context.Context, leaseID ulid.ULID, duration time.Duration) (ulid.ULID, error)

	// ReleaseCapacity returns an existing lease before it expires, gracefully releasing all claimed resources.
	ReleaseCapacity(ctx context.Context, leaseID ulid.ULID) error
}
