package constraintapi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/redis/rueidis"
)

// MigrationPhase represents the current phase of a shard migration for an account.
type MigrationPhase int

const (
	MigrationPhaseIdle MigrationPhase = iota
	MigrationPhaseCopyingGCRA
	MigrationPhaseCopyingLeaseState
	MigrationPhaseConvergingDelta
	MigrationPhaseFinalSync
	MigrationPhaseSwitchingRoute
	MigrationPhaseCleanup
	MigrationPhaseComplete
	MigrationPhaseFailed
)

func (p MigrationPhase) String() string {
	switch p {
	case MigrationPhaseIdle:
		return "idle"
	case MigrationPhaseCopyingGCRA:
		return "copying_gcra"
	case MigrationPhaseCopyingLeaseState:
		return "copying_lease_state"
	case MigrationPhaseConvergingDelta:
		return "converging_delta"
	case MigrationPhaseFinalSync:
		return "final_sync"
	case MigrationPhaseSwitchingRoute:
		return "switching_route"
	case MigrationPhaseCleanup:
		return "cleanup"
	case MigrationPhaseComplete:
		return "complete"
	case MigrationPhaseFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// KeyCopier copies Redis keys from a source to a destination client.
// This interface allows different copy strategies (e.g., DUMP/RESTORE for production,
// type-aware copy for testing with miniredis).
type KeyCopier interface {
	// CopyKeys copies a batch of Redis keys from source to destination.
	// Returns the number of keys successfully copied.
	CopyKeys(ctx context.Context, src, dst rueidis.Client, keys []string, batchSize int) (int, error)
}

// MigrationRequest describes a shard migration for a single account.
type MigrationRequest struct {
	AccountID uuid.UUID

	// SourceClient is the Redis client for the source shard.
	SourceClient rueidis.Client

	// DestinationClient is the Redis client for the destination shard.
	DestinationClient rueidis.Client

	// SwitchRouting is called after all state has been copied and verified.
	// It must atomically update the routing table so the account points to the destination shard.
	// This is called during the brief full-pause window.
	SwitchRouting func(ctx context.Context, accountID uuid.UUID) error

	// Copier is the key copy strategy. Defaults to DumpRestoreCopier.
	Copier KeyCopier

	// CopyBatchSize controls how many keys are pipelined in each copy batch.
	// Defaults to 500.
	CopyBatchSize int

	// ScanCount controls the COUNT hint for SCAN operations.
	// Defaults to 1000.
	ScanCount int64

	// DeltaConvergenceThreshold is the maximum number of dirty keys allowed
	// before proceeding to the final sync phase.
	// Defaults to 200.
	DeltaConvergenceThreshold int

	// MaxDeltaPasses is the maximum number of convergent delta passes before
	// proceeding to final sync regardless of dirty key count.
	// Defaults to 5.
	MaxDeltaPasses int
}

func (r *MigrationRequest) defaults() {
	if r.Copier == nil {
		r.Copier = &DumpRestoreCopier{}
	}
	if r.CopyBatchSize == 0 {
		r.CopyBatchSize = 500
	}
	if r.ScanCount == 0 {
		r.ScanCount = 1000
	}
	if r.DeltaConvergenceThreshold == 0 {
		r.DeltaConvergenceThreshold = 200
	}
	if r.MaxDeltaPasses == 0 {
		r.MaxDeltaPasses = 5
	}
}

// MigrationResult contains the outcome of a migration.
type MigrationResult struct {
	// Phase is the final phase reached.
	Phase MigrationPhase

	// GCRAKeysCopied is the number of GCRA state keys copied.
	GCRAKeysCopied int

	// LeaseStateKeysCopied is the number of lease/concurrency state keys copied.
	LeaseStateKeysCopied int

	// DeltaKeysCopied is the total number of keys re-copied across all delta passes.
	DeltaKeysCopied int

	// DeltaPasses is the number of convergent delta passes performed.
	DeltaPasses int

	// FinalSyncKeys is the number of keys copied in the final sync.
	FinalSyncKeys int

	// TotalDuration is the total migration duration.
	TotalDuration time.Duration

	// FullPauseDuration is the duration of the brief full pause (Phase 4).
	FullPauseDuration time.Duration

	// Error is set if the migration failed.
	Error error
}

// MigrationCoordinator orchestrates the phased migration of an account's constraint
// state from one Redis shard to another.
//
// The migration follows these phases:
//  1. CopyingGCRA: Copy rate limit and throttle GCRA state (frozen because Acquires are blocked)
//  2. CopyingLeaseState: Copy lease-related keys while tracking dirty keys from Extend/Release
//  3. ConvergingDelta: Iteratively re-copy dirty keys until the delta is small
//  4. FinalSync: Brief full pause, copy remaining dirty keys, switch routing
//  5. Cleanup: Remove state from source shard
type MigrationCoordinator struct {
	mu    sync.Mutex
	phase MigrationPhase

	// tracker tracks dirty keys modified by Extend/Release during the copy phases.
	tracker *migrationDirtyTracker

	// pauseExtendRelease is set to true during the final sync phase to block Extend/Release.
	pauseExtendRelease bool
}

// NewMigrationCoordinator creates a new migration coordinator.
func NewMigrationCoordinator() *MigrationCoordinator {
	return &MigrationCoordinator{
		phase: MigrationPhaseIdle,
	}
}

// Phase returns the current migration phase.
func (mc *MigrationCoordinator) Phase() MigrationPhase {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.phase
}

// Tracker returns the dirty key tracker, or nil if not in a tracking phase.
func (mc *MigrationCoordinator) Tracker() *migrationDirtyTracker {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.tracker
}

// IsPaused returns whether all operations (including Extend/Release) should be paused.
func (mc *MigrationCoordinator) IsPaused() bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.pauseExtendRelease
}

// Migrate executes the full phased migration for an account.
// This is a long-running operation that should be called from a dedicated goroutine.
//
// The caller is responsible for:
//   - Blocking Acquire operations for the account before calling this
//   - Routing Extend/Release through the dirty key tracking hooks
//   - Calling SwitchRouting to atomically update the routing table
func (mc *MigrationCoordinator) Migrate(ctx context.Context, req MigrationRequest) MigrationResult {
	req.defaults()
	start := time.Now()

	l := logger.StdlibLogger(ctx).With(
		"account_id", req.AccountID,
		"component", "migration",
	)

	result := MigrationResult{}

	// Phase 1: Copy GCRA state
	mc.setPhase(MigrationPhaseCopyingGCRA)
	l.Info("migration phase 1: copying GCRA state")

	gcraCount, err := mc.copyGCRAState(ctx, req, req.Copier)
	if err != nil {
		return mc.fail(result, start, fmt.Errorf("phase 1 (GCRA copy) failed: %w", err))
	}
	result.GCRAKeysCopied = gcraCount
	l.Info("migration phase 1 complete", "gcra_keys", gcraCount)

	// Phase 2: Copy lease + concurrency state (with dirty tracking)
	mc.setPhase(MigrationPhaseCopyingLeaseState)
	mc.mu.Lock()
	mc.tracker = newMigrationDirtyTracker()
	mc.mu.Unlock()

	l.Info("migration phase 2: copying lease state")

	leaseCount, err := mc.copyLeaseState(ctx, req, req.Copier)
	if err != nil {
		return mc.fail(result, start, fmt.Errorf("phase 2 (lease state copy) failed: %w", err))
	}
	result.LeaseStateKeysCopied = leaseCount

	// Copy scavenger entry
	if err := mc.copyScavengerEntry(ctx, req); err != nil {
		return mc.fail(result, start, fmt.Errorf("phase 2 (scavenger entry) failed: %w", err))
	}
	l.Info("migration phase 2 complete", "lease_keys", leaseCount)

	// Phase 3: Convergent delta sync
	mc.setPhase(MigrationPhaseConvergingDelta)
	l.Info("migration phase 3: convergent delta sync")

	for pass := 0; pass < req.MaxDeltaPasses; pass++ {
		dirtyKeys := mc.tracker.DrainAndReset()
		if len(dirtyKeys) <= req.DeltaConvergenceThreshold {
			l.Info("delta converged", "pass", pass, "remaining_dirty_keys", len(dirtyKeys))
			// Put them back for final sync
			mc.tracker.MarkDirty(dirtyKeys...)
			break
		}

		l.Info("delta pass", "pass", pass, "dirty_keys", len(dirtyKeys))
		copied, err := req.Copier.CopyKeys(ctx, req.SourceClient, req.DestinationClient, dirtyKeys, req.CopyBatchSize)
		if err != nil {
			return mc.fail(result, start, fmt.Errorf("phase 3 (delta pass %d) failed: %w", pass, err))
		}
		result.DeltaKeysCopied += copied
		result.DeltaPasses++
	}

	// Phase 4: Final sync (brief full pause)
	mc.setPhase(MigrationPhaseFinalSync)
	mc.mu.Lock()
	mc.pauseExtendRelease = true
	mc.mu.Unlock()

	fullPauseStart := time.Now()
	l.Info("migration phase 4: final sync (full pause)")

	// Copy remaining dirty keys
	finalDirtyKeys := mc.tracker.DrainAndReset()
	finalCopied, err := req.Copier.CopyKeys(ctx, req.SourceClient, req.DestinationClient, finalDirtyKeys, req.CopyBatchSize)
	if err != nil {
		// On failure during final sync, we need to unpause and rollback
		mc.mu.Lock()
		mc.pauseExtendRelease = false
		mc.mu.Unlock()
		return mc.fail(result, start, fmt.Errorf("phase 4 (final sync) failed: %w", err))
	}
	result.FinalSyncKeys = finalCopied

	// Switch routing
	mc.setPhase(MigrationPhaseSwitchingRoute)
	if err := req.SwitchRouting(ctx, req.AccountID); err != nil {
		mc.mu.Lock()
		mc.pauseExtendRelease = false
		mc.mu.Unlock()
		return mc.fail(result, start, fmt.Errorf("phase 4 (routing switch) failed: %w", err))
	}

	result.FullPauseDuration = time.Since(fullPauseStart)
	l.Info("migration phase 4 complete",
		"final_sync_keys", finalCopied,
		"full_pause_duration", result.FullPauseDuration,
	)

	// Phase 5: Cleanup (background, non-critical)
	mc.setPhase(MigrationPhaseCleanup)
	mc.mu.Lock()
	mc.pauseExtendRelease = false
	mc.tracker = nil
	mc.mu.Unlock()

	l.Info("migration phase 5: cleanup")
	if err := mc.cleanupSource(ctx, req); err != nil {
		// Cleanup failure is non-critical -- state is already on destination
		l.Warn("cleanup failed (non-critical)", "error", err)
	}

	mc.setPhase(MigrationPhaseComplete)
	result.Phase = MigrationPhaseComplete
	result.TotalDuration = time.Since(start)

	l.Info("migration complete",
		"total_duration", result.TotalDuration,
		"full_pause_duration", result.FullPauseDuration,
		"gcra_keys", result.GCRAKeysCopied,
		"lease_keys", result.LeaseStateKeysCopied,
		"delta_keys", result.DeltaKeysCopied,
		"final_sync_keys", result.FinalSyncKeys,
	)

	return result
}

func (mc *MigrationCoordinator) setPhase(phase MigrationPhase) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.phase = phase
}

