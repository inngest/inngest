package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/redis/rueidis"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// activeRunEntry is the JSON-serialized member stored in the ActiveRuns sorted set.
type activeRunEntry struct {
	RunID       string `json:"r"`
	FunctionID  string `json:"f"`
	AccountID   string `json:"a"`
	WorkspaceID string `json:"w"`
	AppID       string `json:"p"`
}

func marshalActiveRunEntry(info osqueue.StaleRunInfo) (string, error) {
	entry := activeRunEntry{
		RunID:       info.RunID.String(),
		FunctionID:  info.FunctionID.String(),
		AccountID:   info.AccountID.String(),
		WorkspaceID: info.WorkspaceID.String(),
		AppID:       info.AppID.String(),
	}
	byt, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("error marshalling active run entry: %w", err)
	}
	return string(byt), nil
}

func unmarshalActiveRunEntry(member string) (osqueue.StaleRunInfo, error) {
	entry := activeRunEntry{}
	if err := json.Unmarshal([]byte(member), &entry); err != nil {
		return osqueue.StaleRunInfo{}, fmt.Errorf("error unmarshalling active run entry: %w", err)
	}

	runID, err := ulid.Parse(entry.RunID)
	if err != nil {
		return osqueue.StaleRunInfo{}, fmt.Errorf("error parsing run ID: %w", err)
	}
	fnID, err := uuid.Parse(entry.FunctionID)
	if err != nil {
		return osqueue.StaleRunInfo{}, fmt.Errorf("error parsing function ID: %w", err)
	}
	acctID, err := uuid.Parse(entry.AccountID)
	if err != nil {
		return osqueue.StaleRunInfo{}, fmt.Errorf("error parsing account ID: %w", err)
	}
	wsID, err := uuid.Parse(entry.WorkspaceID)
	if err != nil {
		return osqueue.StaleRunInfo{}, fmt.Errorf("error parsing workspace ID: %w", err)
	}
	appID, err := uuid.Parse(entry.AppID)
	if err != nil {
		return osqueue.StaleRunInfo{}, fmt.Errorf("error parsing app ID: %w", err)
	}

	return osqueue.StaleRunInfo{
		RunID:       runID,
		FunctionID:  fnID,
		AccountID:   acctID,
		WorkspaceID: wsID,
		AppID:       appID,
	}, nil
}

// TrackActiveRun adds a run to the active runs sorted set, scored by its start time.
func (q *queue) TrackActiveRun(ctx context.Context, info osqueue.StaleRunInfo, startTime time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "TrackActiveRun"), redis_telemetry.ScopeQueue)

	member, err := marshalActiveRunEntry(info)
	if err != nil {
		return err
	}

	kg := q.RedisClient.KeyGenerator()
	cmd := q.RedisClient.unshardedRc.B().Zadd().
		Key(kg.ActiveRuns()).
		ScoreMember().
		ScoreMember(float64(startTime.UnixMilli()), member).
		Build()

	return q.RedisClient.unshardedRc.Do(ctx, cmd).Error()
}

// RemoveActiveRun removes a run from the active runs sorted set.
func (q *queue) RemoveActiveRun(ctx context.Context, info osqueue.StaleRunInfo) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "RemoveActiveRun"), redis_telemetry.ScopeQueue)

	member, err := marshalActiveRunEntry(info)
	if err != nil {
		return err
	}

	kg := q.RedisClient.KeyGenerator()
	cmd := q.RedisClient.unshardedRc.B().Zrem().
		Key(kg.ActiveRuns()).
		Member(member).
		Build()

	return q.RedisClient.unshardedRc.Do(ctx, cmd).Error()
}

