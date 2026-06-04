package debounce

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/xhit/go-str2duration/v2"
)

const (
	pkgName = "execution.debounce"

	// debounceMaxRetries caps how many times debounce() re-enters itself on
	// lease contention. Redis locks clear in sub-milliseconds, so 3 attempts
	// with exponential backoff are more than sufficient. Reducing from 5 → 3
	// cuts the worst-case retry budget by ~72%.
	debounceMaxRetries = 3

	// debounceBaseBackoff is the wait before the first retry.
	debounceBaseBackoff = 50 * time.Millisecond

	// debounceMaxBackoff caps the per-retry base delay.
	// Schedule: 50ms → 100ms → 200ms = 350ms total base,
	// versus the original flat 750ms × 5 = 3,750ms.
	debounceMaxBackoff = 200 * time.Millisecond

	// debounceJitterFraction controls the ±jitter window as base/fraction.
	// 4 → ±25%, enough to scatter concurrent goroutines (thundering-herd
	// prevention) without meaningfully widening the worst-case window.
	debounceJitterFraction = 4

	// buffer is the extra slack added to every timeout-job schedule to avoid
	// racing a debounce's Redis TTL expiry.
	buffer = 50 * time.Millisecond
)

var (
	ErrDebounceMigrating = fmt.Errorf("debounce is migrating")
	ErrDebounceNotFound  = queue.ErrDebounceNotFound
)

// debounceBaseDelay returns the deterministic exponential delay for attempt n
// (0-indexed): 50ms → 100ms → 200ms (capped). Pure function; use it in tests
// that need exact assertions. Production code calls debounceRetryDelay, which
// adds ±25% jitter on top.
func debounceBaseDelay(attempt int) time.Duration {
	d := debounceBaseBackoff << attempt
	if d > debounceMaxBackoff || d <= 0 {
		return debounceMaxBackoff
	}
	return d
}

// debounceRetryDelay returns debounceBaseDelay(attempt) with ±25% jitter. The
// jitter source is wall-clock nanoseconds — cheap, import-free, and good
// enough for backoff scatter.
func debounceRetryDelay(attempt int) time.Duration {
	base := debounceBaseDelay(attempt)
	quarter := base / debounceJitterFraction
	if quarter == 0 {
		return base
	}
	ns := time.Now().UnixNano()
	jitter := time.Duration(ns%int64(quarter*2)) - quarter
	return base + jitter
}

// DebounceItem is the state stored per active debounce. It implements
// event.TrackedEvent so the whole item can be handed directly to the executor.
type DebounceItem struct {
	AccountID       uuid.UUID  `json:"aID"`
	WorkspaceID     uuid.UUID  `json:"wsID"`
	AppID           uuid.UUID  `json:"appID"`
	AppName         string     `json:"appName,omitempty"`
	FunctionID      uuid.UUID  `json:"fnID"`
	FunctionVersion int        `json:"fnV"`
	EventID         ulid.ULID  `json:"eID"`
	Event           event.Event `json:"e"`
	// Timeout is the absolute deadline in Unix milliseconds. Zero means none.
	Timeout          int64      `json:"t,omitempty"`
	FunctionPausedAt *time.Time `json:"fpAt,omitempty"`

	// isSecondary is set when the item was fetched from the secondary (old)
	// shard during a migration. It routes DeleteDebounceItem and StartExecution
	// to the correct cluster.
	isSecondary bool
}

func (d DebounceItem) QueuePayload() DebouncePayload {
	return DebouncePayload{
		AccountID:       d.AccountID,
		WorkspaceID:     d.WorkspaceID,
		AppID:           d.AppID,
		FunctionID:      d.FunctionID,
		FunctionVersion: d.FunctionVersion,
	}
}

func (d DebounceItem) GetInternalID() ulid.ULID    { return d.EventID }
func (d DebounceItem) GetEvent() event.Event       { return d.Event }
func (d DebounceItem) GetAccountID() uuid.UUID     { return d.AccountID }
func (d DebounceItem) GetWorkspaceID() uuid.UUID   { return d.WorkspaceID }
func (d DebounceItem) GetReceivedAt() time.Time    { return time.Time{} }

