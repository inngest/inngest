# Phase 0 — Recon

Read-only investigation. No feature code written.

Everything below is sourced from the tree at `ae52cb99a`. File:line references are load-bearing —
they are where the claims come from.

---

## 1. The data

### 1.1 Where the run/trace view gets its data

GraphQL, `POST /v0/gql`. The run page polls `useGetRun` → `GET_RUN` → a nested `RunTraceSpan` tree.

| Layer | Location |
|---|---|
| Query + fragment | `ui/apps/dev-server-ui/src/coreapi.ts:377` (`TRACE_DETAILS_FRAGMENT`), `:443` (`GET_RUN`), `:497` (`GET_RUN_TRACE`) |
| Generated documents | `ui/apps/dev-server-ui/src/store/generated.ts:1235`, `:1600` |
| Generated types | `ui/apps/dev-server-ui/src/store/generated-types.ts:756` (`RunTraceSpan`) |
| Fetch hooks | `ui/apps/dev-server-ui/src/hooks/useGetRun.ts`, wired via `src/components/SharedContextProvider.tsx` |
| Shared-package hook | `ui/packages/components/src/SharedContext/useGetRun.ts` |
| UI working type | `ui/packages/components/src/RunDetailsV4/types.ts:12` (`Trace`) |
| Renderer entry | `ui/packages/components/src/RunDetailsV4/RunDetailsV4.tsx:216` |
| GQL schema | `pkg/coreapi/gql.schema.graphql:618` |
| Resolvers | `pkg/coreapi/graph/resolvers/runs_v2.go:167,244,264`; field resolver `function_run_v2.resolver.go:169` |
| GQL shaping | `pkg/coreapi/graph/loaders/trace.go:149` (`convertRunSpanToGQL`) |
| Storage read + tree build | `pkg/cqrs/manager/cqrs.go:102` (`GetSpansByRunID`), `:523` (`mapRootSpansFromRows`), `:996` (`sorter`) |
| SQL | `pkg/db/sqlite/sqlc/queries.sql.go:1490`, `pkg/db/postgres/sqlc/queries.sql.go:1484` |
| Go span model | `pkg/cqrs/traces.go:39` (`OtelSpan`); attributes `pkg/tracing/meta/extracted_values_gen.go:18` |

### 1.2 Does the payload contain what a graph needs?

`pkg/coreapi/gql.schema.graphql:618`:

```graphql
type RunTraceSpan {
  appID: UUID!
  functionID: UUID!
  runID: ULID!
  run: FunctionRun!
  spanID: String!
  traceID: String!
  groupID: String        # "used for grouping spans together in the UI, e.g. retries"
  name: String!
  status: RunTraceSpanStatus!
  attempts: Int
  duration: Int
  outputID: String
  queuedAt: Time!
  scheduledAt: Time
  startedAt: Time
  endedAt: Time
  childrenSpans: [RunTraceSpan!]!
  stepOp: StepOp
  stepID: String
  stepInfo: StepInfo
  stepType: String!
  isRoot: Boolean!
  parentSpanID: String
  parentSpan: RunTraceSpan
  isUserland: Boolean!
  userlandSpan: UserlandSpan
  debugRunID: ULID
  debugSessionID: ULID
  debugPaused: Boolean!
  skipReason: String
  skipExistingRunID: String
  metadata: [SpanMetadata!]!
  response: RunTraceSpanResponseInfo
}

enum RunTraceSpanStatus { FAILED QUEUED RUNNING COMPLETED WAITING CANCELLED SKIPPED }
enum StepOp { INVOKE RUN SLEEP WAIT_FOR_EVENT AI_GATEWAY WAIT_FOR_SIGNAL }
```

| Need | Status |
|---|---|
| Step identity | Yes — `spanID`, `stepID`, `name`, `groupID` |
| Parent/child | Yes, structurally — nested `childrenSpans`. `parentSpanID` exists in schema but is **not requested** by the fragment; `parentSpan` is never populated by the loader (always null). |
| Start/end times | Yes — `queuedAt`, `scheduledAt`, `startedAt`, `endedAt`, plus computed `duration` |
| Attempt number | Yes — `attempts` (the max attempt on a rolled-up step) |
| Status | Yes |
| Opcode / step kind | Yes — `stepOp` + `stepType` + the `stepInfo` union |
| Outputs | **No, not inline.** `outputID` is an opaque base64 `cqrs.SpanIdentifier` (`pkg/cqrs/traces.go:520`); the body needs a second `runTraceSpanOutputByID` call returning `{ input, data, error }`. |