// ScavengeStaleRuns finds runs in the active runs index that have been running
// longer than the threshold and have no outstanding queue items. These are
// orphaned runs caused by lost events during rolling deployments.
//
// It also detects runs stuck on invoke timeouts: runs whose only outstanding
// queue items are KindPause jobs for invoke steps (function.finished events
// lost during pod shutdown). These use a longer threshold
// (StaleInvokeRecoveryThreshold) to avoid interfering with normal invocations.
func (q *queue) ScavengeStaleRuns(ctx context.Context, threshold time.Duration) ([]osqueue.StaleRunInfo, error) {
	l := logger.StdlibLogger(ctx)

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ScavengeStaleRuns"), redis_telemetry.ScopeQueue)

	client := q.RedisClient.unshardedRc
	kg := q.RedisClient.KeyGenerator()

	// Find all runs that started before (now - threshold).
	cutoff := fmt.Sprintf("%d", q.Clock.Now().Add(-threshold).UnixMilli())

	cmd := client.B().Zrangebyscore().
		Key(kg.ActiveRuns()).
		Min("-inf").
		Max(cutoff).
		Limit(0, int64(consts.StaleRunScavengerBatchSize)).
		Build()

	members, err := client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error scanning active runs: %w", err)
	}

	if len(members) == 0 {
		return nil, nil
	}

	invokeRecoveryCutoff := q.Clock.Now().Add(-consts.StaleInvokeRecoveryThreshold)

	var staleRuns []osqueue.StaleRunInfo

	for _, member := range members {
		info, err := unmarshalActiveRunEntry(member)
		if err != nil {
			l.Error("error parsing active run entry, removing", "error", err, "member", member)
			// Remove malformed entry
			client.Do(ctx, client.B().Zrem().Key(kg.ActiveRuns()).Member(member).Build())
			continue
		}

		// Check if the run has outstanding queue items.
		outstandingCount, err := q.OutstandingJobCount(ctx, info.WorkspaceID, info.FunctionID, info.RunID)
		if err != nil {
			l.Error("error checking outstanding job count for stale run candidate",
				"error", err,
				"run_id", info.RunID.String(),
			)
			continue
		}

		if outstandingCount == 0 {
			// No outstanding queue items and older than threshold — stale.
			l.Warn("detected stale run candidate",
				"run_id", info.RunID.String(),
				"function_id", info.FunctionID.String(),
				"account_id", info.AccountID.String(),
			)
			staleRuns = append(staleRuns, info)
			continue
		}

		// Run has outstanding items. Check if it's stuck on invoke timeouts
		// (child completed but function.finished event was lost during deployment).
		// Use the ULID-embedded timestamp as the run start time.
		//
		// NOTE: This heuristic cannot distinguish a parent whose child finished
		// (but the event was lost) from a parent whose child is legitimately
		// still running — both have only invoke timeout jobs. The generous
		// StaleInvokeRecoveryThreshold (1 hour) minimises false positives.
		// A future improvement could verify child completion via the CQRS
		// layer before cancelling.
		// Skip high-cardinality runs (>100 items) since RunJobs uses ZSCAN
		// with COUNT=100 and we need a complete view. OutstandingJobCount
		// (ZCARD) and RunJobs (workspace-filtered ZSCAN) can also diverge
		// for stale cross-workspace index entries, so we only proceed when
		// the item count is small enough to fetch in a single page.
		runStart := time.UnixMilli(int64(info.RunID.Time()))
		if outstandingCount > 100 {
			continue
		}
		if !runStart.Before(invokeRecoveryCutoff) {
			continue
		}
		if !q.hasOnlyInvokeTimeoutJobs(ctx, info) {
			continue
		}
		l.Warn("detected run stuck on invoke timeout (child likely completed but event was lost)",
			"run_id", info.RunID.String(),
			"function_id", info.FunctionID.String(),
			"account_id", info.AccountID.String(),
			"outstanding_count", outstandingCount,
		)
		staleRuns = append(staleRuns, info)
	}

	return staleRuns, nil
}

// hasOnlyInvokeTimeoutJobs checks whether all outstanding queue items for a run
// are KindPause timeout jobs for invoke steps (identified by having an
// InvokeCorrelationID on the embedded pause). This detects the case where a
// child function completed but the function.finished event was lost during a
// rolling deployment, leaving the parent stuck on the invoke pause.
//
// The caller must ensure outstandingCount <= 100 before calling this method
// so that RunJobs can fetch a complete view in a single ZSCAN page.
func (q *queue) hasOnlyInvokeTimeoutJobs(ctx context.Context, info osqueue.StaleRunInfo) bool {
	jobs, err := q.RunJobs(ctx, info.WorkspaceID, info.FunctionID, info.RunID, 100, 0)
	if err != nil || len(jobs) == 0 {
		return false
	}

	for _, job := range jobs {
		if !isInvokePauseTimeoutJob(job) {
			return false
		}
	}

	return true
}

// isInvokePauseTimeoutJob returns true if a job is a KindPause timeout for an
// invoke step (has a non-nil InvokeCorrelationID on the embedded pause).
func isInvokePauseTimeoutJob(job osqueue.JobResponse) bool {
	if job.Kind != osqueue.KindPause {
		return false
	}
	qi, ok := job.Raw.(*osqueue.QueueItem)
	if !ok {
		return false
	}
	pt, ok := qi.Data.Payload.(osqueue.PayloadPauseTimeout)
	return ok && pt.Pause.InvokeCorrelationID != nil
}