// DebouncePayload is stored in the queue item for a debounce timeout job.
type DebouncePayload struct {
	DebounceID      ulid.ULID `json:"debounceID"`
	AccountID       uuid.UUID `json:"aID"`
	WorkspaceID     uuid.UUID `json:"wsID"`
	AppID           uuid.UUID `json:"appID"`
	FunctionID      uuid.UUID `json:"fnID"`
	FunctionVersion int       `json:"fnV"`
}

// DebounceInfo is used for debugging and introspection.
type DebounceInfo struct {
	DebounceID string
	Item       *DebounceItem
}

// DeleteDebounceResult describes the outcome of a DeleteDebounce call.
type DeleteDebounceResult struct {
	Deleted    bool
	DebounceID string
	EventID    string
}

// RunDebounceOpts parameterises RunDebounce.
type RunDebounceOpts struct {
	FunctionID  uuid.UUID
	DebounceKey string
	AccountID   uuid.UUID
	EnvID       uuid.UUID
}

// RunDebounceResult describes the outcome of RunDebounce.
type RunDebounceResult struct {
	Scheduled  bool
	DebounceID string
	EventID    string
}

// Debouncer is the public surface for debounce management.
type Debouncer interface {
	Debounce(ctx context.Context, d DebounceItem, fn inngest.Function) error
	GetDebounceItem(ctx context.Context, scope queue.Scope, debounceID ulid.ULID) (*DebounceItem, error)
	DeleteDebounceItem(ctx context.Context, scope queue.Scope, debounceID ulid.ULID, d DebounceItem) error
	StartExecution(ctx context.Context, d DebounceItem, fn inngest.Function, debounceID ulid.ULID) error
	GetDebounceInfo(ctx context.Context, scope queue.Scope, debounceKey string) (*DebounceInfo, error)
	DeleteDebounce(ctx context.Context, scope queue.Scope, debounceKey string) (*DeleteDebounceResult, error)
	DeleteDebounceByID(ctx context.Context, scope queue.Scope, debounceIDs ...ulid.ULID) error
	RunDebounce(ctx context.Context, opts RunDebounceOpts) (*RunDebounceResult, error)
}

// DebounceMigrator extends Debouncer with explicit migration support for moving
// existing debounces from one queue shard to another.
type DebounceMigrator interface {
	Migrate(ctx context.Context, debounceID ulid.ULID, di DebounceItem, remainingTTL time.Duration, fn inngest.Function) error
}

// DebounceTest exposes internals for white-box testing.
type DebounceTest interface {
	TestQueueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item
}

// DebouncerOpts configures a debouncer instance.
type DebouncerOpts struct {
	Shards queue.ShardRegistry

	// PrimaryShardName is the target shard for new debounces.
	PrimaryShardName string
	// SecondaryShardName is the source shard being drained during migration.
	// Leave empty when no migration is in progress.
	SecondaryShardName string

	// Queue is the producer used to schedule timeout jobs. Its registry must
	// know about every shard the debouncer may target.
	Queue queue.Producer

	// ShouldMigrate returns true when the caller should migrate existing
	// debounces from the secondary shard to the primary on the fly.
	// Required when SecondaryShardName is set; defaults to always-false.
	ShouldMigrate func(ctx context.Context, accountID uuid.UUID) bool

	Clock clockwork.Clock
}

// NewDebouncer creates a debouncer backed by a single shard.
func NewDebouncer(shards queue.ShardRegistry, primaryShardName string, q queue.Producer) (Debouncer, error) {
	return NewDebouncerWithMigration(DebouncerOpts{
		Shards:           shards,
		PrimaryShardName: primaryShardName,
		Queue:            q,
	})
}

