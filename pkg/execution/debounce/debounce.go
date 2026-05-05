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
)

var (
	ErrDebounceExists     = fmt.Errorf("a debounce exists for this function")
	ErrDebounceNotFound   = queue.ErrDebounceNotFound
	ErrDebounceInProgress = fmt.Errorf("debounce is in progress")
	ErrDebounceMigrating  = fmt.Errorf("debounce is migrating")
)

var (
	buffer = 50 * time.Millisecond
)

// The general strategy for debounce:
//
// 1. Create a new debounce key.
// 2. Store the current event in the debounce key.
// 3. Create a new queue item for the debounce, linking to the debounce key

// DebounceItem represents a debounce stored within the debounce manager.
//
// DebounceItem fulfils event.TrackedEvent, allowing the use of the entire DebounceItem
// as the triggering event data passed to executor.Schedule.
type DebounceItem struct {
	// AccountID represents the account for the debounce item
	AccountID uuid.UUID `json:"aID"`
	// WorkspaceID represents the workspace for the debounce item
	WorkspaceID uuid.UUID `json:"wsID"`
	// AppID represents the app for the debounce item
	AppID uuid.UUID `json:"appID"`
	// FunctionID represents the function ID that this debounce is for.
	FunctionID uuid.UUID `json:"fnID"`
	// FunctionVersion represents the version of the function that was debounced.
	FunctionVersion int `json:"fnV"`
	// EventID represents the internal event ID that triggers the function.
	EventID ulid.ULID `json:"eID"`
	// Event represents the event data which triggers the function.
	Event event.Event `json:"e"`
	// Timeout is the timeout for the debounce, in unix milliseconds.
	Timeout int64 `json:"t,omitempty"`
	// FunctionPausedAt indicates whether the function is paused.
	FunctionPausedAt *time.Time `json:"fpAt,omitempty"`

	// While we're migrating, it is possible for the debounce timeout to elapse before
	// an old debounce is migrated, and so the debounce will still reside on the secondary cluster.
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

func (d DebounceItem) GetInternalID() ulid.ULID {
	return d.EventID
}

func (d DebounceItem) GetEvent() event.Event {
	return d.Event
}

func (d DebounceItem) GetAccountID() uuid.UUID {
	return d.AccountID
}

func (d DebounceItem) GetWorkspaceID() uuid.UUID {
	return d.WorkspaceID
}

func (d DebounceItem) GetReceivedAt() time.Time {
	return time.Time{}
}

// DebouncePayload represents the data stored within the queue's payload.
type DebouncePayload struct {
	DebounceID ulid.ULID `json:"debounceID"`
	// AccountID represents the account for the debounce item
	AccountID uuid.UUID `json:"aID"`
	// WorkspaceID represents the workspace for the debounce item
	WorkspaceID uuid.UUID `json:"wsID"`
	// AppID represents the app for the debounce item
	AppID uuid.UUID `json:"appID"`
	// FunctionID represents the function ID that this debounce is for.
	FunctionID uuid.UUID `json:"fnID"`
	// FunctionVersion represents the version of the function that was debounced.
	FunctionVersion int `json:"fnV"`
}

// DebounceMigrator exposes functionality to gracefully move existing debounces
// from one queue shard to another, maintaining the existing ttl and timeout.
type DebounceMigrator interface {
	// Migrate existing debounce to primary shard. This requires primary/secondary clusters
	// to be configured in advance.
	Migrate(ctx context.Context, debounceId ulid.ULID, di DebounceItem, remainingTtl time.Duration, fn inngest.Function) error
}

// Debouncer represents an implementation-agnostic function debouncer, delaying function runs
// until a specific time period passes when no more events matching a key are received.
type Debouncer interface {
	Debounce(ctx context.Context, d DebounceItem, fn inngest.Function) error
	GetDebounceItem(ctx context.Context, debounceID ulid.ULID, accountID uuid.UUID) (*DebounceItem, error)
	DeleteDebounceItem(ctx context.Context, debounceID ulid.ULID, d DebounceItem, accountID uuid.UUID) error
	StartExecution(ctx context.Context, d DebounceItem, fn inngest.Function, debounceID ulid.ULID) error
	// GetDebounceInfo retrieves information about the current debounce for a function and debounce key.
	// This is used for debugging and introspection.
	GetDebounceInfo(ctx context.Context, functionID uuid.UUID, debounceKey string) (*DebounceInfo, error)
	// DeleteDebounce deletes the current debounce for a function and debounce key.
	// Returns information about the deleted debounce.
	DeleteDebounce(ctx context.Context, functionID uuid.UUID, debounceKey string) (*DeleteDebounceResult, error)
	// DeleteDebounceByID deletes debounces directly by their IDs.
	// Unlike DeleteDebounce, this does not require function_id or debounce_key.
	// It removes the debounce items from the hash and (best effort) removes the timeout queue items.
	DeleteDebounceByID(ctx context.Context, debounceIDs ...ulid.ULID) error
	// RunDebounce schedules immediate execution of a debounce by creating a timeout job that runs in one second.
	RunDebounce(ctx context.Context, opts RunDebounceOpts) (*RunDebounceResult, error)
}

// DebounceInfo contains information about a debounce for debugging purposes.
type DebounceInfo struct {
	// DebounceID is the ULID of the current debounce.
	DebounceID string
	// Item contains the debounced item, if found.
	Item *DebounceItem
}

// DeleteDebounceResult contains information about a deleted debounce.
type DeleteDebounceResult struct {
	// Deleted indicates whether a debounce was found and deleted.
	Deleted bool
	// DebounceID is the ULID of the deleted debounce, if one was deleted.
	DebounceID string
	// EventID is the ULID of the event that was debounced.
	EventID string
}

// RunDebounceOpts contains options for running a debounce immediately.
type RunDebounceOpts struct {
	FunctionID  uuid.UUID
	DebounceKey string
	AccountID   uuid.UUID
}

// RunDebounceResult contains information about a scheduled debounce execution.
type RunDebounceResult struct {
	// Scheduled indicates whether a debounce was found and scheduled.
	Scheduled bool
	// DebounceID is the ULID of the debounce that was scheduled.
	DebounceID string
	// EventID is the ULID of the event that was debounced.
	EventID string
}

func NewRedisDebouncer(primaryQueueShard queue.QueueShard, primaryQueueManager queue.QueueManager) Debouncer {
	return debouncer{
		c:                   clockwork.NewRealClock(),
		primaryQueueManager: primaryQueueManager,
		primaryQueueShard:   primaryQueueShard,
		shouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return false
		},
	}
}