The UI's `Trace` type (`RunDetailsV4/types.ts:12`) is a deliberate **subset** — it drops
`parentSpanID`, `duration`, `traceID` and `runID`, matching the fragment.

**Two limits worth knowing.** The query hard-codes **5 levels** of `childrenSpans` nesting
(`coreapi.ts:443,497`); anything deeper is silently truncated. And there is a second, unused source:
REST `GET /v2/runs/{run_id}/trace` (`pkg/api/v2/endpoints_runs.go:460`) reuses the same
`GetSpansByRunID` + `ConvertRunSpan` pipeline with **unbounded** recursion and can inline
input/output via `include_output=true` — but its proto drops `attempts`, `groupID`, `stepInfo`,
`parentSpanID`, `scheduledAt`, `isRoot` and `isUserland`, so it is strictly worse for this work.

### 1.3 What is missing for parallelism — the decisive question

**Answer: fan-out is derivable today only as a heuristic. Fan-in is not directly recoverable, but
under the coalescing model it does not need to be.**

**(a) The trace tree is structurally flat.** Every step span is parented to the run root. All step
creation sites in `pkg/execution/executor/executor.go` pass `Parent: runCtx.RootSpan()` — `:4549`
(run), `:4637` (sleep), `:5063` (invoke), `:5291` (waitForEvent), `:5521` (waitForSignal). The shape
is always `executor.run → [step, step, …] → execution/attempt → userland`. There is no parent span
for a parallel batch, no sibling-group node, no branch marker. Children are then sorted purely by
wall-clock start (`pkg/cqrs/manager/cqrs.go:996`), so parallel and sequential steps are
indistinguishable by structure.

**(b) `groupID` is actively misleading here.** It maps to `job.group.id`
(`pkg/tracing/meta/attributes.go:144,214`) from `queue.Item.GroupID`, whose documented purpose is
per-step lineage across retries (`pkg/execution/queue/item.go:324`). And the executor explicitly
gives each parallel opcode its **own** group ID (`executor.go:3819`):

```go
if group.ShouldStartHistoryGroup {
    // Give each opcode its own group ID, since we want to track each
    // parallel step individually.
    runInstanceCopy.item.GroupID = uuid.New().String()
}
```

`ShouldStartHistoryGroup` is true precisely when the SDK response contained more than one opcode —
i.e. `groupID` **diverges exactly on fan-out**. It is the retry-correlation key, and the UI uses it
that way (`traceConversion.ts:352`).

**(c) Three real signals exist server-side; one reaches the client.**

1. **`response.step.ops`** — every opcode returned by one SDK response, `{Op, ID, Name}` each.
   `pkg/tracing/util.go:177`:

   ```go
   // always add all steps received as a debugging attie
   steps := make(meta.ResponseOps, len(resp.Generator))
   for n, s := range resp.Generator {
       steps[n] = meta.ResponseOp{Op: s.Op, ID: s.ID, Name: s.Name}
   }
   meta.AddAttr(rawAttrs, meta.Attrs.ResponseSteps, &steps)
   ```

   `len(steps) > 1` means that response planned a parallel batch, and the list *is* the membership.
   It is already persisted and already parsed into `cqrs.OtelSpan.Attributes.ResponseSteps`
   (`extracted_values_gen.go:97`). **The GQL loader simply never reads it** — `convertRunSpanToGQL`
   (`loaders/trace.go:200`) skips it. Exposing it needs `loaders/trace.go` + `gql.schema.graphql` and
   nothing else: no storage change, no executor change.

2. **`ParallelCoalesceKey`** — the true fan-in identifier. `pkg/execution/executor/util.go:23` is
   `xxhash(runID + sorted member step IDs)`; `executor.go:3775` sets it on every item in the batch,
   and the fan-in discovery job derives its ID from it so exactly one discovery survives:

   ```go
   // ParallelCoalesceKey is a stable key shared by all queue items in the same
   // parallel batch.  When set, all fan-in completions derive the same
   // discovery job ID so that only one succeeds via queue-level idempotency.
   ```

   It lives on `queue.Item` (`item.go:376`) and `state.Pause` (`pause.go:252`) only. `grep
   ParallelCoalesceKey pkg/tracing` returns nothing — **it is never written to a span**, so it never
   reaches CQRS, GraphQL or the UI. Same for `enums.ParallelMode {None, Wait, Race}`.

