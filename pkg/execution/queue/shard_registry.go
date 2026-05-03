package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
)

// ShardRegistry is the read-only surface for components that need to look up
// shards, fan out across the active set, or resolve a shard for a given
// account/queue. It replaces the trio of (queueShardClients map, ShardSelector,
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
	// ShardSelector. Resolve errors if no selector has been configured.
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

// NewSingleShardRegistry is a convenience constructor for the common
// single-shard case (devserver, tests). It seeds the topology with the
// shard, configures a selector that always returns it, and sets it as
// the primary. Returns an error if shard is nil.
func NewSingleShardRegistry(shard QueueShard) (ShardRegistryController, error) {
	if shard == nil {
		return nil, fmt.Errorf("queue: NewSingleShardRegistry requires a non-nil shard")
	}
	return &singleShardRegistry{shard: shard}, nil
}

// singleShardRegistry holds a single, fixed shard. Topology mutations
// (SetPrimary, Add) are no-ops or rejected — the shard is set at
// construction and not updated.
type singleShardRegistry struct {
	shard QueueShard
}

func (r *singleShardRegistry) Primary() QueueShard {
	return r.shard
}

func (r *singleShardRegistry) ByName(name string) (QueueShard, error) {
	if r.shard.Name() != name {
		return nil, ErrQueueShardNotFound
	}
	return r.shard, nil
}

func (r *singleShardRegistry) ByGroup(groupName string) []QueueShard {
	if r.shard.ShardAssignmentConfig().ShardGroup != groupName {
		return nil
	}
	return []QueueShard{r.shard}
}

func (r *singleShardRegistry) Resolve(ctx context.Context, _ uuid.UUID, _ *string) (QueueShard, error) {
	return r.shard, nil
}

func (r *singleShardRegistry) ForEach(ctx context.Context, fn func(context.Context, QueueShard) error) error {
	l := logger.StdlibLogger(ctx).With("shard_name", r.shard.Name())
	if err := fn(logger.WithStdlib(ctx, l), r.shard); err != nil {
		return fmt.Errorf("shard %q: %w", r.shard.Name(), err)
	}
	return nil
}

func (r *singleShardRegistry) SetPrimary(ctx context.Context, shard QueueShard) error {
	if shard == nil || shard.Name() != r.shard.Name() {
		return ErrQueueShardNotFound
	}
	return nil
}

func (r *singleShardRegistry) Add(shard QueueShard) error {
	return fmt.Errorf("single shard registry does not support Add")
}