type DebouncerOpts struct {
	// Destination/Target: New system queue + colocated debounce state shard
	PrimaryQueue      queue.QueueManager
	PrimaryQueueShard queue.QueueShard

	// Source/Old: Default queue cluster
	SecondaryQueue      queue.QueueManager
	SecondaryQueueShard queue.QueueShard

	ShouldMigrate func(ctx context.Context, accountID uuid.UUID) bool

	Clock clockwork.Clock
}

func NewRedisDebouncerWithMigration(o DebouncerOpts) (Debouncer, error) {
	if o.PrimaryQueue == nil || o.PrimaryQueueShard == nil || o.PrimaryQueueShard.Name() == "" {
		return nil, fmt.Errorf("missing primary")
	}

	if o.Clock == nil {
		o.Clock = clockwork.NewRealClock()
	}

	return debouncer{
		c: o.Clock,

		// New
		primaryQueueManager: o.PrimaryQueue,
		primaryQueueShard:   o.PrimaryQueueShard,

		// Old
		secondaryQueueManager: o.SecondaryQueue,
		secondaryQueueShard:   o.SecondaryQueueShard,

		shouldMigrate: o.ShouldMigrate,
	}, nil
}

type debouncer struct {
	c clockwork.Clock
	// New: system queue
	primaryQueueManager queue.QueueManager
	primaryQueueShard   queue.QueueShard

	// Old: default queue
	secondaryQueueManager queue.QueueManager
	secondaryQueueShard   queue.QueueShard

	// shouldMigrate determines if old debounces should be migrated to new cluster on the fly
	shouldMigrate func(ctx context.Context, accountID uuid.UUID) bool
}

