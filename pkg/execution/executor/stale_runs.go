package executor

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

// StaleRunRecovery is a self-contained coordinator that periodically scans for
// stale RUNNING runs (runs with no outstanding queue items that have been active
// longer than StaleRunThreshold). When found, these runs are cancelled via the
// executor, which triggers Finalize() to release concurrency locks and fire
// function.finished events.
//
// This handles orphaned runs caused by lost events during rolling deployments,
// where in-flight events in the in-memory pubsub are lost when pods terminate.
type StaleRunRecovery struct {
	log  logger.Logger
	q    queue.Queue
	exec execution.Executor
}

// NewStaleRunRecovery creates a new StaleRunRecovery coordinator.
func NewStaleRunRecovery(log logger.Logger, q queue.Queue, exec execution.Executor) *StaleRunRecovery {
	return &StaleRunRecovery{
		log:  log.With("component", "stale-run-recovery"),
		q:    q,
		exec: exec,
	}
}

// Run starts the stale run recovery loop. It uses ConfigLease for distributed
// coordination so that only one pod runs this across replicas.
func (r *StaleRunRecovery) Run(ctx context.Context) {
	qp, ok := r.q.(queue.QueueProcessor)
	if !ok {
		r.log.Warn("queue does not implement QueueProcessor, stale run recovery disabled")
		return
	}

	shard := qp.Shard()
	if shard == nil {
		r.log.Warn("no primary queue shard available, stale run recovery disabled")
		return
	}

	scavenger, ok := shard.(queue.StaleRunScavenger)
	if !ok {
		r.log.Warn("queue shard does not support stale run scavenging, stale run recovery disabled")
		return
	}

	leaseKey := "stale-run-recovery"
	leaseDuration := queue.ConfigLeaseDuration

	var leaseID *ulid.ULID
	for {
		var err error
		leaseID, err = shard.ConfigLease(ctx, leaseKey, leaseDuration)
		if err == nil || err == queue.ErrConfigAlreadyLeased {
			break
		}
		r.log.Error("error claiming stale run recovery lease, retrying", "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(leaseDuration):
		}
	}

	leaseTick := time.NewTicker(leaseDuration / 3)
	scavengeTick := time.NewTicker(consts.StaleRunScavengerInterval)

	defer leaseTick.Stop()
	defer scavengeTick.Stop()

	isLeaseHolder := leaseID != nil

	for {
		select {
		case <-ctx.Done():
			return
		case <-scavengeTick.C:
			if !isLeaseHolder {
				continue
			}
			r.scavenge(ctx, scavenger)
		case <-leaseTick.C:
			leaseID, isLeaseHolder = r.renewLease(ctx, shard, leaseKey, leaseDuration, leaseID)
		}
	}
}

func (r *StaleRunRecovery) renewLease(
	ctx context.Context,
	shard queue.QueueShard,
	leaseKey string,
	leaseDuration time.Duration,
	leaseID *ulid.ULID,
) (*ulid.ULID, bool) {
	newLeaseID, err := shard.ConfigLease(ctx, leaseKey, leaseDuration, leaseID)
	if err == queue.ErrConfigAlreadyLeased {
		return nil, false
	}
	if err != nil {
		r.log.Error("error renewing stale run recovery lease", "error", err)
		return nil, false
	}
	return newLeaseID, true
}

func (r *StaleRunRecovery) scavenge(ctx context.Context, scavenger queue.StaleRunScavenger) {
	staleRuns, err := scavenger.ScavengeStaleRuns(ctx, consts.StaleRunThreshold)
	if err != nil {
		r.log.Error("error scanning for stale runs", "error", err)
		return
	}

	if len(staleRuns) == 0 {
		return
	}

	r.log.Info("found stale runs to recover", "count", len(staleRuns))

	for _, run := range staleRuns {
		r.cancelRun(ctx, scavenger, run)
	}
}

func (r *StaleRunRecovery) cancelRun(ctx context.Context, scavenger queue.StaleRunScavenger, run queue.StaleRunInfo) {
	runLogger := r.log.With(
		"run_id", run.RunID.String(),
		"function_id", run.FunctionID.String(),
		"account_id", run.AccountID.String(),
	)

	id := sv2.ID{
		RunID:      run.RunID,
		FunctionID: run.FunctionID,
		Tenant: sv2.Tenant{
			AccountID: run.AccountID,
			EnvID:     run.WorkspaceID,
			AppID:     run.AppID,
		},
	}

	runLogger.Warn("attempting to cancel stale run candidate")

	if err := r.exec.Cancel(ctx, id, execution.CancelRequest{}); err != nil {
		// Cancel returns nil for already-finalized runs (metadata/event not found),
		// so a non-nil error indicates a genuine transient failure (e.g., storage error).
		// Leave the run in the ActiveRuns index so the next scavenge pass retries it.
		runLogger.Error("error cancelling stale run, will retry on next tick", "error", err)
		return
	}

	// Cancel returned nil — run is finalized (or was already gone).
	// finalizeRemoveActiveRun() inside Cancel()/Finalize() normally handles cleanup;
	// this call is a safety net for the rare case where that path failed.
	if removeErr := scavenger.RemoveActiveRun(ctx, run); removeErr != nil {
		runLogger.Error("error removing run from active runs index", "error", removeErr)
	}

	runLogger.Info("stale run cancelled and cleaned up")
}
