# Retiring obsolete maintenance runs

This is **not ordinary cancellation** and does not replace it. These helpers
remove obsolete internal maintenance jobs and run state under an external hold.
The old function-level semaphore namespace must no longer be used by current
code. **All semaphore counters and capacity configuration stay untouched.**
No acquisition/release/scavenger hot path or ownership protocol changes.

1. Save a complete independent inventory and a fixed run-creation cutoff.
   Preserve affected app identities and incoming requests for separately
   authorized replacement reconciliations. Do not replay automatically.
2. Hold only the target maintenance function. Verify dispatch has stopped,
   including expired execution leases. Verify all relevant constraint
   shards, including previous routing, have no indexed capacity lease for it.
   Unknown or missing lease details are not proof of absence. The caller must
   keep the hold and prevent old writers/configuration from returning.
3. Call `RetireMaintenanceRun` for each inventoried single-job run. It checks
   identity and the one-item run index before changing anything; accepts
   missing run metadata/input; refuses any queue lease, other manual semaphore,
   unknown state read, deferred work, or ambiguous/multiple index entries.
   A shard must implement `queue.MaintenanceShard`; a normal `Dequeue` is never
   used. Its maintenance script compares the last observed record and refuses
   any execution/capacity lease or nonempty function lease index atomically
   with deletion. An expired lease still blocks. A changed/vanished job is an
   error, not successful retirement. System-queue items are refused.
   It removes the job, verifies BOTH the run index and raw queue record, then
   deletes and verifies metadata/input. It never releases a semaphore or emits
   finish events. Record each result; errors after mutation may leave partial
   progress. Retry only the same manifest after inspecting the failure.
   Callers can require saved evidence of the retired manual semaphore to exclude
   modern runs. Already-absent jobs/state return absence, not cancellation proof.
4. Independently verify queue/status membership and history. A library return
   does not prove unknown records outside the supplied inventory disappeared.
   The caller may update maintenance history separately, AFTER verified state
   deletion; do not fabricate missing customer events or trigger parent runs.
5. Restore the previous hold state. A fresh reconciliation and any saved-event
   replay are separate operations requiring verification and authorization.

`CheckFunctionLeasesDrained` is read-only and bounded to 1,000 indexed leases
per account/shard. Larger or incomplete inventories refuse the check. It cannot
discover unindexed holders and is NOT proof that a counter is safe to delete.
No counter-deletion API or separate counter-cleanup phase is provided.

Clearing the obsolete backlog does not require garbage-collecting its unused
Redis counters. Retaining those counters avoids making assumptions about old
permit ownership. It does not make rolling back to old writers safe: those
writers could encounter the retained holds. Do not reactivate the obsolete
namespace or resume old-format queued work. Complete inventory, all-shard
availability and external quiescence remain operational prerequisites.