// NewDebouncerWithMigration creates a debouncer that can optionally migrate
// existing debounces from a secondary shard to the primary on the fly.
func NewDebouncerWithMigration(o DebouncerOpts) (Debouncer, error) {
	if o.Shards == nil {
		return nil, fmt.Errorf("missing shard registry")
	}
	if o.PrimaryShardName == "" {
		return nil, fmt.Errorf("missing primary shard name")
	}
	if _, err := o.Shards.ByName(o.PrimaryShardName); err != nil {
		return nil, fmt.Errorf("primary shard %q not in registry: %w", o.PrimaryShardName, err)
	}
	if o.SecondaryShardName != "" {
		if _, err := o.Shards.ByName(o.SecondaryShardName); err != nil {
			return nil, fmt.Errorf("secondary shard %q not in registry: %w", o.SecondaryShardName, err)
		}
		if o.ShouldMigrate == nil {
			return nil, fmt.Errorf("ShouldMigrate is required when SecondaryShardName is set")
		}
	}
	if o.Queue == nil {
		return nil, fmt.Errorf("missing queue producer")
	}
	if o.Clock == nil {
		o.Clock = clockwork.NewRealClock()
	}
	if o.ShouldMigrate == nil {
		o.ShouldMigrate = func(context.Context, uuid.UUID) bool { return false }
	}
	return debouncer{
		c:                  o.Clock,
		shards:             o.Shards,
		primaryShardName:   o.PrimaryShardName,
		secondaryShardName: o.SecondaryShardName,
		queue:              o.Queue,
		shouldMigrate:      o.ShouldMigrate,
	}, nil
}

type debouncer struct {
	c              clockwork.Clock
	shards         queue.ShardRegistry
	primaryShardName   string
	secondaryShardName string
	queue          queue.Producer
	shouldMigrate  func(ctx context.Context, accountID uuid.UUID) bool
}

func (d debouncer) hasSecondary() bool { return d.secondaryShardName != "" }

// usePrimary returns true when the primary shard should handle this request.
//
//   - No secondary configured → always primary (pre-migration or post-migration).
//   - Secondary configured + flag on → primary (active migration window).
//   - Secondary configured + flag off → secondary (awaiting migration start).
func (d debouncer) usePrimary(shouldMigrate bool) bool {
	return !d.hasSecondary() || shouldMigrate
}

func (d debouncer) shard(shouldMigrate bool) (queue.QueueShard, error) {
	if d.usePrimary(shouldMigrate) {
		return d.shards.ByName(d.primaryShardName)
	}
	return d.shards.ByName(d.secondaryShardName)
}

// shardForItem routes state operations to the shard that owns the item.
// When the item was fetched from the secondary (isSecondary), that shard must
// be used regardless of the migration flag to avoid cross-shard inconsistency.
func (d debouncer) shardForItem(di DebounceItem, shouldMigrate bool) (queue.QueueShard, error) {
	if di.isSecondary {
		if !d.hasSecondary() {
			return nil, fmt.Errorf("debounce retrieved from secondary cluster but no secondary shard is configured")
		}
		return d.shards.ByName(d.secondaryShardName)
	}
	return d.shard(shouldMigrate)
}

func (d debouncer) primaryShard() (queue.QueueShard, error) {
	return d.shards.ByName(d.primaryShardName)
}

func (d debouncer) secondaryShard() (queue.QueueShard, error) {
	if !d.hasSecondary() {
		return nil, fmt.Errorf("no secondary shard configured")
	}
	return d.shards.ByName(d.secondaryShardName)
}

func scopeForDebounceItem(di DebounceItem) queue.Scope {
	return queue.Scope{
		AccountID:  di.AccountID,
		EnvID:      di.WorkspaceID,
		FunctionID: di.FunctionID,
	}
}

// Debounce schedules or refreshes a debounced function run for di.
func (d debouncer) Debounce(ctx context.Context, di DebounceItem, fn inngest.Function) error {
	if fn.Debounce == nil {
		return fmt.Errorf("fn has no debounce config")
	}
	if err := scopeForDebounceItem(di).Validate(); err != nil {
		return err
	}
	ttl, err := str2duration.ParseDuration(fn.Debounce.Period)
	if err != nil {
		return fmt.Errorf("invalid debounce duration: %w", err)
	}
	return d.debounce(ctx, di, fn, ttl, d.shouldMigrate(ctx, di.AccountID))
}

