package queue

import (
	"context"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// MaintenanceShard is an optional, fail-closed extension for offline retirement.
// Status indexes and best-effort backlog readers are NOT substitutes for it.
type MaintenanceShard interface {
	MaintenanceLeaseCount(context.Context, Scope) (int64, error)
	// DequeueMaintenance atomically refuses a changed item or ANY execution or
	// capacity lease, including expired leases. Normal Dequeue is not safe here.
	DequeueMaintenance(context.Context, QueueItem) error
}

// MaintenanceRetirement records that this exact manifest passed inspection before
// destructive cleanup. It is not semaphore ownership evidence or run state.
type MaintenanceRetirement struct {
	AccountID          uuid.UUID `json:"account_id"`
	EnvID              uuid.UUID `json:"env_id"`
	AppID              uuid.UUID `json:"app_id"`
	FunctionID         uuid.UUID `json:"function_id"`
	RunID              ulid.ULID `json:"run_id"`
	JobID              string    `json:"job_id"`
	BeforeMS           int64     `json:"before_ms"`
	RetiredSemaphoreID string    `json:"retired_semaphore_id"`
}

// MaintenanceRetirementStore preserves inspection evidence across partial cleanup
// failures. Save/Delete must refuse a different existing manifest. A checkpoint
// is kept until queue and state absence have both been verified.
type MaintenanceRetirementStore interface {
	LoadMaintenanceRetirement(context.Context, string) (*MaintenanceRetirement, error)
	SaveMaintenanceRetirement(context.Context, MaintenanceRetirement) error
	DeleteMaintenanceRetirement(context.Context, MaintenanceRetirement) error
}
