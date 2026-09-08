# Hobby Execution Hard Cap: OSS Implementation Guide

Spec: [Notion](https://app.notion.com/p/inngest/Hobbyist-Execution-Hard-Caps-3d1b64753bbd80b38a9aeaa095419ba9), detailed copy at `../monorepo/HARD_CAP_SPEC.md`.

Goal: Hobby accounts at 50k executions/month stop starting new runs. In-flight runs finish. This repo owns the gate, the error, and the UI label. Cloud owns counting, plan lookup, and enforcement flags.

## How it works today

- Every run-starting path (event, cron, batch, debounce, invoke, replay, checkpoint create) calls `executor.Schedule`, which calls `skipped()` (`pkg/execution/executor/executor.go:834`).
- `skipped()` checks paused, drained, then backlog size. Backlog is the template: `WithFunctionBacklogSizeLimit` opt, `BacklogSizeLimitFn` callback, cloud supplies the callback.
- A skip creates the run span and history, runs lifecycles, and returns the bare sentinel `ErrFunctionSkipped` (`executor.go:1888`). Callers cannot tell why.

## OSS PR

1. **Enum.** Add `SkipReasonAccountExecutionCapHit` to `pkg/enums/skipreason.go`. Run `go generate`.

2. **Executor opt.** Mirror the backlog opt.
   ```go
   type ExecutionCapDecision struct {
       Capped   bool
       ResetsAt time.Time
   }
   type ExecutionCapFn func(ctx context.Context, accountID uuid.UUID) ExecutionCapDecision
   func WithAccountExecutionCap(fn ExecutionCapFn) ExecutorOpt
   ```
   Nil callback or error means not capped. Cloud caches the decision, so call it on every `Schedule`.

3. **Placement.** Check the cap in `Schedule` before singleton handling (`executor.go:1365`), not inside `skipped()`. Singleton cancel mode cancels the in-flight run first. A capped account would lose a running run and get nothing new.

4. **Error shape.** Replace the bare sentinel return with a typed error that still satisfies `errors.Is`.
   ```go
   type SkippedError struct{ Reason enums.SkipReason }
   func (e SkippedError) Error() string      { return "function skipped: " + e.Reason.String() }
   func (e SkippedError) Is(target error) bool { return target == ErrFunctionSkipped }
   ```
   Return it from `handleFunctionSkipped`. Callers read the reason with `errors.As`. Keep `ErrFunctionSkippedIdempotency` as its own sentinel.

5. **Fix equality switches.** These compare with `==` and will stop matching:
   - `pkg/execution/runner/runner.go:713`. Without this, every skip logs as a scheduling error.
   - `pkg/api/apiv1/checkpoint.go:214`.
   Convert both to `errors.Is`.

6. **API errors.**
   - Checkpoint create: map `ErrFunctionSkipped` to 403 with the reason and reset date. It is 500 today.
   - REST v2 invoke (`pkg/api/v2/endpoints_function.go:271`): keep 422, name the reason instead of "paused or draining".

7. **UI label.** Add `AccountExecutionCapHit` to `ui/packages/components/src/utils/skipReasons.ts`.

8. **Dev server.** Env var `INNGEST_DEV_EXECUTION_CAP=1` wires a callback that always returns capped. Enough to demo the skip, label, 403, and 422 locally. No counter.

9. **Tests.** Capped request skips with the new reason. `errors.Is(err, ErrFunctionSkipped)` still true. Runner treats the skip as a no-op. Checkpoint returns 403. Invoke message names the reason. Singleton cancel mode does not cancel when capped.

## inngest-js PR

Checkpoint create retries five times with no status filter (`engine.ts:489`, `promises.ts:249`). Add `shouldRetry` that returns false on 4xx. Ship before enforcement or every capped sync request retries for seconds then throws.

## Cloud follow-up (monorepo, after vendoring)

- `pkg/usagecap`: copy `pkg/extendedtraces/cap.go`. Redis `INCRBY` + `EXPIREAT` per account per month, 30s decision cache, fail open.
- `IsExecutionCapped`: plan slug in `{PlanSlugHobbyFree}`, `executions` limit > 0, overage not allowed, LaunchDarkly flag on, counter >= limit. Read plan through entitlements, not the executor's uncleared account cache.
- Increment at the four `MetricCounterExecutionsTotal` sites. Fix the checkpoint function-finish double count first.
- Wire the opt beside `functionBacklogSizeLimit` (`pkg/execution/executor/executor.go:683`).
- Invoke: fail the parent via `HandleInvokeFailed` like the rate-limit branch (`pkg/execution/execution.go:363`).
- Add the proto state and `skipReasonState` case (`lifecycles/evt_lifecycle.go:568`). Until then the event page shows no state for this skip.
- Limit-hit metric hangs off `OnFunctionSkipped`, which already carries the reason. No new lifecycle hook.

## Not in the POC

Reconcile job, ClickHouse rollup, emails, dashboard banner, usage page changes.