func (d debouncer) usePrimary(shouldMigrate bool) bool {
	// Use primary cluster, if no secondary cluster is configured. This is set
	// before the migration started or after the migration is completed.
	// As soon as both (new) primary and (current) secondary are provided, we must
	// only use the primary if we're actively migrating.
	if d.secondaryQueueManager == nil || d.secondaryQueueShard == nil || d.secondaryQueueShard.Name() == "" {
		return true
	}

	// If migrate feature flag is enabled, use primary
	if shouldMigrate {
		return true
	}

	// If we should not migrate yet, keep using to secondary (previous) cluster!
	// This is necessary to keep writing to the old cluster before the migration is started.
	return false
}

func (d debouncer) queueShard(shouldMigrate bool) queue.QueueShard {
	if d.usePrimary(shouldMigrate) {
		return d.primaryQueueShard
	}
	return d.secondaryQueueShard
}

func (d debouncer) queueManager(shouldMigrate bool) queue.QueueManager {
	if d.usePrimary(shouldMigrate) {
		return d.primaryQueueManager
	}
	return d.secondaryQueueManager
}

// DeleteDebounceItem removes a debounce from the map.
func (d debouncer) DeleteDebounceItem(ctx context.Context, debounceID ulid.ULID, di DebounceItem, accountID uuid.UUID) error {
	// Determine the flag value once and pass down to prevent inconsistent values mid-deletion
	shouldMigrate := d.shouldMigrate(ctx, accountID)

	queueShard := d.queueShard(shouldMigrate)
	if queueShard == nil || queueShard.Name() == "" {
		return fmt.Errorf("queueShard did not return QueueShard")
	}

	// If the new primary is set up and the secondary is draining, for some time old debounces
	// will still exist on the secondary. If a debounce item times out before being migrated,
	// it will be marked with isSecondary in GetDebounceItem(). StartExecution() and DeleteDebounceItem()
	// must then run on the secondary cluster.
	if di.isSecondary {
		if d.secondaryQueueShard == nil || d.secondaryQueueShard.Name() == "" {
			return fmt.Errorf("debounce retrieved from secondary cluster but secondary shard is missing")
		}
		queueShard = d.secondaryQueueShard
	}

	success := "false"
	defer func() {
		metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op":          "deleted",
				"queue_shard": queueShard.Name(),
				"success":     success,
			},
		})
	}()

	if err := queueShard.DebounceDeleteItems(ctx, debounceID); err != nil {
		return fmt.Errorf("could not delete debounce item: %w", err)
	}

	success = "true"
	return nil
}

// GetDebounceItem returns a DebounceItem given a debounce ID.
func (d debouncer) GetDebounceItem(ctx context.Context, debounceID ulid.ULID, accountID uuid.UUID) (*DebounceItem, error) {
	shouldMigrate := d.shouldMigrate(ctx, accountID)

	// Determine the flag value once and pass down to prevent inconsistent values mid-retrieval
	queueShard := d.queueShard(shouldMigrate)

	di, err := getDebounceItem(ctx, queueShard, debounceID)
	if err != nil && !errors.Is(err, ErrDebounceNotFound) {
		return nil, err
	}

	// If we're currently migrating, it is possible that existing debounces on the secondary
	// will time out and execute before being migrated. In this case, we must retrieve the
	// debounce item from the secondary cluster.
	if errors.Is(err, ErrDebounceNotFound) {
		usePrimary := d.usePrimary(shouldMigrate)

		// If we couldn't find the debounce item on the secondary cluster, we're done.
		if !usePrimary {
			return nil, err
		}

		// If we're supposed to use the primary and no secondary is set up, we're done.
		if d.secondaryQueueShard == nil {
			return nil, ErrDebounceNotFound
		}

		// We could not find the debounce item on the primary but a secondary is configured.
		// The debounce item might still be on the secondary cluster, try to retrieve it.
		di, err := getDebounceItem(ctx, d.secondaryQueueShard, debounceID)

		// Mark DebounceItem as being retrieved from the secondary cluster. This is important
		// for StartExecution() and DeleteDebounceItem() to use the correct cluster.
		if di != nil {
			di.isSecondary = true
		}

		return di, err
	}

	return di, nil
}

