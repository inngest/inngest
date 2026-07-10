---
TITLE: Executor Readability Refactor
AUTHOR: Linell Bonnette
STATUS: Draft
---

# Executor Readability Refactor

## Overview

`pkg/execution/executor/executor.go` is **6,160 lines in a single file** and is
the most important file in the codebase: it drives the entire lifecycle of a
function run (scheduling, execution, drivers, pauses, cancel/resume, and the
generator-opcode state machine). It is not one god-function — it is ~90 methods
on `*executor` — but the sheer length makes navigation hard, and a handful of
individual functions are long enough that they are hard to reason about in
isolation.

This plan splits the work into two independent, sequenceable efforts:

1. **File decomposition** — move method clusters into sibling files within the
   same `package executor`. Pure code motion, zero behavior change.
2. **Function decomposition** — break the few oversized functions into named,
   single-purpose helpers, starting with `schedule` (750 lines).

Neither effort changes the package's public surface. Tier 1 is behavior-preserving
by construction. Tier 2 is intended to preserve behavior, but helper extraction
can change defer timing, span send/drop timing, early-return semantics, and nil
handling, so each extraction must preserve explicit invariants and be tested.

## Motivation

- The density comes from *two separate problems* that need *different fixes*:
  a too-long file (a navigation problem, fixed by splitting files) and a few
  too-long functions (a reasoning problem, fixed by extracting helpers).
- Go packages span multiple files freely. Splitting `executor.go` into
  `schedule.go`, `generator.go`, etc. — all still
  `package executor` — is a pure move: method receivers are unchanged, no
  exported/unexported boundaries shift, and **no caller imports change**.
- This convention is already established in this package: `service.go`,
  `finalize.go`, `constraints.go`, `reconstruct.go`, and `validate.go` are all
  `package executor`. This plan finishes a pattern the codebase already uses.

## Guiding Constraints

- **No behavior change in Tier 1.** Every commit is code motion only. A diff
  that moves a method must not edit its body; import-only edits are expected.
- **No intended behavior change in Tier 2.** Treat helper extraction as a
  behavior-risking refactor, not pure motion.
- **Preserve unexported helpers' locality.** When a cluster moves, its
  private helpers move with it unless shared across clusters (shared helpers
  stay in `executor.go` or move to purpose-specific files; do not dump unrelated
  helpers into `util.go`).
- **One cluster per commit** so each is independently reviewable and revertible.
- **Tests stay green after every commit.** Run
  `go test ./pkg/execution/executor/...` between steps.
- Keep comments as-is during motion; comment cleanup is out of scope here.

## Current Structure (reference)

Approximate method clusters in `executor.go` today (line ranges as of this
plan; treat as a guide, re-grep before moving):

| Cluster | Rough lines | Key symbols |
|---|---|---|
| Construction & options | 99–478 | `NewExecutor`, all `WithX`, config structs, `ScheduleStatus` |
| Struct & listener plumbing | 479–590 | `executor` struct, `AddLifecycleListener`, `SetFinalizer` |
| Scheduling | 591–1752 | `Schedule`, `schedule`, `idempotencyKey`, `createCancellationPauses`, `skipped`, `checkBacklogSizeLimit`, `handleFunctionSkipped` |
| Execution & drivers | 1753–2683 | `Execute`, `HandleResponse`, `run`, `executeDriverV1/V2`, `checkCancellation`, `functionFinishedData` |
| Pauses | 2684–3081 | `HandlePauses`, `handlePausesAllNaively`, `handleAggregatePauses`, `handlePause`, `HandleInvokeFinish` |
| Cancel / Resume | 3082–3552 | `Cancel`, `ResumePauseTimeout`, `Resume` |
| Generator state machine | 3553–5357 | `HandleGeneratorResponse`, `handleGeneratorOp` (dispatch) + every `handleGeneratorX` opcode handler |
| Batch | 5367–5560 | `AppendAndScheduleBatch`, `RetrieveAndScheduleBatch` |
| Misc / tracing / spans | 5561–6160 | `GetEvent`, `validateStateSize`, `ResumeSignal`, `execError`, span/metadata helpers |