func (mc *MigrationCoordinator) fail(result MigrationResult, start time.Time, err error) MigrationResult {
	mc.setPhase(MigrationPhaseFailed)
	mc.mu.Lock()
	mc.tracker = nil
	mc.pauseExtendRelease = false
	mc.mu.Unlock()

	result.Phase = MigrationPhaseFailed
	result.TotalDuration = time.Since(start)
	result.Error = err
	return result
}

// copyGCRAState copies all rate limit and throttle GCRA state keys from source to destination.
// This should only be called while Acquires are blocked (GCRA state is frozen).
func (mc *MigrationCoordinator) copyGCRAState(ctx context.Context, req MigrationRequest, copier KeyCopier) (int, error) {
	patterns := AccountGCRAScanPatterns(req.AccountID)
	total := 0
	for _, pattern := range patterns {
		keys, err := scanKeys(ctx, req.SourceClient, pattern, req.ScanCount)
		if err != nil {
			return total, fmt.Errorf("scan failed for pattern %s: %w", pattern, err)
		}
		copied, err := copier.CopyKeys(ctx, req.SourceClient, req.DestinationClient, keys, req.CopyBatchSize)
		if err != nil {
			return total, err
		}
		total += copied
	}
	return total, nil
}

// copyLeaseState copies all lease-related keys (leaseq, ld:*, rs:*, concurrency:*, ik:*) from source to destination.
func (mc *MigrationCoordinator) copyLeaseState(ctx context.Context, req MigrationRequest, copier KeyCopier) (int, error) {
	patterns := AccountLeaseStateScanPatterns(req.AccountID)
	total := 0
	for _, pattern := range patterns {
		keys, err := scanKeys(ctx, req.SourceClient, pattern, req.ScanCount)
		if err != nil {
			return total, fmt.Errorf("scan failed for pattern %s: %w", pattern, err)
		}
		copied, err := copier.CopyKeys(ctx, req.SourceClient, req.DestinationClient, keys, req.CopyBatchSize)
		if err != nil {
			return total, err
		}
		total += copied
	}
	return total, nil
}

