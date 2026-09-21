---
TITLE: Durable and Replicated In-Memory Constraint Store
AUTHOR: Tony HB
STATUS: Draft
---

# Durable and replicated in-memory constraint store

## Context

`pkg/constraintapi/memory` (plan 006) holds every lease, counter and TAT in process memory.
Constraint services will run as a fleet handling hundreds of thousands of requests per second,
deployed as a StatefulSet with a disk, and must survive rollouts and crashes.  What a restart
costs today, from the consumers (`pkg/execution/queue/process.go`, `constraints.go`,
`pkg/connect/lifecycles/semaphore.go`, `pkg/execution/executor/finalize.go`):

| state lost | consumer behaviour | severity |
|---|---|---|
| app semaphore capacity (`AdjustCapacity` on worker sync) | reads 0, every item is semaphore limited, `lease_item.go:389` stops the whole partition; nothing resyncs it until the worker reconnects | fail closed, worst |
| manual release semaphore usage | no scavenger, no TTL, not derivable from leases | permanent drift |
| `AdjustCapacity` idempotency (60 s) | a retried `disconnect-` adjust double decrements, floored at zero | capacity drift |
| in-flight leases | extend returns no lease, `process.go:337` requeues with `AlwaysRetryError` and no attempt burn; a watchdog at the lease's expiry does the same without any call | work redone, bounded by lease duration |
| counters, TATs | usage restarts at zero, throttles burst | over-admission until in-flight work drains |
| releases of forgotten leases | fire and forget, never inspected (`process.go:359`) | invisible |

The durability layer must be exact for semaphore capacity, manual usage and semaphore
idempotency, and as fresh as practical for everything else.  Two capabilities are wanted and
share one mechanism:

- **checkpointing** to the local disk, fsynced every second by default, so a restart or
  rollout loses at most that window;
- **leader/follower replication**: a hot follower applying the leader's changes with
  millisecond lag, taking over when membership (memberlist, owned by the caller) says so.
  Writes go to the one leader; followers serve reads.  No consensus protocol.

The mechanism is a **redo log of physical mutations**.  Every mutating operation already reduces
to counter deltas, absolute TAT states, slot creates and takes, and idempotency writes.
Recording those costs tens of nanoseconds, replaying them is deterministic, and one stream
feeds both the local WAL and remote followers.  Periodic point-in-time snapshots bound the log.

Nothing in the repo does this today: no WAL, fsync, snapshot, memberlist or leader election
exists, and there is no gRPC server or client for the constraint API, only the proto,
`convert.go` and generated code; executors get the manager by in-process injection.  Serving
the API over the network, redirecting writes and wiring memberlist is the caller's layer.  This
work gives it `memory.ErrNotLeader` with the leader address and a `SetRole` hook.

## Decisions taken (with the user)

- Checkpointing lands first and replication second, on the same log.
- The network layer (constraint API server, redirect, memberlist) stays outside this repo.
- The default fsync interval is 1 s.
- Followers serve `Check` and `GetCapacity` from their own state, at most one replication lag
  stale.

## Design

### Why a physical redo log

Re-executing `Acquire` on a follower would not converge: outcomes depend on timing and
contention.  The follower applies effects.  Each effect is commutative, versioned, ordered
within its shard, or exactly-once, so no global order is needed; the log sequence number (LSN)
is a position for replay and catch-up.

| effect | record | follower apply rule |
|---|---|---|
| counter delta (usage cells, `AdjustCapacity`) | `cellKey`, the delta the CAS actually applied | atomic add with the take pattern: add, and if the cell read dead before the add, undo, relookup, retry |
| counter set (`SetCapacity`) | `cellKey`, value | store; ordered with adjusts by sharding capacity writes per cell |
| TAT state (rate limit, throttle, rollback, denied acquires included) | `cellKey`, `tat`, `expiresAt`, `version` | store if `version` is greater; skip if `expiresAt` has passed |
| lease created | `seq`, `expiresAtMS`, `setKey`, request state fields, check idempotency hashes, acquire idempotency record | place the slot at that seq, raise the shard counter to it; skip a tombstoned seq |
| lease taken (release, sweep) | `seq`, the applied deltas | take the slot, apply deltas; an unknown seq becomes a tombstone |
| extend | old `seq`, new `seq`, new expiry, request state fields, extend idempotency record | atomic on the follower |
| constraint set | `setKey`, IDs, config version, constraints with resolved limits | intern; duplicates are harmless |
| idempotency record | `(map, key)`, value, `expiresAtMS` | set; the map is part of the key because `Release` and `ReleaseSemaphore` share the `"rel"` op string |

Records carry hashes, never raw idempotency strings.  Constraint items in set records use the
existing `ConstraintItem` proto through `convert.go`.

**Applied deltas, not requested ones.**  `take` already knows what it applied; `give`,
`adjust` and `set` return only the new value today and must return the old one too, so a give
that clamped at zero logs what it did.  `gcraCell.update` must return the state it stored, or
nil, so a denied acquire whose TAT moved and rolled back inexactly still logs the final state.

