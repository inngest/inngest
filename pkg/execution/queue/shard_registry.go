package queue

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"golang.org/x/sync/errgroup"
)

// shardSelector returns a shard reference for the given queue item. It
// applies a caller-supplied policy to route enqueues to different shards.
type shardSelector func(ctx context.Context, accountID uuid.UUID, queueName *string) (QueueShard, error)

// ShardRegistry is the read-only surface for components that need to look up
// shards, fan out across the active set, or resolve a shard for a given
// account/queue. It replaces the trio of (queueShardClients map, selector,
// primaryQueueShard) that used to be passed independently into queue.New,
// the executor, the singleton store, and various API surfaces.
type ShardRegistry interface {
	// Primary returns the shard this executor is currently leased against,
	// or nil when the registry is in shard-group mode and has not yet
	// claimed a lease.
	Primary() QueueShard

	// ByName returns the shard registered under name, or
	// ErrQueueShardNotFound if absent.
	ByName(name string) (QueueShard, error)

	// ByGroup returns all shards whose ShardAssignmentConfig.ShardGroup
	// matches groupName.
	ByGroup(groupName string) []QueueShard

	// Resolve picks a shard for a given enqueue, applying the registry's
	// shard selector. Resolve errors if no selector has been configured.
	Resolve(ctx context.Context, accountID uuid.UUID, queueName *string) (QueueShard, error)

	// ForEach runs fn against every active shard concurrently, returning
	// the first error encountered. The shard set is snapshotted at call
	// time; mutations during iteration are not observed. The ctx passed
	// to fn carries a logger tagged with shard_name.
	ForEach(ctx context.Context, fn func(context.Context, QueueShard) error) error
}

// QueueShardRegistry is the surface the queue processor itself depends on:
// the read-only ShardRegistry plus SetPrimary, which the shard-lease loop
// calls when it claims a lease. Components that don't run the lease loop
// should depend on ShardRegistry instead.
type QueueShardRegistry interface {
	ShardRegistry

	// SetPrimary updates the leased primary shard.
	SetPrimary(ctx context.Context, shard QueueShard) error
}

// ShardRegistryController is the topology-mutate surface, held by bootstrap
// wiring that owns shard registration. Components outside the queue control
// plane should depend on ShardRegistry (or QueueShardRegistry) instead.
type ShardRegistryController interface {
	QueueShardRegistry

	// Add registers a shard. If a shard with the same name already
	// exists, it is overwritten.
	Add(shard QueueShard) error
}

// shardRegistry is the single ShardRegistryController implementation. It
// manages a named shard topology, an optional primary shard, and delegates
// Resolve to a caller-supplied selector. Construct via NewShardRegistry or,
// for the single-shard case, NewSingleShardRegistry.
type shardRegistry struct {
	mu       sync.RWMutex
	shards   map[string]QueueShard
	primary  QueueShard
	selector shardSelector
}

// ShardRegistryOpt configures a shardRegistry at construction.
type ShardRegistryOpt func(*shardRegistry)

// WithPrimary sets the initial primary shard. The shard must already be
// present in the topology passed to NewShardRegistry; the constructor
// enforces this. In shard-group mode where the primary is claimed at
// runtime by the lease loop, omit this option and let the lease loop call
// SetPrimary later.
func WithPrimary(shard QueueShard) ShardRegistryOpt {
	return func(r *shardRegistry) {
		r.primary = shard
	}
}

// WithShardSelector sets the selector used by Resolve to pick a shard for
// a given enqueue. Required when the topology has more than one shard;
// optional for the single-shard case (a default selector that always
// returns the only shard is installed).
func WithShardSelector(selector shardSelector) ShardRegistryOpt {
	return func(r *shardRegistry) {
		r.selector = selector
	}
}

// NewShardRegistry constructs a registry with the given topology. Use
// WithPrimary to set the leased primary up front (or call SetPrimary later)
// and WithShardSelector to install a Resolve policy. Returns an error if
// shards is empty, a provided primary is not in the topology, or no
// selector was supplied (or the supplied selector was nil).
func NewShardRegistry(
	shards map[string]QueueShard,
	opts ...ShardRegistryOpt,
) (ShardRegistryController, error) {
	if len(shards) == 0 {
		return nil, fmt.Errorf("queue: NewShardRegistry requires at least one shard")
	}
	r := &shardRegistry{
		shards: maps.Clone(shards),
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.primary != nil {
		if _, ok := r.shards[r.primary.Name()]; !ok {
			return nil, fmt.Errorf("queue: primary shard %q not in topology", r.primary.Name())
		}
	}
	if r.selector == nil {
		return nil, fmt.Errorf("queue: NewShardRegistry requires a non-nil shard selector (use WithShardSelector)")
	}
	return r, nil
}

// NewSingleShardRegistry is a convenience constructor for the common
// single-shard case (devserver, tests). It seeds the topology with the
// shard, sets it as the primary, and installs a selector that always
// returns it. Returns an error if shard is nil.
func NewSingleShardRegistry(shard QueueShard) (ShardRegistryController, error) {
	if shard == nil {
		return nil, fmt.Errorf("queue: NewSingleShardRegistry requires a non-nil shard")
	}
	return NewShardRegistry(
		map[string]QueueShard{shard.Name(): shard},
		WithPrimary(shard),
		WithShardSelector(func(context.Context, uuid.UUID, *string) (QueueShard, error) {
			return shard, nil
		}),
	)
}

func (r *shardRegistry) Primary() QueueShard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.primary
}

func (r *shardRegistry) ByName(name string) (QueueShard, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.shards[name]
	if !ok {
		return nil, ErrQueueShardNotFound
	}
	return s, nil
}

func (r *shardRegistry) ByGroup(groupName string) []QueueShard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []QueueShard
	for _, s := range r.shards {
		if s.ShardAssignmentConfig().ShardGroup == groupName {
			out = append(out, s)
		}
	}
	return out
}

func (r *shardRegistry) Resolve(ctx context.Context, accountID uuid.UUID, queueName *string) (QueueShard, error) {
	return r.selector(ctx, accountID, queueName)
}

func (r *shardRegistry) ForEach(ctx context.Context, fn func(context.Context, QueueShard) error) error {
	snapshot := r.snapshot()
	eg, ctx := errgroup.WithContext(ctx)
	for name, s := range snapshot {
		eg.Go(func() error {
			l := logger.StdlibLogger(ctx).With("shard_name", name)
			if err := fn(logger.WithStdlib(ctx, l), s); err != nil {
				return fmt.Errorf("shard %q: %w", name, err)
			}
			return nil
		})
	}
	return eg.Wait()
}

// SetPrimary updates the leased primary shard. The shard must already be
// in the registry — Add it first if not — so the registry stays the source
// of truth for which shards are known. Pass nil to clear the primary.
func (r *shardRegistry) SetPrimary(_ context.Context, shard QueueShard) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if shard == nil {
		r.primary = nil
		return nil
	}
	if _, ok := r.shards[shard.Name()]; !ok {
		return fmt.Errorf("queue: primary shard %q not in topology", shard.Name())
	}
	r.primary = shard
	return nil
}

func (r *shardRegistry) Add(shard QueueShard) error {
	if shard == nil {
		return fmt.Errorf("queue: cannot add nil shard")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shards[shard.Name()] = shard
	return nil
}

func (r *shardRegistry) snapshot() map[string]QueueShard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return maps.Clone(r.shards)
}

var _ ShardRegistryController = (*shardRegistry)(nil)
