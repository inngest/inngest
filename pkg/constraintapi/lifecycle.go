package constraintapi

import (
	"context"
	"sync"
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

type ConstraintApiDebugLifecycles struct {
	AcquireCalls []OnCapacityLeaseAcquiredData
	ExtendCalls  []OnCapacityLeaseExtendedData
	ReleaseCalls []OnCapacityLeaseReleasedData
	l            sync.Mutex
}

func (c *ConstraintApiDebugLifecycles) OnCapacityLeaseAcquired(ctx context.Context, data OnCapacityLeaseAcquiredData) error {
	c.l.Lock()
	defer c.l.Unlock()
	c.AcquireCalls = append(c.AcquireCalls, data)
	return nil
}

func (c *ConstraintApiDebugLifecycles) OnCapacityLeaseExtended(ctx context.Context, data OnCapacityLeaseExtendedData) error {
	c.l.Lock()
	defer c.l.Unlock()
	c.ExtendCalls = append(c.ExtendCalls, data)
	return nil
}

func (c *ConstraintApiDebugLifecycles) OnCapacityLeaseReleased(ctx context.Context, data OnCapacityLeaseReleasedData) error {
	c.l.Lock()
	defer c.l.Unlock()
	c.ReleaseCalls = append(c.ReleaseCalls, data)
	return nil
}

func (c *ConstraintApiDebugLifecycles) Reset() {
	c.l.Lock()
	defer c.l.Unlock()
	c.AcquireCalls = nil
	c.ExtendCalls = nil
	c.ReleaseCalls = nil
}

func NewConstraintAPIDebugLifecycles() *ConstraintApiDebugLifecycles {
	return &ConstraintApiDebugLifecycles{}
}
