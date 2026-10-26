# Verified cancellation recovery

`CancelRequest.RequireCleanupVerification` is opt-in for a bounded maintenance
operation, not an automatic production sweep. Ordinary cancellation is unchanged.
The caller must hold the target maintenance function and verify no work is
executing. Do not use this to cancel customer functions as part of SYSOP recovery.

The mode requires an ownership-aware semaphore manager (`ReleaseOwnedSemaphore`),
then drains and checks the run index before deleting metadata. A read, release,
dequeue or state-delete error is returned so the operator can stop and retry.
Missing metadata refuses cleanup: it cannot establish the target's permit list.
An intact metadata record with missing input can have its verified permits and
jobs removed, but returns an explicit exception and retains its metadata. It
does not fabricate completion events or claim full recovery.

Pass up to 1,000 `KnownQueueItemIDs` from the saved inventory to include jobs whose
run-index entry is missing. Every loaded item must match the target run, account,
environment and function. Its raw queue record is re-read after removal. The run
index alone cannot discover unknown orphan records: retain the independent
inventory and verify queue memberships externally before restoring the hold.

## Dependencies and remaining work

- Deploy ownership-aware acquisition, release and scavenging first; enable
  acquisition tracking only after legacy holds on selected keys are drained or
  quarantined. Old positive usage without ownership evidence fails closed.
- The separate existing PR inngest/inngest#4866 addresses preserving input/state
  when finish-event publishing fails. This PR does not duplicate that work and
  is not sufficient for a fully retryable publish/finalization protocol on its
  own. Do not broaden production cleanup until that path is resolved and tested
  with these changes.
- Cloud/SYSOP must adopt the new option and pass its verified inventory after
  vendoring the library changes. No existing event, service configuration, flag,
  queue, counter or account is changed by merging this library work.
- A fresh reconciliation still needs an end-to-end test after backlog recovery.
  Retain app identities and pause-window requests, and restore gradually; a
  cancellation receipt is not proof a replacement run completed.