func getDebounceItem(ctx context.Context, shard queue.QueueShard, debounceID ulid.ULID) (*DebounceItem, error) {
	byt, err := shard.DebounceGetItem(ctx, debounceID)
	if err != nil {
		return nil, err
	}
	di := &DebounceItem{}
	if err := json.Unmarshal(byt, di); err != nil {
		return nil, fmt.Errorf("error unmarshalling debounce item: %w", err)
	}
	return di, nil
}

// StartExecution swaps out the underlying pointer of the debounce
func (d debouncer) StartExecution(ctx context.Context, di DebounceItem, fn inngest.Function, debounceID ulid.ULID) error {
	dkey, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return err
	}

	newDebounceID := ulid.MustNew(ulid.Now(), rand.Reader)

	// Determine the flag value once and pass down to prevent inconsistent values mid-execution
	shouldMigrate := d.shouldMigrate(ctx, di.AccountID)

	queueShard := d.queueShard(shouldMigrate)
	if queueShard == nil || queueShard.Name() == "" {
		return fmt.Errorf("queueShard did not return QueueShard")
	}

	// If the new primary is set up and the secondary is draining, for some time old debounces
	// will still exist on the secondary. If a debounce item times out before being migrated,
	// it will be marked with isSecondary in GetDebounceItem(). StartExecution() and DeleteDebounceItem()
	// must then run on the secondary cluster.
	if di.isSecondary {
		if d.secondaryQueueShard == nil || d.secondaryQueueShard.Name() == "" {
			return fmt.Errorf("debounce retrieved from secondary cluster but secondary shard is missing")
		}
		queueShard = d.secondaryQueueShard
	}

	status := "unknown"
	defer func() {
		metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op":          "start",
				"queue_shard": queueShard.Name(),
				"status":      status,
			},
		})
	}()

	res, err := queueShard.DebounceStartExecution(ctx, fn.ID, dkey, newDebounceID, debounceID)
	if err != nil {
		status = "error"
		return err
	}

	switch res {
	// If another Start() or prepareMigration() raced us, we must not process the
	// debounce again.
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

// Migrate debounces a given function with the given DebounceItem.
func (d debouncer) Migrate(ctx context.Context, debounceId ulid.ULID, di DebounceItem, remainingTtl time.Duration, fn inngest.Function) error {
	if fn.Debounce == nil {
		return fmt.Errorf("fn has no debounce config")
	}

	// Determine the flag value once and pass down to prevent inconsistent values mid-migration
	shouldMigrate := d.shouldMigrate(ctx, di.AccountID)
	if !shouldMigrate {
		return fmt.Errorf("expected migration mode to be enable")
	}

	return d.debounce(ctx, di, fn, remainingTtl, 0, shouldMigrate)
}

// Debounce debounces a given function with the given DebounceItem.
func (d debouncer) Debounce(ctx context.Context, di DebounceItem, fn inngest.Function) error {
	if fn.Debounce == nil {
		return fmt.Errorf("fn has no debounce config")
	}
	ttl, err := str2duration.ParseDuration(fn.Debounce.Period)
	if err != nil {
		return fmt.Errorf("invalid debounce duration: %w", err)
	}

	// Determine the flag value once and pass down to prevent inconsistent values while debouncing
	shouldMigrate := d.shouldMigrate(ctx, di.AccountID)

	return d.debounce(ctx, di, fn, ttl, 0, shouldMigrate)
}

func (d debouncer) debounce(ctx context.Context, di DebounceItem, fn inngest.Function, ttl time.Duration, n int, shouldMigrate bool) error {
	newDebounceID := ulid.MustNew(ulid.Timestamp(d.c.Now()), rand.Reader)
	var foundDebounce bool

	l := logger.StdlibLogger(ctx).With(
		"fn_id", di.FunctionID.String(),
		"evt_id", di.EventID.String(),
		"ttl", ttl,
		"timeout", fn.Debounce.Timeout,
	)

	if d.primaryQueueShard != nil {
		l = l.With(
			"primary_shard_name", d.primaryQueueShard.Name(),
		)
	}

	if d.secondaryQueueShard != nil {
		l = l.With(
			"secondary_shard_name", d.secondaryQueueShard.Name(),
		)
	}

	//
	// Enable debounce migration
	// 1. Check if debounce exists on old default cluster and atomically disable execution on old debounce, if exists
	// 2. Write new debounce to new system cluster with debounce ID, timeout, ttl
	// 3. Enqueue new timeout item on new system queue
	// 4. Dequeue (delete) old timeout item from default queue shard
	//
	// Notes
	// - New debounces will be created on _new_ system cluster.
	// - `shouldMigrate` should only be set to true once this code is running on all services invoking `Debounce()`
	// - Subsequent calls to this method will attempt to create/update debounces on the new system, this is desired.
	// - We must carry over the previous timeout to ensure debounces don't run longer than intended.
	//
	hasSecondary := d.secondaryQueueShard != nil && d.secondaryQueueShard.Name() != "" && d.secondaryQueueManager != nil
	if shouldMigrate && hasSecondary {
		debounceID, debounceTimeout, err := d.prepareMigration(ctx, di, fn)
		if err != nil {
			return fmt.Errorf("could not prepare debounce migration: %w", err)
		}

		if debounceID != nil {
			l = l.With(
				"debounce_id", *debounceID,
				"timeout", time.UnixMilli(debounceTimeout).String(),
			)

			l.Debug("found debounce to migrate")

			foundDebounce = true
			newDebounceID = *debounceID

			// Preserve previous timeout
			di.Timeout = debounceTimeout

			// Delete debounce state from old cluster
			if err := d.secondaryQueueShard.DebounceDeleteItems(ctx, newDebounceID); err != nil {
				l.Error("unable to delete old debounce after migration", "err", err)
				return nil
			}

			// Delete debounce timeout from old cluster
			queueItemId := queue.HashID(ctx, debounceID.String())
			err = d.secondaryQueueShard.RemoveQueueItem(
				ctx,
				// Debounce timeouts are stored in a system queue
				queue.KindDebounce,
				queueItemId,
			)
			if err != nil {
				l.Error("could not remove queue item", "item_id", queueItemId)
			} else {
				l.Debug("deleted migrated debounce")
			}

			err = d.secondaryQueueShard.DebounceDeleteMigratingFlag(ctx, newDebounceID)
			if err != nil {
				l.Error("unable to delete debounce migrating flag", "err", err)
			}

			metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"op":          "migration_prepared",
					"queue_shard": d.secondaryQueueShard.Name(),
				},
			})
		}
	}

	// Call new debounce immediately.  If this returns ErrDebounceExists then
	// update the debounce.  This ensures that checking and creating a debounce
	// is atomic, and two individual threads/workers cannot create debounces simultaneously.
	existingDebounceID, err := d.newDebounce(ctx, di, fn, ttl, shouldMigrate, newDebounceID)
	if err == nil {
		return nil
	}
	if err != ErrDebounceExists {
		if shouldMigrate && foundDebounce {
			l.Error("unexpected error while creating debounce on primary cluster during migration", "err", err)
		}

		// There was an unkown error creating the debounce.
		return err
	}
	if existingDebounceID == nil {
		if shouldMigrate && foundDebounce {
			l.Error("unexpected missing existing debounce ID after creating on primary cluster", "err", err)
		}

		return fmt.Errorf("expected debounce ID when debounce exists")
	}

	// A debounce must already exist for this fn.  Update it.
	err = d.updateDebounce(ctx, di, fn, ttl, *existingDebounceID, shouldMigrate)
	if err == context.DeadlineExceeded || err == ErrDebounceInProgress || err == ErrDebounceNotFound {
		if n == 5 {
			l.Error("unable to update debounce", "error", err)
			// Only recurse 5 times.
			return fmt.Errorf("unable to update debounce: %w", err)
		}
		// Re-invoke this to see if we need to extend the debounce or continue.
		// Wait 50 milliseconds for the current lock and job to have evaluated.
		//
		// TODO: Instead of this, make debounce creation and updating atomic within the queue.
		// This needs to modify queue items and partitions directly.
		<-time.After(750 * time.Millisecond)
		return d.debounce(ctx, di, fn, ttl, n+1, shouldMigrate)
	}

	return err
}

