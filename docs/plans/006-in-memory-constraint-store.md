---
TITLE: In-Memory Constraint Store
AUTHOR: Tony HB
STATUS: Active
---

# In-memory constraint store: `pkg/constraintapi/memory`

## Context

`pkg/constraintapi` enforces rate limits, throttles, concurrency and semaphores.  Every
operation (`Check`, `Acquire`, `ExtendLease`, `Release`, scavenging, semaphore capacity
management) is a Lua script executed in Valkey.  All keys share the `{cs}` hash tag, so the
whole constraint store runs on one Valkey shard and one thread.

Concurrency constraints are the only constraint kind that stores per-lease state inside the
constraint (a ZSET per constraint key, scored by lease expiry).  Semaphores already model
the same thing with two counters (`cap`, `usage`).  If concurrency constraints are also
represented as semaphores, every constraint becomes either a GCRA cell (one `tat` value) or
a semaphore (two counters), and the entire store fits in process memory as atomic counters.
Valkey stops being a dependency for constraints, and constraint evaluation runs lock free
in parallel.

Goal: add a second `CapacityManager` + `SemaphoreManager` implementation backed by process
memory, in a **new package**, behind the same API, without deleting or changing the
behaviour of the Redis implementation, so both can be constructed and run side by side.


## Decisions taken (with the user)

- **New package** `pkg/constraintapi/memory`.  GCRA arithmetic in a leaf package
  `pkg/constraintapi/gcra` (no imports of `constraintapi`) so the existing Lua GCRA test
  tables in `constraintapi` validate the Go port without an import cycle.
- **Go API only, no wiring.**  `memory.NewManager` is the integration point.  The dev
  server (`pkg/devserver/devserver.go:345-393`, the only production construction site)
  keeps using Redis.  Cloud selects the backend itself.
- **No account locks.  Constraint state is atomics.**  Usage and capacity are
  `atomic.Int64`; the GCRA TAT is `atomic.Uint64` (float64 bits) updated by CAS.
  Cross-constraint atomicity is replaced by optimistic take-and-rollback (never over-admits).
- **Leases are sequence numbered.**  Each lease gets a monotonic `uint64` seq encoded in
  the ULID entropy; the ULID timestamp stays the expiry.  Lease records live in a paged
  slab indexed by seq (no hashing, no map, write-once slots).
- **Sweeper-only expiry.**  One goroutine per manager reclaims expired leases every tick
  (default 100ms) through the normal `Release` path with the scavenger `LeaseSource`, so
  hooks and metrics behave like the Redis scavenger.  No per-operation drain.  The manager
  does not implement `CapacityLeaseScavenger`; it exposes `Scavenge(ctx)` for tests and
  for callers that want to drive it.
- **Expiry index is a `[]uint64` of seqs per expiry second.**  No bitmaps, no new dependency.
- **Idempotency is striped TTL maps** keyed by xxhash `uint64`, swept by the same goroutine.
- **Simple first.**  Denser encodings, interning and the scavenger interface are recorded
  under "Room to improve" with their numbers.

## Reference: what the Lua does (parity checklist)

| Concern | Redis behaviour | Source |
|---|---|---|
| Constraint order | `sortConstraints`; limiting/exhausted indices refer to the sorted list | `sort.go`, `acquire.go:227` |
| GCRA | `rateLimit` (ns) / `throttle` (ms); quantity 0 simulates; rate limit burst = `limit/10` (int division); throttle `maxBurst = l + b - 1` from the *unclamped* config limit; `gcra.lua` clamps `limit >= 1` internally; `retry_at` defaults to `now + emission`, overridden only when `diff < 0 and increment <= dvt`; on the limited path the TAT is **not** written; on success TTL = `%d` of `max(ttl/1e9, 1)` (truncates) | `lua/helper/gcra.lua`, `acquire.lua:234-252`, `lua.go:287,354` |
| GCRA quirk to keep | when stored TAT is in the past, `ttl < 0` and `remaining > burst+1`; pass 2 with `quantity = granted` may hit the limited path and skip the `SET` while leases are still granted.  Pin with a Lua test before porting, then match. | `gcra.lua:64-112` |
| Concurrency | usage = `ZCOUNT key nowMS +inf` (inclusive: expiry == now still counts); capacity = `limit - usage`; retryAt = `now + 2s` | `acquire.lua:73,239-244` |
| Semaphore | `remaining = cap - usage`; `remaining < weight` → exhausted, else `floor(remaining/weight)`; weight `<= 0` → 1; pass 2 `INCRBY weight*granted`; **no retryAt** (semaphore-only exhaustion yields zero `RetryAfter`); unset capacity reads as 0 | `acquire.lua:253-268,380-395` |
| Two-pass acquire | pass 1: `availableCapacity = requested`, limiting iff `cap < available` (strict), exhausted iff `cap <= 0`; if `available <= 0` → status 2, **nothing written, nothing cached**; pass 2 mutates every constraint by `granted`, recomputes exhausted + retryAt | `acquire.lua:135-423` |
| `skipGCRA` | if constraint-check idempotency exists for `req.IdempotencyKey`, rate limit/throttle are skipped (not exhausted, capacity `max(available,1)` in pass 1, `1` in pass 2) | `acquire.lua:167,229,347` |
| Bookkeeping | lease details `{lik, rid, req}`; request state (`redisRequestState`) with `g`, `a`; account leases ZSET; scavenger shard ZSET; constraint-check idempotency set for the request key **and** every hashed lease key (TTL 5 min); leases are the prefix of `LeaseIdempotencyKeys` in request order | `acquire.lua:425-473` |
| Op idempotency | `acq` key = `hash(IdempotencyKey + "-" + fingerprint)`, TTL `int(min(5s, Duration).Seconds())`; `ext`/`rel` TTL 5s; cached **only on success** (acquire status 3, extend 4, release 3); replay returns the stored response with `OperationIdempotencyHit`; `RequestID` is fresh even on replay | `acquire.go:270-272`, `extend.lua:210`, `release.lua:176` |
| Release | idempotent by lease ID: status 1 no lease, 2 no request state, 3 ok; concurrency `ZREM`; semaphore `DECRBY` when `rel == 0` or `Source.Location == CallerLocationLeaseScavenge`, clamped at 0; `a--`, delete request state at 0; `Usage` contains **only concurrency** entries with `l` = limit stored at acquire time | `lua/release.lua` |
| Extend | ULID timestamp `< nowMS` (strict) → status 1 **before** any lookup; no details → 2; no request state → 3; ok → 4 with new ULID at `now + Duration`; `Usage` concurrency-only as above | `lua/extend.lua` |
| Check | never reads its `chk` key (write-only); `availableCapacity` starts nil so the first constraint is always limiting; `break` when capacity hits 0; usage: rate limit raw `u` (may be negative), concurrency/throttle `max(min(l - cap, l), 0)`; **no `k == 4` branch, so a semaphore reports as exhausted with retryAt 0** | `lua/check.lua` |
| Scavenge | every 500ms (`consts.ConstraintAPIScavengerTick`): `Release` expired leases with `Source{ServiceConstraintScavenger, CallerLocationLeaseScavenge}` and `IdempotencyKey = leaseID`; metrics via `ScavengeResult.Report` | `scavenge.go:337-378`, `devserver.go:393` |
| Semaphore manager | `setcap`/`adjcap`/`rel` idempotency keys, **unhashed**, 60s; replay returns first result with `Applied=false`; adjust and release clamp at 0; `GetCapacity` of unknown = `(0, 0, nil)` | `semaphore_manager.go`, `lua/semaphore_*.lua` |
| Centralized acquire cache | optional Redis cache of exhausted constraints | `acquire.go:232-260` |