The generator dispatch (`handleGeneratorOp`) is already well-factored: a clean
switch delegating to per-opcode handlers. It is the model to replicate, not
something to change.

## Phases

### Phase 1: File Decomposition (Tier 1 — pure code motion)

Target layout (all `package executor`):

```
pkg/execution/executor/
  executor.go       # executor struct, listener plumbing, now, central glue
  options.go        # NewExecutor, WithX opts, config structs, ScheduleStatus
  schedule.go       # Schedule, schedule, idempotency, cancellation pauses, skip/backlog
  execute.go        # Execute, HandleResponse, run, executeDriverV1/V2, checkCancellation
  pauses.go         # HandlePauses, aggregate/naive, handlePause, HandleInvokeFinish
  cancel_resume.go  # Cancel, Resume, ResumePauseTimeout, ResumeSignal
  generator.go      # generator dispatch and compact handlers
  generator_*.go    # optional split if generator remains too large by handler family
  batch.go          # AppendAndScheduleBatch, RetrieveAndScheduleBatch
  tracing.go        # span/metadata/timing/defer-span helpers
  finalize.go       # existing; function-finished event helpers live here
```

Shared helper ownership to decide before moving:

| Symbol(s) | Likely owner |
|---|---|
| `functionFinishedData`, `correlationID` | `finalize.go` |
| `createEagerCancellationForTimeout` | `cancel_resume.go` or `timeouts.go` |
| `shouldEnqueueDiscovery`, `hasPlanOp` | `generator.go` / `generator_discovery.go` |
| `fnDriver`, `extractTraceCtx` | `execute.go` |
| `validateStateSize`, `newExpressionEvaluator` | generator helper file |
| `createMetadataSpan*`, `emit*Span`, `updateDeferSpans` | `tracing.go` |

Each is its own commit:

- [ ] Move construction/options → `options.go`
- [ ] Move scheduling cluster → `schedule.go`
- [ ] Move execution/drivers cluster → `execute.go`
- [ ] Move pauses cluster → `pauses.go`
- [ ] Move cancel/resume cluster → `cancel_resume.go`
- [ ] Move generator state machine → `generator.go` or `generator_*.go`
- [ ] Move batch cluster → `batch.go`
- [ ] Move tracing/span helpers → `tracing.go`
- [ ] Confirm `executor.go` retains only the struct, listener plumbing, and
      genuinely shared helpers
- [ ] `go test ./pkg/execution/executor/...` and `make lint` green after each commit

Expected outcome: no file over ~1,900 lines, each answering one question, and
Tier 2 diffs become legible against a smaller surrounding file.

### Phase 2: Function Decomposition — `schedule` (Tier 2, flagship)

**Prerequisites (from measured baselines — `schedule` is only 40% covered):**

- [ ] Add a shared executor test-builder + driver/`FunctionLoader` fake to
      `pkg/execution/executor` (none exists today; unblocks characterizing the
      0%/thin functions)
- [ ] Add characterization tests for `schedule`'s branch exits (rate-limit /
      debounce / skip) before extracting
- [x] Add `BenchmarkSchedule` in `tests/execution/executor` and capture a
      `benchstat` baseline (354.7µs/op ± 4%, 661.3KiB/op, 3.455k allocs/op;
      `schedule_bench_test.go`)

`schedule` (currently ~lines 934–1684, 750 lines) is a mostly linear pipeline,
but it has fragile early returns and span/state cleanup. Prefer a private
per-call struct (for example `scheduleAttempt`) over helpers with long parameter
lists. Candidate seams (from existing in-function comments):

