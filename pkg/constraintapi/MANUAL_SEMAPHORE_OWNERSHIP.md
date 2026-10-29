# Single-record retry experiment

Local experiment only, not an approved rollout or legacy-backlog repair.
The original two-record implementation remains on `fix/manual-semaphore-ownership`.

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

This removes the separate attempt key, the second request protocol marker, and
the strict recovery API. It does not remove RunID validation, batch-prefix
rules, other capacity checks, or the protection against caching a run-specific
refusal as shared exhaustion. Held admissions still have no expiry.

Default remains off. Compatible acquisition, ordinary release, finalization and
scavenger code must be installed before enabling. Buckets must be drained of
legacy holds first; missing ownership is not evidence that old usage is safe to
decrement. The permanent bucket marker still forbids unsafe legacy release after
a finalized record expires. Mixed legacy writers on an enabled bucket are not
supported. This experimental record format is incompatible with the unshipped
two-record draft. Do not enable either design in production from this branch.

Use the existing `BenchmarkManualSemaphoreOwnership` on a dedicated disposable
localhost Redis with `SEMAPHORE_TEST_REDIS_ADDR`. Compare the draft and experiment
on the same engine. Server CPU and retained memory still need measurement; fewer
records do not prove production latency is unaffected.
