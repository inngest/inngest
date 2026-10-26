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
3. Use `ReleaseOwnedSemaphore` for recovery, or opt ordinary callers into
   `WithStrictSemaphoreOwnership()`. Positive legacy usage without a matching
   ownership record returns `ErrSemaphoreOwnershipUnknown` without a decrement.
   This is a quarantine condition, not proof the run never held a permit.

Disabling acquisition tracking does not restore legacy release on marked keys;
new untracked holds there would fail closed. Do not disable it or roll
release/scavenger binaries back while those keys are active. These library changes do not enable the protocol in
Cloud or authorize a migration, counter reset, or cleanup of existing runs.

Old runs are not backfilled automatically. Their safe recovery still requires
an explicit hold, independently verified holders, and a recovery manifest.