- [ ] `applyRateLimit` — rate-limit gate (~960–1027)
- [ ] `applyDebounce` — debounce gate (~1031–1051)
- [ ] `normalizeEvents` — event IDs, sessions, marshaling, session limit (~1057–1096)
- [ ] `buildRunConfig` — priority factor, span ID, `sv2.Config`, cron schedule (~1098–1230)
- [ ] concurrency/throttle/singleton setup (~1231–1302)
- [ ] `createRunState` — state creation (~1303–1321)
- [ ] `evaluateSkip` — paused/draining/backlog skip checks (~1322 onward)
- [ ] span emission + skip handling
- [ ] `enqueueStart` (async) vs sync run-mode branch (~1568–end)
- [ ] Reduce `schedule` body to orchestration only; verify tests green

Invariants to preserve during extraction:

- `sendSpans`/`Drop` timing and all early-return `(runID, metadata, error)` values.
- constraint → rate-limit → debounce → normalize → state → skip → span → enqueue ordering.
- queue duplicate cleanup, tombstone/idempotency handling, and singleton skip/cancel behavior.
- skipped runs send run spans before lifecycle handling; sync runs send spans and lifecycles without enqueueing.
- no helper silently owns lifecycle goroutines, span finalization, or state deletion unless named for that side effect.

### Phase 3: Function Decomposition — remaining large functions

Same treatment, in descending size order. Only extract where a clear seam
exists; do not force it.

- [ ] `Execute` (~434 lines)
- [ ] `Resume` (~223 lines)
- [ ] `handleGeneratorWaitForEvent` (~203 lines)
- [ ] `handleGeneratorInvokeFunction` (~185 lines)
- [ ] `handleGeneratorWaitForSignal` (~177 lines)
- [ ] `handleGeneratorAIGateway` (~171 lines)
- [ ] `HandleResponse` (~165 lines)
- [ ] `ResumePauseTimeout` (~142 lines)

## Non-Goals

- No changes to the `execution.Executor` interface or any exported symbol.
- No logic, ordering, or error-handling changes anywhere.
- No comment rewrites, renames of existing exported methods, or dependency
  changes.
- Not splitting `executor` into multiple packages — this stays one package.

## Safety & Verification

Tier 1 and Tier 2 need **different bars**. Go 1.25's per-iteration loop
variables already eliminate the classic loop-closure trap.

**Tier 1 (file motion) — effectively compiler-proven.** Go makes unused imports
and variables compile errors, so a dropped reference cannot build.

- `make test` (already `-race -count=1`) + `make lint` green per commit.
- Each commit is pure motion: moved lines plus import edits only, no body edits
  (`git diff --color-moved=blocks --ignore-space-change` review).

**Tier 2 (function extraction) — behavior can change silently; passing unit
tests does NOT catch these.**

- **`defer` audit (highest risk):** an extracted block's `defer` fires at the
  *helper's* return, not the caller's — silently changing `span.End()` / unlock
  timing and `recover` scope. 22 defers in the file, 3 in `schedule`. Keep
  defers in the caller unless proven safe.
- **Early / named returns:** rethread control flow; never lift `return`s
  verbatim into a helper.
- **Coverage first:** measure per-function coverage *before* extracting; where
  thin, add characterization tests that pin current behavior first.
- **e2e per change:** `make e2e-golang` exercises the real run lifecycle
  (sleep/wait/invoke/parallel/retry/cancel/throttle/singleton) — the actual
  engine net that unit tests miss.
- **Perf baseline:** no executor benchmarks exist today. Add one over
  `schedule`/`Execute` and diff `-benchmem` with `benchstat` before/after — this
  is the hottest path and alloc regressions are invisible to tests.

## Measured Baselines (2026-07-09)

Investigated before committing to the plan. These numbers are what make the
"tests pass" bar untrustworthy on their own.

### Unit-test coverage — `pkg/execution/executor` overall 21.9%

Per-target `go tool cover -func` (profile in scratchpad `exec_cover.out`):

