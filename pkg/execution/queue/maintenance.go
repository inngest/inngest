package queue

import "context"

// MaintenanceShard is an optional, fail-closed extension for offline retirement.
// Status indexes and best-effort backlog readers are NOT substitutes for it.
type MaintenanceShard interface {
	MaintenanceLeaseCount(context.Context, Scope) (int64, error)
	// DequeueMaintenance atomically refuses a changed item or ANY execution or
	// capacity lease, including expired leases. Normal Dequeue is not safe here.
	DequeueMaintenance(context.Context, QueueItem) error
}