func (d debouncer) queueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item {
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

func (d debouncer) newDebounce(ctx context.Context, di DebounceItem, fn inngest.Function, ttl time.Duration, shouldMigrate bool, newDebounceID ulid.ULID) (*ulid.ULID, error) {
	now := d.c.Now()

	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return nil, err
	}

	// Set initial timeout value on the debounce item if configured for the function and not already set by the migration flow
	if di.Timeout == 0 {
		// Ensure we set the debounce's max lifetime.
		if timeout := fn.Debounce.TimeoutDuration(); timeout != nil {
			di.Timeout = now.Add(*timeout).UnixMilli()
		}
	}

	if di.Timeout != 0 {
		// In case the timeout is shorter than the debounce period, adjust the ttl
		// This is required for carrying over timeouts during a migration.
		// Example: Existing debounce with a timeout in 5s exists in the old cluster. We send another event,
		// which migrates the debounce over, keeping the timeout intact. If we have a debounce period of 10s,
		// we still want to run the debounce timeout job after 5s.
		untilTimeout := time.UnixMilli(di.Timeout).Sub(now) // how much time until timeout
		if ttl > untilTimeout {
			ttl = untilTimeout.Round(time.Second) // round to the nearest second
		}
		if ttl <= 0 {
			ttl = 1 * time.Second
		}
	}

	queueShard := d.queueShard(shouldMigrate)
	if queueShard == nil || queueShard.Name() == "" {
		return nil, fmt.Errorf("queueShard did not return QueueShard")
	}

	byt, err := json.Marshal(di)
	if err != nil {
		return nil, fmt.Errorf("error marshalling debounce: %w", err)
	}

	existingID, err := queueShard.DebounceCreate(ctx, fn.ID, key, newDebounceID, byt, ttl)
	if err != nil {
		return nil, fmt.Errorf("error creating debounce: %w", err)
	}

	if existingID == nil {
		// Enqueue the debounce job with extra buffer.  This ensures that we never
		// attempt to start a debounce during the debounce's expiry (race conditions), and the extra
		// second lets an updateDebounce call on TTL 0 finish, as the buffer is the updateDebounce
		// deadline.
		qi := d.queueItem(ctx, di, newDebounceID)

		queueManager := d.queueManager(shouldMigrate)
		if queueManager == nil {
			return nil, fmt.Errorf("queueManager did not return QueueManager")
		}

		err = queueManager.Enqueue(ctx, qi, now.Add(ttl).Add(buffer).Add(time.Second), queue.EnqueueOpts{
			// Debounce timeout items must live on the same Redis instance as the state.
			ForceQueueShardName: queueShard.Name(),
		})
		if err != nil {
			return &newDebounceID, fmt.Errorf("error enqueueing debounce job: %w", err)
		}

		metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op":          "created",
				"queue_shard": queueShard.Name(),
			},
		})

		return &newDebounceID, nil
	}

	return existingID, ErrDebounceExists
}

