package routing

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/jonboulle/clockwork"
	"google.golang.org/protobuf/proto"
)

const (
	connectionSnapshotTTL        = 250 * time.Millisecond
	connectionSnapshotMaxEntries = 1024
	connectionSnapshotMaxBytes   = 32 << 20
)

type connectionSnapshotKey struct {
	envID uuid.UUID
	appID uuid.UUID
}

type connectionSnapshot struct {
	connections []*connectpb.ConnMetadata
	token       *connectionSnapshot
	expiresAt   time.Time
	bytes       int64
}

type connectionSnapshotLoad struct {
	done        chan struct{}
	finished    chan struct{}
	cancel      context.CancelFunc
	generation  uint64
	waiters     int
	closed      bool
	invalidated bool
	snapshot    *connectionSnapshot
	err         error
}

type connectionSnapshotState struct {
	key        connectionSnapshotKey
	generation uint64
	waiters    int
	load       *connectionSnapshotLoad
	snapshot   *connectionSnapshot
	element    *list.Element
}

type connectionSnapshotCache struct {
	mu         sync.Mutex
	clock      clockwork.Clock
	states     map[connectionSnapshotKey]*connectionSnapshotState
	lru        *list.List
	bytes      int64
	maxEntries int
	maxBytes   int64
}

func newConnectionSnapshotCache() *connectionSnapshotCache {
	return &connectionSnapshotCache{
		clock:      clockwork.NewRealClock(),
		states:     make(map[connectionSnapshotKey]*connectionSnapshotState),
		lru:        list.New(),
		maxEntries: connectionSnapshotMaxEntries,
		maxBytes:   connectionSnapshotMaxBytes,
	}
}

func (c *connectionSnapshotCache) get(ctx context.Context, manager state.StateManager, key connectionSnapshotKey) (*connectionSnapshot, bool, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		c.mu.Lock()
		current := c.states[key]
		if current == nil {
			current = &connectionSnapshotState{key: key}
			c.states[key] = current
		}
		if current.snapshot != nil {
			if c.clock.Now().Before(current.snapshot.expiresAt) {
				c.lru.MoveToFront(current.element)
				snapshot := current.snapshot
				c.mu.Unlock()
				return cloneConnectionSnapshot(snapshot), true, nil
			}
			c.removeSnapshot(current)
		}
		load := current.load
		if load == nil {
			// each waiter owns its cancellation.  the shared read stops only
			// when every waiter leaves or the snapshot is invalidated.
			loadCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
			load = &connectionSnapshotLoad{
				done:       make(chan struct{}),
				finished:   make(chan struct{}),
				cancel:     cancel,
				generation: current.generation,
			}
			current.load = load
			go c.load(loadCtx, manager, current, load)
		}
		load.waiters++
		current.waiters++
		c.mu.Unlock()

		select {
		case <-ctx.Done():
			c.mu.Lock()
			c.releaseWaiter(current, load)
			c.mu.Unlock()
			return nil, false, ctx.Err()
		case <-load.done:
			c.mu.Lock()
			if err := ctx.Err(); err != nil {
				c.releaseWaiter(current, load)
				c.mu.Unlock()
				return nil, false, err
			}
			invalidated := load.invalidated || load.generation != current.generation
			if load.snapshot != nil && len(load.snapshot.connections) > 0 && !c.clock.Now().Before(load.snapshot.expiresAt) {
				invalidated = true
			}
			snapshot, err := load.snapshot, load.err
			c.releaseWaiter(current, load)
			c.mu.Unlock()
			if invalidated {
				continue
			}
			if err != nil {
				return nil, false, err
			}
			return cloneConnectionSnapshot(snapshot), false, nil
		}
	}
}