3. **Shared `queuedAt` — the one usable client-side signal.** All opcodes from one SDK response are
   stamped with a single shared handling timestamp, deliberately (`executor.go:3768`):

   ```go
   // One handling timestamp for the whole response: handlers run
   // concurrently below, so per-handler clocks would give parallel opcodes
   // distinct span queue times and a nondeterministic trace order.
   handledAt := e.now()
   groups.PriorityGroup.HandledAt = handledAt
   groups.OtherGroup.HandledAt = handledAt
   ```

   It reaches the span via `opcodeHandledAt` (`:3861`) → `tracing.AddTimingAttrs(attrs, handledAt,
   handledAt, …)` (`:4545`) and surfaces as `RunTraceSpan.queuedAt`. **Siblings of a parallel batch
   share a byte-identical `queuedAt`.**

   It is a heuristic, not an invariant. It cannot separate two batches planned in the same instant,
   and it breaks for checkpoint-fed opcodes where `HandledAt` is zero and the handler falls back to
   its own clock (`OpcodeGroup.HandledAt` doc, `executor/util.go:45`).

**(d) Why fan-in does not need its own signal, mostly.** When `Metadata.ShouldCoalesceParallelism`
holds (`state/v2/state_metadata.go:100` — SDK request version ≥ 2), discovery waits for the whole
batch before planning the next. So every member of level *N+1* genuinely depends on every member of
level *N*, and a synthetic join node between levels is **accurate rather than approximate**.
`ParallelMode.Race` is the exception: race mode skips coalescing and each branch schedules its own
discovery (`executor.go:3996`), so levels go jagged and the model degrades to a chain.

**(e) A fourth signal, noted for later.** OTel `follows_from` links are attached from each span to
its originating queue item (`pkg/tracing/tracer.go:201`), persisted into `spans.links`
(`tracer_sqlc.go:209,244`) and even selected by `getSpansByRunID` — then silently dropped, because
`mapSpanFromRow` (`cqrs.go:408`) never reads the key and `cqrs.OtelSpan`/`RawOtelSpan` have no
`Links` field. Real causal edges, written and discarded.

**(f) Prior art that this was a known concept.** The legacy path explicitly filtered planning spans
out (`pkg/run/trace.go:330`, "ignore parallelism planning spans"), and the loader still hides
discovery spans (`loaders/trace.go:239`: `showSpan := span.Name != meta.SpanNameStepDiscovery`).

**Decision for the PoC:** derive fan-out batches from `queuedAt` equality, UI-only, no backend
change — carrying a prominent `TODO(canvas)` naming `ResponseSteps` as the cheap correct fix and
`ParallelCoalesceKey` as the complete one. Phase 1 verifies the equality holds against a real
captured payload *before* the derivation is written; if it does not hold, this becomes a UI +
backend project and I stop and say so.

### 1.4 How are `sleep` and `waitForEvent` represented?

Both are a **single `executor.step` span** parented to the run root, created the moment the opcode is
handled — not a start/end pair.

**Sleep** (`executor.go:4595`): computes `until := e.now().Add(dur)`, stamps
`DynamicStatus = StepStatusSleeping`, then creates a follow-on `SpanNameStepDiscovery` span with
`StartTime: until`. Duration is carried as `step.sleep.duration`
(`meta.Attrs.StepSleepDuration`). `sleepUntil` is **not stored** — it is recomputed in the loader
(`loaders/trace.go:280`):

```go
case models.StepOpSleep:
    if span.Attributes.StepSleepDuration != nil {
        gqlSpan.StepInfo = &models.SleepStepInfo{
            SleepUntil: span.GetQueuedAtTime().Add(*span.Attributes.StepSleepDuration),
        }
    }
```

**WaitForEvent**: same shape, attributes `step.wait_for_event.{name,if,matched_id}` plus
`step.wait.{expiry,expired}`, mapped at `loaders/trace.go:288` into
`WaitForEventStepInfo { eventName, expression, timeout, foundEventID, timedOut }`.

**Gotcha:** `StepStatusSleeping` and `StepStatusWaiting` both collapse to GraphQL `WAITING`
(`loaders/trace.go:138`). **You cannot tell a sleep from a wait by `status` — you must read
`stepOp`.**

