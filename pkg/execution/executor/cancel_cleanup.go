package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
)

var ErrCancellationCleanupUnverified = errors.New("cancellation cleanup could not be verified")

// cleanupAndVerifyCancellation requires an external hold with no executing work.
// It checks indexed and inventoried records, not unknown orphans outside both.
func (e *executor) cleanupAndVerifyCancellation(ctx context.Context, opts execution.FinalizeOpts) error {
	if !opts.Optional.Cancel || len(opts.Optional.KnownQueueItemIDs) > 1000 {
		return fmt.Errorf("%w: invalid bounded cancellation request", ErrCancellationCleanupUnverified)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Phase 1: inspect the complete bounded index and inventory before mutation.
	id := opts.Metadata.ID
	scope := queue.Scope{AccountID: id.Tenant.AccountID, EnvID: id.Tenant.EnvID, FunctionID: id.FunctionID}
	shard, err := e.shards.Resolve(ctx, scope, nil)
	if err != nil {
		return fmt.Errorf("resolve cleanup shard: %w", err)
	}
	// A nil entry retains an already-absent inventoried ID for final verification.
	items := map[string]*queue.QueueItem{}
	check := func(item *queue.QueueItem) error {
		if item == nil || item.Data.Identifier.RunID != id.RunID || item.Data.Identifier.AccountID != id.Tenant.AccountID || item.Data.Identifier.WorkspaceID != id.Tenant.EnvID || item.FunctionID != id.FunctionID {
			return fmt.Errorf("%w: queue item identity mismatch", ErrCancellationCleanupUnverified)
		}
		if item.LeaseID != nil || item.ID == queue.JobIDFromContext(ctx) {
			return fmt.Errorf("%w: leased or executing item requires separate recovery", ErrCancellationCleanupUnverified)
		}
		items[item.ID] = item
		return nil
	}
	count, err := shard.OutstandingJobCount(ctx, scope, id.RunID)
	if err != nil {
		return err
	}
	if count > 32000 {
		return fmt.Errorf("%w: run exceeds bounded cleanup limit", ErrCancellationCleanupUnverified)
	}
	for offset := 0; offset < count; offset += 1000 {
		jobs, err := e.queue.RunJobs(ctx, shard.Name(), scope, id.RunID, 1000, int64(offset))
		if err != nil {
			return err
		}
		for _, job := range jobs {
			item, _ := job.Raw.(*queue.QueueItem)
			if err := check(item); err != nil {
				return err
			}
		}
	}
	if len(items) != count {
		return fmt.Errorf("%w: run index is incomplete or changed during inspection", ErrCancellationCleanupUnverified)
	}
	for _, itemID := range opts.Optional.KnownQueueItemIDs {
		item, err := e.queue.LoadQueueItem(ctx, shard.Name(), itemID)
		if errors.Is(err, queue.ErrQueueItemNotFound) {
			items[itemID] = nil
			continue
		}
		if err != nil {
			return err
		}
		if err := check(item); err != nil {
			return err
		}
	}

	// Phase 2: release proven-owned permits and remove the inspected jobs.
	for _, sem := range opts.Metadata.Config.Semaphores {
		if sem.Release != constraintapi.SemaphoreReleaseManual {
			continue
		}
		manager, ok := e.semaphoreManager.(constraintapi.OwnedSemaphoreManager)
		if !ok {
			return fmt.Errorf("%w: ownership-aware semaphore manager required", ErrCancellationCleanupUnverified)
		}
		if err := manager.ReleaseOwnedSemaphore(ctx, id.Tenant.AccountID, sem.ID, sem.EvaluatedKeyHash, id.RunID.String()); err != nil {
			return fmt.Errorf("%w: manual permit release: %w", ErrCancellationCleanupUnverified, err)
		}
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		err := e.queue.Dequeue(ctx, shard.Name(), *item)
		if err != nil && !errors.Is(err, queue.ErrQueueItemNotFound) {
			return fmt.Errorf("dequeue cleanup item: %w", err)
		}
	}

	// Phase 3: verify both the index and every indexed/inventoried raw record.
	remaining, err := shard.OutstandingJobCount(ctx, scope, id.RunID)
	if err != nil {
		return err
	}
	if remaining != 0 {
		return fmt.Errorf("%w: %d indexed jobs remain", ErrCancellationCleanupUnverified, remaining)
	}
	for itemID := range items {
		_, err := e.queue.LoadQueueItem(ctx, shard.Name(), itemID)
		if !errors.Is(err, queue.ErrQueueItemNotFound) {
			return fmt.Errorf("%w: item %s remains or cannot be checked: %v", ErrCancellationCleanupUnverified, itemID, err)
		}
	}
	return nil
}
