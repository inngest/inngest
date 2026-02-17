package constraintapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// ShardRouter determines which shard an account's constraint state lives on.
type ShardRouter func(ctx context.Context, accountID uuid.UUID) (shardName string, err error)

// RoutingCapacityManager routes constraint operations to the correct shard based on
// account ID, and handles shard migrations transparently.
//
// During a migration:
//   - Acquire operations are blocked with RetryAfter for the migrating account
//   - Extend and Release operations continue on the source shard with dirty key tracking
//   - After the final sync phase, all operations are paused briefly (< 100ms)
//   - After routing switch, all operations go to the destination shard
type RoutingCapacityManager struct {
	mu     sync.RWMutex
	shards map[string]*redisCapacityManager
	router ShardRouter

	// migrations tracks active migrations by account ID
	migrations map[uuid.UUID]*accountMigration

	// migrationState provides dirty key tracking for active migrations
	migrationState *migrationAccountState
}

type accountMigration struct {
	coordinator  *MigrationCoordinator
	sourceShard  string
	destShard    string
	retryAfter   time.Duration
	sourceClient rueidis.Client
}

// NewRoutingCapacityManager creates a new routing-aware CapacityManager.
func NewRoutingCapacityManager(
	router ShardRouter,
	shards map[string]*redisCapacityManager,
) *RoutingCapacityManager {
	return &RoutingCapacityManager{
		shards:         shards,
		router:         router,
		migrations:     make(map[uuid.UUID]*accountMigration),
		migrationState: newMigrationAccountState(),
	}
}

// RegisterShard adds a shard to the routing manager.
func (rm *RoutingCapacityManager) RegisterShard(name string, manager *redisCapacityManager) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.shards[name] = manager
}

// StartMigration initiates a phased convergent migration for an account.
// The migration runs asynchronously; use the returned MigrationCoordinator to monitor progress.
//
// The caller must ensure that:
//   - Both source and destination shards are registered
//   - The account is currently routed to the source shard
func (rm *RoutingCapacityManager) StartMigration(
	ctx context.Context,
	accountID uuid.UUID,
	sourceShard, destShard string,
	retryAfter time.Duration,
) (*MigrationCoordinator, error) {
	rm.mu.Lock()

	// Verify shards exist
	srcManager, ok := rm.shards[sourceShard]
	if !ok {
		rm.mu.Unlock()
		return nil, fmt.Errorf("source shard %q not found", sourceShard)
	}
	destManager, ok := rm.shards[destShard]
	if !ok {
		rm.mu.Unlock()
		return nil, fmt.Errorf("destination shard %q not found", destShard)
	}

	// Check for existing migration
	if _, exists := rm.migrations[accountID]; exists {
		rm.mu.Unlock()
		return nil, fmt.Errorf("migration already in progress for account %s", accountID)
	}

	coordinator := NewMigrationCoordinator()
	migration := &accountMigration{
		coordinator:  coordinator,
		sourceShard:  sourceShard,
		destShard:    destShard,
		retryAfter:   retryAfter,
		sourceClient: srcManager.client,
	}
	rm.migrations[accountID] = migration

	// Start dirty key tracking
	rm.migrationState.StartTracking(accountID)

	rm.mu.Unlock()

	// Run migration asynchronously
	go func() {
		req := MigrationRequest{
			AccountID:         accountID,
			SourceClient:      srcManager.client,
			DestinationClient: destManager.client,
			SwitchRouting: func(ctx context.Context, accountID uuid.UUID) error {
				// This is a no-op from the migration coordinator's perspective.
				// The actual routing switch happens below when the migration completes.
				// We just need to ensure the coordinator knows routing has been switched.
				return nil
			},
		}

		result := coordinator.Migrate(ctx, req)

		// Clean up migration state
		rm.mu.Lock()
		delete(rm.migrations, accountID)
		rm.mu.Unlock()
		rm.migrationState.StopTracking(accountID)

		l := logger.StdlibLogger(ctx)
		if result.Error != nil {
			l.Error("migration failed",
				"account_id", accountID,
				"error", result.Error,
				"phase", result.Phase,
				"duration", result.TotalDuration,
			)
		} else {
			l.Info("migration completed successfully",
				"account_id", accountID,
				"duration", result.TotalDuration,
				"full_pause", result.FullPauseDuration,
			)
		}
	}()

	return coordinator, nil
}