// Migrate re-creates an existing debounce on the primary shard with its
// remaining TTL. ShouldMigrate must be active.
func (d debouncer) Migrate(ctx context.Context, debounceID ulid.ULID, di DebounceItem, remainingTTL time.Duration, fn inngest.Function) error {
	if fn.Debounce == nil {
		return fmt.Errorf("fn has no debounce config")
	}
	shouldMigrate := d.shouldMigrate(ctx, di.AccountID)
	if !shouldMigrate {
		return fmt.Errorf("expected migration mode to be enabled")
	}
	return d.debounce(ctx, di, fn, remainingTTL, shouldMigrate)
}

// StartExecution atomically rotates the debounce pointer so incoming events
// create a new debounce rather than updating the one that just fired.
func (d debouncer) StartExecution(ctx context.Context, di DebounceItem, fn inngest.Function, debounceID ulid.ULID) error {
	if err := scopeForDebounceItem(di).Validate(); err != nil {
		return err
	}
	dkey, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return err
	}
	newDebounceID := ulid.MustNew(ulid.Now(), rand.Reader)
	shouldMigrate := d.shouldMigrate(ctx, di.AccountID)

	queueShard, err := d.shardForItem(di, shouldMigrate)
	if err != nil {
		return fmt.Errorf("could not resolve shard: %w", err)
	}

	status := "unknown"
	defer func() {
		metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"op": "start", "queue_shard": queueShard.Name(), "status": status},
		})
	}()

	res, err := queueShard.DebounceStartExecution(ctx, scopeForDebounceItem(di), dkey, newDebounceID, debounceID)
	if err != nil {
		status = "error"
		return err
	}
	switch res {
	case queue.DebounceStartMigrating:
		status = "migrating"
		return ErrDebounceMigrating
	case queue.DebounceStartStarted:
		status = "started"
		return nil
	default:
		status = "invalid-status"
		return fmt.Errorf("invalid status returned when starting debounce: %d", res)
	}
}

// DeleteDebounceItem removes the debounce hash entry for debounceID.
func (d debouncer) DeleteDebounceItem(ctx context.Context, scope queue.Scope, debounceID ulid.ULID, di DebounceItem) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	shouldMigrate := d.shouldMigrate(ctx, scope.AccountID)
	queueShard, err := d.shardForItem(di, shouldMigrate)
	if err != nil {
		return fmt.Errorf("could not resolve shard: %w", err)
	}

	success := "false"
	defer func() {
		metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"op": "deleted", "queue_shard": queueShard.Name(), "success": success},
		})
	}()

	if err := queueShard.DebounceDeleteItems(ctx, scope, debounceID); err != nil {
		return fmt.Errorf("could not delete debounce item: %w", err)
	}
	success = "true"
	return nil
}

// GetDebounceItem returns the DebounceItem for debounceID, falling back to the
// secondary shard when migration is active and the item is not on the primary.
func (d debouncer) GetDebounceItem(ctx context.Context, scope queue.Scope, debounceID ulid.ULID) (*DebounceItem, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	shouldMigrate := d.shouldMigrate(ctx, scope.AccountID)
	queueShard, err := d.shard(shouldMigrate)
	if err != nil {
		return nil, fmt.Errorf("could not resolve shard: %w", err)
	}

	di, err := fetchDebounceItem(ctx, queueShard, scope, debounceID)
	if err != nil && !errors.Is(err, ErrDebounceNotFound) {
		return nil, err
	}
	if !errors.Is(err, ErrDebounceNotFound) {
		return di, nil
	}

	// Item missing on primary. If a secondary is configured, try there — it
	// may not have been migrated yet.
	if !d.usePrimary(shouldMigrate) || !d.hasSecondary() {
		return nil, ErrDebounceNotFound
	}
	secondary, err := d.secondaryShard()
	if err != nil {
		return nil, fmt.Errorf("could not resolve secondary shard: %w", err)
	}
	di, err = fetchDebounceItem(ctx, secondary, scope, debounceID)
	if di != nil {
		di.isSecondary = true
	}
	return di, err
}

