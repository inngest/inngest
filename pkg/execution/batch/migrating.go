package batch

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

// WriteToNextFunc reports whether an account's new batch appends go to the next
// cluster. That cluster is the account's primary, and reads check it first. It
// is called on every operation so it can change at runtime (e.g. via feature
// flag).
type WriteToNextFunc func(ctx context.Context, accountID uuid.UUID) bool

// NewMigratingBatchManager creates a BatchManager that migrates batching state
// from current to next. Each append goes to exactly one cluster, chosen per
// account by writeToNext, and reads always cover both clusters so batches drain
// from whichever cluster created them.
//
// current is required. If next is nil, current is returned directly with zero
// overhead. If writeToNext is nil, every append goes to current.
func NewMigratingBatchManager(current, next BatchManager, writeToNext WriteToNextFunc) (BatchManager, error) {
	l := logger.StdlibLogger(context.Background())
	if current == nil {
		return nil, fmt.Errorf("current batch manager is required")
	}
	if next == nil {
		l.Info("next batch manager is nil, batch migration is off; delegating all operations to current")
		return current, nil
	}

	l.Info("next batch manager is set, batch migration is on; appends go to each account's primary and reads fall back to the other cluster")
	return &migratingBatchManager{
		current:     current,
		next:        next,
		writeToNext: writeToNext,
	}, nil
}

type migratingBatchManager struct {
	current     BatchManager
	next        BatchManager
	writeToNext WriteToNextFunc
}

// writeManager returns the account's primary cluster, which receives its appends
// and is read first, and the fallback cluster that reads check when the
// primary doesn't have the batch.
func (m *migratingBatchManager) writeManager(ctx context.Context, accountID uuid.UUID) (primary, fallback BatchManager) {
	if m.writeToNext != nil && m.writeToNext(ctx, accountID) {
		return m.next, m.current
	}
	return m.current, m.next
}

func (m *migratingBatchManager) Append(ctx context.Context, bi BatchItem, fn inngest.Function) (*BatchAppendResult, error) {
	primary, _ := m.writeManager(ctx, bi.AccountID)
	return primary.Append(ctx, bi, fn)
}

// BulkAppend routes on the first item's account. The append buffer groups items
// by function, so every item in a call belongs to the same account.
func (m *migratingBatchManager) BulkAppend(ctx context.Context, items []BatchItem, fn inngest.Function) (*BulkAppendResult, error) {
	if len(items) == 0 {
		return m.current.BulkAppend(ctx, items, fn)
	}
	primary, _ := m.writeManager(ctx, items[0].AccountID)
	return primary.BulkAppend(ctx, items, fn)
}

// StartExecution checks the fallback cluster only when the primary reports the
// batch absent. Errors are returned as-is so the job retries: treating one as
// absent would let the caller delete a batch that may still exist.
func (m *migratingBatchManager) StartExecution(ctx context.Context, accountID, functionID uuid.UUID, batchID ulid.ULID, batchPointer string) (string, error) {
	primary, fallback := m.writeManager(ctx, accountID)
	result, err := primary.StartExecution(ctx, accountID, functionID, batchID, batchPointer)
	if err != nil || result != enums.BatchStatusAbsent.String() {
		return result, err
	}
	logger.StdlibLogger(ctx).Warn("batch absent in primary cluster, falling back for StartExecution",
		"account_id", accountID,
		"function_id", functionID,
		"batch_id", batchID,
	)
	return fallback.StartExecution(ctx, accountID, functionID, batchID, batchPointer)
}

