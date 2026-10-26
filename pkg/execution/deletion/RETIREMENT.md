# Retiring obsolete maintenance runs

This is **not ordinary cancellation** and does not replace it. These helpers
exist for an externally held, obsolete internal maintenance workload whose
old function-level semaphore namespace is no longer used by the current code.
No acquisition/release/scavenger hot path or ownership protocol changes.

1. Save a complete independent inventory and a fixed run-creation cutoff.
   Preserve affected app identities and incoming requests for separately
   authorized replacement reconciliations. Do not replay automatically.
2. Hold only the target maintenance function. Verify dispatch has stopped,
   including expired in-progress queue records. Verify all relevant constraint
   shards, including previous routing, have no indexed capacity lease for it.
   Unknown or missing lease details are not proof of absence. The caller must
   keep the hold and prevent old writers/configuration from returning.
3. Call `RetireMaintenanceRun` for each inventoried single-job run. It checks
   identity and the one-item run index before changing anything; accepts
   missing run metadata/input; refuses any queue lease, other manual semaphore,
   unknown state read, deferred work, or ambiguous/multiple index entries.
   It removes the job, verifies BOTH the run index and raw queue record, then
   deletes and verifies metadata/input. It never releases a semaphore or emits
   finish events. Record each result; errors after mutation may leave partial
   progress. Retry only the same manifest after inspecting the failure.
4. Independently verify queue/status membership and history. A library return
   does not prove unknown records outside the supplied inventory disappeared.
   The caller may update maintenance history separately, AFTER verified state
   deletion; do not fabricate missing customer events or trigger parent runs.
5. ONLY after the old backlog is gone and the function's partition/backlogs
   are empty, retire its obsolete keyed semaphore usage buckets. The dedicated
   helper checks all account lease entries atomically before each deletion,
   including expired entries and other functions referencing that bucket.
   A zero/missing counter never bypasses the check. It does not reset a live
   counter, delete other namespaces, or reconstruct historical ownership.
   The inert shared capacity configuration is retained.
6. Restore the previous hold state. A fresh reconciliation and any saved-event
   replay are separate operations requiring verification and authorization.

The lease inspection is bounded to 1,000 indexed leases per account/shard and
refuses larger or incomplete inventories. Drain/scavenge safely first; do not
raise limits or delete lease evidence to force a pass. There is no automatic
override for leased or unclassifiable work. All-shard availability, complete
inventory, external quiescence, and retirement of the old namespace are real
operational prerequisites, not guaranteed by an empty run index.

Do not roll back to old writers, resume old-format queued work, or reactivate
the retired semaphore namespace after removing its usage counters.