There is also a known in-tree quirk at `executor.go:2216`: *"This is going to drop any sleep
requests, because `DriverResponseAttrs` forces the drop field if `resp.IsDiscoveryResponse()` is
true."*

This matches the brief's instinct: sleeps and waits are **edges with duration**, not nodes.

### 1.5 How are retries represented?

**N sibling spans, rolled up twice — once server-side, once client-side.**

Server-side (`loaders/trace.go:403`): when the parent is `executor.step` or
`executor.step.discovery`, each non-userland `executor.execution` child is renamed `Attempt N` and
folded into the parent, lifting `endedAt`, `status` and the max `attempts`. A lone successful attempt
is collapsed away entirely (`:502`).

Client-side (`traceConversion.ts:352`, `collectRollupGroups`): buckets the run root's direct children
by `stepID` into `Map<attempt, Trace>`, additionally absorbing same-`groupID` spans that have an
`outputID` but no `stepID` (network failures, finalization). `rollupStepAttempts` (`:434`) then
synthesises a virtual `${stepID}-rollup` span spanning first-attempt `queuedAt` → last-attempt
`endedAt`, carrying the last attempt's status and output, with the attempts as children.

**Consequence:** the canvas derivation should run on **`traceRollup`'s output**, not the raw trace.
That inherits correct retry collapsing, finalization naming (`Finalization` vs `Function error`) and
the EXE-1992 duplicate-span fix for free, instead of reimplementing three subtle behaviours.

---

## 2. The code

### 2.1 Where the Dev Server UI lives, how it is built and run

`ui/apps/dev-server-ui` — **TanStack Start + Vite + TanStack Router, Tailwind v3**, pnpm workspace
(`ui/pnpm-workspace.yaml`). `ui/README.md` still says "Next.js apps"; it is stale. Shared library is
`ui/packages/components` (`@inngest/components`), aliased in `vite.config.ts` straight at
`../../packages/components/src` (source, not built).

```
go run ./cmd dev --no-discovery       # Go dev server, :8288
cd ui/apps/dev-server-ui && pnpm dev  # vite dev + graphql codegen watcher
```

`.env.development` sets `VITE_PUBLIC_API_BASE_URL=http://localhost:8288`. For production the UI is
built into the binary: `make build-ui` → `pkg/devserver/static/` → `//go:embed` in
`pkg/devserver/ui.go`.

**Transport:** `graphql-request` + RTK Query (`src/store/baseApi.ts`), with TanStack Query wrapping
the shared hooks. Not Apollo, not urql — `ui/apps/dashboard` uses urql, `dev-server-ui` does not.
Codegen: `codegen.ts` reads `../../../pkg/coreapi/**/*.graphql` and emits `store/generated.ts` +
`store/generated-types.ts`.

**The cross-app seam** is `SharedContextProvider.tsx`, which injects app-specific implementations
(`getRun`, `getRunTrace`, `getTraceResult`, `rerun`, `rerunFromStep`, `cancelRun`, `invokeRun`,
`pathCreator`, `booleanFlag`) plus `cloud: false`. The dashboard's equivalent sets `cloud: true`.
**`useShared().cloud` is the existing dev-only gate** — `RunDetailsV4/Actions.tsx:26` already uses
it.

### 2.2 The component to reuse on node selection

Chain: `routes/_dashboard/run/index.tsx` → `RunDetailsV4.tsx` → left column `Tabs` →
`TimelineV4Wrapper` → `Timeline` → `TimelineBarRenderer` → `TimelineBar`; right column `StepInfo` or
`TopInfo`.

**`RunDetailsV4/StepInfo.tsx:168`** is the panel:

```ts
export const StepInfo = ({
  selectedStep,          // StepInfoType = { trace: Trace; runID: string }
  pollInterval,
  tracesPreviewEnabled,
  debug = false,
  isDurableEndpoint,
}) => ...
```

It fetches its own input/output via `useGetTraceResult({ traceID: trace.outputID })`. **Hand it a
`Trace` and a `runID` and it renders.** For root-node clicks the counterpart is `TopInfo.tsx:96`.

`TimelineV4Wrapper` (inside `RunDetailsV4.tsx:66`) is the adapter to copy: `traceRollup` → build a
`Map<spanID, Trace>` via `traceWalk` → `traceToTimelineData` → `onSelectStep(spanID)` → look up →
`selectStep({ trace, runID })`. A graph view drops in exactly where `<Timeline data onSelectStep/>`
sits, emits the same `spanID` string, and the right-hand panel updates for free.

