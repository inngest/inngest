package constraintapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type OnCapacityLeaseAcquiredData struct {
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	FunctionID uuid.UUID

	Configuration ConstraintConfig
	Constraints   []ConstraintItem

	RequestedAmount int
	Duration        time.Duration
	Source          LeaseSource

	GrantedLeases       []CapacityLease
	LimitingConstraints []ConstraintItem
	FairnessReduction   int
	RetryAfter          time.Time
}

type OnCapacityLeaseExtendedData struct {
	AccountID  uuid.UUID
	OldLeaseID ulid.ULID
	NewLeaseID ulid.ULID
	Duration   time.Duration
}

type OnCapacityLeaseReleasedData struct {
	AccountID uuid.UUID
	LeaseID   ulid.ULID
}

type ConstraintAPILifecycleHooks interface {
	OnCapacityLeaseAcquired(ctx context.Context, data OnCapacityLeaseAcquiredData) error
	OnCapacityLeaseExtended(ctx context.Context, data OnCapacityLeaseExtendedData) error
	OnCapacityLeaseReleased(ctx context.Context, data OnCapacityLeaseReleasedData) error
}