Consumer contracts that must hold (from `pkg/execution/queue/constraints.go`,
`process.go`, `pkg/execution/executor/constraints.go`, `finalize.go`,
`pkg/connect/lifecycles/semaphore.go`, `pkg/registration/process.go`,
`pkg/debugapi/semaphore.go`):

- Lease ID ULID timestamp **is** the expiry (`constraints.go:414,447,600`, `process.go:229,245`).
- `ExtendLease` returns `LeaseID == nil, err == nil` for every non-success status (`process.go:326-339`).
- `Release` never errors for a missing lease (`process.go:195-203`).
- `ExhaustedConstraints` order = ascending sorted index, deduped; `ConvertLimitingConstraint` takes the last match (`constraints.go:111-164`).
- `RetryAfter` zero when nothing exhausted or when in the past (`acquire.go:407-410`).
- `ErrAccountNotFound` must never be returned for an unseen account.
- `l.IdempotencyKey` on returned leases is the original unhashed key (`constraints.go:337-344`).
- Concurrent `Acquire` calls with the same idempotency key must all receive the same leases (`concurrency_race_test.go:465`).
- `AdjustCapacity` idempotent per connection id (`lifecycles/semaphore.go:67,105`); `SetCapacity` then `GetCapacity` in one request must see the new value (`debugapi/semaphore.go:148-153`).

**Process boundary.**  The Redis store is shared by executor, connect gateway, registration,
finalize and debugapi.  A process-local memory store only works when all of them run in
one process (the dev server does).  For a multi-process deployment it must sit behind the
existing gRPC service (`proto/constraintapi/v1/service.proto`, `convert.go`).  State is
lost on restart; no persistence in this work.  Lease IDs from another manager instance
(different nonce) are treated as unknown leases.

## Design

### Packages and files

```
pkg/constraintapi/gcra/          leaf package, pure arithmetic, stdlib only
    gcra.go                      Throttle(...), RateLimit(...), Result
    gcra_test.go
pkg/constraintapi/memory/        imports constraintapi, gcra, x/sync/singleflight
    manager.go                   Manager, Option, NewManager, Close
    cells.go                     semaphoreCell, gcraCell, take/give/update
    leaseid.go                   seq <-> ULID encoding, nonce
    slab.go                      paged slab of lease slots
    expiry.go                    per-second seq buckets
    ttlmap.go                    striped TTL map (idempotency)
    acquire.go  check.go  extend.go  release.go
    semaphore.go                 SemaphoreManager methods
    sweeper.go                   Scavenge, sweeps, the goroutine
    README.md
    *_test.go                    package memory_test (external) so miniredis + the Redis
                                 manager can be used for conformance without a cycle
```

`memory.Manager` implements `constraintapi.CapacityManager` and
`constraintapi.SemaphoreManager` on one value, so semaphore management and leasing share
state the way the two Redis structs share keys.

```go
func NewManager(opts ...Option) (*Manager, error)
func (m *Manager) Close() error                       // stops the sweeper
func (m *Manager) Scavenge(ctx context.Context) (reclaimed int, err error)
func WithShardName(string) Option                     // required, metric tags
func WithClock(clockwork.Clock) Option
func WithLifecycles(...constraintapi.ConstraintAPILifecycleHooks) Option
func WithEnableHighCardinalityInstrumentation(constraintapi.EnableHighCardinalityInstrumentation) Option
func WithOperationIdempotencyTTL / WithConstraintCheckIdempotencyTTL / WithCheckIdempotencyTTL(time.Duration) Option
func WithSweepInterval(time.Duration) Option          // default 100ms
```