func (d debouncer) prepareMigration(ctx context.Context, di DebounceItem, fn inngest.Function) (*ulid.ULID, int64, error) {
	// Replace existing debounce pointer with fake debounce ID so timeout jobs don't
	now := d.c.Now()
	fakeDebounceID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return nil, 0, err
	}

	return d.secondaryQueueShard.DebouncePrepareMigration(ctx, fn.ID, key, fakeDebounceID)
}

// updateDebounce updates the currently pending debounce to point to the new event ID.  It pushes
// out the debounce's TTL, and re-enqueues the job to initialize fns from the debounce.
func (d debouncer) updateDebounce(ctx context.Context, di DebounceItem, fn inngest.Function, ttl time.Duration, debounceID ulid.ULID, shouldMigrate bool) error {
	now := d.c.Now()

	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return err
	}

	// NOTE: This function has a deadline to complete.  If this fn doesn't complete within the deadline,
	// eg, network issues, we must check if the debounce expired and re-attempt the entire thing, allowing
	// us to either update or create a new debounce depending on the current time.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	queueManager := d.queueManager(shouldMigrate)
	if queueManager == nil {
		return fmt.Errorf("queueManager did not return QueueManager")
	}

	queueShard := d.queueShard(shouldMigrate)
	if queueShard.Name() == "" {
		return fmt.Errorf("queueShard did not return QueueShard")
	}

	byt, err := json.Marshal(di)
	if err != nil {
		return fmt.Errorf("error marshalling debounce: %w", err)
	}

	newTTL, status, err := queueShard.DebounceUpdate(
		ctx,
		fn.ID,
		key,
		debounceID,
		byt,
		ttl,
		queue.HashID(ctx, debounceID.String()),
		now,
		di.Event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("error updating debounce: %w", err)
	}
	switch status {
	case queue.DebounceUpdateInProgress:
		// The debounce is in progress or has just finished.  Requeue.
		return ErrDebounceInProgress
	case queue.DebounceUpdateOutOfOrder:
		// The event is out-of-order and a newer event exists within the debounce.
		// Do nothing.
		return nil
	case queue.DebounceUpdateNotFound:
		// The queue item is not found with the debounceID
		// enqueue a new item
		qi := d.queueItem(ctx, di, debounceID)

		return queueManager.Enqueue(ctx, qi, now.Add(ttl).Add(buffer).Add(time.Second), queue.EnqueueOpts{
			// Debounce timeout items must live on the same Redis instance as the state.
			ForceQueueShardName: queueShard.Name(),
		})
	case queue.DebounceUpdateOK:
		// Debounces should have a maximum timeout;  updating the debounce returns
		// the timeout to use.
		actualTTL := time.Second * time.Duration(newTTL)
		err = queueManager.RequeueByJobID(
			ctx,
			queueShard,
			debounceID.String(),
			now.Add(actualTTL).Add(buffer).Add(time.Second),
		)
		if err == queue.ErrQueueItemAlreadyLeased {
			logger.StdlibLogger(ctx).Warn(ErrDebounceInProgress.Error(),
				"error", err,
				"ttl", newTTL,
			)
			// This is in progress.
			return ErrDebounceInProgress
		}
		if err != nil {
			return fmt.Errorf("error requeueing debounce job '%s': %w", debounceID, err)
		}

		metrics.IncrQueueDebounceOperationCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op":          "updated",
				"queue_shard": queueShard.Name(),
			},
		})

		return nil
	default:
		return fmt.Errorf("unknown debounce update status: %d", status)
	}
}

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