### 2.3 Selection state

Three layers, none of them URL params:

1. **Cross-component: a module-level pub/sub emitter.** `runDetailsUtils.ts:109` —
   `stepSelectionEmitter` with a `Set<Listener>`, exposed as `useStepSelection({ runID })` returning
   `{ selectedStep, selectStep }`. Listeners are filtered by `runID` so expanded rows in the runs
   table do not clobber each other. No context, no Redux, no router.
2. **Visual highlight:** plain local `useState` inside `Timeline.tsx:861` (`selectedStepId`),
   deliberately duplicated. Also local there: `expandedBars: Set<string>` and
   `viewStartOffset`/`viewEndOffset` for brush zoom.
3. **Which run is open:** `useSearchParam('runID')` (`hooks/useSearchParams.ts`, TanStack Router).

**Selected step is not deep-linkable today.** If the canvas should preserve selection in the URL,
`useSearchParam` is the existing idiom.

### 2.4 Primitives, tokens and layout conventions

- **Tokens:** `AppRoot/tokens.css` — 410 lines of `:root` / `.dark` custom properties as
  space-separated RGB triplets (palette ramps `--color-carbon-*`, `--color-matcha-*`, … plus
  semantic `--color-background-canvas-base`, `--color-foreground-base`, `--color-border-subtle`).
  Split out in `ae52cb99a`. Imported by `AppRoot/globals.css`, loaded in dev-server-ui via `?url`
  links in `routes/__root.tsx`.
- **Styling:** Tailwind v3, class-based dark mode, **no CSS modules, no CSS-in-JS**.
  `tailwind.config.ts` in `packages/components` is the shared preset and maps every token to a
  Tailwind colour — notably `status.{failed,running,queued,completed,cancelled,paused}`,
  `borderColor.{subtle,muted,contrast}`, `backgroundColor.{canvasBase,canvasSubtle,surfaceBase,…}`,
  `textColor.{basis,subtle,light,muted}`. Rule from `ui/README.md`: never raw colours (`bg-white`),
  always tokens (`bg-canvasBase`). Merge with `cn` (`utils/classNames.ts`, tailwind-merge).
- **Primitives:** `Button`, `Pill`, `Card`, `Modal`, `Tooltip`, `HoverCard`, `DropdownMenu`,
  `Popover`, `Select`, `Switch`, `SegmentedControl`, `Skeleton`, `Table`, `Resizable`.
  Domain-flavoured and directly relevant: `Status/statusClasses.tsx`
  (`getStatusBackgroundClass`/`getStatusTextClass`), `Status/StatusDot.tsx`,
  `DetailsCard/Element.tsx`, `NewCodeBlock`, `icons/` (`@remixicon/react`).
- **Directly reusable for later phases:** `RunDetailsV4/TimeBrush.tsx` is a working draggable range
  selector with handles, drag-to-move, click-to-create and a reset button — the Phase 3 scrubber.
  `RunDetailsV4/utils/timing.ts` holds `TIMELINE_CONSTANTS` (`ROW_HEIGHT_PX: 28`,
  `INDENT_WIDTH_PX: 20`, `MIN_BAR_WIDTH_PX: 2`, `TRANSITION_MS: 150`).

### 2.5 Testing setup, and what actually runs in CI

**Runner: vitest everywhere. No jest, no playwright, no cypress** (the only playwright entries in
`ui/pnpm-lock.yaml` are transitive storybook deps).

- `ui/packages/components`: `vitest.config.ts` with `@vitejs/plugin-react` and `environment:
  'jsdom'`; `@testing-library/react ^16.3.0`. **No `@testing-library/jest-dom`, no setupFiles** —
  tests use raw vitest matchers.
- `ui/apps/dev-server-ui`: has a `test` script but **zero `.test.*` files and no `vitest.config.ts`**.
- Storybook exists only in `ui/packages/components/.storybook/` (17 stories, all generic primitives;
  none for `RunDetailsV4`).

**CI (`.github/workflows/`):**

