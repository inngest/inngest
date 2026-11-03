# Verified cancellation recovery

`CancelRequest.RequireCleanupVerification` is opt-in for a bounded maintenance
operation, not an automatic production sweep. Ordinary cancellation is unchanged.
The caller must hold the target maintenance function and verify no work is
executing. Do not use this to cancel customer functions as part of SYSOP recovery.

The mode checks the complete bounded job list before touching permits or jobs.
It then requires an ownership-aware semaphore manager (`ReleaseOwnedSemaphore`),
removes the jobs, and verifies the run index and raw records before deleting
metadata. A read, release, dequeue or state-delete error is returned so the
operator can stop and retry. Failures after cleanup starts can leave partial
progress; permit release is idempotent. Metadata deletion is not attempted until
queue/permit verification succeeds; a state-delete error must still be inspected.
Missing metadata or input refuses cleanup before permits or jobs are touched.
Jobs with any lease marker (including an expired marker), or the executing job
itself, are left for separate recovery. This mode does not reconstruct missing
state or fabricate completion events.

Pass up to 1,000 `KnownQueueItemIDs` from the saved inventory to include jobs whose
run-index entry is missing. Every loaded item must match the target run, account,
environment and function. Its raw queue record is re-read after removal. The run
index alone cannot discover unknown orphan records: retain the independent
inventory and verify queue memberships externally before restoring the hold.

## Dependencies and remaining work

- Deploy ownership-aware acquisition, release and scavenging first; enable
  acquisition tracking only after legacy holds on selected keys are drained or
  quarantined. Untracked legacy keys fail closed even when usage is zero:
  an old capacity lease could still be scavenged later.
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