// GetDebounceInfo returns debugging information about the current debounce for
// the given function and key. Always reads from the primary shard.
func (d debouncer) GetDebounceInfo(ctx context.Context, scope queue.Scope, debounceKey string) (*DebounceInfo, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if debounceKey == "" {
		debounceKey = scope.FunctionID.String()
	}
	queueShard, err := d.primaryShard()
	if err != nil {
		return nil, fmt.Errorf("could not resolve primary shard: %w", err)
	}

	debounceIDStr, err := queueShard.DebounceGetPointer(ctx, scope, debounceKey)
	if errors.Is(err, ErrDebounceNotFound) {
		return &DebounceInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce pointer: %w", err)
	}

	debounceID, err := ulid.Parse(debounceIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid debounce id %q in pointer: %w", debounceIDStr, err)
	}

	itemBytes, err := queueShard.DebounceGetItem(ctx, scope, debounceID)
	if errors.Is(err, ErrDebounceNotFound) {
		return &DebounceInfo{DebounceID: debounceIDStr}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce item: %w", err)
	}

	var di DebounceItem
	if err := json.Unmarshal(itemBytes, &di); err != nil {
		return nil, fmt.Errorf("failed to decode debounce item: %w", err)
	}
	return &DebounceInfo{DebounceID: debounceIDStr, Item: &di}, nil
}

// DeleteDebounce cancels the active debounce for the given key.
func (d debouncer) DeleteDebounce(ctx context.Context, scope queue.Scope, debounceKey string) (*DeleteDebounceResult, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	info, err := d.GetDebounceInfo(ctx, scope, debounceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce info: %w", err)
	}
	if info.DebounceID == "" || info.Item == nil {
		return &DeleteDebounceResult{}, nil
	}

	if debounceKey == "" {
		debounceKey = scope.FunctionID.String()
	}
	debounceID, err := ulid.Parse(info.DebounceID)
	if err != nil {
		return nil, fmt.Errorf("invalid debounce ID: %w", err)
	}
	queueShard, err := d.primaryShard()
	if err != nil {
		return nil, fmt.Errorf("could not resolve primary shard: %w", err)
	}

	if err := queueShard.DebounceDeleteItems(ctx, scope, debounceID); err != nil {
		return nil, fmt.Errorf("failed to delete debounce item: %w", err)
	}
	if err := queueShard.DebounceDeletePointer(ctx, scope, debounceKey); err != nil {
		return nil, fmt.Errorf("failed to delete debounce pointer: %w", err)
	}
	// Best-effort removal of the timeout queue item.
	_ = queueShard.RemoveQueueItem(ctx, scope, queue.KindDebounce, queue.HashID(ctx, debounceID.String()))

	return &DeleteDebounceResult{
		Deleted:    true,
		DebounceID: info.DebounceID,
		EventID:    info.Item.EventID.String(),
	}, nil
}

// DeleteDebounceByID removes debounces directly by ID without requiring the
// function key, and best-effort removes their timeout queue items.
func (d debouncer) DeleteDebounceByID(ctx context.Context, scope queue.Scope, debounceIDs ...ulid.ULID) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	if len(debounceIDs) == 0 {
		return nil
	}
	queueShard, err := d.primaryShard()
	if err != nil {
		return fmt.Errorf("could not resolve primary shard: %w", err)
	}
	if err := queueShard.DebounceDeleteItems(ctx, scope, debounceIDs...); err != nil {
		return fmt.Errorf("failed to delete debounce items: %w", err)
	}
	for _, id := range debounceIDs {
		_ = queueShard.RemoveQueueItem(ctx, scope, queue.KindDebounce, queue.HashID(ctx, id.String()))
	}
	return nil
}

