package batch

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

// MigrationMode controls how a migratingBatchManager routes operations between
// the current and next BatchManager instances during a cluster migration.
type MigrationMode int

const (
	// MigrationModeCurrentOnly routes all reads and writes to the current cluster.
	// This is the default mode used before migration starts.
	MigrationModeCurrentOnly MigrationMode = iota

	// MigrationModeDualRead writes to the current cluster but reads from both,
	// checking the next cluster first. This allows instances that have already
	// switched to writing to the next cluster to have their data found.
	MigrationModeDualRead

	// MigrationModeWriteToNext writes to the next cluster and reads from both,
	// checking the next cluster first. Old batches on the current cluster remain
	// readable until they are processed.
	MigrationModeWriteToNext
)

// MigrationModeFunc returns the current migration mode. It is called on every
// operation so the mode can change at runtime (e.g. via feature flag).
type MigrationModeFunc func(ctx context.Context) MigrationMode

// MigratingBatchManagerOpt configures a migratingBatchManager.
type MigratingBatchManagerOpt func(m *migratingBatchManager)

// WithMigratingLogger sets the logger for migration-related log messages.
func WithMigratingLogger(l logger.Logger) MigratingBatchManagerOpt {
	return func(m *migratingBatchManager) {
		m.log = l
	}
}

// NewMigratingBatchManager creates a BatchManager that delegates to two underlying
// managers based on a runtime migration mode.
//
// If next is nil or mode is nil, current is returned directly with zero overhead.
func NewMigratingBatchManager(current, next BatchManager, mode MigrationModeFunc, opts ...MigratingBatchManagerOpt) BatchManager {
	if next == nil || mode == nil {
		return current
	}

	m := &migratingBatchManager{
		current: current,
		next:    next,
		mode:    mode,
		log:     logger.StdlibLogger(context.Background()),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type migratingBatchManager struct {
	current BatchManager
	next    BatchManager
	mode    MigrationModeFunc
	log     logger.Logger
}

// writeManager returns the manager that should receive writes for the current mode.
func (m *migratingBatchManager) writeManager(ctx context.Context) BatchManager {
	if m.mode(ctx) == MigrationModeWriteToNext {
		return m.next
	}
	return m.current
}

// isDualRead returns true when reads should check both clusters.
func (m *migratingBatchManager) isDualRead(ctx context.Context) bool {
	mode := m.mode(ctx)
	return mode == MigrationModeDualRead || mode == MigrationModeWriteToNext
}

func (m *migratingBatchManager) Append(ctx context.Context, bi BatchItem, fn inngest.Function) (*BatchAppendResult, error) {
	return m.writeManager(ctx).Append(ctx, bi, fn)
}

func (m *migratingBatchManager) BulkAppend(ctx context.Context, items []BatchItem, fn inngest.Function) (*BulkAppendResult, error) {
	return m.writeManager(ctx).BulkAppend(ctx, items, fn)
}

func (m *migratingBatchManager) StartExecution(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID, batchPointer string) (string, error) {
	if !m.isDualRead(ctx) {
		return m.current.StartExecution(ctx, functionID, batchID, batchPointer)
	}

	result, err := m.next.StartExecution(ctx, functionID, batchID, batchPointer)
	if err == nil && result != enums.BatchStatusAbsent.String() {
		return result, nil
	}
	if err != nil {
		m.log.WarnContext(ctx, "migrating batch: next StartExecution failed, falling back to current",
			"error", err,
		)
	}
	return m.current.StartExecution(ctx, functionID, batchID, batchPointer)
}

func (m *migratingBatchManager) RetrieveItems(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID) ([]BatchItem, error) {
	if !m.isDualRead(ctx) {
		return m.current.RetrieveItems(ctx, functionID, batchID)
	}

	items, err := m.next.RetrieveItems(ctx, functionID, batchID)
	if err == nil && len(items) > 0 {
		return items, nil
	}
	if err != nil {
		m.log.WarnContext(ctx, "migrating batch: next RetrieveItems failed, falling back to current",
			"error", err,
		)
	}
	return m.current.RetrieveItems(ctx, functionID, batchID)
}

func (m *migratingBatchManager) ScheduleExecution(ctx context.Context, opts ScheduleBatchOpts) error {
	return m.writeManager(ctx).ScheduleExecution(ctx, opts)
}

func (m *migratingBatchManager) DeleteKeys(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID) error {
	if !m.isDualRead(ctx) {
		return m.current.DeleteKeys(ctx, functionID, batchID)
	}

	// Delete is idempotent, so call both clusters.
	errNext := m.next.DeleteKeys(ctx, functionID, batchID)
	errCurrent := m.current.DeleteKeys(ctx, functionID, batchID)
	return errors.Join(errNext, errCurrent)
}

func (m *migratingBatchManager) GetBatchInfo(ctx context.Context, functionID uuid.UUID, batchKey string) (*BatchInfo, error) {
	if !m.isDualRead(ctx) {
		return m.current.GetBatchInfo(ctx, functionID, batchKey)
	}

	info, err := m.next.GetBatchInfo(ctx, functionID, batchKey)
	if err == nil && info != nil && info.BatchID != "" {
		return info, nil
	}
	if err != nil {
		m.log.WarnContext(ctx, "migrating batch: next GetBatchInfo failed, falling back to current",
			"error", err,
		)
	}
	return m.current.GetBatchInfo(ctx, functionID, batchKey)
}

func (m *migratingBatchManager) DeleteBatch(ctx context.Context, functionID uuid.UUID, batchKey string) (*DeleteBatchResult, error) {
	if !m.isDualRead(ctx) {
		return m.current.DeleteBatch(ctx, functionID, batchKey)
	}

	result, err := m.next.DeleteBatch(ctx, functionID, batchKey)
	if err == nil && result != nil && result.Deleted {
		return result, nil
	}
	if err != nil {
		m.log.WarnContext(ctx, "migrating batch: next DeleteBatch failed, falling back to current",
			"error", err,
		)
	}
	return m.current.DeleteBatch(ctx, functionID, batchKey)
}

func (m *migratingBatchManager) RunBatch(ctx context.Context, opts RunBatchOpts) (*RunBatchResult, error) {
	if !m.isDualRead(ctx) {
		return m.current.RunBatch(ctx, opts)
	}

	result, err := m.next.RunBatch(ctx, opts)
	if err == nil && result != nil && result.Scheduled {
		return result, nil
	}
	if err != nil {
		m.log.WarnContext(ctx, "migrating batch: next RunBatch failed, falling back to current",
			"error", err,
		)
	}
	return m.current.RunBatch(ctx, opts)
}

func (m *migratingBatchManager) Close() error {
	return errors.Join(m.current.Close(), m.next.Close())
}
