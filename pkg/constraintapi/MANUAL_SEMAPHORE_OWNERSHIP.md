# Manual semaphore ownership rollout

Legacy manual semaphores keep a shared counter and a 60-second release marker.
Neither proves that a particular run acquired capacity. Never infer ownership
from `StartedAt`, and never reset a shared counter to make cleanup appear done.

The opt-in protocol records each run's acquired weight in the acquisition Lua
transaction. Finalization and force-release/scavenging consume the same record.
A released record is kept for seven days. A retry after that retention window
refuses to decrement positive usage if ownership cannot be established; it does
not become an ownership-blind release. Held records have no expiry.
An ownership-mode marker on the usage key prevents legacy fallback after a
released run's record expires. That marker is retained, not silently aged out.

## Ordering

1. Deploy the new acquisition, manual-release, and scavenger code everywhere,
   leaving acquisition tracking disabled (the default). Old release binaries
   must not consume tracked permits: they do not update ownership records.
2. Enable `WithManualSemaphoreOwnership()` on the selected capacity managers.
   First drain or explicitly quarantine legacy holds on the selected keys:
   once marked, untracked positive usage cannot be automatically attributed.
   Manual acquisitions then require a run ID for each lease. There is no new
   wire/RPC field: the protocol marker lives in internal Redis request state.
3. Use `ReleaseOwnedSemaphore` for recovery. Untracked legacy keys without a
   matching ownership record return `ErrSemaphoreOwnershipUnknown`, even when
   usage is zero: an old lease could still be scavenged later. No decrement occurs.
   This is a quarantine condition, not proof the run never held a permit.

Disabling acquisition tracking does not restore legacy release on marked keys;
new untracked holds there would fail closed. Do not disable it or roll
release/scavenger binaries back while those keys are active. These library changes do not enable the protocol in
Cloud or authorize a migration, counter reset, or cleanup of existing runs.

Old runs are not backfilled automatically. Their safe recovery still requires
an explicit hold, independently verified holders, and a recovery manifest.

## Start-job retries

With ownership enabled, new acquisitions also record which capacity request
currently claims each run's manual permit. Ordinary release keeps the permit
but marks that claim released, atomically with removing the old lease. A retry
of the same run can then reuse the recorded weight rather than acquiring it
twice. Renewing a lease does not change its capacity request ID.

This closes a retry deadlock: an unsuccessful start could release its capacity
lease while leaving its manual permit held. Its retry was blocked by that hold,
and the scavenger had no lease record left to reclaim.

An active claim cannot be reused. Other runs, changed weights, finalized or
scavenged runs, and untracked legacy holds remain blocked. Every other constraint
is still checked; owning a run permit does not supply worker or step capacity.
Batch acquisition only grants an eligible prefix and rejects duplicate runs or
duplicate manual semaphore constraints before writing.

Deploy retry-aware acquisition, ordinary release, finalization and scavenging
everywhere before enabling ownership. The internal request marker `msr` keeps
older ownership requests from being treated as retry-aware. Existing owner
records without a claim are not adopted, and existing legacy SYSOP jobs are
not repaired or replayed by deployment. Do not downgrade these participants
while retry-aware holds remain active. Releasing a large old backlog still
requires separate authorization and bounded pressure-monitored recovery.

## Performance before enabling

Tracking has a real Redis CPU and memory cost. Owner/claim reads and writes are
batched per manual constraint; tracking-off acquisitions do not perform those
operations or an extra usage read before INCRBY. Batching does not make the
additional records or their seven-day retention free.

Use a dedicated disposable localhost Redis/Valkey, with no other Lua callers:

```sh
SEMAPHORE_TEST_REDIS_ADDR=127.0.0.1:<test-port> ENV=test \
  go test -run '^$' -bench '^BenchmarkManualSemaphoreOwnership$' \
  -benchtime=1000x -count=3 ./pkg/constraintapi
```

The benchmark covers one job/one manual constraint and twenty jobs/three manual
constraints with tracking off/on. `redis-us/op` is server EVALSHA CPU from command
statistics; `ns/op` includes the Go caller and round trip. Cleanup of only the
benchmark's random account is outside the timer. Never use production or a
production port-forward. Compare revisions on the same quiet engine.

Before enabling an account, estimate active and retained ownership records using
its actual permit volume and key lengths, and check the shard's Redis CPU and
memory headroom. The benchmark is not a production tail-latency guarantee. Keep
the rollout small until real pressure and memory growth are understood; do not
shorten retention or remove correctness checks to meet a performance target.