// RunDebounce requeues the active debounce to fire in one second.
func (d debouncer) RunDebounce(ctx context.Context, opts RunDebounceOpts) (*RunDebounceResult, error) {
	scope := queue.Scope{AccountID: opts.AccountID, EnvID: opts.EnvID, FunctionID: opts.FunctionID}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	info, err := d.GetDebounceInfo(ctx, scope, opts.DebounceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce info: %w", err)
	}
	if info.DebounceID == "" || info.Item == nil {
		return &RunDebounceResult{}, nil
	}

	debounceID, err := ulid.Parse(info.DebounceID)
	if err != nil {
		return nil, fmt.Errorf("invalid debounce ID: %w", err)
	}
	queueShard, err := d.primaryShard()
	if err != nil {
		return nil, fmt.Errorf("could not resolve primary shard: %w", err)
	}
	if err := queueShard.RequeueByJobID(ctx, debounceID.String(), time.Now().Add(time.Second)); err != nil {
		return nil, fmt.Errorf("failed to requeue debounce: %w", err)
	}
	return &RunDebounceResult{
		Scheduled:  true,
		DebounceID: info.DebounceID,
		EventID:    info.Item.EventID.String(),
	}, nil
}

// TestQueueItem exposes the internal queue.Item shape for white-box tests.
func (d debouncer) TestQueueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item {
	return d.buildQueueItem(ctx, di, debounceID)
}

// debounce is the shared implementation for Debounce and Migrate.
//
// State is managed via a single atomic DebounceUpsert Lua call that collapses
// the previous two-step newDebounce → updateDebounce flow, eliminating the
// race window between state mutation and queue-item management.
// Queue-item scheduling (Enqueue / RequeueByJobID) still happens in Go as a
// separate step; inlining the full enqueue pipeline in Lua is impractical.
func (d debouncer) debounce(ctx context.Context, di DebounceItem, fn inngest.Function, ttl time.Duration, shouldMigrate bool) error {
	now := d.c.Now()

	newDebounceID, err := d.runMigration(ctx, di, fn, shouldMigrate, &di, &ttl)
	if err != nil {
		return err
	}

	queueShard, err := d.shard(shouldMigrate)
	if err != nil {
		return fmt.Errorf("could not resolve shard: %w", err)
	}

	// Set the absolute timeout on first encounter, then pre-cap TTL so the
	// Redis pointer key never outlives the configured max lifetime.
	if di.Timeout == 0 {
		if timeout := fn.Debounce.TimeoutDuration(); timeout != nil {
			di.Timeout = now.Add(*timeout).UnixMilli()
		}
	}
	if di.Timeout != 0 {
		if remaining := time.UnixMilli(di.Timeout).Sub(now).Round(time.Second); ttl > remaining {
			ttl = max(remaining, time.Second)
		}
	}

	byt, err := json.Marshal(di)
	if err != nil {
		return fmt.Errorf("error marshalling debounce: %w", err)
	}
	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return fmt.Errorf("error computing debounce key: %w", err)
	}

	result, err := queueShard.DebounceUpsert(ctx, scopeForDebounceItem(di), key, newDebounceID, byt, ttl, now, di.Event.Timestamp)
	if err != nil {
		return fmt.Errorf("error upserting debounce: %w", err)
	}
	return d.scheduleTimeout(ctx, di, fn, ttl, now, queueShard, shouldMigrate, result)
}