| Function | Coverage | Verdict |
|---|---|---|
| `handleGeneratorStep` | 75.0% | WELL covered — safe-ish with existing tests |
| `ResumePauseTimeout` | 57.1% | THIN |
| `handleGeneratorAIGateway` | 52.7% | THIN |
| `Resume` | 48.6% | THIN |
| `schedule` | 40.4% | THIN |
| `Schedule`, `Execute`, `HandleResponse`, `handleGeneratorSleep`, `handleGeneratorWaitForEvent`, `handleGeneratorWaitForSignal`, `handleGeneratorInvokeFunction` | 0.0% | **DANGEROUS** |

**7 of 12 targets — including `Execute` and `HandleResponse` — have zero unit
coverage.** Do not extract from a 0% function without a characterization test or
e2e coverage backing it.

### e2e net — `tests/golang` is the primary safety net

- Run: Terminal A `make run` (dev server on :8288, `--tick=50`, `TEST_MODE=true`);
  Terminal B `make e2e-golang`, or targeted `go test ./tests/golang -v -count=1 -run 'TestSleep'`.
  Go suite needs only the Go toolchain — no Next.js/Docker. (`SDK_URL`/`API_URL`
  in the Makefile target are vestigial for the Go suite; it hardcodes `127.0.0.1:8288`.)
- Behavior→target mapping (use to scope `-run` while refactoring):
  Sleep→`TestSleep`; WaitForEvent/Resume/timeout→`TestWait|TestTimeout`;
  Invoke→`TestInvoke`; AIGateway→`TestStepInfer`; Step/parallel→`TestFunctionSteps|TestParallelSteps`;
  Schedule flow-control→`TestThrottle|TestFunctionWithRateLimit|TestDebounce|TestSingleton|TestConcurrency`;
  Cancel→`TestEventCancellation|TestPauseCancelFunction`.
- **e2e gap:** `handleGeneratorWaitForSignal` and `ResumeSignal` have **no
  `tests/golang` coverage** (0% unit coverage too). Refactoring signal handling
  has no net — add tests first or defer it.

### Perf baseline — feasible, nothing exists yet

- No executor benchmarks anywhere. `benchstat` not installed → run via
  `go run golang.org/x/perf/cmd/benchstat@latest base.txt new.txt`.
- `BenchmarkSchedule` is **Easy–Moderate**: put it in `tests/execution/executor`
  (not `pkg/...`, which lacks scaffolding), reusing `createInmemoryRedis`,
  `mockDriverV1`, `fakeLifecycle` from `executor_test.go`; the "in-memory" story
  is miniredis + sqlite, so it measures relative regressions, not pure allocs.
  `BenchmarkExecute` is Moderate (needs a pre-scheduled run per iteration).
- Recipe: `go test -run=NONE -bench=BenchmarkSchedule -benchmem -benchtime=1000x -count=6 ./tests/execution/executor > base.txt`, refactor, rerun → `new.txt`, `benchstat`.

### Characterization-test machinery — mostly already exists

*Revised 2026-07-09 after a parallel investigation of the actual test surfaces
(supersedes the earlier "must be built first" framing).*

- The in-package (`pkg/execution/executor`) unit tests use ad-hoc inline
  `&executor{...}` structs and one-method-override fakes (`mockRunContext`,
  `stubRunService`, `stubQueue`, `stubPauseMgr`). Good for pure helpers and
  handlers that take a `RunContext`; cannot reach `schedule`/`Execute`/`Resume`
  full paths (too many real collaborators).
- **The needed builder and driver fake already exist** — just in the sibling
  `tests/execution/executor` package: `deferTestInfra` (`defer_test.go`) wires
  sqlite cqrs + function loader, two miniredis instances, a real queue, and a
  real `pauses.Manager`; `newExecutor`/`newExecutorWithQueue`/`scheduleRun`
  build and drive a run; `mockDriverV1` returns canned generator/error
  responses. This harness already characterizes several "0%" functions
  end-to-end (`Execute` via `TestExecutorReturnsResponseWhenNonRetriableError`
  and `TestCapacityErrorRetriesWhenAttemptsExhausted`; `schedule`
  flow-control via `TestExecutorScheduleRateLimit`/`...BacklogSizeLimit`).
