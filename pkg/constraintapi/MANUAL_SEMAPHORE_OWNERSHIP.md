# Manual permits and start retries

Tracking is opt-in and off by default. It does not repair legacy holds or
authorize backlog recovery.

With `WithManualSemaphoreOwnership()` enabled, one record per run and manual
constraint stores `weight:requestID`. The request ID already exists in the lease
details and stays unchanged through lease renewal.

- Acquire writes the record with the usage increment in the same Lua operation.
- Ordinary release changes it to `weight:` while deleting the step lease. The
  run still holds its admission, but its next start attempt may reuse it.
- A retry requires the same weight and an empty request ID. It claims the record
  without incrementing usage. A live attempt cannot be reused.
- Finalization or scavenging consumes the held weight once, then writes `0:`
  with the existing seven-day retention. Cleanup of a different active request
  is rejected before writes. Finalization never trusts its supplied weight.

Run IDs, eligible batch prefixes and all other capacity checks remain required.
Held admissions have no expiry.

Shared exhaustion cannot rule out reuse of an owned manual admission. Redis
acquisition excludes these constraints from its shared cache. In-process cache
wrappers must do the same using `WithConstraintCacheShouldCacheForAccount` for
the same enrolled accounts. Other constraints and legacy accounts keep caching.

Default remains off. Compatible acquisition, ordinary release, finalization and
scavenger code must be installed before enabling. Buckets must be drained of
legacy holds first; missing ownership is not evidence that old usage is safe to
decrement. The permanent bucket marker still forbids unsafe legacy release after
a finalized record expires. Mixed legacy writers on an enabled bucket are not
supported. This record format differs from the earlier unshipped two-record
draft; those records are not converted automatically.

Use the existing `BenchmarkManualSemaphoreOwnership` on a dedicated disposable
localhost Redis with `SEMAPHORE_TEST_REDIS_ADDR`. Compare revisions on the same
engine. Owned manual requests must reach Redis even when the shared limit is
full, so cache-hit avoidance cannot be assumed in load sizing. Fewer records do
not prove production latency is unaffected; check CPU and memory headroom before
enabling an account.
