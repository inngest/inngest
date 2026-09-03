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
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
)

const (
	pkgName = "constraintapi.memory"

	// DefaultSweepInterval is how often the sweeper reclaims expired leases.
	// an expired lease still counts toward usage until the next sweep.
	DefaultSweepInterval = 100 * time.Millisecond

	// housekeepEvery is the number of sweeps between idempotency map sweeps,
	// page frees and cell deletes.
	housekeepEvery = 50

	// stripeCount is the number of operation locks.  two operations with the
	// same idempotency key share a lock, so a retry sees the first result.
	// unrelated keys share a lock one time in stripeCount.
	stripeCount = 4096
)

// paddedMutex keeps neighbouring locks on separate cache lines so cores
// taking different locks do not bounce the same line.
type paddedMutex struct {
	sync.Mutex
	_ [56]byte
}

var (
	_ constraintapi.CapacityManager  = (*Manager)(nil)
	_ constraintapi.SemaphoreManager = (*Manager)(nil)
)

// Manager holds every constraint counter, lease and idempotency record for one
// process.  construct it with NewManager and stop it with Close.
type Manager struct {
	shardName string
	clock     clockwork.Clock
	nonce     uint32

	lifecycles                           []constraintapi.ConstraintAPILifecycleHooks
	enableHighCardinalityInstrumentation constraintapi.EnableHighCardinalityInstrumentation

	operationIdempotencyTTL       time.Duration
	constraintCheckIdempotencyTTL time.Duration
	checkIdempotencyTTL           time.Duration
	sweepInterval                 time.Duration

	// sems maps a usage or capacity key to its counter.  gcra maps a rate
	// limit or throttle state key to its TAT cell.  sets maps a request shape
	// to its interned constraint set.
	sems sync.Map // cellKey -> *semaphoreCell
	gcra sync.Map // cellKey -> *gcraCell
	sets sync.Map // cellKey -> *constraintSet

	slab   slab
	expiry *expiryIndex

	acqIdem   *ttlMap[*acquireRecord]
	extIdem   *ttlMap[*constraintapi.CapacityExtendLeaseResponse]
	relIdem   *ttlMap[*constraintapi.CapacityReleaseResponse]
	checkIdem *ttlMap[struct{}]
	semIdem   *ttlMap[int64]

	locks [stripeCount]paddedMutex
	stats *stats

	stop      chan struct{}
	done      chan struct{}
	wg        sync.WaitGroup
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

// NewManager builds a Manager and starts its sweeper and metrics goroutines.
func NewManager(opts ...Option) (*Manager, error) {
	m := &Manager{
		operationIdempotencyTTL:       constraintapi.OperationIdempotencyTTL,
		constraintCheckIdempotencyTTL: constraintapi.ConstraintCheckIdempotencyTTL,
		checkIdempotencyTTL:           constraintapi.CheckIdempotencyTTL,
		sweepInterval:                 DefaultSweepInterval,
		acqIdem:                       newTTLMap[*acquireRecord](),
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
	m.stats = newStats(m.shardName)

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.stats.run(m.stop)
	}()
	if m.sweepInterval > 0 {
		m.wg.Add(1)
		go m.run()
	}
	go func() {
		m.wg.Wait()
		close(m.done)
	}()

	return m, nil
}

// Close stops the sweeper and the metrics goroutine and waits for them.
// leases and counters stay in memory until the Manager is garbage collected.
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

// lock takes the operation lock for a hashed idempotency key.
func (m *Manager) lock(key uint64) *sync.Mutex {
	mu := &m.locks[key%stripeCount].Mutex
	mu.Lock()
	return mu
}

// newRequestID is a fresh ULID at now.  request IDs identify a call in logs
// and need no unpredictability, so the entropy comes from math/rand.
func (m *Manager) newRequestID(now time.Time) ulid.ULID {
	var id ulid.ULID
	_ = id.SetTime(ulid.Timestamp(now))
	binary.BigEndian.PutUint64(id[6:14], rand.Uint64())
	binary.BigEndian.PutUint16(id[14:16], uint16(rand.Uint32()))
	return id
}

// cellKey identifies one counter.  it is the 128 bit hash of the counter's
// identity tuple, which is the same information the Redis key strings carry.
// two hashes with different seeds make a collision between two live keys
// practically impossible.
type cellKey struct {
	a, b uint64
}

const cellSeed = 0x9E3779B97F4A7C15

func cellKeyOf(kind byte, accountID uuid.UUID, p1, p2 string) cellKey {
	var arr [128]byte
	b := arr[:0]
	b = append(b, kind)
	b = appendUUID(b, accountID)
	b = appendStr(b, p1)
	b = appendStr(b, p2)
	return cellKey{a: xxhash.Sum64(b), b: sumSeeded(b, cellSeed)}
}

// scopedKey is the cell key of one scoped constraint state, a rate limit or
// throttle TAT or a concurrency counter: the kind, the account, the scope and
// the ID the scope points at, and the key expression and evaluated key
// hashes, the same identity the Redis key carries.  concurrency mode is not
// part of it, as in Redis.
func scopedKey(kind byte, accountID uuid.UUID, scope int, entity uuid.UUID, exprHash, evaluatedHash string) cellKey {
	if exprHash == "" {
		evaluatedHash = ""
	}
	var arr [128]byte
	b := arr[:0]
	b = append(b, kind)
	b = appendUUID(b, accountID)
	b = appendInt(b, scope)
	b = appendUUID(b, entity)
	b = appendStr(b, exprHash)
	b = appendStr(b, evaluatedHash)
	return cellKey{a: xxhash.Sum64(b), b: sumSeeded(b, cellSeed)}
}

// semaphoreCells returns the usage and capacity keys of one semaphore.
// capacity is shared by every evaluated key of the semaphore, usage is not.
func semaphoreCells(accountID uuid.UUID, id, evaluatedKeyHash string) (usage, capacity cellKey) {
	return cellKeyOf('u', accountID, id, evaluatedKeyHash), cellKeyOf('c', accountID, id, "")
}

// sem returns the live counter for key, creating it on first use.  a dead
// counter left behind by housekeeping is dropped and replaced.
func (m *Manager) sem(key cellKey) *semaphoreCell {
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

// gcraCell returns the live TAT cell for key, creating it on first use.  a
// dead cell left behind by housekeeping is dropped and replaced.
func (m *Manager) gcraCell(key cellKey) *gcraCell {
	for {
		v, ok := m.gcra.Load(key)
		if !ok {
			v, _ = m.gcra.LoadOrStore(key, &gcraCell{})
		}
		c := v.(*gcraCell)
		if c.alive() {
			return c
		}
		m.gcra.CompareAndDelete(key, c)
	}
}

// peekSem reads a counter without creating it.  unknown reads as zero.
func (m *Manager) peekSem(key cellKey) int64 {
	v, ok := m.sems.Load(key)
	if !ok {
		return 0
	}
	val, _ := v.(*semaphoreCell).load()
	return val
}

// the append helpers build the bytes of one hash key in a caller's stack
// array so the key is hashed once.  one Sum64 over the buffer costs a
// fraction of a Write call per field.  every variable length field is length
// prefixed so two field lists never hash the same by lining up differently.

func appendU64(b []byte, v uint64) []byte {
	return binary.LittleEndian.AppendUint64(b, v)
}

func appendInt(b []byte, v int) []byte {
	return appendU64(b, uint64(v))
}

func appendStr(b []byte, s string) []byte {
	b = appendU64(b, uint64(len(s)))
	return append(b, s...)
}

func appendBytes(b []byte, p []byte) []byte {
	b = appendU64(b, uint64(len(p)))
	return append(b, p...)
}

func appendUUID(b []byte, u uuid.UUID) []byte {
	return append(b, u[:]...)
}

// sumSeeded is the seeded second hash of a key buffer.
func sumSeeded(b []byte, seed uint64) uint64 {
	var d xxhash.Digest
	d.ResetWithSeed(seed)
	_, _ = d.Write(b)
	return d.Sum64()
}

// opKey is the idempotency map key for one operation.  the account ID is
// part of the hash, so two accounts using the same key never collide, the
// way Redis keys embed the account scope.
func opKey(accountID uuid.UUID, op, key string) uint64 {
	var arr [128]byte
	b := arr[:0]
	b = appendUUID(b, accountID)
	b = appendStr(b, op)
	b = appendStr(b, key)
	return xxhash.Sum64(b)
}

// idemExpiry is when an idempotency record set at nowMS with ttl is gone.
// the TTL truncates to whole seconds like Redis EX.
func idemExpiry(nowMS int64, ttl time.Duration) int64 {
	return nowMS + int64(ttl.Seconds())*1000
}