**Sharding the buffers.**  Records for one lease chain (acquire, its extends, its release or
sweep) go to the shard chosen by the acquire key and stored on `requestState`, and the drainer
keeps per-shard order, so a release never overtakes its acquire and an extend record's request
state is at hand.  Records for a semaphore capacity cell go to the shard chosen by the cell
key, so `set` and `adjust` stay ordered.  Manual `ReleaseSemaphore` deltas may cross shards;
they commute.  A set definition is emitted in a shard the first time that shard logs a lease
using it (a per-set shard bitmap), inside the locked section, so it precedes the acquire.

### Hot path cost

One append of a ~64 B record into one of 64 padded shard buffers inside the stripe-locked
section, the same pattern and cost as `stats.record`.  The drainer takes the buffers every few
milliseconds, assigns LSNs, encodes, appends to the WAL and hands batches to follower streams.
Callers never wait on disk or network.  If the drainer falls behind, buffers grow to a hard cap
and the manager refuses writes rather than losing records.

### Snapshot

Every mutating path holds one of the 4096 stripe locks for its whole mutation section: the
acquire body, release, extend, the three semaphore writes, and the sweeper through release.
Set interning and zero-cell creation happen outside a stripe but are logically empty.
Housekeeping's kills and frees are logically empty but observable, so it runs under a
housekeeping mutex, and the whole sweeper tick and external `Scavenge` take that mutex too
because `Scavenge` blocks in `doRelease` on a stripe.  The snapshotter takes the housekeeping
mutex first, then stripes 0 to 4095, drains the buffers to the WAL, notes the last LSN as the
snapshot's, copies state skipping dead cells, and unlocks.  Reads continue.  Expected stall: 10
to 20 ms at 500k live leases; snapshots run every 60 s and at shutdown, and the 1 s durability
target comes from the fsync interval.

Snapshot contents: nonce, per-shard sequence high-water marks, every set referenced by a live
slot, live slots grouped by request with the request state fields, counter cells, TAT cells,
and the acquire, extend, semaphore and check idempotency maps.  The expiry index is never
copied; it is rebuilt from live slots on restore because `drain` removes buckets before the
releases they name run.  The release idempotency map is not kept: a replayed release finds no
slot and is a no-op, the same outcome.

### Restore and repair

Load the newest valid snapshot, replay WAL records above its LSN, truncate a torn tail on CRC
failure, rebuild the expiry index, then one `Scavenge` reclaims leases that expired while the
process was down through the normal path.  The nonce is restored, so lease IDs held by
executors stay valid.  A store with no snapshot draws a fresh nonce as today.

Repair runs on restore and on promotion: walk live slots to their request state, set and
resolved constraints; recount usage cells touched only by concurrency and auto-release
semaphores, one per concurrency slot and the weight per semaphore slot, and set them; clamp
any negative cell to zero; leave cells touched by any manual-release constraint as stored,
since release mode is not part of the cell key.  Then advance each shard's counter past the
highest seq seen for it.  This is the recount job noted in plan 006 and it also absorbs any
record lost at failover.

### Replication

The leader streams WAL batches over a gRPC server stream.  A follower connects with its last
LSN and the term it knows; the leader serves from retained segments or answers "snapshot
required", after which the follower fetches the latest snapshot and streams from its LSN.
Followers apply into their own manager in follower mode: mutating calls return
`memory.ErrNotLeader` with the leader address, constructed retryable because `queue.ShouldRetry`
(`queue.go:207`) permanently dequeues an item on a non-retryable `errs.InternalError`; reads
are served locally, at most one replication lag stale.  Followers run the idempotency map
sweeps and page frees but neither the sweeper nor cell and set kills, because the leader keeps
cells and sets alive through reads and denied acquires that log nothing.  Followers write
applied records to their own WAL and take their own snapshots, so a promoted follower has a
full local history and a restarting follower resumes from its own LSN.

Replication is asynchronous: the leader never waits.  The loss window on failover is the
replication lag, milliseconds, the same class as the fsync window on a crash.

### Failover and terms

Membership names the leader and hands the manager a term that increases on every change:
`SetRole(role, term, leaderAddr)`.  Promotion runs repair, rebuilds the expiry index, starts
the sweeper and one housekeeping round, continues LSNs, opens the WAL for appending, accepts
writes and serves the stream.  A leader that steps down stops writing before it stops serving.
A follower accepts a stream only from a term at least as high as any it has seen.  A demoted
leader rejoining resyncs from the new leader's snapshot rather than computing a divergence
point; it costs one snapshot transfer and is simple to get right.

What this does not give: a partition can leave two members believing they lead.  Each accepts
writes, their states diverge, and when it heals the lower term resyncs and the grants it made
in that window are forgotten.  For a constraint store that is bounded over-admission for the
partition's duration.  Raft would remove it at the cost of a round trip per write.  Accepted.

### What survives what

| event | leases | counters and TATs | idempotency | loss |
|---|---|---|---|---|
| crash, restore from disk | valid, same nonce | exact to the last fsync, then repaired | acquire, extend, semaphore, check | grants in the last second: their releases no-op, their extends fail once and requeue, counters under-count them until their work ends |
| clean rollout | valid | exact | exact | none: shutdown drains, fsyncs and snapshots |
| failover to follower | valid | exact to the last applied record, then repaired | same | grants in the replication lag |