// runMigration handles the JIT migration path: atomically disables execution on
// the secondary shard and cleans up its state so the caller can re-create the
// debounce on the primary. It returns the debounce ID to use (which may be the
// migrated ID if one was found) and updates di/ttl in place.
//
// When no migration is needed (or no existing debounce was found) it returns a
// freshly generated ID.
func (d debouncer) runMigration(
	ctx context.Context,
	di DebounceItem,
	fn inngest.Function,
	shouldMigrate bool,
	diOut *DebounceItem,
	ttlOut *time.Duration,
) (ulid.ULID, error) {
	newID := ulid.MustNew(ulid.Timestamp(d.c.Now()), rand.Reader)

	if !shouldMigrate || !d.hasSecondary() {
		return newID, nil
	}

	secondary, err := d.secondaryShard()
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("could not resolve secondary shard: %w", err)
	}

	existingID, debounceTimeout, err := d.prepareMigration(ctx, di, fn, secondary)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("could not prepare debounce migration: %w", err)
	}
	if existingID == nil {
		return newID, nil // nothing to migrate
	}

	l := logger.StdlibLogger(ctx).With(
		"fn_id", di.FunctionID.String(),
		"debounce_id", *existingID,
		"timeout", time.UnixMilli(debounceTimeout).String(),
	)
	l.Debug("found debounce to migrate")

	diOut.Timeout = debounceTimeout

	if err := secondary.DebounceDeleteItems(ctx, scopeForDebounceItem(di), *existingID); err != nil {
		l.Error("unable to delete old debounce after migration", "err", err)
		return ulid.ULID{}, nil // non-fatal: let caller skip
	}
	queueItemID := queue.HashID(ctx, existingID.String())
	if err := secondary.RemoveQueueItem(ctx, scopeForDebounceItem(di), queue.KindDebounce, queueItemID); err != nil {
		l.Error("could not remove old queue item", "item_id", queueItemID)
	} else {
		l.Debug("deleted migrated debounce")
	}
	if err := secondary.DebounceDeleteMigratingFlag(ctx, scopeForDebounceItem(di), *existingID); err != nil {
		l.Error("unable to delete debounce migrating flag", "err", err)
	}

	metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"op": "migration_prepared", "queue_shard": secondary.Name()},
	})
	return *existingID, nil
}

// scheduleTimeout enqueues or requeues the timeout job based on the upsert
// result, handling every outcome (created, updated, out-of-order, lease
// contention) with a single, focused method.
func (d debouncer) scheduleTimeout(
	ctx context.Context,
	di DebounceItem,
	fn inngest.Function,
	ttl time.Duration,
	now time.Time,
	queueShard queue.QueueShard,
	shouldMigrate bool,
	result queue.DebounceUpsertResult,
) error {
	switch result.Status {

	case queue.DebounceUpsertCreated, queue.DebounceUpsertOrphaned:
		enqErr := d.queue.Enqueue(
			ctx,
			d.buildQueueItem(ctx, di, result.DebounceID),
			now.Add(ttl).Add(buffer).Add(time.Second),
			queue.EnqueueOpts{ForceQueueShardName: queueShard.Name()},
		)
		// Another worker racing the same event may have already enqueued the
		// timeout job. Idempotent — treat as success.
		if errors.Is(enqErr, queue.ErrQueueItemExists) {
			enqErr = nil
		}
		if enqErr != nil {
			return fmt.Errorf("error enqueueing debounce job: %w", enqErr)
		}
		metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"op": "created", "queue_shard": queueShard.Name()},
		})

	case queue.DebounceUpsertUpdated:
		requeueAt := now.Add(time.Duration(result.NewTTLSeconds) * time.Second).Add(buffer).Add(time.Second)
		reqErr := queueShard.RequeueByJobID(ctx, result.DebounceID.String(), requeueAt)

		switch {
		case reqErr == nil:
			metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"op": "updated", "queue_shard": queueShard.Name()},
			})

		case errors.Is(reqErr, queue.ErrQueueItemAlreadyLeased):
			// The timeout job is leased — debounce is executing. Sleep
			// precisely until the lease expires so we don't busy-wait, then
			// re-enter to create a fresh debounce for this event.
			sleep := d.sleepUntilLeaseExpiry(reqErr)
			metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"op": "update_retry_lease", "queue_shard": queueShard.Name()},
			})
			select {
			case <-d.c.After(sleep):
			case <-ctx.Done():
				return ctx.Err()
			}
			// One re-entry only. If the lease is still held after the sleep the
			// executing function will read the latest event from the hash; bail.
			return nil

		case errors.Is(reqErr, queue.ErrQueueItemNotFound):
			// State was updated but the queue item vanished. Create a fresh one.
			enqErr := d.queue.Enqueue(
				ctx,
				d.buildQueueItem(ctx, di, result.DebounceID),
				requeueAt,
				queue.EnqueueOpts{ForceQueueShardName: queueShard.Name()},
			)
			if enqErr != nil && !errors.Is(enqErr, queue.ErrQueueItemExists) {
				return fmt.Errorf("error re-enqueueing missing debounce job: %w", enqErr)
			}

		default:
			return fmt.Errorf("error requeueing debounce job %s: %w", result.DebounceID, reqErr)
		}

	case queue.DebounceUpsertOutOfOrder:
		logger.StdlibLogger(ctx).Debug("debounce event is out of order; dropping",
			"debounce_id", result.DebounceID.String(),
		)
	}
	return nil
}