- **Net machinery investment is small**, not a new framework — see the
  Characterization Test Plan below (M1–M6). The critical caveat: `mockDriverV1`
  returns the *same* response on every call, so multi-attempt / retry-then-succeed
  scripting is not possible without a new scripted driver (defer until a test
  demands it).
- Cheapest early wins: the signal path (zero net) and the AIGateway failure
  branches (no failure-path coverage), both drivable with the existing harness
  plus small fakes.

## Characterization Test Plan (synthesized 2026-07-09)

Output of a five-way parallel investigation. Goal: only **load-bearing** tests —
each must catch a regression that helper extraction could silently introduce.
Tests that merely re-encode a switch, assert a just-set struct field, or test
that Go works are explicitly rejected below.

### Cross-cutting findings

1. **`sendSpans`/`Drop` timing is NOT directly observable** (the plan's
   highest-flagged risk). `tracing.DroppableSpan` is a concrete struct; both
   `Send()` and `Drop()` call the underlying `span.End()`, and the recording
   tracer captures span *creation*, not send/drop. Do not write span-timing
   assertions. Anchor the defer/early-return risk on **coupled observables** that
   change when timing is wrong: lifecycle hook not re-fired, event not re-sent,
   no duplicate state write, enqueue count. These are the load-bearing proxies.
2. **`checkCancellation` is effectively dead in OSS** — `WithCancellationChecker`
   is never wired outside tests. Do not test it.
3. **The signal path is testable now** — `deferTestInfra` wires a real pause
   manager implementing `PauseBySignalID`, so a full write→consume→enqueue signal
   round-trip is drivable. It has zero net (0% unit + 0% e2e), so it is the top
   priority.
4. **The generator handlers contain no `defer`s** — the defer-timing risk does
   not apply to them; their extraction risks are early returns, lifecycle firing
   conditions, and opcode state-write correctness.

### Prerequisite machinery (build once, shared)

| # | Addition | Notes |
|---|---|---|
| M1 | Move `deferTestInfra` + `mockDriverV1` + `createInmemoryRedis` → shared `infra_test.go` | pure file move within `tests/execution/executor` |
| M2 | Channel-based lifecycle recorder embedding `NoopLifecyceListener` | hooks fire in goroutines — buffer + drain-with-timeout; assert counts/args, not timing |
| M3 | Capturing queue wrapper (retain `[]queue.Item` + `at`; inject `QueueItemExistsError` on demand) | extends `enqueueCountingQueue` |
| M4 | `mockDriverV1` invocation counter | for "driver never called" asserts |
| M5 | Fake `exechttp.RequestExecutor` (single method) via `WithHTTPClient` | unblocks all AIGateway failure tests |
| M6 | SaveStep-capturing `RunService` wrapper | mirror existing `pendingCapturingState` |

### Tier A — zero safety net; land before extracting these

| Test | Surface | Pins |
|---|---|---|
| `TestSignalRoundTrip_WriteConsumeEnqueue` | integration | write signal pause → `PauseBySignalID` → consume → enqueue next edge; `SignalStepReturn` data plumbed through `ResumeRequest` |
| `TestResumeSignal_NoMatch_Expired_RacedLease` | inline unit (stub `pm.PauseBySignalID`) | grace-period delete decision + the `ErrPauseLeased`/`ErrPauseNotFound`/`ErrRunNotFound` swallow contract (`MatchedSignal=false`, nil err) |
| `TestWaitForSignal_Conflict_AlreadyExists_QueueExists` | inline unit (stub `pm.Write`) | three error dispositions: fail step / continue-to-enqueue / no-op replay |
| `TestAIGateway_RetryableFailure_NoSaveStepNoEnqueue` | integration (M5, M6) | early retry return bypasses `SaveStep` + discovery enqueue |
| `TestAIGateway_NonRetryable_WrapsErrorSavesEnqueues` | integration (M5, M6) | error-wrapped payload (`StateErrorKey`) falls through the shared SaveStep+discovery tail |