Defaults mirror `NewRedisCapacityManager` (`redis.go:127-153`).

### Edits to `pkg/constraintapi` (additive or rename, behaviour preserving)

| Edit | Why |
|---|---|
| `sort.go`: add `func SortConstraints(constraints []ConstraintItem) { sortConstraints(constraints) }` | memory must sort identically; existing callers and `sort_test.go` untouched |
| `lua.go`: extract the limit lookup (`lua.go:283-357`) into exported `func (ci ConstraintItem) ResolveLimits(config ConstraintConfig) ConstraintLimits` (`Limit, Burst, Period int`), called by `ToSerializedConstraintItem` | both backends resolve the same limit for the same request |
| `semaphore_manager.go`: rename `semaphoreIdempotencyTTL` → `SemaphoreIdempotencyTTL` | single source of truth for the 60s window |
| `gcra_test.go`: `gcraRunner{run(opts), seed(key, tat)}` with Lua and `gcra` implementations; tables run per backend via `t.Run`; add a non-divisible case (limit 7 / 1 min) and a "TAT in the past inflates remaining" case pinning current Lua behaviour | validates the port with the existing ~1600 lines of expectations |
| `SEMAPHORES.md`: one sentence pointing at `memory/README.md` | discoverability |

Not exported: `internalDebugState`; memory responses leave it zero so `Debug()` returns nil
(only external use is a `t.Log`, `tests/execution/constraintapi/constraintapi_test.go:209`).
Not reused: `buildRequestState`; memory builds its fingerprint from
`ToSerializedConstraintItem` (exported) with xxhash (fingerprints are per backend).
Reused as is: `constraintapi.ScavengeResult` and its `Report` for scavenger metrics.

### Concurrency as a semaphore

A concurrency constraint is a `semaphoreCell` of weight 1 with auto release, keyed by the
same identity as the Redis in-progress key (account, scope, the ID the scope points at, key
expression hash, evaluated key hash; mode excluded as in Redis), whose capacity is the
request's configured limit instead of a counter.  Acquire `take`s, release `give`s once by
the slot CAS, extend moves the slot and leaves the counter, the sweeper releases through the
normal path.  It keeps the Lua's `limit - usage` for its units, so a limit lowered below the
usage reports negative `AvailableCapacity` from `Check` as Redis does; the semaphore formula
floors at zero and divides by weight.  Two differences from the ZSET: usage drops on the
sweep tick rather than the millisecond a lease expires, and a counter does not heal a lost
decrement the way a ZSET recomputes from its members.  A recount of live slots per cell is
possible as a repair job and is not in this work.

### Constraint cells (`cells.go`)

```go
type semaphoreCell struct{ v atomic.Int64 }      // one per UsageKey / InProgressLeasesKey / CapacityKey
type gcraCell struct {
    tat       atomic.Uint64 // math.Float64bits; 0 = absent
    expiresAt atomic.Int64  // unix ms; read as absent when <= now
}
// Manager fields
sems sync.Map // string -> *semaphoreCell, LoadOrStore on first use
gcra sync.Map // string -> *gcraCell
```

Keys are the existing exported builders (`InProgressLeasesKey`, `UsageKey`, `CapacityKey`,
`StateKey`), which embed the account ID, so identity rules match Redis exactly.

```go
// take adds w*q and returns how many of q fit under cap, rolling back the rest exactly.
func (c *semaphoreCell) take(capacity, w int64, q int) int {
    added := w * int64(q); after := c.v.Add(added); before := after - added
    fit := clamp((capacity - before) / w, 0, q)        // acquire.lua:261-266
    if fit < q { c.v.Add(-w * int64(q - fit)) }
    return fit
}
// give decrements by w, never below zero.  a CAS loop, not Add + fix-up, because two
// concurrent double releases fixing up would leave the counter at +1.
func (c *semaphoreCell) give(w int64)
// update runs f on the current tat and CASes the result; retries on contention.
func (c *gcraCell) update(f func(tat float64, present bool) (newTAT float64, store bool, ttlSec int64))
```

The transient overshoot in `take` can make a concurrent request under-grant for a few
nanoseconds.  Never over-admission.  Semaphore rollbacks are exact; a GCRA rollback (only
when a later constraint shrinks a partial grant) subtracts `diff * emission` by CAS and is
approximate under contention by at most one emission interval per unit, always in the
restrictive direction.

### Lease IDs and the slab (`leaseid.go`, `slab.go`)

ULID layout (16 bytes): `[0:6]` expiry ms (unchanged, the consumer contract),
`[6:14]` `uint64` seq big endian, `[14:16]` manager nonce (random at `NewManager`).
`decode(id) (expiresAtMS int64, seq uint64, nonce uint16)`.  A foreign nonce → unknown lease.

```go
const pageBits = 16                           // 65536 slots per page, 1.5 MB
type slot struct {
    state       atomic.Uint32                 // 0 empty, 1 live, 2 taken
    expiresAtMS int64
    req         *requestState                 // account, env, fn, app IDs and constraints live here
}
type page struct { createdAtMS int64; live atomic.Int64; slots [1 << pageBits]slot }
type slab struct {
    next  atomic.Uint64          // next seq
    pages sync.Map               // seq >> pageBits -> *page
}
func (s *slab) alloc() (seq uint64, sl *slot)  // next.Add(1); LoadOrStore page; fill; state.Store(1); page.live++
func (s *slab) get(seq uint64) *slot            // nil when page gone or state != 1
func (s *slot) take() bool                      // CAS 1 -> 2; the winner owns the record and the counters
```