// RetrieveItems checks the fallback cluster only when the primary returns no items.
// Errors are returned as-is so the job retries: an empty batch is skipped, so
// treating an error as a miss would silently drop the batch.
func (m *migratingBatchManager) RetrieveItems(ctx context.Context, accountID, functionID uuid.UUID, batchID ulid.ULID) ([]BatchItem, error) {
	primary, fallback := m.writeManager(ctx, accountID)
	items, err := primary.RetrieveItems(ctx, accountID, functionID, batchID)
	if err != nil || len(items) > 0 {
		return items, err
	}
	logger.StdlibLogger(ctx).Warn("batch empty in primary cluster, falling back for RetrieveItems",
		"account_id", accountID,
		"function_id", functionID,
		"batch_id", batchID,
	)
	return fallback.RetrieveItems(ctx, accountID, functionID, batchID)
}

// ScheduleExecution doesn't touch either store; it only enqueues the
// schedule-batch job, which finds the batch in whichever cluster holds it.
func (m *migratingBatchManager) ScheduleExecution(ctx context.Context, opts ScheduleBatchOpts) error {
	primary, _ := m.writeManager(ctx, opts.AccountID)
	return primary.ScheduleExecution(ctx, opts)
}

// DeleteKeys deletes from both clusters. Batch IDs are unique ULIDs, so
// deleting from the cluster that doesn't hold the batch is a no-op.
func (m *migratingBatchManager) DeleteKeys(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID) error {
	errNext := m.next.DeleteKeys(ctx, functionID, batchID)
	errCurrent := m.current.DeleteKeys(ctx, functionID, batchID)
	return errors.Join(errNext, errCurrent)
}

// GetBatchInfo returns the open batch behind next's pointer, falling back to
// current's when next has none.
func (m *migratingBatchManager) GetBatchInfo(ctx context.Context, functionID uuid.UUID, batchKey string) (*BatchInfo, error) {
	info, err := m.next.GetBatchInfo(ctx, functionID, batchKey)
	if err != nil || (info != nil && info.BatchID != "") {
		return info, err
	}
	// GetBatchInfo has no account to pick a primary, so it always probes next
	// first. Falling back to current is the expected path for accounts still
	// writing there, so it is not worth a warning.
	logger.StdlibLogger(ctx).Debug("no batch in next cluster, falling back to current for GetBatchInfo",
		"function_id", functionID,
		"batch_key", batchKey,
	)
	return m.current.GetBatchInfo(ctx, functionID, batchKey)
}

// DeleteBatch deletes the open batch behind each cluster's pointer. Right after
// a flag flip both clusters can hold one for the same pointer, so it never stops
// at the first. ItemCount is summed across clusters, and BatchID is next's when
// next deleted a batch, otherwise current's.
func (m *migratingBatchManager) DeleteBatch(ctx context.Context, functionID uuid.UUID, batchKey string) (*DeleteBatchResult, error) {
	resNext, errNext := m.next.DeleteBatch(ctx, functionID, batchKey)
	resCurrent, errCurrent := m.current.DeleteBatch(ctx, functionID, batchKey)

	result := &DeleteBatchResult{}
	for _, res := range []*DeleteBatchResult{resNext, resCurrent} {
		if res == nil || !res.Deleted {
			continue
		}
		if !result.Deleted {
			result.BatchID = res.BatchID
		}
		result.Deleted = true
		result.ItemCount += res.ItemCount
	}
	return result, errors.Join(errNext, errCurrent)
}

// RunBatch runs the open batch behind the primary cluster's pointer, falling
// back to the fallback cluster's when the primary has none.
func (m *migratingBatchManager) RunBatch(ctx context.Context, opts RunBatchOpts) (*RunBatchResult, error) {
	primary, fallback := m.writeManager(ctx, opts.AccountID)
	result, err := primary.RunBatch(ctx, opts)
	if err != nil || (result != nil && result.Scheduled) {
		return result, err
	}
	logger.StdlibLogger(ctx).Warn("no batch scheduled from primary cluster, falling back for RunBatch",
		"account_id", opts.AccountID,
		"function_id", opts.FunctionID,
		"batch_key", opts.BatchKey,
	)
	return fallback.RunBatch(ctx, opts)
}

func (m *migratingBatchManager) Close() error {
	return errors.Join(m.current.Close(), m.next.Close())
}
