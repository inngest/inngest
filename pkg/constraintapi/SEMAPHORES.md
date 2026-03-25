# Semaphores

Semaphores are a constraint kind (`ConstraintKindSemaphore`, kind=4) that use counters for O(1)
capacity checks.  They power worker concurrency and function concurrency.


## Naming convention

Semaphore names are always prefixed to avoid collisions:

- `app:<uuid>` — worker concurrency (one per app, capacity managed by connect lifecycle)
- `fn:<uuid>` — function concurrency (one per function)
- `hash:<xxhash>` — user-defined names (xxhash of user string)

## How they work

A semaphore has a capacity and acquired weight, split across two keys.  In Redis, this is:

```
{cs}:<account_scope>:sem:<name>:cap    → INT (total capacity)
{cs}:<account_scope>:sem:<name>:usage:<usagevalue>  → INT (current acquired weight)
```

NOTE: This means that we only store capacity once per raw expression, but have many keys for
each evaluated result of the expression.  We can always look up capacity for evaluated expressions
based off of the key prefix.

During `Acquire` (in `acquire.lua`), we read both keys.  If `capacity - usage < weight`,
the constraint is exhausted and no leases are granted.  Otherwise, `usage` is incremented by
`weight * granted`.

## Release modes

Unlike other constraints, semaphores can be manually released.  This is required for fn concurrency,
as semaphores must be held over the lifetime of a run, ie. many jobs.

This is controlled by `SemaphoreReleaseMode`:

- **`SemaphoreReleaseAuto` (0)** — The usage counter is decremented when the constraint API lease
  is released (`release.lua`).  Used for worker concurrency where each step independently acquires
  and releases a slot.  The constraint API scavenger handles expired leases: when it calls `Release`,
  `release.lua` decrements the counter automatically.

- **`SemaphoreReleaseManual` (1)** — The usage counter is NOT decremented when the constraint API
  lease is released.  Instead, it is decremented explicitly via `SemaphoreManager.ReleaseSemaphore()`
  during run finalization.  Used for function concurrency where the semaphore is held for the entire
  run lifetime.  Only the start job includes the semaphore constraint — subsequent steps do not.

NOTE:  If a constraint lease fails, scavenge will ALWAYS release semaphore capacity.

## Lifecycle

### Worker concurrency (auto-release)

1. Worker connects → connect lifecycle calls `SemaphoreManager.AdjustCapacity(+N)` with worker ID
   as idempotency key
2. Queue item is dequeued → `acquire.lua` checks capacity and INCRBYs usage
3. Step completes → `release.lua` DECRBYs usage (auto-release)
4. Worker disconnects → connect lifecycle calls `AdjustCapacity(-N)`

### Function concurrency (manual-release)

1. Function config specifies a semaphore with `release=manual`
2. Start job is enqueued with the semaphore in `Item.Semaphores`
3. Start job's semaphore constraints are stored in run metadata (`Identifier.Semaphores`)
   so they persist in Redis across the run lifetime
4. Start job is dequeued → `acquire.lua` checks capacity and INCRBYs usage
5. Step completes → `release.lua` sees `rel=1`, skips DECRBY
6. Subsequent steps do NOT include the semaphore constraint (the run already holds the slot)
7. Run finalizes → `Finalize()` reads semaphores from run metadata, calls
   `SemaphoreManager.ReleaseSemaphore()` with retry for each manual-release semaphore,
   using the run ID as the idempotency key
8. State is deleted after semaphore release

## SemaphoreManager

The `SemaphoreManager` interface handles capacity management and manual release.  All methods use
idempotent Lua scripts (check idempotency key → execute → set idempotency key with TTL).

Acquisition is NOT part of `SemaphoreManager` — it always happens in the `acquire.lua` pipeline
during constraint checks, keeping it consistent and atomic with other constraints.

## Releasing

- **Auto-release:** The constraint API scavenger finds expired capacity leases and calls `Release()`.
  `release.lua` handles the DECRBY for kind=4/rel=0.
- **Manual-release:** The run eventually reaches `Finalize()` (via completion, failure, or timeout
  cancellation), and the semaphore is released.