### Tier B — thin coverage; each maps to a concrete extraction seam

| Test | Surface | Pins |
|---|---|---|
| `TestHandleResponse_EmptyNoneOps_RedrivesDiscovery` | integration (M3) | first early-return in `HandleResponse`; stuck-run bug if reordered |
| `TestExecute_InternalDriverError_SkipsHandleResponse` | integration (M4) | `resp==nil && err!=nil` early return before `HandleResponse` (nil-deref/panic risk); may need the `ErrNoRuntimeDriver` path — verify V1 can yield nil resp |
| `TestExecute_FirstAttemptOnly_FiresFunctionStartedOnce` | integration (M2) | `item.Attempt==0` gate around start lifecycle + `StartedAt`/`RequestVersion` metadata |
| `TestResume_TimeoutVsEvent_DequeueDivergence` | integration (M2, M3) | `IsTimeout` early-return skips the timeout-job dequeue block |
| `TestResumePauseTimeout_Duplicate_LeavesPause` | integration | `ErrDuplicateResponse` short-circuits before both enqueue and `pm.Delete` (anti-double-execution) |
| `TestSchedule_QueueDuplicate_DeletesUnownedState` | integration (M3) | conditional `smv2.Delete` based on `keepState` (state leak vs loss) |
| `TestSchedule_SyncRunMode_CreatesStateWithoutEnqueue` | integration (M2, M3) | sync fork sends spans + lifecycle without enqueueing |

### Tier C — opportunistic (happy paths largely e2e-covered)

WaitForEvent expression-interpolation write-back (1a); idempotent-replay-no-double-lifecycle
for waitEvent/sleep/invoke (1b/3b/2a); invoke correlation-ID wiring (2b); sleep
discovery-span determinism (3a); `schedule` debounce + singleton-skip gates
(inline unit); `HandleResponse` completion + fail-early branches (P5/P6).

### Explicitly NOT worth testing

Opcode/duration/expires **parse-error** guards; nil / empty-string guards;
span-attribute string assertions; metrics-counter side effects; config-field
population (`SpanID`, cron, priority factor); coalesce-key formatting (already
pinned by `TestResumeCoalesceJobID` etc.); rate-limit / backlog skip (already
pinned by `TestExecutorScheduleRateLimit` / `...BacklogSizeLimit`);
`checkCancellation` (dead code in OSS); DAG>1 / `RequestVersion==0` /
`ef.Paused` one-line guards.

### Extraction gating (do not refactor ahead of coverage)

- `handleGeneratorWaitForSignal` + `ResumeSignal`: **no net at all** — Tier A must
  land first.
- `handleGeneratorAIGateway` failure/retry tail: 52% coverage is *all* the success
  path — land the two Tier A AIGateway tests before touching it.
- Sleep / WaitForEvent / Invoke **idempotency early returns**: e2e covers only the
  happy paths — land the relevant Tier B/C idempotency tests before touching those
  return points.

## Risks & Mitigations

- **Accidental body edits during motion.** Mitigate by moving one cluster per
  commit and diffing for content changes.
- **Shared unexported helpers.** Before moving a cluster, grep for callers of
  its private helpers; if used across clusters, leave them in `executor.go`.
- **Parameter soup in Tier 2.** Mitigate with small result structs or a private
  per-call state object instead of helpers with many positional arguments.
- **Hidden lifecycle/span ordering changes.** Mitigate by documenting invariants
  before extraction and keeping side-effecting helper names explicit.
- **Merge conflicts against active work on `executor.go`.** This file is
  high-churn; land Phase 1 quickly in small commits and rebase often, or
  coordinate a brief freeze.