The slot does not store the lease ID (derivable from expiry, seq, nonce), the account ID
(in `requestState`), or the hashed lease idempotency key and run ID that Redis keeps in
`ld:*`: no response or lifecycle hook reads them back (`release.lua:160-171`,
`extend.lua:195-205`, `lifecycle.go`).

Seqs never repeat, so a slot is written once and taken at most once: no ABA, no locks.
`Release`, `ExtendLease` and the sweeper all go through `take`, which is what makes them
safe against each other.  A page is freed by housekeeping when it is not its shard's
allocation page and every slot in it reads taken; a late `Release` for a freed page finds no
page → status 1, same as Redis after cleanup.

```go
type requestState struct {                      // referenced by pointer from each slot
    accountID, envID, functionID, appID uuid.UUID
    sortedConstraints []constraintapi.ConstraintItem
    limits            []constraintapi.ConstraintLimits
    usageKeys         []string
    configVersion, requestedAmount, grantedAmount int
    active            atomic.Int64                // garbage collected at 0
    maximumLifetime   time.Duration
    source            constraintapi.LeaseSource
}
```

No request-ID map: Lua needed `rs:<requestID>` because it has no pointers.

### Expiry index (`expiry.go`)

```go
type expiryBucket struct { mu sync.Mutex; seqs []uint64 }
type expiryIndex struct {
    buckets sync.Map       // expiry second -> *expiryBucket
    swept   atomic.Int64   // last second drained whole
}
func (e *expiryIndex) add(expiresAtMS int64, seq uint64)   // LoadOrStore bucket; lock; append
```

`ExtendLease` appends the new seq to its bucket; the old seq stays in its old bucket with
`state == 2` and is skipped when that bucket drains.  New leases always expire at least
`MinimumDuration` (2s) ahead, so a bucket whose second has passed never receives entries
again, which makes `swept` monotonic.

### Sweeper (`sweeper.go`)

`NewManager` starts one goroutine on `clock.NewTicker(sweepInterval)`; `Close` stops it.
Each tick runs `Scavenge(ctx)`; every 50th tick (~5s) also runs the housekeeping below.

`Scavenge(ctx)`:

1. For each second `s` in `swept+1 .. nowSec-1`: `LoadAndDelete` the bucket; for each seq
   with `slab.get(seq) != nil`, call `m.Release(ctx, &CapacityReleaseRequest{IdempotencyKey:
   leaseID.String(), AccountID: slot.req.accountID, LeaseID: encode(...), Source:
   LeaseSource{ServiceConstraintScavenger, CallerLocationLeaseScavenge}})`, exactly as
   `scavenge.go:346-354`.  Every entry in such a bucket is expired, so no per-slot time
   check is needed.  Set `swept = s`.
2. For the bucket of `nowSec`: lock, scan, `Release` slots with `expiresAtMS < now`
   (strict), leave the entries (they are skipped as taken when the bucket drains whole).
3. `constraintapi.ScavengeResult{TotalExpiredLeasesCount, ReclaimedLeases}.Report(ctx, shardName)`
   and `HistogramConstraintAPIScavengerLeaseAge` per lease as `scavenge.go:338-344`.

Reclaim latency is at most one sweep interval (default 100ms; Redis today is a 500ms
tick).  Going through `Release` means hooks, metrics, the `rel` idempotency entry and the
forced manual-semaphore release (`release.lua:121-133`) all behave as in Redis.  Tests
call `clock.Advance(d)` then `Scavenge(ctx)`.

Housekeeping (every ~5s): `ttlMap.sweep(now)` on the three idempotency maps; free
eligible slab pages; delete zero-valued semaphore cells (`CompareAndDelete` after
`Load() == 0`; recreated by `LoadOrStore`) and expired GCRA cells.

### Idempotency (`ttlmap.go`)

```go
type ttlMap[V any] struct{ stripes [64]struct{ mu sync.RWMutex; m map[uint64]entry[V] } }
type entry[V any] struct { expiresAtMS int64; v V }
func (t *ttlMap[V]) get(now, key uint64) (V, bool)         // stripe = key & 63; expired reads as absent
func (t *ttlMap[V]) set(key uint64, v V, expiresAtMS int64)
func (t *ttlMap[V]) sweep(now int64)                        // delete expired, stripe by stripe
```

| map | key | value | TTL |
|---|---|---|---|
| `opIdem` | xxhash(`"acq:"` + key + `"-"` + fingerprint), `"ext:"`/`"rel:"` + key | stored response (`any`) | 5s; acquire `int(min(5s, Duration).Seconds())` |
| `checkIdem` | xxhash(idempotency key) | `struct{}` | 5min |
| `semIdem` | xxhash(`"setcap:"`/`"adjcap:"`/`"rel:"` + raw key) | `int64` | 60s |

A `singleflight.Group` keyed by the 8-byte hash string wraps every mutating operation so
concurrent callers with the same key share one execution (the guarantee in
`concurrency_race_test.go:465`).  Replays return a shallow copy of the stored response with
`OperationIdempotencyHit = true`.

### Acquire (`acquire.go`)