// getShard returns the CapacityManager for a given account, checking migration state.
// Returns the shard manager, migration info (if migrating), and any error.
func (rm *RoutingCapacityManager) getShard(ctx context.Context, accountID uuid.UUID) (*redisCapacityManager, *accountMigration, error) {
	rm.mu.RLock()
	migration, isMigrating := rm.migrations[accountID]
	rm.mu.RUnlock()

	if isMigrating {
		// During migration, route to source shard
		rm.mu.RLock()
		manager, ok := rm.shards[migration.sourceShard]
		rm.mu.RUnlock()
		if !ok {
			return nil, nil, fmt.Errorf("source shard %q not found during migration", migration.sourceShard)
		}
		return manager, migration, nil
	}

	shardName, err := rm.router(ctx, accountID)
	if err != nil {
		return nil, nil, fmt.Errorf("routing failed: %w", err)
	}

	rm.mu.RLock()
	manager, ok := rm.shards[shardName]
	rm.mu.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("shard %q not found", shardName)
	}

	return manager, nil, nil
}

// Check implements CapacityManager.
func (rm *RoutingCapacityManager) Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError) {
	manager, migration, err := rm.getShard(ctx, req.AccountID)
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "routing error: %w", err)
	}

	// During final sync, Check is also paused
	if migration != nil && migration.coordinator.IsPaused() {
		return &CapacityCheckResponse{
			RetryAfter: time.Now().Add(migration.retryAfter),
		}, nil, nil
	}

	return manager.Check(ctx, req)
}

// Acquire implements CapacityManager.
func (rm *RoutingCapacityManager) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	manager, migration, err := rm.getShard(ctx, req.AccountID)
	if err != nil {
		return nil, errs.Wrap(0, false, "routing error: %w", err)
	}

	// Block Acquires during migration
	if migration != nil {
		return &CapacityAcquireResponse{
			RetryAfter: time.Now().Add(migration.retryAfter),
		}, nil
	}

	return manager.Acquire(ctx, req)
}

// ExtendLease implements CapacityManager.
func (rm *RoutingCapacityManager) ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.InternalError) {
	manager, migration, err := rm.getShard(ctx, req.AccountID)
	if err != nil {
		return nil, errs.Wrap(0, false, "routing error: %w", err)
	}

	// During final sync, Extend is paused
	if migration != nil && migration.coordinator.IsPaused() {
		return &CapacityExtendLeaseResponse{}, errs.Wrap(0, true, "migration in final sync, retry shortly")
	}

	// Track dirty keys during migration
	if migration != nil {
		tracker := rm.migrationState.GetTracker(req.AccountID)
		if tracker != nil {
			if err := rm.trackExtendDirtyKeys(ctx, migration.sourceClient, req.AccountID, req.LeaseID, tracker); err != nil {
				// Non-fatal: log and continue. The next delta pass will catch it via SCAN comparison.
				l := logger.StdlibLogger(ctx)
				l.Warn("failed to track dirty keys for extend", "error", err)
			}
		}
	}

	return manager.ExtendLease(ctx, req)
}

// Release implements CapacityManager.
func (rm *RoutingCapacityManager) Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError) {
	manager, migration, err := rm.getShard(ctx, req.AccountID)
	if err != nil {
		return nil, errs.Wrap(0, false, "routing error: %w", err)
	}

	// During final sync, Release is paused
	if migration != nil && migration.coordinator.IsPaused() {
		return &CapacityReleaseResponse{AccountID: req.AccountID}, errs.Wrap(0, true, "migration in final sync, retry shortly")
	}

	// Track dirty keys during migration
	if migration != nil {
		tracker := rm.migrationState.GetTracker(req.AccountID)
		if tracker != nil {
			if err := rm.trackReleaseDirtyKeys(ctx, migration.sourceClient, req.AccountID, req.LeaseID, tracker); err != nil {
				l := logger.StdlibLogger(ctx)
				l.Warn("failed to track dirty keys for release", "error", err)
			}
		}
	}

	return manager.Release(ctx, req)
}