| Workflow | What it runs |
|---|---|
| `components_test.yml` | `pnpm lint` **and `pnpm test`** in `ui/packages/components` — only on changes under that path |
| `dev_server_ui.yml` | `pnpm build` (which includes `tsc --noEmit`). No lint, no tests. |
| `dashboard_ui.yml` | `pnpm type-check` only |
| `ui.yml` | `pnpm audit` only |
| `e2e.yml` | has `paths-ignore: ui/**`, so it does not fire on UI-only changes |

**`ui/packages/components` is the only place UI tests run in CI. There is no browser or e2e UI job at
all.** That is where the canvas derivation and its tests must live.

**Gotcha:** `Timeline.test.tsx` has to `vi.mock('../Button', …)` because self-referencing
`@inngest/components/*` imports do not resolve under vitest.

Existing trace-view tests (~3,900 lines) live in `RunDetailsV4/`: `Timeline.test.tsx`,
`TimelineBar.test.tsx`, `TimeBrush.test.tsx`, `StepInfo.test.tsx`, `types.test.ts`,
`utils/traceConversion.test.ts`.

### 2.6 Existing graph / timeline / DAG visualisation, and available dependencies

**No graph or layout library anywhere in `ui/`.** Searched every `package.json` and the lockfile for
`reactflow`, `@xyflow/*`, `dagre`, `elkjs`, `cytoscape`, `vis-network`, `d3` — zero direct
dependencies. The only `d3` hits are `@types/d3-*` transitives of `recharts@2.4.3`.

What does exist:

- **`RunDetailsV4/`** — a hand-rolled waterfall. `TimelineBar.tsx` is 100% absolutely-positioned,
  percentage-sized Tailwind `<div>`s with `BAR_STYLES` keyed off status colours. **No SVG at all.**
  Documented in `RunDetailsV4/README.md`.
- **`RunDetailsShared/`** — the older inline span waterfall (`InlineSpans.tsx`, `Span.tsx`), largely
  superseded.
- **`Chart/Chart.tsx`** — an ECharts wrapper with token-colour resolution.
- Inline `<svg>` appears in 61 files, essentially all of them `src/icons/*`. **There is no
  hand-rolled SVG diagram anywhere in the repo.**

Usable without a new dependency: `echarts ^5.5.1` (has `series-graph` with fixed x/y layout and
`series-tree`), `framer-motion ^10.16.4`, `react-use ^17.4.0` (`useMeasure`, `useMouse`),
`@remixicon/react`, `date-fns`, the Radix hover-card/tooltip/popover set.

---

## 3. The fixtures

### 3.1 The SDK app

`tests/js` — a private Next.js 15 app (`inngest-js-tests`, `inngest@3.11.1-pr-411.4`), served at
`http://127.0.0.1:3000/api/inngest`, run with `pnpm dev`. Handler
`tests/js/src/pages/api/inngest.ts`. It is the **only** SDK app in the repo; there is no `examples/`
directory.

| File | Trigger | Produces |
|---|---|---|
| `sdk_function.ts` | `tests/function.test` | no steps — single root span |
| `sdk_step_test.ts` | `tests/step.test` | `step.run` ×2 with a **`step.sleep("2s")`** between |
| `sdk_parallel_test.ts` | `tests/parallel.test` | **`Promise.all([a, b, c])` then a sequential `d`** — exactly the fan-out/fan-in shape |
| `sdk_retry_test.ts` | `tests/retry.test` | **step-level failure + retry**, then a function-level failure + retry |
| `non_retryable.ts` | `tests/no-retry.test` | `NonRetriableError` caught by userland — step fails, run succeeds |
| `sdk_cancel_test.ts` | `tests/cancel.test` | `step.sleep("10s")` + `cancelOn: cancel/please` |
| `sdk_wait_for_event_test.ts` | `tests/wait.test` | **`step.waitForEvent` with a 10s timeout** and an `if` expression |

Plus `src/app/api/durable/{sync,async}/route.ts` on `inngest@4.3.0` (durable endpoints).

**The gap: there is no `step.invoke` / nested-function case in `tests/js`.** Invoke exists only
Go-side in `tests/golang/invoke_test.go`. Nor is there an agent-style loop.

So the app covers parallel, sleep, waitForEvent, retry, failure and cancel — six of the eight shapes
the brief asks for — and is missing invoke/nested and loops.

### 3.2 The better capture harness