1. `req.Valid()`, request ID, `now`, `leaseExpiry`, latency metric as `acquire.go:175-230`.
2. `constraintapi.SortConstraints(req.Constraints)` in place; hashed lease keys via
   `util.XXHash`; `ResolveLimits` and cell key per sorted constraint; fingerprint = xxhash
   of the JSON of `[]SerializedConstraintItem` + IDs + config version + amount + hashed
   lease keys + run IDs + max lifetime + source.
3. `flight.Do(acqKey, ...)`: inside, `opIdem.get`; on hit return the stored response.
4. `skipGCRA = checkIdem.get(xxhash(req.IdempotencyKey))`.
5. **Snapshot pass**: atomic loads only, the exact `acquire.lua:225-302` arithmetic.
   `available <= 0` → status 2, no mutation, no cache, `RetryAfter` from the snapshot.
6. **Commit pass** with `granted = available`, counters first (exact rollback), then the
   ≤2 GCRA constraints: semaphore `fit = cell.take(capacity, w, granted)`, shrink and mark
   limiting; GCRA `cell.update` with `quantity = granted`.  When the helper refuses, the
   refusal is one of two things: with the TAT in the past, pass one saw the inflated
   remaining and `acquire.lua` grants without writing, which is kept; otherwise another
   request moved the TAT between the passes and the take is retried with what fits now,
   down to nothing.  `skipGCRA` skips as `acquire.lua:347`.  Roll earlier constraints back
   by `taken_i - granted`; a GCRA rollback moves the TAT back by that many emission
   intervals.  `granted == 0` → status 2.
7. **Report pass**: atomic loads for `Usage`, exhausted set, `retryAt` as
   `acquire.lua:397-422`; indices in sorted order.
8. Bookkeeping: `slab.alloc()` per granted lease (prefix of `LeaseIdempotencyKeys`), lease
   ID from `(leaseExpiry, seq, nonce)`, `expiry.add`, one shared `requestState{active =
   granted}`, `checkIdem.set` for the request key and each hashed lease key.
9. `opIdem.set` (status 3 only).
10. Outside the flight: hooks, metrics, `RetryAfter` clamping, status handling as
    `acquire.go:407-559`, pkg name `constraintapi.memory` in metric tags.

Under no contention the snapshot and commit passes see the same values and the result is
identical to the Lua.  The centralized acquire cache (`WithEnableAcquireCache`) is not
implemented; `constraintapi.NewConstraintCache` (`cache.go`) still wraps the memory manager.

### Check (`check.go`)

Snapshot pass only, with the `check.lua` rules: first constraint always limiting, `break`
at 0, per-kind usage formula, `RetryAfter` only from exhausted constraints.  Semaphores are
evaluated read-only (a deliberate improvement over `check.lua`'s missing branch; `Check`
conformance cases exclude semaphores).  The `chk` entry is written when
`checkIdempotencyTTL > 0` but never read, matching Redis.

### ExtendLease (`extend.go`)

`flight.Do(extKey)`: `opIdem.get`; `decode(req.LeaseID)`; `expiresAtMS < now` → status 1
before any lookup; foreign nonce, no page, or `!slot.take()` → status 2 (lost to a
concurrent release or sweep); new `slab.alloc()` sharing `slot.req`, ID at
`now + Duration`, `expiry.add`; `Usage` from loads for concurrency constraints only with
the stored limit; status 4; cache on 4 only.  Redis status 3 cannot occur (record and
request state are one allocation).  `MaximumLifetime` stored, not enforced, matching
`extend.lua`.

### Release (`release.go`)

`flight.Do(relKey)`: `opIdem.get`; decode; foreign nonce, no page, or `!slot.take()` →
status 1; else concurrency `give(1)`; semaphore `give(weight)` when auto or
`Source.Location == CallerLocationLeaseScavenge`; `req.active.Add(-1)`;
`page.live.Add(-1)`.  Response carries IDs, `CreationSource`, concurrency-only `Usage`
from loads.  Cache on 3 only.  Hooks outside the flight.

### SemaphoreManager (`semaphore.go`)

`SetCapacity` = `Store`; `AdjustCapacity` = `Add` then CAS clamp at 0; `ReleaseSemaphore`
= `give(weight)`; `GetCapacity` = two loads (unknown → `0, 0, nil`).  Each write is wrapped
in `flight.Do` with `semIdem` at `constraintapi.SemaphoreIdempotencyTTL`; replay returns
the stored value with `Applied = false`.  Same metrics as `redisSemaphoreManager.recordMetrics`.

### GCRA port (`pkg/constraintapi/gcra`)

Line-for-line port of `rateLimit`/`throttle`:

```go
type Result struct {
    Limited bool; Limit, Remaining int; Usage int64
    ResetAfter, RetryAfter, RetryAt, EmissionInterval, DVT, TAT, NewTAT, Increment, AllowAt, Diff, Next float64
}
func Throttle(tat float64, present bool, nowMS, periodMS float64, limit, burst, quantity int) (res Result, newTAT float64, ttlSeconds int64, store bool)
func RateLimit(tat float64, present bool, nowNS, periodNS float64, limit, burst, quantity int) (...)
```

All arithmetic and the stored TAT are `float64` (`emission = period/limit` is fractional
whenever `limit` does not divide `period`; Redis and miniredis both round-trip the double
exactly).  Integer conversions copy Lua: `toInteger(v) = math.Floor(v + 0.5)`,
`math.min(math.ceil(ttl/emission), limit)` unclamped below zero, TTL `%d` truncation with a
floor of 1.  Pure function, so `gcraCell.update` can call it repeatedly in its CAS loop.