// copyScavengerEntry copies the account's entry in the scavenger shard sorted set.
func (mc *MigrationCoordinator) copyScavengerEntry(ctx context.Context, req MigrationRequest) error {
	scavengerKey := ScavengerShardKey()

	// Read the score (earliest lease expiry) from source
	scoreCmd := req.SourceClient.B().Zscore().Key(scavengerKey).Member(req.AccountID.String()).Build()
	score, err := req.SourceClient.Do(ctx, scoreCmd).AsFloat64()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// No scavenger entry -- account has no active leases
			return nil
		}
		return fmt.Errorf("failed to read scavenger score: %w", err)
	}

	// Write the entry on the destination
	addCmd := req.DestinationClient.B().Zadd().Key(scavengerKey).ScoreMember().ScoreMember(score, req.AccountID.String()).Build()
	if err := req.DestinationClient.Do(ctx, addCmd).Error(); err != nil {
		return fmt.Errorf("failed to write scavenger entry: %w", err)
	}

	return nil
}

// DumpRestoreCopier copies Redis keys using DUMP/RESTORE commands.
// This is the production implementation that works with any Redis key type
// without needing to know the type.
type DumpRestoreCopier struct{}

func (c *DumpRestoreCopier) CopyKeys(ctx context.Context, src, dst rueidis.Client, keys []string, batchSize int) (int, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	copied := 0
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		// Pipeline DUMP commands on source
		dumpCmds := make([]rueidis.Completed, len(batch))
		for j, key := range batch {
			dumpCmds[j] = src.B().Dump().Key(key).Build()
		}
		dumpResults := src.DoMulti(ctx, dumpCmds...)

		// Pipeline PTTL commands on source to preserve TTLs
		pttlCmds := make([]rueidis.Completed, len(batch))
		for j, key := range batch {
			pttlCmds[j] = src.B().Pttl().Key(key).Build()
		}
		pttlResults := src.DoMulti(ctx, pttlCmds...)

		// Pipeline RESTORE commands on destination
		restoreCmds := make([]rueidis.Completed, 0, len(batch))
		for j, key := range batch {
			dumpData, err := dumpResults[j].AsBytes()
			if err != nil {
				if rueidis.IsRedisNil(err) {
					// Key was deleted between SCAN and DUMP -- skip
					continue
				}
				return copied, fmt.Errorf("DUMP failed for key %s: %w", key, err)
			}

			pttl, err := pttlResults[j].AsInt64()
			if err != nil {
				return copied, fmt.Errorf("PTTL failed for key %s: %w", key, err)
			}
			// PTTL returns -1 for no expiry, -2 for missing key
			if pttl == -2 {
				continue // Key was deleted
			}
			if pttl < 0 {
				pttl = 0 // No expiry
			}

			restoreCmds = append(restoreCmds,
				dst.B().Restore().Key(key).Ttl(pttl).SerializedValue(rueidis.BinaryString(dumpData)).Replace().Build(),
			)
		}

		if len(restoreCmds) > 0 {
			restoreResults := dst.DoMulti(ctx, restoreCmds...)
			for j, res := range restoreResults {
				if err := res.Error(); err != nil {
					return copied, fmt.Errorf("RESTORE failed for batch item %d: %w", j, err)
				}
			}
			copied += len(restoreCmds)
		}
	}

	return copied, nil
}