`tests/golang` needs no external app: `golang_sdk_util.go`'s `NewSDKHandler(t, appID, …)` spins an
in-process `inngestgo` client and HTTP server and self-registers against
`http://127.0.0.1:8288/fn/register`. `tests/client/function_run.go` already has `WaitForRunTraces`,
`RunTraces`, `RunSpanOutput` and `RunTrigger`. Existing coverage includes `parallel_test.go`
(`group.Parallel` over 4 concurrent `step.Run`), `sleep_test.go`, `wait_test.go`, `invoke_test.go`,
`retry_test.go`, `step_error_test.go`, `failure_test.go`, `cancel_test.go`.

**Recommendation: capture fixtures from `tests/golang` rather than adding functions to `tests/js`.**
It leaves the shared e2e app untouched, gives us the `step.invoke` shape `tests/js` cannot produce,
and is deterministic and repeatable in one `go test` invocation.

### 3.3 Existing fixtures

**There are no trace/run JSON payload fixtures anywhere in `ui/`.** Every `RunDetailsV4` test builds
data inline via TS factories (`createTrace(overrides)` in `traceConversion.test.ts`, object literals
in `Timeline.test.tsx`).

The only committed JSON test data is `utils/historyParser/testData/` — 9 files
(`parallelSteps.json`, `sleeps.json`, `timesOutWaitingForEvent.json`, …) for the **legacy v1 history
format**, dated 2023, unrelated to traces. And `ui/apps/dev-server-ui/mock/` is dead code — nothing
imports it.

So committing real captured payloads is new ground here, and worth doing properly.

---

## 4. Proposed architecture

**Where the graph derivation lives:** `ui/packages/components/src/RunDetailsV4/canvas/graph.ts` — a
pure TypeScript module with no React import, depending only on `../types` and `../runDetailsUtils`.

Three reasons for that location. It sits beside `utils/traceConversion.ts`, which is the same class
of thing — a `Trace` → view-model transform — and which we **chain off rather than duplicate**. It is
the only package where CI actually runs tests, which matters because this is the tier where the
correctness lives. And it keeps the whole feature inside one directory plus one conditional in
`RunDetailsV4.tsx`, so `rm -rf canvas/` and deleting four lines removes it cleanly.

**The pipeline reuses everything that already exists:**

```
useGetRun                      existing
  → traceRollup(trace)         existing — attempts, finalization, EXE-1992
  → toCanvasGraph(rolled)      NEW, pure, Phase 1
  → toFlowElements(graph)      NEW, thin, Phase 2
  → <Canvas onSelectStep>      NEW, Phase 2
  → selectStep({trace, runID}) existing emitter
  → <StepInfo>                 existing, untouched
```

**Placement:** a second tab beside "Trace" inside `RunDetailsV4.tsx`, gated on
`!useShared().cloud` — the idiom `Actions.tsx` already uses. The right-hand column is not touched at
all, which is what makes "same panel, same data, no parallel implementation" free rather than
expensive.

**The model** (`canvas/graph.types.ts`):

```ts
type CanvasNode = {
  id: string;              // spanID for real nodes; synthetic for trigger/result/join
  kind: 'trigger' | 'step' | 'invoke' | 'join' | 'result';
  label: string;
  status: 'QUEUED'|'RUNNING'|'COMPLETED'|'FAILED'|'CANCELLED'|'WAITING'|'SKIPPED';
  level: number;           // x — 0 is the trigger
  lane: number;            // y — index within the level
  spanID?: string;         // → traceMap → StepInfo
  stepID?: string | null;
  stepOp?: string | null;
  attempts: number;
  queuedAt: number; startedAt: number | null; endedAt: number | null;
  batchKey: string | null; // fan-out identity (derived — see §1.3)
};

type CanvasEdge = {
  id: string; from: string; to: string;
  kind: 'sequence' | 'fanOut' | 'fanIn';
  // sleep / waitForEvent are edges with duration, not nodes
  waitKind?: 'sleep' | 'waitForEvent' | 'waitForSignal';
  waitMs?: number;
  spanID?: string;         // a wait edge stays selectable
};

type CanvasGraph = {
  nodes: CanvasNode[]; edges: CanvasEdge[];
  levels: string[][];      // node ids per level — the layout is this
  minTime: number; maxTime: number | null;   // null while in flight
  warnings: string[];      // e.g. 'fan-out batches inferred from queuedAt equality'
};
```

`levels` is the layout: no DAG library is needed because the level structure falls out of the
derivation. Phase 3's time travel is a second pure function, `graphAt(graph, t)`, so it is testable
on the same fixtures.

