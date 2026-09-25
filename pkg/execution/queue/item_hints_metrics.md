# Item hint metrics

These metrics depend on the buffered item-hint implementation. Enabling metrics
does not enable the optional hint source or change ordinary queue processing.

`inngest_fast_path_execution_latency` is a millisecond histogram of the queue
clock at worker start minus the finalized item's `EnqueuedAt`. It is emitted once
for a hinted dispatch after the worker waits until `AtMS`, immediately before
normal work begins. It includes backlog/capacity and intentional scheduled delay;
it does not subtract sojourn time like `queue_item_latency_duration`.

The in-memory dispatch marker is set only by the hint dispatch callback. Buffer
admission and successful leasing do not emit this histogram. Ordinary scanning,
continuations dispatched by scanning, and hints rejected before dispatch do not
emit it. Missing or future enqueue timestamps skip the sample without changing
execution. The endpoint is executor work start, not remote SDK execution start.

Tags are `queue_shard` and `queue_backend`, derived from the owned queue shard.
Explicit buckets resolve millisecond starts and delays through ten minutes.

`inngest_queue_item_hint_total` retains its existing eligibility and dispatched
outcomes and adds `queue_backend`. Failed leases formerly grouped as
`not_dispatched` now use the existing lease result: `already_leased`,
`no_worker_capacity`, `throttled`, `concurrency_limited`,
`custom_concurrency_limited`, `semaphore_limited`, `not_found`, `lease_contention`,
`lease_error`, or `dropped`. Other unsuccessful dispatches remain `not_dispatched`.
An active lease in the supplied snapshot is `already_leased`; another worker
winning the backend lease is `lease_contention`. Missing and generation-stale
items may both be `not_found`, matching the backend's existing result.

For rollout, graph p50/p95/p99 with sample counts and these outcomes. Verify that
the exporter and Datadog retain the shard/backend tags and millisecond units.
This histogram samples successful hinted starts only. Use existing ordinary
queue health as a guardrail; cohort-wide latency including scanning fallback is
still required to establish customer benefit. Functional metric tests do not
measure a production latency improvement.
