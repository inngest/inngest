# Defers

Schedule another Inngest function ("deferred run") from inside the current run. Reported by the SDK as `OpcodeDeferAdd` / `OpcodeDeferAbort`. Deferred runs are enqueued at parent-run finalize via `inngest/deferred.schedule` events.

## Key files

- [`pkg/execution/state/redis_state/lua/saveDefer.lua`](../pkg/execution/state/redis_state/lua/saveDefer.lua) — atomic write + cap enforcement
- [`pkg/execution/state/redis_state/lua/setDeferStatus.lua`](../pkg/execution/state/redis_state/lua/setDeferStatus.lua) — abort transition; releases input bytes from the aggregate budget
- [`pkg/execution/state/redis_state/lua/saveRejectedDefer.lua`](../pkg/execution/state/redis_state/lua/saveRejectedDefer.lua) — sentinel write for soft-fail
- [`pkg/execution/state/redis_state/redis_state.go`](../pkg/execution/state/redis_state/redis_state.go) — Go entry points: `SaveDefer`, `SetDeferStatus`, `SaveRejectedDefer`, `LoadDefers`, `LoadDefersMeta`
- [`pkg/execution/defers/defers.go`](../pkg/execution/defers/defers.go) — `SaveFromOp` / `AbortFromOp`: shared soft-fail ladder used by both executor and checkpoint paths
- [`pkg/execution/executor/executor.go`](../pkg/execution/executor/executor.go) — `handleGeneratorDeferAdd`, `handleGeneratorDeferAbort`
- [`pkg/execution/executor/finalize.go`](../pkg/execution/executor/finalize.go) — emits `inngest/deferred.schedule` for each AfterRun defer
- [`pkg/execution/driver/driver.go`](../pkg/execution/driver/driver.go) — `MarshalV1` populates the SDKRequest's `Defers` map (uses `LoadDefersMeta` to avoid loading inputs)

## Key types & constants

- `enums.DeferStatus` — `AfterRun` (will schedule), `Aborted` (user-aborted), `Rejected` (system-rejected). Aborted and Rejected are terminal.
- `consts.MaxDefersPerRun = 20` — distinct hashedIDs per run.
- `consts.MaxDeferInputSize = 4 MB` — per-defer input.
- `consts.MaxDeferInputAggregateSize = 4 MB` — total inputs across all defers in a run. Tracked separately from `DefaultMaxStateSizeLimit` so defers can never blow up the run's state-size budget.

## Lazy ops

Defers are reported as "lazy ops" (`OpcodeDeferAdd`, `OpcodeDeferAbort`); identified by `enums.OpcodeIsLazy`. Differ from normal step ops in five ways:

- **Piggyback on a host op.** They never travel alone; they ride alongside an op like `StepRun` or `RunComplete` that drives the run forward.
- **Processed inline.** No queue item, no follow-up SDK request. The executor handles them synchronously in `handleGeneratorDeferAdd` / `handleGeneratorDeferAbort` and they're done as soon as those return.
- **Excluded from the pending-step set.** `OpcodeGroups.IDs()` filters them out before `SavePending`. If they leaked in, then the run will hang (because they don't call `SaveStep`).
- **Don't count toward parallel-step gating or history grouping.** `nonLazyOpCount` skips them, so `ForceStepPlan` and per-step trace emission only see real steps.
- **Routed to the priority group.** `opGroups` puts them first so they drain before `RunComplete` finalizes and deletes state. If we don't process them first, there's a chance the defer state is deleted before building the `inngest/deferred.schedule` events.

## Storage layout

Three Redis hashes per run, all under a shared `{...:runID}` hash tag so multi-key Lua works in cluster mode:
- `defers-meta:{fnID}:{runID}` — `{hashedID → metaJSON}` (FnSlug, HashedID, ScheduleStatus)
- `defers-input:{fnID}:{runID}` — `{hashedID → rawInput}` (only for AfterRun)
- the run-metadata hash carries `defer_input_size` for aggregate-cap accounting

## Happy path

1. SDK emits `OpcodeDeferAdd` (lazy op piggybacked on a host op).
2. Executor `handleGeneratorDeferAdd` calls `SaveDefer` → `saveDefer.lua` writes meta + input atomically and bumps `defer_input_size`.
3. Future SDK requests carry the entry in `SDKRequest.Defers` (built from `LoadDefersMeta`, so multi-MB inputs aren't re-sent every step).
4. Parent run finalizes: `LoadDefers` returns all defers; `finalize.go` emits one `inngest/deferred.schedule` event per `AfterRun` defer. Other statuses are skipped.

## Abort path

1. SDK emits `OpcodeDeferAbort`.
2. Executor calls `SetDeferStatus(Aborted)` → `setDeferStatus.lua` flips the meta status, `HDEL`s the input, decrements `defer_input_size`.
3. Meta entry stays. `saveDefer.lua` is insert-only (any existing entry is a no-op), so SDK retransmits of the original DeferAdd dedupe automatically.
4. Finalize skips Aborted entries.

## Unhappy path (soft-fail)

A defer must never fail its parent run. Each rejection logs a warning, increments `IncrDefersRejectedCounter` (with a `reason` tag), and writes a `Rejected` sentinel where possible so SDK retransmits dedupe.

| Reason             | Trigger                                         | Sentinel?               |
| ------------------ | ----------------------------------------------- | ----------------------- |
| `per_defer_size`   | Single input > `MaxDeferInputSize`              | Yes (`SaveRejectedDefer`) |
| `aggregate_size`   | Run total would exceed `MaxDeferInputAggregateSize` | Yes (written by `saveDefer.lua`) |
| `per_run_count`    | Run already at `MaxDefersPerRun` distinct hashedIDs | No (no room). SDK retransmits absorbed until run completes |
| `invalid_opts`     | Malformed `DeferAddOpts` (missing FnSlug, etc.) | No                      |

## Todo

### Suppress SDK retransmits after `MaxDefersPerRun` is hit

**Problem.** Once a run reaches the count cap, `saveDefer.lua` can't write a Rejected sentinel for any new hashedID because the meta hash is full. The SDK keeps re-emitting `OpcodeDeferAdd` for that hashedID until the run finalizes. Wasted SDK to server traffic; not a failure.

**Proposed fix.** Add a flag (e.g. `defers_full bool`) to [`SDKRequest`](../pkg/execution/driver/request.go), set in `MarshalV1` when `LoadDefersMeta` returns `MaxDefersPerRun` entries. The SDK reads the flag and stops emitting any new `OpcodeDeferAdd` for the rest of the run.