**Rendering:** `@xyflow/react`, one new dependency, chosen because Phases 4–5 put *buttons on nodes*
and xyflow's custom nodes are plain React components that can use the existing Tailwind tokens
directly. ECharts is already present but renders nodes as canvas symbols, which would force an HTML
overlay for the controls — two positioning systems for the thing that is the actual point of the
feature. No layout library alongside it.
`TODO(canvas)`: xyflow ships its own stylesheet that knows nothing about `tokens.css`; scoping it to
the canvas container is acceptable for a PoC, but a real version should either restyle it or drop to
hand-rolled divs plus one SVG edge overlay, matching `TimelineBar`'s existing idiom.

---

## 5. Open questions

1. **Loops.** Forty iterations rendered left-to-right will be unreadable, and agent runs are the
   strategic case. Not solved in Phase 2 — Phase 2 shows the naive render so we can judge how bad it
   is. The likely answer is collapsing repeated sub-shapes into one node with an iteration count and
   a stepper, which needs a shape-hashing pass over `levels`.
2. **Fan-in identity.** Answered in §1.3: derivable-ish today from `queuedAt`; correctly via
   `ResponseSteps` (a ~2-file Go change) or `ParallelCoalesceKey` (span attribute + schema). **No SDK
   change is needed either way** — that was the risk, and it is not real. Race mode is the case that
   genuinely breaks levelling.
3. **Nested/invoked functions.** `InvokeStepInfo` carries the child `runID`, so **link out** is
   nearly free and is what the PoC should do. Inline sub-graph is the nicer end state and needs a
   second `useGetRun` plus a way past the 5-level nesting cap.
4. **The 5-level `childrenSpans` cap** (`coreapi.ts:443,497`) is a real ceiling for any future
   inlining of sub-runs.

---

## 6. Incidental findings worth recording

- **A `DebugRun` / `DebugSession` feature was removed one day before this brief** — `2b7c5e45f`,
  *"Remove unused/incomplete run debug feature"*, closes EXE-2191. It deleted `DebugRunLoader`,
  `DebugSessionLoader`, the `createDebugSession` mutation and the `useCreateDebugSession` /
  `useGetDebugRun` / `useGetDebugSession` hooks. **`debugRunID` / `debugSessionID` / `debugPaused`
  remain on `RunTraceSpan`, and the `rerun` mutation still takes `debugSessionID` / `debugRunID`.**
  Worth understanding what that was before building near it.
- **Phase 5's engine already exists.** `rerun(runID, fromStep: { stepID, input })`
  (`gql.mutations.graphql`, resolver `function_run.resolver.go:272`) →
  `pkg/execution/executor/reconstruct.go`, which loads the original run's spans, memoizes every step
  output *before* `fromStepID` into `newState.Steps`, and puts a mutated input for the from-step in
  `newState.StepInputs`. It creates a **new** run and preserves `OriginalRunID` for n+1 re-runs.
  `SharedContext/useRerunFromStep.ts` is already wired in both apps.
  Three things the Phase 5 design doc must resolve: the mutation overrides a step's **input** while
  Phase 4 stages an **output** (faking an output means memoizing step N with fake data and re-running
  from N+1, which probably wants a `MemoizedOverrides` field on `RerunFromStepInput`); it always
  forks rather than mutating in place; and `reconstruct.go` cuts `newState.Steps` at the from-step in
  **start-time order**, which is already an approximation for parallel steps.
- `resolveRerunStepID` (`reconstruct.go:112`) already returns `ErrRerunStepNotFound` /
  `ErrRerunStepAmbiguous` — that is the "code changed shape since the run started" case, partly
  handled.

---

## 7. Verification commands

```bash
# unit tests (the only UI tests CI runs)
cd ui/packages/components && pnpm test
cd ui/packages/components && pnpm lint
cd ui/apps/dev-server-ui && pnpm type-check

# manual, every phase
go run ./cmd dev --no-discovery          # :8288
cd tests/js && pnpm dev                  # :3000
cd ui/apps/dev-server-ui && pnpm dev     # UI against :8288
# then send tests/parallel.test, tests/step.test, tests/retry.test,
# tests/wait.test, tests/cancel.test and open the run
```

There is **no browser e2e infrastructure for the UI** and I will not build any. Manual screenshots at
each phase instead.
