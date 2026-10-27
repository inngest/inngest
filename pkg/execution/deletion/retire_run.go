package deletion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
)

var ErrRunRetirementUnsafe = errors.New("maintenance run retirement is unsafe")

// RunRetirement is an inventoried, single-job maintenance run. This is NOT
// cancellation: it emits no finish events and never releases a semaphore.
// The caller must retain the manifest, leave obsolete semaphore counters alone,
// and arrange any required history update or replacement work.
type RunRetirement struct {
	ID                      sv2.ID
	JobID                   string
	Before                  time.Time
	RetiredSemaphoreID      string
	RequireRetiredSemaphore bool
	DryRun                  bool
}

type RunRetirementResult struct {
	Metadata   *sv2.Metadata    `json:"metadata,omitempty"`
	Item       *queue.QueueItem `json:"item,omitempty"`
	InputFound bool             `json:"input_found"`
}

// RetireMaintenanceRun removes one obsolete maintenance job and its run state.
// guard must verify an external hold, no executing work or capacity leases, and
// that the current function no longer acquires RetiredSemaphoreID. Keep the hold
// until this call finishes. Never use this API for ordinary customer runs.
// Failed/uncertain reads refuse cleanup; partial deletion is safe to retry with
// the same manifest. Missing metadata/input are allowed, not fabricated.
func RetireMaintenanceRun(ctx context.Context, shard queue.QueueShard, store sv2.RunService, req RunRetirement, guard func(context.Context) error) (*RunRetirementResult, error) {
	id := req.ID
	if shard == nil || store == nil || guard == nil || req.JobID == "" || id.RunID.IsZero() || id.FunctionID == uuid.Nil || id.Tenant.AccountID == uuid.Nil || id.Tenant.EnvID == uuid.Nil || id.Tenant.AppID == uuid.Nil || req.Before.IsZero() || id.RunID.Timestamp().After(req.Before) {
		return nil, fmt.Errorf("%w: incomplete or out-of-window manifest", ErrRunRetirementUnsafe)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := guard(ctx); err != nil {
		return nil, err
	}
	scope := queue.Scope{AccountID: id.Tenant.AccountID, EnvID: id.Tenant.EnvID, FunctionID: id.FunctionID}
	result := &RunRetirementResult{}
	md, err := store.LoadMetadata(ctx, id)
	if err == nil {
		if md.ID != id {
			return nil, fmt.Errorf("%w: metadata identity mismatch", ErrRunRetirementUnsafe)
		}
		result.Metadata = &md
		if err := checkRetiredSemaphores(md.Config.Semaphores, req.RetiredSemaphoreID); err != nil {
			return nil, err
		}
	} else if !metadataAbsent(err) {
		return nil, err
	}
	// Deferred work can survive partially missing metadata too. Never infer
	// that it is absent merely because the metadata record is gone.
	defers, err := store.LoadDefers(ctx, id)
	if err != nil && !metadataAbsent(err) {
		return nil, err
	}
	if len(defers) != 0 {
		return nil, fmt.Errorf("%w: run has deferred work", ErrRunRetirementUnsafe)
	}
	_, err = store.LoadEvents(ctx, id)
	result.InputFound = err == nil
	if err != nil && !errors.Is(err, state.ErrEventNotFound) && !metadataAbsent(err) {
		return nil, err
	}
	item, err := shard.LoadQueueItem(ctx, req.JobID)
	if err == nil {
		if item == nil || item.ID != req.JobID || item.FunctionID != id.FunctionID || sv2.IDFromV1(item.Data.Identifier) != id || item.LeaseID != nil || item.CapacityLease != nil || item.QueueName != nil || item.ID == queue.JobIDFromContext(ctx) {
			return nil, fmt.Errorf("%w: wrong identity or leased/executing job", ErrRunRetirementUnsafe)
		}
		switch item.Data.Kind {
		case queue.KindStart, queue.KindEdge, queue.KindEdgeError, queue.KindSleep:
		default:
			return nil, fmt.Errorf("%w: unsupported job kind", ErrRunRetirementUnsafe)
		}
		if err := checkRetiredSemaphores(item.Data.Semaphores, req.RetiredSemaphoreID); err != nil {
			return nil, err
		}
		if err := checkRetiredSemaphores(item.Data.Identifier.Semaphores, req.RetiredSemaphoreID); err != nil {
			return nil, err
		}
		result.Item = item
	} else if !errors.Is(err, queue.ErrQueueItemNotFound) {
		return nil, err
	}
	count, err := shard.OutstandingJobCount(ctx, scope, id.RunID)
	if err != nil {
		return nil, err
	}
	if count < 0 || count > 1 || (count == 1 && result.Item == nil) {
		return nil, fmt.Errorf("%w: ambiguous run index", ErrRunRetirementUnsafe)
	}
	if count == 1 {
		jobs, err := shard.RunJobs(ctx, scope, id.RunID, 2, 0)
		if err != nil {
			return nil, err
		}
		if len(jobs) != 1 || jobs[0].JobID != req.JobID {
			return nil, fmt.Errorf("%w: run index does not match manifest", ErrRunRetirementUnsafe)
		}
	}
	if result.Metadata == nil && result.Item == nil && !result.InputFound {
		return result, nil // idempotent absence is not a new cancellation
	}
	if req.RequireRetiredSemaphore {
		var semaphores []constraintapi.Semaphore
		if result.Metadata != nil {
			semaphores = append(semaphores, result.Metadata.Config.Semaphores...)
		}
		if result.Item != nil {
			semaphores = append(semaphores, result.Item.Data.Semaphores...)
			semaphores = append(semaphores, result.Item.Data.Identifier.Semaphores...)
		}
		found := false
		for _, sem := range semaphores {
			found = found || (req.RetiredSemaphoreID != "" && sem.ID == req.RetiredSemaphoreID && sem.Release == constraintapi.SemaphoreReleaseManual)
		}
		if !found {
			return nil, fmt.Errorf("%w: no retired manual semaphore evidence", ErrRunRetirementUnsafe)
		}
	}
	maintenance, ok := shard.(queue.MaintenanceShard)
	if result.Item != nil && !ok {
		return nil, fmt.Errorf("%w: shard lacks atomic maintenance dequeue", ErrRunRetirementUnsafe)
	}
	// Inspect first; no permit-release or ordinary finalization path is called.
	if req.DryRun {
		return result, nil
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	if result.Item != nil {
		// A vanished/changed record is not proof that this caller retired it.
		if err := maintenance.DequeueMaintenance(ctx, *result.Item); err != nil {
			return nil, err
		}
	}
	remaining, err := shard.OutstandingJobCount(ctx, scope, id.RunID)
	if err != nil {
		return nil, err
	}
	if remaining != 0 {
		return nil, fmt.Errorf("%w: run index remains", ErrRunRetirementUnsafe)
	}
	if _, err := shard.LoadQueueItem(ctx, req.JobID); !errors.Is(err, queue.ErrQueueItemNotFound) {
		return nil, fmt.Errorf("%w: queue record remains or cannot be checked: %v", ErrRunRetirementUnsafe, err)
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	if err := store.Delete(ctx, id); err != nil && !metadataAbsent(err) {
		return nil, err
	}
	if _, err := store.LoadMetadata(ctx, id); !metadataAbsent(err) {
		return nil, fmt.Errorf("%w: metadata remains or cannot be checked: %v", ErrRunRetirementUnsafe, err)
	}
	if _, err := store.LoadEvents(ctx, id); !errors.Is(err, state.ErrEventNotFound) && !metadataAbsent(err) {
		return nil, fmt.Errorf("%w: input remains or cannot be checked: %v", ErrRunRetirementUnsafe, err)
	}
	return result, nil
}

func metadataAbsent(err error) bool {
	return errors.Is(err, sv2.ErrMetadataNotFound) || errors.Is(err, state.ErrRunNotFound)
}

func checkRetiredSemaphores(semaphores []constraintapi.Semaphore, retired string) error {
	for _, sem := range semaphores {
		if sem.Release == constraintapi.SemaphoreReleaseManual && (retired == "" || sem.ID != retired) {
			return fmt.Errorf("%w: manual semaphore is not retired", ErrRunRetirementUnsafe)
		}
	}
	return nil
}