func (c *connectionSnapshotCache) load(ctx context.Context, manager state.StateManager, current *connectionSnapshotState, load *connectionSnapshotLoad) {
	defer close(load.finished)
	defer load.cancel()
	connections, err := manager.GetConnectionsByAppID(ctx, current.key.envID, current.key.appID)
	var snapshot *connectionSnapshot
	if err == nil {
		snapshot = &connectionSnapshot{
			connections: cloneConnections(connections),
			token:       &connectionSnapshot{},
		}
		snapshot.bytes = connectionSnapshotBytes(snapshot.connections)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if load.invalidated || load.generation != current.generation || current.load != load {
		return
	}
	current.load = nil
	if snapshot != nil {
		snapshot.expiresAt = c.clock.Now().Add(connectionSnapshotTTL)
	}
	load.snapshot, load.err = snapshot, err
	if snapshot != nil && len(snapshot.connections) > 0 && snapshot.bytes <= c.maxBytes {
		current.snapshot = snapshot
		current.element = c.lru.PushFront(current)
		c.bytes += snapshot.bytes
		for c.lru.Len() > c.maxEntries || c.bytes > c.maxBytes {
			oldest := c.lru.Back().Value.(*connectionSnapshotState)
			c.removeSnapshot(oldest)
			c.pruneState(oldest)
		}
	}
	load.closed = true
	close(load.done)
	c.pruneState(current)
}

func (c *connectionSnapshotCache) invalidate(key connectionSnapshotKey, observed *connectionSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.states[key]
	if current == nil {
		return
	}
	if observed != nil && (current.snapshot == nil || current.snapshot.token != observed.token) {
		return
	}
	current.generation++
	c.removeSnapshot(current)
	if load := current.load; load != nil {
		current.load = nil
		c.invalidateLoad(load)
	}
	c.pruneState(current)
}

func (c *connectionSnapshotCache) refresh(ctx context.Context, manager state.StateManager, key connectionSnapshotKey, observed *connectionSnapshot) (*connectionSnapshot, bool, error) {
	c.invalidate(key, observed)
	return c.get(ctx, manager, key)
}

func (c *connectionSnapshotCache) releaseWaiter(current *connectionSnapshotState, load *connectionSnapshotLoad) {
	load.waiters--
	current.waiters--
	if load.waiters == 0 && current.load == load {
		current.load = nil
		c.invalidateLoad(load)
	}
	c.pruneState(current)
}

func (c *connectionSnapshotCache) invalidateLoad(load *connectionSnapshotLoad) {
	load.invalidated = true
	load.cancel()
	if !load.closed {
		load.closed = true
		close(load.done)
	}
}

func (c *connectionSnapshotCache) removeSnapshot(current *connectionSnapshotState) {
	if current.snapshot == nil {
		return
	}
	c.bytes -= current.snapshot.bytes
	c.lru.Remove(current.element)
	current.snapshot, current.element = nil, nil
}

func (c *connectionSnapshotCache) pruneState(current *connectionSnapshotState) {
	// completed reads keep their generation until their waiters leave.
	// invalidation must also stop those waiters from returning an old record.
	if current.snapshot == nil && current.load == nil && current.waiters == 0 {
		delete(c.states, current.key)
	}
}

func cloneConnectionSnapshot(snapshot *connectionSnapshot) *connectionSnapshot {
	return &connectionSnapshot{
		connections: cloneConnections(snapshot.connections),
		token:       snapshot.token,
		expiresAt:   snapshot.expiresAt,
		bytes:       snapshot.bytes,
	}
}

func cloneConnections(connections []*connectpb.ConnMetadata) []*connectpb.ConnMetadata {
	clones := make([]*connectpb.ConnMetadata, len(connections))
	for i, connection := range connections {
		if connection != nil {
			clones[i] = proto.Clone(connection).(*connectpb.ConnMetadata)
		}
	}
	return clones
}

func connectionSnapshotBytes(connections []*connectpb.ConnMetadata) int64 {
	// retained bytes include protobuf wire bytes, 512 bytes per record,
	// 64 bytes per map entry, eight bytes per slice item, and 256 bytes per entry.
	// this estimate bounds retention, not caller copies or active reads.
	bytes := int64(256 + 8*len(connections))
	for _, connection := range connections {
		bytes += int64(proto.Size(connection)) + 512
		bytes += int64(len(connection.GetAllWorkerGroups())+len(connection.GetSyncedWorkerGroups())) * 64
	}
	return bytes
}
