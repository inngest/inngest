package debounce

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/xhit/go-str2duration/v2"
)

const (
	pkgName = "execution.debounce"
)

//go:embed lua/*
var embedded embed.FS

var (
	ErrDebounceExists     = fmt.Errorf("a debounce exists for this function")
	ErrDebounceNotFound   = fmt.Errorf("debounce not found")
	ErrDebounceInProgress = fmt.Errorf("debounce is in progress")
	ErrDebounceMigrating  = fmt.Errorf("debounce is migrating")
)

var (
	buffer = 50 * time.Millisecond
	// scripts stores all embedded lua scripts on initialization
	scripts = map[string]*rueidis.Lua{}
	include = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)
)

func init() {
	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}
	readRedisScripts("lua", entries)
}

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
}

// DebounceInfo contains information about a debounce for debugging purposes.
type DebounceInfo struct {
	// DebounceID is the ULID of the current debounce.
	DebounceID string
	// Item contains the debounced item, if found.
	Item *DebounceItem
}

func NewRedisDebouncer(primaryDebounceClient *redis_state.DebounceClient, primaryQueueShard queue.QueueShard, primaryQueueManager queue.QueueManager) Debouncer {
	return debouncer{
		c:                     clockwork.NewRealClock(),
		primaryDebounceClient: primaryDebounceClient,
		primaryQueueManager:   primaryQueueManager,
		primaryQueueShard:     primaryQueueShard,
		shouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return false
		},
	}
}

type DebouncerOpts struct {
	// Destination/Target: New system queue + colocated debounce state shard
	PrimaryDebounceClient *redis_state.DebounceClient
	PrimaryQueue          queue.QueueManager
	PrimaryQueueShard     queue.QueueShard

	// Source/Old: Default queue cluster
	SecondaryDebounceClient *redis_state.DebounceClient
	SecondaryQueue          queue.QueueManager
	SecondaryQueueShard     queue.QueueShard

	ShouldMigrate func(ctx context.Context, accountID uuid.UUID) bool

	Clock clockwork.Clock
}

func NewRedisDebouncerWithMigration(o DebouncerOpts) (Debouncer, error) {
	if o.PrimaryQueue == nil || o.PrimaryQueueShard == nil || o.PrimaryQueueShard.Name() == "" || o.PrimaryDebounceClient == nil {
		return nil, fmt.Errorf("missing primary")
	}

	if o.Clock == nil {
		o.Clock = clockwork.NewRealClock()
	}

	return debouncer{
		c: o.Clock,

		// New
		primaryDebounceClient: o.PrimaryDebounceClient,
		primaryQueueManager:   o.PrimaryQueue,
		primaryQueueShard:     o.PrimaryQueueShard,

		// Old
		secondaryDebounceClient: o.SecondaryDebounceClient,
		secondaryQueueManager:   o.SecondaryQueue,
		secondaryQueueShard:     o.SecondaryQueueShard,

		shouldMigrate: o.ShouldMigrate,
	}, nil
}

type debouncer struct {
	c clockwork.Clock
	// New: system queue
	primaryDebounceClient *redis_state.DebounceClient
	primaryQueueManager   queue.QueueManager
	primaryQueueShard     queue.QueueShard

	// Old: default queue
	secondaryDebounceClient *redis_state.DebounceClient
	secondaryQueueManager   queue.QueueManager
	secondaryQueueShard     queue.QueueShard

	// shouldMigrate determines if old debounces should be migrated to new cluster on the fly
	shouldMigrate func(ctx context.Context, accountID uuid.UUID) bool
}