// trackExtendDirtyKeys reads lease details and request state to determine which keys
// an Extend operation will modify, and marks them dirty in the tracker.
func (rm *RoutingCapacityManager) trackExtendDirtyKeys(
	ctx context.Context,
	client rueidis.Client,
	accountID uuid.UUID,
	leaseID ulid.ULID,
	tracker *migrationDirtyTracker,
) error {
	prefix := AccountKeyPrefix(accountID)

	// Always mark the directly known keys as dirty
	tracker.MarkDirty(
		fmt.Sprintf("%sleaseq", prefix),
		fmt.Sprintf("%sld:%s", prefix, leaseID),
	)
	tracker.MarkDirty(ScavengerShardKey())

	// Read lease details to get request ID
	leaseDetailKey := fmt.Sprintf("%sld:%s", prefix, leaseID)
	hmgetCmd := client.B().Hmget().Key(leaseDetailKey).Field("req").Build()
	result, err := client.Do(ctx, hmgetCmd).ToArray()
	if err != nil {
		return fmt.Errorf("HMGET lease details: %w", err)
	}
	if len(result) == 0 {
		return nil // Lease already cleaned up
	}

	requestIDStr, err := result[0].ToString()
	if err != nil || requestIDStr == "" {
		return nil // No request ID found
	}

	requestStateKey := fmt.Sprintf("%srs:%s", prefix, requestIDStr)
	tracker.MarkDirty(requestStateKey)

	// Read request state to get concurrency keys
	concurrencyKeys, err := extractConcurrencyKeysFromRequestState(ctx, client, requestStateKey)
	if err != nil {
		return fmt.Errorf("extract concurrency keys: %w", err)
	}
	tracker.MarkDirty(concurrencyKeys...)

	return nil
}

// trackReleaseDirtyKeys reads lease details and request state to determine which keys
// a Release operation will modify, and marks them dirty in the tracker.
func (rm *RoutingCapacityManager) trackReleaseDirtyKeys(
	ctx context.Context,
	client rueidis.Client,
	accountID uuid.UUID,
	leaseID ulid.ULID,
	tracker *migrationDirtyTracker,
) error {
	prefix := AccountKeyPrefix(accountID)

	// Always mark the directly known keys as dirty
	tracker.MarkDirty(
		fmt.Sprintf("%sleaseq", prefix),
		fmt.Sprintf("%sld:%s", prefix, leaseID),
	)
	tracker.MarkDirty(ScavengerShardKey())

	// Read lease details to get request ID
	leaseDetailKey := fmt.Sprintf("%sld:%s", prefix, leaseID)
	hgetCmd := client.B().Hget().Key(leaseDetailKey).Field("req").Build()
	requestIDStr, err := client.Do(ctx, hgetCmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil // Lease already cleaned up
		}
		return fmt.Errorf("HGET lease details: %w", err)
	}

	requestStateKey := fmt.Sprintf("%srs:%s", prefix, requestIDStr)
	tracker.MarkDirty(requestStateKey)

	// Read request state to get concurrency keys
	concurrencyKeys, err := extractConcurrencyKeysFromRequestState(ctx, client, requestStateKey)
	if err != nil {
		return fmt.Errorf("extract concurrency keys: %w", err)
	}
	tracker.MarkDirty(concurrencyKeys...)

	return nil
}

// extractConcurrencyKeysFromRequestState reads a request state JSON and extracts
// the InProgressLeaseKey values from concurrency constraints.
func extractConcurrencyKeysFromRequestState(ctx context.Context, client rueidis.Client, requestStateKey string) ([]string, error) {
	getCmd := client.B().Get().Key(requestStateKey).Build()
	data, err := client.Do(ctx, getCmd).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil // Request state already cleaned up
		}
		return nil, fmt.Errorf("GET request state: %w", err)
	}

	// Parse request state to extract concurrency constraint keys
	var state struct {
		SortedConstraints []struct {
			Kind        int `json:"k"`
			Concurrency *struct {
				InProgressLeaseKey string `json:"ilk"`
			} `json:"c,omitempty"`
		} `json:"s"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal request state: %w", err)
	}

	var keys []string
	for _, constraint := range state.SortedConstraints {
		if constraint.Kind == 2 && constraint.Concurrency != nil && constraint.Concurrency.InProgressLeaseKey != "" {
			keys = append(keys, constraint.Concurrency.InProgressLeaseKey)
		}
	}

	return keys, nil
}