### Known divergences from Redis

Documented in `README.md` and pinned in the conformance suite as memory-only expectations.

1. An expired lease counts toward usage until the sweeper reclaims it (≤ one sweep
   interval, default 100ms); `ZCOUNT` stops counting at the millisecond.  Reclaim itself is
   faster than today's 500ms tick.
2. Under contention a request may be under-granted for a few nanoseconds of another
   request's overshoot (never over-granted).
3. Lease IDs are sequential, not random (internal identifiers, not user facing).
4. `Check` evaluates semaphores; `check.lua` reports them as exhausted.

## Performance and memory targets

The original estimate was 5–10µs per uncontended `Acquire`, dominated by a JSON fingerprint,
against 50–200µs plus a network hop for the Lua acquire serialized on one Valkey thread.  The
measured hot path is now one hash over the request, one `sync.Map` lookup for the interned
constraint set, two loads and one RMW per constraint, one slab slot write, one bucket append
and three idempotency map writes: 0.8µs median single threaded, 0.45µs per op across 8 CPUs.
`Release`/`ExtendLease` decode the ULID, look up the page, CAS the slot and give the counters
back, no hashing beyond the idempotency key.  The p99 guard in `acquire_bench_test.go` runs
behind `CONSTRAINT_BENCH_GUARD`.

### Measured (semaphore acquire, Valkey 8 in Docker vs memory, Apple Silicon)

`go test ./pkg/constraintapi/memory/ -run '^$' -bench 'Acquire.*/semaphore/' -benchmem -cpu 1,8`
with `CONSTRAINT_BENCH_REDIS_ADDR` pointing at Valkey:

| | Valkey 1 CPU | memory 1 CPU | Valkey 8 CPUs | memory 8 CPUs |
|---|---|---|---|---|
| mean per op | 172µs | 2.0µs | 55µs | 0.45µs |
| p50 | 170µs | 0.79µs | 433µs | 1.7µs |
| p99 | 239µs | 3.3µs | 742µs | 9.0µs |
| allocs per op | 290 / 21 KB | 15 / 1.7 KB | 290 / 21 KB | 14 / 1.6 KB |

`BenchmarkBatch` (`-benchtime=1x -cpu 8`) runs one batch through 8 workers per CPU that pull
requests from a shared counter, so it measures the manager.  An earlier driver that started
one goroutine per request measured the Go scheduler instead: the 100k batch took 53ms that
way against 27ms through the pool, and was removed.

| batch | Valkey wall | memory wall | Valkey req/s | memory req/s | Valkey p99 | memory p99 |
|---|---|---|---|---|---|---|
| 10,000 | 371ms | 4.4ms | 27k | 2.26M | 3.9ms | 0.9ms |
| 100,000 | 3.88s | 27ms | 26k | 3.66M | 4.7ms | 0.18ms |

History at 8 CPUs, memory side: 0.52µs mean and 25µs p99 after the metrics pass, 0.47µs and
15µs after the contention pass (items 7 to 9), 0.45µs and 9µs after interning and compact
records (items 10 and 11).  The 10k batch is short enough that page and map growth and the
first GC cycle dominate it.

What got the hot path there, in order of payoff:

1. **Metrics off the hot path** (`stats.go`).  Every metrics call built a tag map, reflected
   over it, sorted an attribute set and formatted the metric name: half the allocated bytes
   and a fifth of CPU.  Counters are now atomics flushed once a second as one call per
   series.  Histogram observations are appended to one of 16 mutex protected buffers and
   recorded by one goroutine every 100ms through `metrics.PreparedHistogram`, a small
   addition to `pkg/telemetry/metrics` that binds an instrument to one attribute set so
   recording allocates nothing.  Values are exact, at most a second late; a full buffer
   drops and logs the count.
2. **Cells keyed by a 128 bit hash** of the identity tuple instead of the `Sprintf` key
   strings, two xxhash passes with different seeds.  No key strings are built anywhere.
3. **Fingerprint hashed field by field** into xxhash, no JSON, no
   `ToSerializedConstraintItem`.
4. **Striped mutexes per idempotency key** instead of `singleflight`.  A retry waits for the
   first call and replays its record.  This also matches Redis for rejected requests, which
   `singleflight` shared but Redis recomputes.
5. **Slot owners clear `req`** and **pages are freed once empty for 5s** (two housekeeping
   sightings) instead of 3h after creation, with 8k slot pages.  Before, a taken slot pinned
   its request state for the life of the page, so the live heap and every GC cycle grew
   with the allocation rate.
6. Request IDs from `math/rand/v2`, no closures or per call slices in the acquire body.
7. **One hash per key.**  Key bytes are appended into a caller's stack array and hashed once
   with `Sum64`, instead of a `Digest.Write` per field.  The buffer must not be a struct
   that slices its own array, or escape analysis moves it to the heap.
8. **No reload after `take`.**  `take` returns the counter value with the grant accounted,
   and the report pass recomputes from that and the capacity read in pass one.  Two loads
   and one RMW per constraint on the hot cell instead of five loads and one RMW.
9. **No single points of contention.**  Expiry buckets hold 16 padded slices chosen by seq.
   Sequence numbers come from 16 padded counters, the shard in the top byte of the seq, so
   each shard fills its own pages.  `page.live` is gone; housekeeping frees a page when
   every slot reads taken, which also needs no grace period.  Operation locks are 4096
   padded mutexes and the idempotency maps 256 padded stripes with plain mutexes.