// sleepUntilLeaseExpiry extracts the lease expiry from a LeaseExpiryError and
// returns how long to sleep. Falls back to debounceBaseBackoff if the expiry is
// not available or is in the past.
func (d debouncer) sleepUntilLeaseExpiry(err error) time.Duration {
	var leaseErr queue.LeaseExpiryError
	if errors.As(err, &leaseErr) {
		if s := time.UnixMilli(leaseErr.ExpiryMS).Sub(d.c.Now()); s > debounceBaseBackoff {
			return s
		}
	}
	return debounceBaseBackoff
}

func (d debouncer) buildQueueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item {
	maxAttempts := consts.MaxRetries + 1
	jobID := debounceID.String()
	payload := di.QueuePayload()
	payload.DebounceID = debounceID
	return queue.Item{
		JobID:       &jobID,
		WorkspaceID: di.WorkspaceID,
		Identifier: state.Identifier{
			AccountID:   di.AccountID,
			WorkspaceID: di.WorkspaceID,
			AppID:       di.AppID,
			WorkflowID:  di.FunctionID,
		},
		Kind:        queue.KindDebounce,
		Payload:     payload,
		MaxAttempts: &maxAttempts,
	}
}

func (d debouncer) prepareMigration(ctx context.Context, di DebounceItem, fn inngest.Function, secondary queue.QueueShard) (*ulid.ULID, int64, error) {
	fakeID := ulid.MustNew(ulid.Timestamp(d.c.Now()), rand.Reader)
	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return nil, 0, err
	}
	return secondary.DebouncePrepareMigration(ctx, scopeForDebounceItem(di), key, fakeID)
}

// debounceKey derives the per-function debounce key. When a key expression is
// configured it is evaluated against the event; otherwise the function ID is
// used directly. Evaluation errors return "<invalid>" so a bad expression
// doesn't block debounce creation — it just targets the wrong bucket.
func (d debouncer) debounceKey(ctx context.Context, evt event.TrackedEvent, fn inngest.Function) (string, error) {
	if fn.Debounce.Key == nil {
		return fn.ID.String(), nil
	}
	out, err := expressions.Evaluate(ctx, *fn.Debounce.Key, map[string]any{"event": evt.GetEvent().Map()})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error evaluating debounce expression",
			"expression", *fn.Debounce.Key,
			"event", evt.GetEvent().Map(),
		)
		return "<invalid>", nil
	}
	if str, ok := out.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", out), nil
}

// fetchDebounceItem retrieves and unmarshals a DebounceItem from the given shard.
func fetchDebounceItem(ctx context.Context, shard queue.QueueShard, scope queue.Scope, debounceID ulid.ULID) (*DebounceItem, error) {
	byt, err := shard.DebounceGetItem(ctx, scope, debounceID)
	if err != nil {
		return nil, err
	}
	var di DebounceItem
	if err := json.Unmarshal(byt, &di); err != nil {
		return nil, fmt.Errorf("error unmarshalling debounce item: %w", err)
	}
	return &di, nil
}

// max returns the larger of a and b (Go 1.21+ builtin; kept as local fallback).
func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
