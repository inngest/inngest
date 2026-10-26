package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
)

var ErrCancellationCleanupUnverified = errors.New("cancellation cleanup could not be verified")

// Implemented by ownership-aware semaphore managers. A separate interface keeps
// old managers source-compatible, but they cannot silently service strict work.
type ownedSemaphoreReleaser interface {
	ReleaseOwnedSemaphore(context.Context, uuid.UUID, string, string, string, int64) error
}

// prepareVerifiedCancellation is opt-in; it requires the operator to hold the
// target function externally. It proves removal of indexed and inventoried jobs,
// not the absence of unknown raw queue records outside every index.
func (e *executor) prepareVerifiedCancellation(ctx context.Context, opts execution.FinalizeOpts) error {
	if !opts.Optional.Cancel || len(opts.Optional.KnownQueueItemIDs) > 1000 {
		return fmt.Errorf("%w: invalid bounded cancellation request", ErrCancellationCleanupUnverified)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, sem := range opts.Metadata.Config.Semaphores {
		if sem.Release != constraintapi.SemaphoreReleaseManual {
			continue
		}
		manager, ok := e.semaphoreManager.(ownedSemaphoreReleaser)
		if !ok {
			return fmt.Errorf("%w: ownership-aware semaphore manager required", ErrCancellationCleanupUnverified)
		}
		if err := manager.ReleaseOwnedSemaphore(ctx, opts.Metadata.ID.Tenant.AccountID, sem.ID, sem.EvaluatedKeyHash, opts.Metadata.ID.RunID.String(), sem.Weight); err != nil {
			return fmt.Errorf("%w: release permit: %w", ErrCancellationCleanupUnverified, err)
		}
	}
	return e.removeVerifiedRunJobs(ctx, opts)
}

func (e *executor) removeVerifiedRunJobs(ctx context.Context, opts execution.FinalizeOpts) error {
	id := opts.Metadata.ID
	scope := queue.Scope{AccountID: id.Tenant.AccountID, EnvID: id.Tenant.EnvID, FunctionID: id.FunctionID}
	shard, err := e.shards.Resolve(ctx, scope, nil)
	if err != nil {
		return fmt.Errorf("resolve cleanup shard: %w", err)
	}
	skippedCurrent := false
	remove := func(item *queue.QueueItem) error {
		if item == nil || item.Data.Identifier.RunID != id.RunID || item.Data.Identifier.AccountID != id.Tenant.AccountID || item.Data.Identifier.WorkspaceID != id.Tenant.EnvID || item.FunctionID != id.FunctionID {
			return fmt.Errorf("%w: queue item identity mismatch", ErrCancellationCleanupUnverified)
		}
		if current := queue.JobIDFromContext(ctx); current != "" && item.ID == current {
			skippedCurrent = true
			return nil
		}
		err := e.queue.Dequeue(ctx, shard.Name(), *item)
		if err != nil && !errors.Is(err, queue.ErrQueueItemNotFound) {
			return fmt.Errorf("dequeue cleanup item: %w", err)
		}
		return nil
	}
	for _, itemID := range opts.Optional.KnownQueueItemIDs {
		item, err := e.queue.LoadQueueItem(ctx, shard.Name(), itemID)
		if errors.Is(err, queue.ErrQueueItemNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if err := remove(item); err != nil {
			return err
		}
	}
	// Bounded drain, not three best-effort scans. A non-empty or unreadable
	// index is an error, and metadata is not deleted on that error.
	for range 32 {
		if err := ctx.Err(); err != nil {
			return err
		}
		jobs, err := e.queue.RunJobs(ctx, shard.Name(), scope, id.RunID, 1000, 0)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			item, _ := job.Raw.(*queue.QueueItem)
			if err := remove(item); err != nil {
				return err
			}
		}
		remaining, err := shard.OutstandingJobCount(ctx, scope, id.RunID)
		if err != nil {
			return err
		}
		allowed := 0
		if skippedCurrent {
			allowed = 1
		}
		if remaining <= allowed {
			if remaining > 0 {
				last, err := e.queue.RunJobs(ctx, shard.Name(), scope, id.RunID, 2, 0)
				if err != nil {
					return err
				}
				if len(last) != 1 {
					return ErrCancellationCleanupUnverified
				}
				item, _ := last[0].Raw.(*queue.QueueItem)
				if item == nil || item.ID != queue.JobIDFromContext(ctx) {
					return ErrCancellationCleanupUnverified
				}
			}
			// Independently re-read inventoried records, including orphans that
			// never appeared in RunJobs. A response alone is not verification.
			for _, itemID := range opts.Optional.KnownQueueItemIDs {
				if skippedCurrent && itemID == queue.JobIDFromContext(ctx) {
					continue
				}
				_, err := e.queue.LoadQueueItem(ctx, shard.Name(), itemID)
				if !errors.Is(err, queue.ErrQueueItemNotFound) {
					return fmt.Errorf("%w: inventoried item %s remains or cannot be checked: %v", ErrCancellationCleanupUnverified, itemID, err)
				}
			}
			return nil
		}
		if len(jobs) == 0 {
			return fmt.Errorf("%w: index counts %d jobs but no readable jobs were returned", ErrCancellationCleanupUnverified, remaining)
		}
	}
	return fmt.Errorf("%w: bounded queue drain exhausted", ErrCancellationCleanupUnverified)
}