// GetDebounceInfo retrieves information about the current debounce for a function and debounce key.
func (d debouncer) GetDebounceInfo(ctx context.Context, functionID uuid.UUID, debounceKey string) (*DebounceInfo, error) {
	// Use function ID as debounce key if not specified
	if debounceKey == "" {
		debounceKey = functionID.String()
	}

	// Always use the primary shard for debugging - this is read-only
	queueShard := d.primaryQueueShard
	if queueShard == nil {
		return nil, fmt.Errorf("primary queue shard not configured")
	}

	// Read the debounce ID from the pointer
	debounceIDStr, err := queueShard.DebounceGetPointer(ctx, functionID, debounceKey)
	if errors.Is(err, ErrDebounceNotFound) {
		// No active debounce
		return &DebounceInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce pointer: %w", err)
	}

	debounceID, err := ulid.Parse(debounceIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid debounce id %q in pointer: %w", debounceIDStr, err)
	}

	itemBytes, err := queueShard.DebounceGetItem(ctx, debounceID)
	if errors.Is(err, ErrDebounceNotFound) {
		// Debounce ID exists in pointer but item not found in hash
		return &DebounceInfo{DebounceID: debounceIDStr}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce item: %w", err)
	}

	var di DebounceItem
	if err := json.Unmarshal(itemBytes, &di); err != nil {
		return nil, fmt.Errorf("failed to decode debounce item: %w", err)
	}

	return &DebounceInfo{
		DebounceID: debounceIDStr,
		Item:       &di,
	}, nil
}

