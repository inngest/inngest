// Package memory is a process local constraint store.  it implements
// constraintapi.CapacityManager and constraintapi.SemaphoreManager on one
// value with atomic counters and no locks on the hot path.
//
// state lives in this process only.  every service that touches a lease must
// run in the same process, or reach this manager through the constraintapi
// gRPC service.  a restart loses every lease and counter.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"golang.org/x/sync/singleflight"
)

const (
	pkgName = "constraintapi.memory"

	// DefaultSweepInterval is how often the sweeper reclaims expired leases.
	// an expired lease still counts toward usage until the next sweep.
	DefaultSweepInterval = 100 * time.Millisecond

	// housekeepEvery is the number of sweeps between idempotency map sweeps,
	// page frees and cell deletes.
	housekeepEvery = 50
)

var (
	_ constraintapi.CapacityManager  = (*Manager)(nil)
	_ constraintapi.SemaphoreManager = (*Manager)(nil)
)

// Manager holds every constraint counter, lease and idempotency record for one
// process.  construct it with NewManager and stop it with Close.
type Manager struct {
	shardName string
	clock     clockwork.Clock
	nonce     uint16

	lifecycles                           []constraintapi.ConstraintAPILifecycleHooks
	enableHighCardinalityInstrumentation constraintapi.EnableHighCardinalityInstrumentation

	operationIdempotencyTTL       time.Duration
	constraintCheckIdempotencyTTL time.Duration
	checkIdempotencyTTL           time.Duration
	sweepInterval                 time.Duration

	// sems maps a usage or capacity key to its counter.  gcra maps a rate
	// limit or throttle state key to its TAT cell.
	sems sync.Map
	gcra sync.Map

	slab   slab
	expiry *expiryIndex

	acqIdem   *ttlMap[*acquireResult]
	extIdem   *ttlMap[*constraintapi.CapacityExtendLeaseResponse]
	relIdem   *ttlMap[*constraintapi.CapacityReleaseResponse]
	checkIdem *ttlMap[struct{}]
	semIdem   *ttlMap[int64]

	flight singleflight.Group

	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

// Option configures a Manager.
type Option func(m *Manager)

// WithShardName sets the shard tag on every metric.  required.
func WithShardName(name string) Option {
	return func(m *Manager) { m.shardName = name }
}

// WithClock replaces the wall clock.  tests pass a fake clock and drive
// Scavenge themselves.
func WithClock(clock clockwork.Clock) Option {
	return func(m *Manager) { m.clock = clock }
}

// WithLifecycles registers hooks that run after every acquire, extend and
// release, including the ones the sweeper performs.
func WithLifecycles(lifecycles ...constraintapi.ConstraintAPILifecycleHooks) Option {
	return func(m *Manager) { m.lifecycles = lifecycles }
}

// WithEnableHighCardinalityInstrumentation gates function level metric tags.
func WithEnableHighCardinalityInstrumentation(fn constraintapi.EnableHighCardinalityInstrumentation) Option {
	return func(m *Manager) { m.enableHighCardinalityInstrumentation = fn }
}

// WithOperationIdempotencyTTL sets how long a successful acquire, extend or
// release replays its response.  acquire uses the smaller of this and the
// lease duration.
func WithOperationIdempotencyTTL(ttl time.Duration) Option {
	return func(m *Manager) { m.operationIdempotencyTTL = ttl }
}

// WithConstraintCheckIdempotencyTTL sets how long an acquire idempotency key
// and its lease keys skip rate limit and throttle checks on a later acquire.
func WithConstraintCheckIdempotencyTTL(ttl time.Duration) Option {
	return func(m *Manager) { m.constraintCheckIdempotencyTTL = ttl }
}

// WithCheckIdempotencyTTL sets the TTL of the write only Check record.  zero
// disables the write.
func WithCheckIdempotencyTTL(ttl time.Duration) Option {
	return func(m *Manager) { m.checkIdempotencyTTL = ttl }
}

// WithSweepInterval sets the sweeper tick.  zero disables the background
// sweeper, and expired leases are reclaimed only when Scavenge is called.
func WithSweepInterval(d time.Duration) Option {
	return func(m *Manager) { m.sweepInterval = d }
}

// NewManager builds a Manager and starts its sweeper.
func NewManager(opts ...Option) (*Manager, error) {
	m := &Manager{
		operationIdempotencyTTL:       constraintapi.OperationIdempotencyTTL,
		constraintCheckIdempotencyTTL: constraintapi.ConstraintCheckIdempotencyTTL,
		checkIdempotencyTTL:           constraintapi.CheckIdempotencyTTL,
		sweepInterval:                 DefaultSweepInterval,
		acqIdem:                       newTTLMap[*acquireResult](),
		extIdem:                       newTTLMap[*constraintapi.CapacityExtendLeaseResponse](),
		relIdem:                       newTTLMap[*constraintapi.CapacityReleaseResponse](),
		checkIdem:                     newTTLMap[struct{}](),
		semIdem:                       newTTLMap[int64](),
		stop:                          make(chan struct{}),
		done:                          make(chan struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	if m.shardName == "" {
		return nil, fmt.Errorf("missing shard name")
	}
	if m.clock == nil {
		m.clock = clockwork.NewRealClock()
	}

	nonce, err := randomNonce()
	if err != nil {
		return nil, fmt.Errorf("could not draw manager nonce: %w", err)
	}
	m.nonce = nonce
	m.expiry = newExpiryIndex(m.nowMS())

	if m.sweepInterval > 0 {
		go m.run()
	} else {
		close(m.done)
	}

	return m, nil
}

// Close stops the sweeper and waits for it.  leases and counters stay in
// memory until the Manager is garbage collected.
func (m *Manager) Close() error {
	m.closeOnce.Do(func() { close(m.stop) })
	<-m.done
	return nil
}

func (m *Manager) nowMS() int64 {
	return m.clock.Now().UnixMilli()
}

func (m *Manager) log(ctx context.Context) logger.Logger {
	return logger.StdlibLogger(ctx).With("shard", m.shardName)
}

// sem returns the live counter for key, creating it on first use.  a dead
// counter left behind by housekeeping is dropped and replaced.
func (m *Manager) sem(key string) *semaphoreCell {
	for {
		v, ok := m.sems.Load(key)
		if !ok {
			v, _ = m.sems.LoadOrStore(key, &semaphoreCell{})
		}
		c := v.(*semaphoreCell)
		if _, alive := c.load(); alive {
			return c
		}
		m.sems.CompareAndDelete(key, c)
	}
}

// peekSem reads a counter without creating it.  unknown reads as zero.
func (m *Manager) peekSem(key string) int64 {
	v, ok := m.sems.Load(key)
	if !ok {
		return 0
	}
	val, _ := v.(*semaphoreCell).load()
	return val
}

// hashKey is the idempotency map key for one operation.  the account ID is
// part of the hash, so two accounts using the same key never collide, the
// way Redis keys embed the account scope.
func hashKey(accountID uuid.UUID, op string, parts ...string) uint64 {
	var d xxhash.Digest
	d.Reset()
	_, _ = d.Write(accountID[:])
	_, _ = d.WriteString(op)
	for _, p := range parts {
		_, _ = d.WriteString("\x00")
		_, _ = d.WriteString(p)
	}
	return d.Sum64()
}

// flightKey is the singleflight key for a hashed operation key.
func flightKey(op byte, h uint64) string {
	var b [9]byte
	b[0] = op
	for i := 0; i < 8; i++ {
		b[1+i] = byte(h >> (8 * i))
	}
	return string(b[:])
}

// idemExpiry is when an idempotency record set at nowMS with ttl is gone.
// the TTL truncates to whole seconds like Redis EX.
func idemExpiry(nowMS int64, ttl time.Duration) int64 {
	return nowMS + int64(ttl.Seconds())*1000
}