10. **Interned constraint sets** (`constraintSet`).  One set per request shape, keyed by a
    128 bit hash of the IDs and of every constraint's identity and resolved limits in
    request order.  The key covers the limits rather than the config version, so a request
    carrying different limits under the same version gets its own set, the way Redis stores
    limits per request.  A request does one hash, one `sync.Map` lookup and atomics.  Sort,
    `ResolveLimits` for the set, and the cell lookups run only when a set is built, and the
    caller's constraint slice is no longer sorted in place.  Request state is ~64 B.
    Housekeeping drops a set no request used since its last round.
11. **Compact acquire records.**  The idempotency map holds seqs, indices and usage pairs
    (~150 B), and the response is rebuilt from the record and the replaying request.  The
    constraints in a response come from the request's own list, positioned by the set's
    evaluation order.

### Where the time goes now

At 8 CPUs about half of CPU is the garbage collector and the runtime scavenger's `madvise`,
driven by ~1.6 KB allocated per acquire: ~1.1 KB is the manager's (the 352 B
`CapacityAcquireResponse` is a third of it, then the acquire record, request state, leases and
usage slices) and the rest is idempotency map growth and the benchmark's own request.  The
gap between the 0.8µs median and the 2.0µs mean single threaded is GC assist charged to the
allocating goroutine.  The one contended RMW per request on a hot semaphore cell costs 50 to
100ns, which caps a single key near 10 to 20M req/s; nothing below changes that.

### Remaining optimizations

Ordered by expected payoff.  None changes the `CapacityManager` API unless stated.

- [ ] **Request state in an arena.**  Slots hold a `uint32` index into a slab of
  `requestState` instead of a pointer, so pages hold no pointers and the GC never scans them.
  The arena frees an entry when `active` reaches zero.  Removes one allocation per acquire and
  the scan cost of every live page.  Same design as the slot slab, one more free list.
- [ ] **Lossy or absent check idempotency.**  The two `checkIdem` inserts per acquire only let a
  retry skip GCRA.  Either time bucketed Bloom filters at ~2.5 B per entry, or plan item 10
  below (pass the prior lease ID on the item lease acquire) so no map is needed.  Removes two
  map writes and the largest map from the hot path.  The Bloom variant admits a false skip
  with the filter's error rate, which only means a retry is not throttled once.
- [ ] **Pool the usage slice.**  `Usage` is rebuilt per response from the record; a
  `sync.Pool` of `[]ConstraintUsage` capped at `MaxConstraints` avoids one allocation.  The
  response struct itself is the API's and cannot be pooled without a release call.
- [ ] **Soft memory limit.**  `GOMEMLIMIT` or `debug.SetGCPercent` where the manager runs.  A
  heap allowed to grow between cycles halves GC frequency and stops the scavenger returning
  and refaulting memory that shows as `madvise`.  Deployment configuration, not code.
- [ ] **Shard the manager by account.**  N independent managers selected by account hash,
  each with its own maps, slab, expiry index and stats.  Every cross account structure stops
  contending; the single hot key case is unchanged.  Lease IDs need the shard in the nonce.
- [ ] **Spread benchmark.**  Many keys and accounts.  The single key benchmark is the worst
  case for cache lines and the best case for map locality, so it hides map and GC costs a
  fleet pays.
- [ ] **Batch acquire for backlog refill.**  One lock, one hash prefix and one set lookup for
  up to `MaximumAmount` leases.  API addition on both backends.
- [ ] **Coalesce zero observations.**  `RequestLatency` is always 0ms in process and
  `RetryAfter` is 0 for every grant.  Counting them and recording n zeros in the drain cuts the
  observation buffers' work to the few non zero values.

Memory per lease (3-constraint request, `Amount = 1`):