// TypeAwareCopier copies Redis keys by inspecting the key type and using
// type-specific commands (GET/SET, HGETALL/HSET, ZRANGEBYSCORE/ZADD, etc.).
// This works with miniredis for testing and serves as a fallback copier.
type TypeAwareCopier struct{}

func (c *TypeAwareCopier) CopyKeys(ctx context.Context, src, dst rueidis.Client, keys []string, batchSize int) (int, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	copied := 0
	for _, key := range keys {
		// Get key type
		typeCmd := src.B().Type().Key(key).Build()
		keyType, err := src.Do(ctx, typeCmd).ToString()
		if err != nil {
			return copied, fmt.Errorf("TYPE failed for key %s: %w", key, err)
		}

		if keyType == "none" {
			continue // Key was deleted
		}

		// Get TTL
		pttlCmd := src.B().Pttl().Key(key).Build()
		pttl, err := src.Do(ctx, pttlCmd).AsInt64()
		if err != nil {
			return copied, fmt.Errorf("PTTL failed for key %s: %w", key, err)
		}

		switch keyType {
		case "string":
			getCmd := src.B().Get().Key(key).Build()
			val, err := src.Do(ctx, getCmd).ToString()
			if err != nil {
				if rueidis.IsRedisNil(err) {
					continue
				}
				return copied, fmt.Errorf("GET failed for key %s: %w", key, err)
			}
			setCmd := dst.B().Set().Key(key).Value(val).Build()
			if err := dst.Do(ctx, setCmd).Error(); err != nil {
				return copied, fmt.Errorf("SET failed for key %s: %w", key, err)
			}

		case "hash":
			hgetallCmd := src.B().Hgetall().Key(key).Build()
			fields, err := src.Do(ctx, hgetallCmd).AsStrMap()
			if err != nil {
				return copied, fmt.Errorf("HGETALL failed for key %s: %w", key, err)
			}
			if len(fields) == 0 {
				continue
			}
			hsetArgs := make([]string, 0, len(fields)*2)
			for field, val := range fields {
				hsetArgs = append(hsetArgs, field, val)
			}
			// Use HSET with field-value pairs
			hsetCmd := dst.B().Hset().Key(key).FieldValue()
			for field, val := range fields {
				hsetCmd = hsetCmd.FieldValue(field, val)
			}
			if err := dst.Do(ctx, hsetCmd.Build()).Error(); err != nil {
				return copied, fmt.Errorf("HSET failed for key %s: %w", key, err)
			}

		case "zset":
			zrangeCmd := src.B().Zrange().Key(key).Min("0").Max("-1").Withscores().Build()
			members, err := src.Do(ctx, zrangeCmd).AsZScores()
			if err != nil {
				return copied, fmt.Errorf("ZRANGE failed for key %s: %w", key, err)
			}
			if len(members) == 0 {
				continue
			}
			zaddCmd := dst.B().Zadd().Key(key).ScoreMember()
			for _, m := range members {
				zaddCmd = zaddCmd.ScoreMember(m.Score, m.Member)
			}
			if err := dst.Do(ctx, zaddCmd.Build()).Error(); err != nil {
				return copied, fmt.Errorf("ZADD failed for key %s: %w", key, err)
			}

		default:
			return copied, fmt.Errorf("unsupported key type %q for key %s", keyType, key)
		}

		// Set TTL if applicable
		if pttl > 0 {
			pexpireCmd := dst.B().Pexpire().Key(key).Milliseconds(pttl).Build()
			if err := dst.Do(ctx, pexpireCmd).Error(); err != nil {
				return copied, fmt.Errorf("PEXPIRE failed for key %s: %w", key, err)
			}
		}

		copied++
	}

	return copied, nil
}