## Files

- `pkg/constraintapi/memory/log.go`: record types, sharded buffers keyed per lease chain and
  per capacity cell, set emission bitmap, drainer, LSN, sink.
- `pkg/constraintapi/memory/snapshot.go`: housekeeping-then-stripes quiesce, encode, decode,
  restore, expiry rebuild, repair.
- `pkg/constraintapi/memory/wal.go`: segments (`wal-<firstLSN>.log`, 64 MB), framing
  `uvarint length, crc32c, protobuf`, group commit on the fsync interval, retention past the
  last snapshot, tail truncation; snapshot files `snap-<lsn>.pb` written to a temp name and
  renamed.
- `pkg/constraintapi/memory/follower.go`: apply rules, tombstones, follower mode, roles,
  terms, `ErrNotLeader`, promotion and demotion.
- `pkg/constraintapi/memory/replication.go`: stream server and follower client.
- `proto/constraintapi/v1/replication.proto`, a `buf.gen.yaml` line in the Makefile
  `protobuf` target; messages reuse `ConstraintItem` and `ConstraintConfig`.
- `cells.go`: `give`, `adjust`, `set` return the old value; `gcraState` gains `version`;
  `update` returns the stored state.  `slab.go`: `allocAt(seq)` and per-shard high-water
  marks.  `acquire.go`, `release.go`, `extend.go`, `semaphore.go`: record emission inside the
  locked sections; `requestState` carries its log shard.  `sweeper.go`: the housekeeping
  mutex around the tick.  `manager.go`: `WithDataDir`, `WithFsyncInterval`,
  `WithSnapshotInterval`, `WithRole`; remove the unused `WithCheckIdempotencyTTL`.

## Implementation order (one commit each, tests green after every step)

Milestone 1, log and checkpoint:

- [ ] 1. `cells.go`: `give`, `adjust`, `set` return the old value; `gcraState.version`;
   `update` returns the stored state.  `slab.go`: `allocAt(seq)`, per-shard high-water marks.
   Unit tests.
- [ ] 2. `log.go`: record types, sharded buffers, drainer, LSN, an in-memory sink.  Emission
   from `acquire.go`, `release.go`, `extend.go`, `semaphore.go`, `sweeper.go` inside the locked
   sections; `requestState.shard`.  Tests: the records of a load test, applied to a fresh
   manager by the follower rules, equal the source manager's state.
- [ ] 3. `follower.go` apply rules with tombstones and pending sets; adversarial interleaving
   tests (release before acquire across shards, clamped give, denied acquire with TAT churn,
   dead cell during apply).
- [ ] 4. `wal.go`: segments, framing, CRC, group commit on `WithFsyncInterval`, retention, tail
   truncation.  Tests for torn tails and rotation.
- [ ] 5. `snapshot.go`: housekeeping mutex around the sweeper tick and `Scavenge`, the quiesce,
   encode and decode, expiry rebuild, repair, `WithDataDir`, `WithSnapshotInterval`, the
   shutdown hook.  Test: snapshot during a load test equals a paused copy.
- [ ] 6. Restore on `NewManager` with a data dir; every conformance script with a snapshot,
   restart and restore inserted between steps still matches Redis; the crash test.
- [ ] 7. `BenchmarkAcquire` with the log enabled against plan 006's numbers.  Doc updates.

Milestone 2, replication:

- [ ] 8. `proto/constraintapi/v1/replication.proto`, Makefile line, generated code.
- [ ] 9. `replication.go`: the stream server as a `service.Service` on the
   `pkg/debugapi/service.go` pattern, the follower client, snapshot catch-up.
- [ ] 10. Roles: `WithRole`, `SetRole(role, term, leaderAddr)`, `ErrNotLeader` (retryable),
    follower housekeeping rules, promotion (repair, expiry rebuild, sweeper start), demotion
    resync from the leader's snapshot.  Tests: two managers in one process under load, follower
    equals leader after drain; promotion mid-load; demotion; pre-failover lease IDs extend on
    the new leader.

Milestone 3, operations:

- [ ] 11. Metrics for LSN, fsync latency, snapshot stall, follower lag, refused writes;
    retention tuning; the recount as a periodic repair job on the leader.

## Verification

- `go test -race ./pkg/constraintapi/memory/` including every new test above.
- Crash test: run the pooled batch driver against a manager with a data dir, `kill -9` at a
  random point, restart and restore, then check every lease recorded before the last fsync is
  known, every counter equals its recount, and no counter is negative.
- Two managers in one process: leader under load, follower state equals leader state after
  drain; promotion mid-load with repair; demotion resync; pre-failover lease IDs extend on
  the new leader.
- `BenchmarkAcquire` with the log enabled: the append stays under 50 ns and adds no
  allocation.
- Manual: two processes, a flag naming the leader, kill the leader mid-load, promote,
  confirm extends of pre-failover leases succeed on the new leader.