| Structure | Bytes | Lifetime |
|---|---|---|
| slab slot | 24 | lease; the page is freed once every slot in it is taken |
| expiry bucket entry | 8 | until the bucket's second passes |
| `requestState` (shared by the request's leases) | ~64 | until all its leases release |
| `constraintSet` (shared by every request of one shape) | ~450 | while requests use it |
| `checkIdem` entries (1 + Amount) | 2 × ~40 | 5 min |
| `acqIdem` record (seqs, indices, usage pairs) | ~150 | 5 s |
| constraint cells | ~80 per distinct key | until zero |

About 350 B per lease, versus roughly 2–3 KB of Redis keys today (`ld:*` hash, `rs:*`
JSON, ZSET members, `ik:*` strings).

## Room to improve (not in this work)

Performance items live under "Remaining optimizations" above.  Items 1 to 4 of the original
list (interned sets, direct fingerprint hashing, compact idempotency values, hashed cell
keys) are done.  The rest, each local to one file and needing no API change unless stated:

1. **Roaring expiry buckets** (`expiry.go`): seqs per second compress to ~2 B/entry if
   the index ever matters; adds `github.com/RoaringBitmap/roaring/v2`.
2. **Millisecond reclaim** (`expiry.go`): a bounded drain of the current second's bucket
   on each operation would restore `ZCOUNT`'s exact expiry semantics.
3. **`CapacityLeaseScavenger`**: implementing it needs `scavengerOpt`/`scavengerOptions`
   exported in `constraintapi`; then `NewLeaseScavengerService` can drive the memory
   manager like the Redis one.
4. **Differential GCRA test**: load `lua/test/*.lua` into miniredis from the memory package
   and compare against the Go port on fuzzed inputs.
5. **Dense `checkIdem` via seq** (API addition on both backends): pass the prior lease ID
   on the ItemLease `Acquire` so the constraint-check idempotency set can be a bitmap of
   seqs, or be dropped.

## Implementation order (one commit each, tests green after every step)

Tests were written first against a stub skeleton, so every test in `memory/` and `gcra/`
existed and failed before any implementation.  Steps below that are still open have their
tests in place and red.

- [x] 1. Sync `docs/plans/006-in-memory-constraint-store.md` with this plan (checklist form).
   `constraintapi` exports: `SortConstraints`, `ResolveLimits` (+ table test asserting it
   matches serialized `Limit/Burst/Period` for every kind and scope),
   `SemaphoreIdempotencyTTL`.  Existing tests unchanged.
- [x] 2. `pkg/constraintapi/gcra` + `gcra_test.go` refactor (all 27 table cases run against
   the Lua in miniredis and the Go port through `forEachGCRARunner`; the non-divisible and
   past-TAT cases pin the Lua and pass on both): `gcraRunner` with Lua and Go
   implementations; tables per backend via `t.Run`; non-divisible and past-TAT cases.
- [x] 3. `memory/cells.go`, `ttlmap.go` + unit tests: `take`/`give` under `-race` with 16
   goroutines (never exceeds capacity, never negative, exact rollback), `gcraCell.update`
   CAS contention, TTL map expiry boundary and sweep.
- [x] 4. `memory/leaseid.go`, `slab.go`, `expiry.go` + unit tests: encode/decode round trip and
   timestamp preservation, foreign nonce, slot `take` exactly-once under 16 goroutines,
   page free eligibility, bucket drain with stale seqs, `swept` monotonicity.
- [x] 5. `memory/manager.go`, `semaphore.go`, `sweeper.go` skeleton (goroutine, `Close`):
   options, constructor, `SemaphoreManager`.  Port `TestSemaphoreManager`,
   `TestSemaphoreGetCapacity`, `TestSemaphoreAdjustCapacityClampsToZero`
   (`semaphore_test.go:353-684`) using `SetCapacity` instead of `r.Set`.
- [x] 6. `memory/acquire.go` + `memory/conformance_test.go` harness (every scenario green on
   both backends):
   `backend{cm CapacityManager; sm SemaphoreManager; scavenge func(ctx); clock *clockwork.FakeClock; advance(d)}`
   with miniredis (`advance` also calls `r.FastForward`/`r.SetTime`; `scavenge` calls the
   Redis `Scavenge`) and memory.  Scenarios: account/fn/custom concurrency, throttle, rate
   limit, semaphore auto/manual, weight > 1, mixed, partial grant, exhaustion with
   `RetryAfter`, semaphore-only exhaustion with zero `RetryAfter`, idempotent replay,
   status 2 not cached, `skipGCRA`.  Compare `len(Leases)`, `LimitingConstraints`,
   `ExhaustedConstraints`, `Usage`, `RetryAfter` (±1ms), `OperationIdempotencyHit`, and
   `LeaseID.Time()` == expiry on both backends.
- [x] 7. `memory/release.go` + `memory/extend.go`.  Conformance: extend then release with the
   old ID (status 1, no double decrement), extend expired (nil lease ID, nil error),
   idempotent replays, no-op not cached, `Usage` concurrency-only with stored limit,
   foreign-nonce lease IDs.
- [x] 8. `memory/sweeper.go` `Scavenge` + housekeeping.  Conformance: expired leases reclaimed
   after `advance` + `scavenge` on both backends, manual semaphore force-release (port
   `TestSemaphoreScavengeManualRelease`), acquire during scavenge (port
   `concurrency_race_test.go:338`), hooks fired once per reclaimed lease; a test drives
   time past all TTLs and asserts pages, cells, buckets and maps return to empty.
- [x] 9. `memory/check.go` (conformant with `check.lua` for rate limit, throttle and
   concurrency; semaphores are read like any counter; the write-only `chk` record is not kept).
- [x] 10. `memory/race_test.go` (`-race`) (green; `BenchmarkAcquire`, `BenchmarkAcquireRelease`
    and `BenchmarkBatch` measured on every shape, p99 guard behind `CONSTRAINT_BENCH_GUARD`,
    real Valkey via `CONSTRAINT_BENCH_REDIS_ADDR`): N goroutines across M accounts and K shared keys
    acquiring, extending, releasing with the sweeper running; invariants: usage == live
    lease count after a final scavenge, no negative counters, atomic high-water mark never
    exceeds the limit, concurrent identical idempotent requests receive identical leases
    (port `concurrency_race_test.go:23-113`, `:465-`).  `memory/acquire_bench_test.go`
    with the p99 guard and `-benchmem`.
- [ ] 11. `memory/README.md`; one sentence in `SEMAPHORES.md`.

## Verification

- `go test ./pkg/constraintapi/...` green at every commit; existing Redis tests untouched
  except the `gcra_test.go` runner refactor.
- `go test -race -cpu 1,4,16 ./pkg/constraintapi/memory/...`.
- `go test ./pkg/constraintapi/memory/ -run '^$' -bench 'Acquire' -benchmem -cpu 1,16`
  against the miniredis benchmark in `pkg/constraintapi`.
- `make lint`; `go mod tidy` reports no changes (no new dependencies).
- Manual: in a scratch `main`, construct `memory.NewManager`, acquire past a concurrency
  limit, wait past the lease duration, observe capacity return through `Check` and
  `GetCapacity` without calling anything else.