func (d debouncer) usePrimary(shouldMigrate bool) bool {
	// Use primary cluster, if no secondary cluster is configured. This is set
	// before the migration started or after the migration is completed.
	// As soon as both (new) primary and (current) secondary are provided, we must
	// only use the primary if we're actively migrating.
	if d.secondaryDebounceClient == nil || d.secondaryQueueManager == nil || d.secondaryQueueShard == nil || d.secondaryQueueShard.Name() == "" {
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

func (d debouncer) client(shouldMigrate bool) *redis_state.DebounceClient {
	if d.usePrimary(shouldMigrate) {
		return d.primaryDebounceClient
	}
	return d.secondaryDebounceClient
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

	client := d.client(shouldMigrate)
	if client == nil {
		return fmt.Errorf("client did not return DebounceClient")
	}

	queueShard := d.queueShard(shouldMigrate)
	if queueShard == nil || queueShard.Name() == "" {
		return fmt.Errorf("queueShard did not return QueueShard")
	}

	// If the new primary is set up and the secondary is draining, for some time old debounces
	// will still exist on the secondary. If a debounce item times out before being migrated,
	// it will be marked with isSecondary in GetDebounceItem(). StartExecution() and DeleteDebounceItem()
	// must then run on the secondary cluster.
	if di.isSecondary {
		if d.secondaryDebounceClient == nil || d.secondaryQueueShard == nil || d.secondaryQueueShard.Name() == "" {
			return fmt.Errorf("debounce retrieved from secondary cluster but debounce client or shard are missing")
		}

		client = d.secondaryDebounceClient
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

	err := d.deleteDebounceItem(ctx, debounceID, client)
	if err != nil {
		return fmt.Errorf("could not delete debounce item: %w", err)
	}

	success = "true"

	return nil
}

func (d debouncer) deleteDebounceItem(ctx context.Context, debounceID ulid.ULID, client *redis_state.DebounceClient) error {
	keyDbc := client.KeyGenerator().Debounce(ctx)
	cmd := client.Client().B().Hdel().Key(keyDbc).Field(debounceID.String()).Build()
	err := client.Client().Do(ctx, cmd).Error()
	if rueidis.IsRedisNil(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error removing debounce: %w", err)
	}

	return nil
}

func (d debouncer) deleteMigratingFlag(ctx context.Context, debounceID ulid.ULID, client *redis_state.DebounceClient) error {
	keyDbc := client.KeyGenerator().DebounceMigrating(ctx)
	cmd := client.Client().B().Hdel().Key(keyDbc).Field(debounceID.String()).Build()
	err := client.Client().Do(ctx, cmd).Error()
	if rueidis.IsRedisNil(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error removing debounce migrating flag: %w", err)
	}

	return nil
}

// GetDebounceItem returns a DebounceItem given a debounce ID.
func (d debouncer) GetDebounceItem(ctx context.Context, debounceID ulid.ULID, accountID uuid.UUID) (*DebounceItem, error) {
	shouldMigrate := d.shouldMigrate(ctx, accountID)

	// Determine the flag value once and pass down to prevent inconsistent values mid-retrieval
	client := d.client(shouldMigrate)

	keyDbc := client.KeyGenerator().Debounce(ctx)

	getDebounce := func(client *redis_state.DebounceClient) (*DebounceItem, error) {
		cmd := client.Client().B().Hget().Key(keyDbc).Field(debounceID.String()).Build()
		byt, err := client.Client().Do(ctx, cmd).AsBytes()
		if rueidis.IsRedisNil(err) {
			return nil, ErrDebounceNotFound
		}

		di := &DebounceItem{}
		if err := json.Unmarshal(byt, &di); err != nil {
			return nil, fmt.Errorf("error unmarshalling debounce item: %w", err)
		}

		return di, nil
	}

	di, err := getDebounce(client)
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
		if d.secondaryDebounceClient == nil {
			return nil, ErrDebounceNotFound
		}

		// We could not find the debounce item on the primary but a secondary is configured.
		// The debounce item might still be on the secondary cluster, try to retrieve it.
		di, err := getDebounce(d.secondaryDebounceClient)

		// Mark DebounceItem as being retrieved from the secondary cluster. This is important
		// for StartExecution() and DeleteDebounceItem() to use the correct cluster.
		if di != nil {
			di.isSecondary = true
		}

		return di, err

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

	client := d.client(shouldMigrate)
	if client == nil {
		return fmt.Errorf("client did not return DebounceClient")
	}

	queueShard := d.queueShard(shouldMigrate)
	if queueShard == nil || queueShard.Name() == "" {
		return fmt.Errorf("queueShard did not return QueueShard")
	}

	// If the new primary is set up and the secondary is draining, for some time old debounces
	// will still exist on the secondary. If a debounce item times out before being migrated,
	// it will be marked with isSecondary in GetDebounceItem(). StartExecution() and DeleteDebounceItem()
	// must then run on the secondary cluster.
	if di.isSecondary {
		if d.secondaryDebounceClient == nil || d.secondaryQueueShard == nil || d.secondaryQueueShard.Name() == "" {
			return fmt.Errorf("debounce retrieved from secondary cluster but debounce client or shard are missing")
		}

		client = d.secondaryDebounceClient
		queueShard = d.secondaryQueueShard
	}

	keys := []string{
		client.KeyGenerator().DebouncePointer(ctx, fn.ID, dkey),

		// Make sure this debounce is not currently being migrated. If so, we need to prevent timeout execution.
		// The migration will go on to delete debounce state and the timeout item.
		client.KeyGenerator().DebounceMigrating(ctx),
	}
	args := []string{
		newDebounceID.String(),
		debounceID.String(),
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

	res, err := scripts["start"].Exec(
		ctx,
		client.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		status = "error"
		return err
	}

	switch res {
	// If another Start() or prepareMigration() raced us, we must not process the
	// debounce again.
	case -1:
		status = "migrating"
		return ErrDebounceMigrating
	case 0, 1:
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
	hasSecondary := d.secondaryQueueShard != nil && d.secondaryQueueShard.Name() != "" && d.secondaryQueueManager != nil && d.secondaryDebounceClient != nil
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
			err := d.deleteDebounceItem(ctx, newDebounceID, d.secondaryDebounceClient)
			if err != nil {
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

			err = d.deleteMigratingFlag(ctx, newDebounceID, d.secondaryDebounceClient)
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

	client := d.client(shouldMigrate)
	if client == nil {
		return nil, fmt.Errorf("client did not return DebounceClient")
	}

	keyPtr := client.KeyGenerator().DebouncePointer(ctx, fn.ID, key)
	keyDbc := client.KeyGenerator().Debounce(ctx)

	byt, err := json.Marshal(di)
	if err != nil {
		return nil, fmt.Errorf("error marshalling debounce: %w", err)
	}

	out, err := scripts["newDebounce"].Exec(
		ctx,
		client.Client(),
		[]string{keyPtr, keyDbc},
		[]string{newDebounceID.String(), string(byt), strconv.Itoa(int(ttl.Seconds()))},
	).ToString()
	if err != nil {
		return nil, fmt.Errorf("error creating debounce: %w", err)
	}

	if out == "0" {
		// Enqueue the debounce job with extra buffer.  This ensures that we never
		// attempt to start a debounce during the debounce's expiry (race conditions), and the extra
		// second lets an updateDebounce call on TTL 0 finish, as the buffer is the updateDebounce
		// deadline.
		qi := d.queueItem(ctx, di, newDebounceID)

		queueManager := d.queueManager(shouldMigrate)
		if queueManager == nil {
			return nil, fmt.Errorf("queueManager did not return QueueManager")
		}

		queueShard := d.queueShard(shouldMigrate)
		if queueShard == nil || queueShard.Name() == "" {
			return nil, fmt.Errorf("queueShard did not return QueueShard")
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

	existingID, err := ulid.Parse(out)
	if err != nil {
		// This was not a ULID, so we have no idea what was returned.
		return nil, fmt.Errorf("unknown new debounce return value: %s", out)
	}
	return &existingID, ErrDebounceExists
}

func (d debouncer) prepareMigration(ctx context.Context, di DebounceItem, fn inngest.Function) (*ulid.ULID, int64, error) {
	// Replace existing debounce pointer with fake debounce ID so timeout jobs don't
	now := d.c.Now()
	fakeDebounceID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return nil, 0, err
	}

	keyPtr := d.secondaryDebounceClient.KeyGenerator().DebouncePointer(ctx, fn.ID, key)
	keyDbc := d.secondaryDebounceClient.KeyGenerator().Debounce(ctx)
	keyMigrating := d.secondaryDebounceClient.KeyGenerator().DebounceMigrating(ctx)

	out, err := scripts["prepareMigration"].Exec(
		ctx,
		d.secondaryDebounceClient.Client(),
		[]string{keyPtr, keyDbc, keyMigrating},
		[]string{fakeDebounceID.String()},
	).ToAny()
	if err != nil {
		return nil, 0, fmt.Errorf("error running script: %w", err)
	}

	returnedSet, ok := out.([]any)
	if !ok {
		return nil, 0, fmt.Errorf("expected to receive one or more set items")
	}

	if len(returnedSet) < 1 {
		return nil, 0, fmt.Errorf("expected at least one item")
	}

	status, ok := returnedSet[0].(int64)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected return value, expected status int")
	}

	// No previous debounce exists
	if status == 0 {
		return nil, 0, nil
	}

	if status != 1 || len(returnedSet) < 2 {
		return nil, 0, fmt.Errorf("expected status 1 with at least two return items")
	}

	debounceIdStr, ok := returnedSet[1].(string)
	if !ok {
		return nil, 0, fmt.Errorf("expected debounceID as second item")
	}

	existingID, err := ulid.Parse(debounceIdStr)
	if err != nil {
		// This was not a ULID, so we have no idea what was returned.
		return nil, 0, fmt.Errorf("unknown new debounce return value: %s", debounceIdStr)
	}

	var timeoutUnixMillis int64

	if len(returnedSet) == 3 {
		timeoutUnixMillis, ok = returnedSet[2].(int64)
		if !ok {
			return nil, 0, fmt.Errorf("expected timeout int")
		}
	}

	return &existingID, timeoutUnixMillis, nil
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

	client := d.client(shouldMigrate)
	if client == nil {
		return fmt.Errorf("client did not return DebounceClient")
	}

	queueManager := d.queueManager(shouldMigrate)
	if queueManager == nil {
		return fmt.Errorf("queueManager did not return QueueManager")
	}

	queueShard := d.queueShard(shouldMigrate)
	if queueShard.Name() == "" {
		return fmt.Errorf("queueShard did not return QueueShard")
	}

	keyPtr := client.KeyGenerator().DebouncePointer(ctx, fn.ID, key)
	keyDbc := client.KeyGenerator().Debounce(ctx)
	byt, err := json.Marshal(di)
	if err != nil {
		return fmt.Errorf("error marshalling debounce: %w", err)
	}

	out, err := scripts["updateDebounce"].Exec(
		ctx,
		client.Client(),
		[]string{
			keyPtr,
			keyDbc,
			client.KeyGenerator().QueueItem(),
		},
		[]string{
			debounceID.String(),
			string(byt),
			strconv.Itoa(int(ttl.Seconds())),
			queue.HashID(ctx, debounceID.String()),
			strconv.Itoa(int(now.UnixMilli())),
			strconv.Itoa(int(di.Event.Timestamp)),
		},
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error creating debounce: %w", err)
	}
	switch out {
	case -1:
		// The debounce is in progress or has just finished.  Requeue.
		return ErrDebounceInProgress
	case -2:
		// The event is out-of-order and a newer event exists within the debounce.
		// Do nothing.
		return nil
	case -3:
		// The queue item is not found with the debounceID
		// enqueue a new item
		qi := d.queueItem(ctx, di, debounceID)

		return queueManager.Enqueue(ctx, qi, now.Add(ttl).Add(buffer).Add(time.Second), queue.EnqueueOpts{
			// Debounce timeout items must live on the same Redis instance as the state.
			ForceQueueShardName: queueShard.Name(),
		})
	default:
		// Debounces should have a maximum timeout;  updating the debounce returns
		// the timeout to use.
		actualTTL := time.Second * time.Duration(out)
		err = queueManager.RequeueByJobID(
			ctx,
			queueShard,
			debounceID.String(),
			now.Add(actualTTL).Add(buffer).Add(time.Second),
		)
		if err == queue.ErrQueueItemAlreadyLeased {
			logger.StdlibLogger(ctx).Warn(ErrDebounceInProgress.Error(),
				"error", err,
				"ttl", out,
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

	// Always use the primary client for debugging - this is read-only
	client := d.primaryDebounceClient
	if client == nil {
		return nil, fmt.Errorf("debounce client not configured")
	}

	// Get the debounce pointer (which contains the debounce ID)
	debouncePointerKey := client.KeyGenerator().DebouncePointer(ctx, functionID, debounceKey)

	// Read the debounce ID from the pointer
	debounceIDStr, err := client.Client().Do(ctx, client.Client().B().Get().Key(debouncePointerKey).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// No active debounce
			return &DebounceInfo{
				DebounceID: "",
				Item:       nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to get debounce pointer: %w", err)
	}

	// Get the debounce item from the hash
	debounceHashKey := client.KeyGenerator().Debounce(ctx)
	itemBytes, err := client.Client().Do(ctx, client.Client().B().Hget().Key(debounceHashKey).Field(debounceIDStr).Build()).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// Debounce ID exists in pointer but item not found in hash
			return &DebounceInfo{
				DebounceID: debounceIDStr,
				Item:       nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to get debounce item: %w", err)
	}

	// Parse the debounce item
	var di DebounceItem
	if err := json.Unmarshal(itemBytes, &di); err != nil {
		return nil, fmt.Errorf("failed to decode debounce item: %w", err)
	}

	return &DebounceInfo{
		DebounceID: debounceIDStr,
		Item:       &di,
	}, nil
}

func readRedisScripts(path string, entries []fs.DirEntry) {
	for _, e := range entries {
		// NOTE: When using embed go always uses forward slashes as a path
		// prefix. filepath.Join uses OS-specific prefixes which fails on
		// windows, so we construct the path using Sprintf for all platforms
		if e.IsDir() {
			entries, _ := embedded.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
			readRedisScripts(path+"/"+e.Name(), entries)
			continue
		}

		byt, err := embedded.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading redis lua script: %w", err))
		}

		name := path + "/" + e.Name()
		name = strings.TrimPrefix(name, "lua/")
		name = strings.TrimSuffix(name, ".lua")
		val := string(byt)

		// Add any includes.
		items := include.FindAllStringSubmatch(val, -1)
		if len(items) > 0 {
			// Replace each include
			for _, include := range items {
				byt, err = embedded.ReadFile(fmt.Sprintf("lua/includes/%s", include[1]))
				if err != nil {
					panic(fmt.Errorf("error reading redis lua include: %w", err))
				}
				val = strings.ReplaceAll(val, include[0], string(byt))
			}
		}
		scripts[name] = rueidis.NewLuaScript(val)
	}
}

type DebounceTest interface {
	TestQueueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item
}

func (d debouncer) TestQueueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item {
	return d.queueItem(ctx, di, debounceID)
}