// cleanupSource removes all account keys from the source shard after migration.
func (mc *MigrationCoordinator) cleanupSource(ctx context.Context, req MigrationRequest) error {
	// Remove scavenger entry
	scavengerKey := ScavengerShardKey()
	remCmd := req.SourceClient.B().Zrem().Key(scavengerKey).Member(req.AccountID.String()).Build()
	if err := req.SourceClient.Do(ctx, remCmd).Error(); err != nil {
		return fmt.Errorf("failed to remove scavenger entry: %w", err)
	}

	// Enumerate and delete all account keys using UNLINK (async delete)
	pattern := AccountKeyScanPattern(req.AccountID)
	keys, err := scanKeys(ctx, req.SourceClient, pattern, req.ScanCount)
	if err != nil {
		return fmt.Errorf("scan failed during cleanup: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete in batches using UNLINK for async deletion
	batchSize := 1000
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]
		unlinkCmd := req.SourceClient.B().Unlink().Key(batch...).Build()
		if err := req.SourceClient.Do(ctx, unlinkCmd).Error(); err != nil {
			return fmt.Errorf("UNLINK failed: %w", err)
		}
	}

	return nil
}

// scanKeys uses SCAN to enumerate all keys matching a pattern on a Redis client.
func scanKeys(ctx context.Context, client rueidis.Client, pattern string, count int64) ([]string, error) {
	var allKeys []string
	var cursor uint64

	for {
		scanCmd := client.B().Scan().Cursor(cursor).Match(pattern).Count(count).Build()
		res, err := client.Do(ctx, scanCmd).AsScanEntry()
		if err != nil {
			return nil, fmt.Errorf("SCAN failed: %w", err)
		}

		allKeys = append(allKeys, res.Elements...)

		cursor = res.Cursor
		if cursor == 0 {
			break
		}
	}

	return allKeys, nil
}