// DeleteDebounce deletes the current debounce for a function and debounce key.
func (d debouncer) DeleteDebounce(ctx context.Context, functionID uuid.UUID, debounceKey string) (*DeleteDebounceResult, error) {
	// Get the debounce info first
	info, err := d.GetDebounceInfo(ctx, functionID, debounceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce info: %w", err)
	}

	if info.DebounceID == "" || info.Item == nil {
		// No active debounce to delete
		return &DeleteDebounceResult{}, nil
	}

	debounceID, err := ulid.Parse(info.DebounceID)
	if err != nil {
		return nil, fmt.Errorf("invalid debounce ID: %w", err)
	}

	// Use function ID as debounce key if not specified (same as GetDebounceInfo)
	if debounceKey == "" {
		debounceKey = functionID.String()
	}

	queueShard := d.primaryQueueShard
	if queueShard == nil || queueShard.Name() == "" {
		return nil, fmt.Errorf("primary queue shard not configured")
	}

	if err := queueShard.DebounceDeleteItems(ctx, debounceID); err != nil {
		return nil, fmt.Errorf("failed to delete debounce item: %w", err)
	}
	if err := queueShard.DebounceDeletePointer(ctx, functionID, debounceKey); err != nil {
		return nil, fmt.Errorf("failed to delete debounce pointer: %w", err)
	}

	// Try to remove the queue item (best effort)
	queueItemId := queue.HashID(ctx, debounceID.String())
	_ = queueShard.RemoveQueueItem(ctx, queue.KindDebounce, queueItemId)

	return &DeleteDebounceResult{
		Deleted:    true,
		DebounceID: info.DebounceID,
		EventID:    info.Item.EventID.String(),
	}, nil
}

// DeleteDebounceByID deletes debounces directly by their IDs.
func (d debouncer) DeleteDebounceByID(ctx context.Context, debounceIDs ...ulid.ULID) error {
	if len(debounceIDs) == 0 {
		return nil
	}

	queueShard := d.primaryQueueShard
	if queueShard == nil || queueShard.Name() == "" {
		return fmt.Errorf("primary queue shard not configured")
	}

	if err := queueShard.DebounceDeleteItems(ctx, debounceIDs...); err != nil {
		return fmt.Errorf("failed to delete debounce items: %w", err)
	}

	// Best-effort remove timeout queue items
	for _, id := range debounceIDs {
		queueItemId := queue.HashID(ctx, id.String())
		_ = queueShard.RemoveQueueItem(ctx, queue.KindDebounce, queueItemId)
	}

	return nil
}

// RunDebounce schedules immediate execution of a debounce by creating a timeout job that runs in one second.
func (d debouncer) RunDebounce(ctx context.Context, opts RunDebounceOpts) (*RunDebounceResult, error) {
	// Get the debounce info first
	info, err := d.GetDebounceInfo(ctx, opts.FunctionID, opts.DebounceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get debounce info: %w", err)
	}

	if info.DebounceID == "" || info.Item == nil {
		// No active debounce to run
		return &RunDebounceResult{
			Scheduled:  false,
			DebounceID: "",
			EventID:    "",
		}, nil
	}

	debounceID, err := ulid.Parse(info.DebounceID)
	if err != nil {
		return nil, fmt.Errorf("invalid debounce ID: %w", err)
	}

	// Use the primary queue manager
	queueManager := d.primaryQueueManager
	if queueManager == nil {
		return nil, fmt.Errorf("queue manager not configured")
	}

	queueShard := d.primaryQueueShard
	if queueShard == nil || queueShard.Name() == "" {
		return nil, fmt.Errorf("queue shard not configured")
	}

	// Requeue the debounce to run in 1 second
	err = queueManager.RequeueByJobID(
		ctx,
		queueShard,
		debounceID.String(),
		time.Now().Add(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to requeue debounce: %w", err)
	}

	return &RunDebounceResult{
		Scheduled:  true,
		DebounceID: info.DebounceID,
		EventID:    info.Item.EventID.String(),
	}, nil
}

type DebounceTest interface {
	TestQueueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item
}

func (d debouncer) TestQueueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item {
	return d.queueItem(ctx, di, debounceID)
}
