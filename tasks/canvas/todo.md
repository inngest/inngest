# Run Canvas — running log

> **New session? Read `HANDOVER.md` first** — it has the environment setup, the state of both
> repos, and the open items. This file is the decision log (D1–D35), oldest first, and is where the
> reasoning behind each choice lives.

One item in progress at a time. Decisions and discovered constraints recorded as they land.

## Phase 0 — recon

- [x] Locate the run/trace data source, types, resolvers, storage
- [x] Answer: is fan-out/fan-in derivable from today's payload?
- [x] Answer: how are sleep / waitForEvent represented?
- [x] Answer: how are retries represented?
- [x] Locate the Dev Server UI, its build, its transport
- [x] Identify the step detail panel to reuse on node selection
- [x] Identify tokens, primitives, layout conventions
- [x] Identify testing setup and what CI actually runs
- [x] Search for any existing graph/tree/timeline/DAG visualisation or dependency
- [x] Survey fixture apps and existing committed payloads
- [x] Write `tasks/canvas/00-recon.md`
- [x] Reviewed — approved to continue

## Phase 1 — graph derivation, pure and tested

- [x] Capture fixtures by running the real thing (dev server + `tests/js`), not a Go harness —
      faster, and D5 was revised for it
- [x] **Verified empirically: the `queuedAt` signal does NOT survive.** See D1-revised
- [x] Added canvas run shapes to `tests/js/src/inngest/canvas_shapes.ts` (invoke, failure,
      wait-timeout, loop, wide fan-out, parallel chains, slow/in-flight)
- [x] Captured and committed 16 real payloads under `canvas/__fixtures__/` + a README
- [x] `canvas/graph.types.ts` — the model
- [x] `canvas/graph.ts` — `toCanvasGraph(rolledTrace)`
- [x] `canvas/graph.test.ts` — 25 table-driven tests over the committed fixtures
- [x] `pnpm vitest run` green — 360/360 across the package
- [ ] Missing fixture: a **cancelled** run (two attempts raced the run to completion)

## Phase 2 — read-only canvas

- [x] Added `@xyflow/react` (^12.11.6)
- [x] `canvas/toFlowElements.ts`, `canvas/Canvas.tsx`, `canvas/CanvasNode.tsx`
- [x] Canvas tab in `RunDetailsV4.tsx`, gated on `!useShared().cloud`
- [x] Node click → `selectStep` → existing `StepInfo`, untouched — verified in the browser
- [x] Screenshots of every run shape, including the 40-iteration agent loop
- [x] Lint clean; `type-check` clean for both dev-server-ui and dashboard
- [x] **Review round 1 applied** — see D8

## Review round 1 — remaining

- [ ] Discovery step as a real selectable node (**blocked on backend**, see D9)
- [ ] Loop grouping — collapse repeated sub-shapes
- [ ] Prediction layer for in-flight runs: what's running, what we think is next
- [ ] Animate new nodes as the canvas polls
- [ ] Verify batch-event stacking against a real batched run (code is written, untested)

## Phase 3 — scrubber and time travel

- [ ] `graphAt(graph, t)` pure + tested
- [ ] Scrubber below the graph, above the trace, reusing `TimeBrush.tsx`
- [ ] Document what a node shows at *t* when it failed then succeeded on retry
- [ ] Screenshot sequence with overlapping parallel steps
- [ ] STOP

## Phase 4 — edit output / fake output

- [ ] Node controls that stage an override; no execution
- [ ] STOP

## Phase 5 — re-run

- [ ] `tasks/canvas/05-rerun-design.md` **before any code**
- [ ] STOP

---

## Decisions

**D1 (superseded) — fan-out identity from `queuedAt` equality.**
Rejected after measuring real payloads. Kept here because the reasoning matters.

**D1-revised — fan-out identity from observed interval overlap on the rolled-up trace.**
Measured against real runs, not read off the source:

- The executor emits **two spans per step**. On a real fan-out, the three parallel steps shared
  `groupID=13f70615` *and* `queuedAt=…08.442Z` on their **planned** spans, and had divergent
  groupIDs and `queuedAt` on their **execution** spans.
- `traceRollup` buckets by `stepID` into `Map<attempt, Trace>` and last-write-wins, so the
  **execution** spans survive. Both candidate signals are destroyed by the rollup the UI already
  depends on.
- Worse, before rollup both signals **false-positive**: on a strictly sequential
  `step.run → step.sleep("2s") → step.run`, the sleep and the step after it shared both `groupID`
  and `queuedAt`, and the second step's span spanned the entire sleep.
- After rollup the intervals are clean. Parallel: `b[502–539] a[504–550] c[506–551]` overlap,
  `d[808–827]` does not. Sequential: `[357–432] [525–10525] [10526–10543]` — no overlap anywhere.

So: level steps by interval overlap on the rolled-up trace. Correct on every fixture, immune to the
span artifacts, and semantically what the feature is for. Documented in a long `TODO(canvas)` block
at the top of `graph.ts`, with the two proper fixes: expose `ResponseSteps` (already parsed into
`cqrs.OtelSpan`, the GQL loader just never reads it) or write `ParallelCoalesceKey` as a span
attribute.

**Known limit:** overlap is an observation, not the structure. Genuinely-parallel steps that happen
not to overlap — a concurrency limit of 1, or one branch finishing before the other is dequeued —
render as sequential, and nothing in the payload can tell us otherwise.

**D2 — placement: a second tab in `RunDetailsV4.tsx`, gated on `!useShared().cloud`.**
Reuses the existing global selection emitter (`runDetailsUtils.ts:109`) and the existing `StepInfo`
panel with zero changes to either. Whole feature = one directory + one conditional.

**D3 — rendering: `@xyflow/react`, one new dependency, no layout library.**
Custom nodes are plain React components, which is what Phases 4–5 need for per-node buttons. The
model is already levelled so `x = level`, `y = lane` — no DAG layout needed.
`TODO(canvas)`: xyflow's own stylesheet does not know about `tokens.css`.

**D4 — derive from `traceRollup`'s output, not the raw trace.** Confirmed by measurement, and load
bearing for more than expected: rollup is also what de-duplicates the two spans the executor emits
per step. Without it the graph doubles every node.

**D5 (revised) — capture fixtures by running the real stack, not a Go harness.**
The plan was `tests/golang` + `NewSDKHandler`. Running the dev server plus `tests/js` directly and
querying GraphQL was far faster for a paired session, and the missing shapes were added to
`tests/js/src/inngest/canvas_shapes.ts` — additive, so the existing e2e functions are untouched.
Recapture instructions are in `canvas/__fixtures__/README.md`.

**D7 — the loop problem is confirmed unusable, not merely ugly.**
A 40-iteration agent loop is 80 sequential levels (~21,700px wide). Fitted into the pane that is a
zoom of ~0.03 — the graph renders as a grey hairline with no legible nodes at all. Screenshot kept.
This is the strongest argument for collapsing repeated sub-shapes into one node with an iteration
count and a stepper. Deliberately not solved yet.

**D6 — fitting the graph took three fixes, not one.**
1. Node `width`/`height` are declared in `toFlowElements` rather than measured — React Flow's initial
   `fitView` runs before custom nodes are measured, and without them a 12-wide fan-out overflowed on
   first paint.
2. A `ResizeObserver`-driven refit, because `Tabs` mounts the canvas into a container that is still
   zero-height at mount.
3. `minZoom={0.05}` — React Flow's default floor is **0.5**, which is not far enough out for a wide
   fan-out or a long loop to fit at all. This was the one that actually mattered, and it looked like
   a layout bug rather than a zoom clamp.

**D8 — review round 1 reversed two earlier calls.**
- **Canvas is not a tab.** It sits above the trace with a collapse toggle (persisted in
  `localStorage`), because the graph is the orienting view and the trace is the detail — both should
  be on screen. This replaces D2's tab placement; the `!cloud` gate is unchanged.
- **Waits are nodes, not edges.** The brief originally said sleeps and waits are "edges with
  duration, not nodes"; on seeing it, the call was that a `step.sleep` is something the user *writes*,
  so the canvas should mirror the code. Wait nodes are dashed (echoing the timeline's `vertical-lines`
  pattern for sleep/wait/invoke bars) and turn red on timeout.
- Trigger is now the **event**, neutral grey rather than green — an event is not a success — showing
  the event name and id, and stacking for batches. Data comes from the existing `getTrigger` loader,
  not the trace.
- Result node is **filled**, not outlined.
- Selection is driven by the shared emitter and passed into React Flow's `selected`, so selecting a
  row in the trace also highlights the node, and vice versa.
- Edges are `straight` when both ends share a centre line and `smoothstep` only where the graph
  actually changes lane, which removes the spurious bends.
- Icons and status colours are taken from the timeline's vocabulary (`TimelineBar`'s ICON_MAP,
  `Status/statusClasses`) so the two views read as one thing.

**D9 — the discovery node is blocked on the same backend change as parallelism.**
The blank circle between a fan-in and the next level is exactly where the executor's discovery step
runs, and it is the natural place to show what that discovery planned plus its execution stats. It
cannot be built today: `loaders/trace.go` sets `showSpan := span.Name != meta.SpanNameStepDiscovery`
and then `gqlSpan.Omit = true`, so discovery spans never reach the client at all. Surfacing them
(they already carry `response.step.ops` / `meta.Attrs.ResponseSteps`, already parsed into
`cqrs.OtelSpan.Attributes.ResponseSteps`) would deliver **both** the discovery node and exact
fan-out membership in one change.

**Parallel-chains attribution — honest answer: no, not reliably.** Level alignment is real because
discovery coalesces, so `left-1`/`right-1` and then `left-2`/`right-2` do land on the right levels.
But branch *membership* is not recoverable: nothing in the payload says `left-2` follows `left-1`
rather than `right-1`. The canvas therefore draws a junction rather than continuous branch lanes.
`ResponseSteps` would fix this too.

## Constraints discovered

- `GET_RUN` hard-codes **5 levels** of `childrenSpans` nesting. Deeper is silently truncated.
- Outputs are **not** in the trace payload — `outputID` needs a second `runTraceSpanOutputByID` call.
  `StepInfo` already does this itself.
- `groupID` is **not** a parallel-batch ID; the executor deliberately gives each parallel opcode its
  own (`executor.go:3819`). It correlates retries.
- `StepStatusSleeping` and `StepStatusWaiting` both collapse to GraphQL `WAITING`
  (`loaders/trace.go:138`) — read `stepOp` to tell them apart.
- Discovery spans are hidden from GQL output (`loaders/trace.go:239`).
- OTel `follows_from` links are persisted to `spans.links` and then dropped on read — `OtelSpan` has
  no `Links` field. Real causal edges, discarded.
- `ui/packages/components` is the **only** package where CI runs UI tests, and only on changes under
  that path. There is no browser/e2e UI job at all.
- vitest cannot resolve self-referencing `@inngest/components/*` imports — existing tests work around
  it with `vi.mock`.
- `ui/README.md` claims Next.js; the dev-server UI is TanStack Start + Vite.
- Local only for this work: no commits, no pushes, no branches, no PRs.

---

## Round 2 findings

**D10 — the `plannedSteps` change was built, tested against real runs, and reverted.**
Exposing `response.step.ops` on `RunTraceSpan` was supposed to give exact fan-out membership and
unblock a real discovery node. It does neither with the SDK we actually run against, and the reason
is not in the Go code at all.

What was built (saved as `tasks/canvas/planned-steps.patch`, ~30 lines + gqlgen output):
- `RunTraceSpan.plannedSteps: [RunStep!]` in the schema, populated from
  `span.Attributes.ResponseSteps`.
- A `collectPlannedSteps` walk over each omitted discovery subtree, promoting a plan onto every
  visible step span it named. This was needed because the attribute is stamped on the *execution*
  span nested under the discovery span, and the whole discovery subtree is dropped
  (`gqlSpan.Omit = true`).

Why it is inert: instrumenting the loader and running a real `Promise.all([a,b,c])` showed
`respSteps=1` on **every** span — never a batch. `tests/js` runs `inngest@3.11.1-pr-411.4`, whose
`PREFERRED_EXECUTION_VERSION` is `ExecutionVersion.V1`
(`node_modules/inngest/components/execution/InngestExecution.d.ts:48`). A V1 SDK reports **one
opcode per response**, so:
- `response.step.ops` never contains more than one entry, and
- `Metadata.ShouldCoalesceParallelism` (which needs request version ≥ 2) never fires, so there is no
  coalesced fan-in either.

Reverted rather than kept: it adds public GraphQL surface that provably does nothing today. Re-land
`planned-steps.patch` once we can point the dev server at a V2+ SDK — `tests/js` already has
`inngest-v4` (4.3.0) installed under an alias, so rewriting the canvas fixture functions against
that client is the cheapest way to verify.

**This also corrects an earlier claim of mine.** I said the chains fixture levels correctly "because
discovery coalesces". It does not coalesce on this SDK. The levels are correct purely because the
interval-overlap heuristic works — which is better news for the heuristic than the original
reasoning was.

**D11 — branch membership: investigated, answer is "partly, and not for free".**
Not recoverable from the trace today. It *is* implicitly present in the interleaving of `md.Stack`
(completion order) with the SDK's response op order: the inngest-js engine resumes exactly one
memoised step per tick and reports newly discovered steps in discovery order, so zipping the two
server-side reconstructs the pairing. Exact when each branch contributes exactly one new step;
ambiguous when a branch contributes zero or several. Upgrading it to sound needs an
internal-protocol-only SDK field (`opts.parentStepId`, riding the existing free-form
`GeneratorOpcode.Opts any`) — no user-facing API change and no change to step-ID hashing.
`AsyncLocalStorage` cannot do it: continuation context is captured at the `await`, not the resolve.
Caveat: all of that ordering behaviour is inngest-js-specific and was read from `engine.ts`, which
is a newer engine than the V1 one our fixture app actually runs.

**D12 — selection is now genuinely bidirectional.**
`Timeline` kept its own local `selectedStepId`, so canvas→trace highlighting did not work. It now
reads the shared emitter (`useStepSelection({ runID })`) and `RunDetailsV4` passes `runID` down.
Bar ids are rolled-up span ids, which is exactly what the canvas emits, so they match without
translation.

**D13 — node design round 2.**
Per-step-type icons and an explicit monospace `step.run` / `step.waitForEvent` / `step.sleep` /
`step.invoke` / `step.waitForSignal` / `step.ai.infer` label on every node, so the canvas reads as
the code the user wrote. Event node lost its event id (chatty) and gained a card stack plus an
"N events" count for batches. Result node is filled. Selection ring is the `--color-border-contrast`
token read directly — `ring-contrast` is not a real class, since the tailwind preset extends
`borderColor` but not `ringColor`, and it was silently falling back to Tailwind's default blue.

Colour contract, which is a **new** distinction the trace does not draw: green = success,
red = failure, grey = timed out. The trace shows `timedOut` only as a text field and leaves the span
status as `COMPLETED`, so the canvas carries the timeout separately. Applies to invoke timeouts too.

---

## Round 3 — v4 verified, `plannedSteps` landed

**D14 — the SDK was the whole story, and v4 fixes it.**
Added a second fixture app on `inngest-v4` (4.3.0) at `tests/js/src/pages/api/inngest-v4.ts`, with
shapes in `tests/js/src/inngest/canvas_shapes_v4.ts`. It registers as its own app (`canvas-v4`)
alongside the v3 one, so nothing existing changed.

Measured, same instrumentation as before:

| SDK | `respSteps` on the fan-out discovery |
|---|---|
| v3 (`ExecutionVersion.V1`) | `1` — always |
| v4 (`ExecutionVersion.V2`) | **`3`** for `Promise.all([a,b,c])` |

So `plannedSteps` is now live and exact:
- `v4-parallel`: `a`,`b`,`c` all carry `PLAN=[a,b,c]`; `d` carries none.
- `v4-chains`: `[left-1,right-1]` then `[left-2,right-2]`.
- `v4-sequential`: no plans at all — the sleep produces no false positive.

**One more loader bug had to be fixed to get there.** The first attempt still returned nothing,
because a discovery span with a single completed execution child gets collapsed —
`gqlSpan.ChildrenSpans = gqlSpan.ChildrenSpans[0].ChildrenSpans` — which discards the execution span
*before* the promotion walk runs. The attribute now gets lifted in that same branch, alongside the
existing `Response` and `Metadata` lifts.

**D15 — the derivation now has two paths, and says which one it used.**
`toLevelsFromPlans` is authoritative and preferred; `toLevels` (overlap) is the fallback.
`CanvasGraph.parallelismSource` is `'sdk' | 'inferred' | 'none'`, surfaced as a `grouping: exact` /
`grouping: inferred` badge. On an inferred run the canvas **starts collapsed** with a line saying
exact grouping needs inngest-js v4+, so it never quietly presents a guess as fact. A user toggle
overrides and persists.

Deliberately *not* hidden entirely on v3: the canvas is still correct for sequential v3 runs, and
the inferred grouping matched the SDK's own answer on every fixture we can compare. Collapsing it
outright would hide a working view.

**Still missing at every SDK version: branch membership.** A plan says `left-2` and `right-2` were
planned together; nothing says `left-2` follows `left-1`. See D11.

---

## Round 4 — adversarial shapes

Eight deliberately awkward v4 functions in `canvas_shapes_v4.ts`, captured as `gnarly-*.json`
fixtures with assertions in `canvas/gnarly.test.ts`. Six were exactly right first time.

**D16 — `Promise.race` is NOT a canvas bug. I was wrong.**
I initially flagged the race fixture as "a known lie" because `after-race` is drawn after a join of
both branches. The docs say otherwise — `website/pages/docs/guides/step-parallelism.mdx:110`:

> With optimized parallelism (the default in v4), `Promise.race` waits for all parallel steps to
> complete before resolving. If you need early resolution behavior, use `group.parallel()`.

So the canvas is drawing what actually happens. The comment has been corrected. **Untested case:**
`group.parallel({ mode: 'race' })`, which is where early resolution and `ParallelMode.Race` actually
apply — that is the shape that might genuinely misdraw.

**D17 — duplicate step names: fixed, using the SDK's own collision index.**
Two `Promise.all` branches both calling `step.run("work")` rendered as two nodes a reader cannot
tell apart. **The trace view has the identical problem** — two rows named `work`, nothing to
separate them — so there was no existing convention to copy.

The disambiguator already existed and was simply never exposed: the SDK's `resolveStepIdCollision`
assigns an auto-incrementing index, stored as `step.userland.id` / `step.userland.index`
(`meta.Attrs.StepUserlandID/StepUserlandIndex`, written in `tracing/util.go`). Now surfaced as
`RunTraceSpan.userlandStepID` / `userlandStepIndex`, and rendered as `work #1` / `work #2`.

**Gotcha worth remembering:** `userlandStepIndex` is **0-based and present on unique steps too** —
every step in a run with no collisions reports index 0. So the index alone cannot flag a duplicate.
Collisions are detected by grouping the run's steps on `userlandStepID` and only numbering buckets
with more than one member. There is a regression test for exactly this.

**D18 — `foreignAsync` is a genuine limitation, and it fails honestly.**
`Promise.all([delayed(), prompt()])` where `delayed()` awaits a plain 250ms `setTimeout` before
calling `step.run("late")`. Because the SDK only reports steps it has actually reached, `early` is
reported alone in one response and `late` alone in a later one. There is no batch to record, so
`plannedSteps` is empty and `parallelismSource` is `'none'` — the canvas renders them sequentially
and says the grouping is not from the SDK, rather than inventing a fan-out. Structurally they *were*
concurrent in the user's source; nothing at this layer can know that.

---

## Round 5 — branch membership: attempted client-side, measured, rejected

**D19 — the client-side zip is ~94% correct, which is not good enough.**

The idea: `plannedSteps` preserves the SDK's *discovery* order (the executor builds it with
`for n, s := range resp.Generator`, unsorted), and the engine resumes one memoised step per tick in
completion order, each resume discovering that branch's next step. So the k-th step to finish in a
level should pair with the k-th newly planned step in the next — recoverable entirely client-side
from data we already have, with no backend or SDK change.

First check looked perfect: 12/12 hops across six jittered 3x3 runs, with completion order genuinely
varying between runs. That was luck.

Proper stress — 34 runs varying width (2-8 branches), depth (2-4), jitter (0/120/200ms), plus ragged
depths so branches end early:

| Guard | Correct | Wrong | Declined |
|---|---|---|---|
| none | 252 | **15** (5.6%) | 6 |
| require >=3ms between completions | 62 | **2** | 48 (44% of hops) |

**Root cause:** `endedAt` is a lossy proxy for the order that actually matters — `md.Stack`, the
order the executor *saved* each response. Several failures were steps ending within 1ms of each
other (`43.806Z, 43.807Z, 43.808Z`); others had larger gaps but still disagreed, because the delay
between a step ending and its response being saved is not constant.

**Disabled.** A wrong branch edge is worse than none — it asserts a causal link that did not happen.
`resolveBranches` is now a documented stub returning null and the canvas draws junctions.

**Next: do it server-side.** `HandleGeneratorResponse` (executor.go) holds `i.md.Stack` —
authoritative completion order, no timestamps involved — and the ordered `resp.Generator`. Pairing
the k-th newly planned op with the k-th of the most recent completions there removes the entire
class of error measured above. Emit as a `step.parent_id` span attribute, expose on `RunTraceSpan`,
and have the canvas use it when present. The fully sound version remains an internal-protocol
`opts.parentStepId` from the SDK, which would also cover the uneven-width hops this has to decline.

---

## Round 6 — branch membership, done server-side

**D20 — the executor computes it; the client validates it.** Two-part design, because
neither half is sufficient alone.

**Executor** (`parentStepIDs`, `pkg/execution/executor/util.go`, ~20 lines): pairs the k-th newly
planned step with the k-th of the most recent completions in `md.Stack`, emitted as a
`step.parent_id` span attribute.

Why the two orders are guaranteed to agree — this is the whole soundness argument, and it is why
this works where the client-side version did not:

> `d.Execute(ctx, e.smv2, i.md, ...)` marshals `i.md.Stack` into the SDK request
> (`driver.MarshalV1` → `FunctionStack.Stack`), and `HandleGeneratorResponse` runs against that
> same immutable `i.md`. The SDK resumed in exactly the order we sent it. No staleness window,
> no timestamp anywhere.

Guards: declines when there are fewer than 2 new steps (nothing to attribute), and when the stack
holds fewer completions than new steps (the first fan-out of a run, whose steps follow the trigger).

**Known gap the executor cannot close:** "the most recent N completions" assumes those N were
planned as one batch. A branch that itself fans out breaks it — one resumption discovers two steps
— and `md.Stack` records completion order but not batch boundaries. Same for a ragged fan-out where
a branch has ended: the real parents are a subset of the previous level, so everything shifts.

**Client** (`resolveBranches`, `canvas/graph.ts`): rejects unless the attribution is a bijection
onto a whole plan — every member of the level names a parent, the parents are distinct, and together
they are exactly the full membership of one previous plan. `plannedSteps` makes that checkable on
the client, and it is not checkable in the executor.

### Measurements

| Stage | Correct | Wrong | Declined |
|---|---|---|---|
| Client-side, endedAt (Round 5) | 252 | **15** | 6 |
| Executor raw, all shapes | 275 | 6 | 149 |
| Executor raw, balanced only | 264 | **0** | 132 |
| **Executor + client validation** | 267 | **0** | 14 |
| **End-to-end through `toCanvasGraph`, 80 runs** | **542 branch edges** | **0** | 28 hops |

The 80-run pass covered 2–8 branches, depth 2–4, jitter 0–200ms, ragged depths, parallel chains,
3×3 jittered chains, and the nested-fan-out-in-a-branch shape built specifically to break it. That
last shape resolved **zero** hops every time — correctly declined, not wrongly drawn.

### Connect

Works identically, and this was checked rather than assumed. `connectdriver.go:86` and
`httpdriver.go:72` make the same `driver.MarshalV1(ctx, sl, s, step, idx, ...)` call with the same
metadata, and there is a single `HandleGeneratorResponse` call site (`executor.go:2347`) that both
drivers' responses flow through. The attribution is computed above the driver layer, so it is
transport-agnostic.

### Tests

- `pkg/execution/executor/parent_step_test.go` — 8 cases including the ragged gap, pinned as a
  known-wrong output so it is visible rather than folklore.
- `canvas/branches.test.ts` — accepts balanced fan-outs and parallel chains, declines ragged,
  declines nested-fan-out-in-branch, declines the first fan-out of a run.

### Coverage, stated honestly

Branch membership is **sound but incomplete**: it is never wrong, and it is often absent.
Categorising all 240 hops across the 80-run corpus:

| Hop kind | Count | |
|---|---|---|
| Attributable and **resolved** | 122 | branch lanes drawn |
| Attributable but **missed** — uneven widths | 28 | a branch ended; falls back to a junction |
| Attributable but missed — other | 0 | |
| Not attributable: first fan-out of a run | 80 | by design; those steps follow the trigger |
| Not attributable: single-wide level | 10 | sequential, no branch to name |

**81.3% coverage of attributable hops. 0% error rate.**

Per shape (resolved / uneven-miss / other-miss):
`deep3` 8/0/0 · `chains` 4/0/0 · `stress` 110/22/0 · `nested-in-branch` 0/6/0.

The one remaining miss category is **uneven widths** — a fan-out where one branch has finished, so
the parents are a subset of the previous level. Both the executor's pairing and the client's
bijection check refuse it, correctly. Closing it needs a per-opcode parent from the SDK; nothing
server-side can recover it, because `md.Stack` does not record batch boundaries.

Also not covered at all: **any run on an SDK below execution version 2** (inngest-js v3 and
earlier). Those report one opcode per response, so there are no plans, no batches, and no branch
membership — such runs already fall back to the inferred-overlap layout with a `grouping: inferred`
badge.

---

## Round 7 — branch lanes turned off again. The model was wrong, not the code.

**D21 — correcting D20.** I reported "542 branch edges, 0 wrong" over 80 runs. That was true for
that corpus, and the corpus was missing the shape that breaks it:

```js
const [x] = await Promise.all([step.run("fork"), step.run("bystander")]);
await Promise.all([step.run("kid-1"), step.run("kid-2")]);   // both use x
```

Both kids become reachable only once **both** parents resolve, so the executor pairs them
positionally against `[fork, bystander]` and one kid is attributed to `bystander`. Measured over 5
runs of a pathological fixture: **11 correct, 5 wrong**, flipping between runs of identical code.

**The bijection check does not catch it** — the parents named really are exactly the previous level.
And no client-side check can: from outside, "two independent chains" and "a join that re-fans out"
produce identical traces, because discovery is coalesced either way.

**An SDK-side parent hint would not fix it either.** The SDK attributes a newly discovered step to
whichever memoised step it had just resumed; here the kids become reachable only after the *second*
parent resolves, so the SDK names whichever finished last — the same coin-flip.

**The real problem is the model.** Branch lanes assume a tree; a run is a DAG. A step after
`Promise.all([a, b])` genuinely has two parents and no single-parent field can express that. Fixing
it properly means capturing which step promises a continuation actually awaited — real dependency
tracking in the SDK, not positional inference. That is the `async_hooks` route the original
investigation rejected as Node-only with per-promise overhead.

**Decision:** the executor plumbing stays (`parentStepID` is real information and correct for
genuinely independent chains), but `resolveBranches` returns null and the canvas draws junctions.
A junction is always true; a lane can be false, and a false lane sends you debugging the wrong branch.

Pinned in `canvas/branches.test.ts`: no lanes on any shape, the attribution still arrives, and the
`fork`/`bystander` counter-example is asserted so re-enabling means confronting it.

**Also worth recording — how the pathological corpus works.** Ground truth is encoded in the step
name: a step called `kid-1<=fork` declares that it should be attributed to `fork`. That makes
grading automatic for shapes where the right answer is not obvious by eye, and is how the
counter-example was found at all. See `v4Pathological` in `tests/js/src/inngest/canvas_shapes_v4.ts`.

---

## Round 8 — the discriminator, found by prototype

**D22 — "which resume window was this step discovered in" solves it.**

The engine resumes memoised steps **one per tick** (`engine.ts:2320-2333`). That single fact makes
the two shapes that look identical from outside distinguishable *from inside the SDK*:

| shape | new steps vs resume windows |
|---|---|
| independent chains | N steps across N **distinct** windows |
| fork/bystander join | 2 steps in the **same** window |
| ragged (branch ended) | some window produced **nothing** |
| nested fan-out in a branch | one window produced **two** — a real fan-out |

So the datum we need is not a guessed parent, it is **the id of the memoised step being resumed when
this step was discovered**. That is not inferable server-side: the report only shows the final list.

### Prototype

`tasks/canvas/proto-resume-window.mjs` models the resume loop and checks six shapes. Rule tested:
attribute each step to its resume window; trust it only when every member of the previous level
produced at least one child.

```
independent chains          -> DRAW EDGES (L1->L2, R1->R2) then (L2->L3, R2->R3)
fork / bystander            -> junction          (the counter-example, correctly declined)
ragged (one branch ends)    -> junction          (A1 produced, B1 did not)
nested fan-out in a branch  -> DRAW EDGES (F1->F2a, F1->F2b, P1->P2)
io between parent and child -> junction          (IOP produced nothing that replay)
chain deeper than the drain -> DRAW EDGES (DP->DC, SH1->SH2)
```

The nested case is strictly better than anything before: it expresses a genuine **fan-out from a
node**, which a single-parent-per-level model could not represent at all.

**Caveat:** this is a model of the engine, not the engine. It does not reproduce
`resolveAfterPending(100)` or the every-10th `resolveNextTick()` macrotask escape. The deep-chain
result in particular needs confirming against the real thing.

### The change this implies

**SDK (inngest-js), ~6 lines, internal protocol only, no user-facing API, no hashing change:**
1. `let currentResumeStepId: string | undefined` in `createStepTools`.
2. Set it immediately before `.handle()` in the resume loop (`engine.ts:2327`); undefined during the
   initial body run.
3. Stamp it on the step in `pushStepToReport` (`engine.ts:2355`).
4. Emit it in `filterNewSteps` on the existing free-form `opts` — the same channel `ParallelMode`
   already rides.

**Executor: net negative.** Delete `parentStepIDs` and its positional inference entirely; read the
op's opts like `GeneratorOpcode.ParallelMode()` does (`opcode.go:699`) and stamp
`meta.Attrs.StepParentID`. Roughly -25 lines, +5.

**Client:** replace the bijection check with the producer rule — draw edges iff every step names a
window and every member of the previous level produced at least one child.

**Why this succeeds where the previous attempt failed:** it stops inferring parenthood from position
or timing and reports observed causality from the only place that can see it. And because a window
can legitimately produce two children, the model becomes a DAG rather than a tree — which was the
actual defect in D21.

---

## Round 9 — the SDK change, built and measured

**D23 — `opts.discoveredAfter` works, and it is a small change.**

**inngest-js** (`947a244e`, v4.19.0; local tree was 2.5 months stale and was reset to origin/main
first). Two files, ~45 lines, no user-facing API, no hashing change:
- `FoundStep.discoveredAfter?: string`
- `currentlyResuming` set before `.handle()` in the resume loop, cleared when nothing is left to
  resume, stamped in `pushStepToReport`, emitted on the existing free-form `opts` in
  `filterNewSteps`.

**Executor: net negative.** `parentStepIDs` now just reads `op.DiscoveredAfter()`; the ~30 lines of
positional `md.Stack` inference are gone.

**Loader gotcha, same trap as `groupID` and `queuedAt` before it:** the parent is stamped on the span
created when the step was *planned*, and a *later* span records the same step completing. Rollup
keeps the latter. So the loader now collects `stepID -> parentStepID` across every span of the run
and re-stamps it. Third time this exact shape of bug has appeared — worth remembering that anything
step-scoped must be gathered across spans, not read off one.

### Measured, 69 runs on the patched SDK

| shape | edges correct | wrong | hops resolved | declined |
|---|---|---|---|---|
| nested-in-branch | 15 | 0 | **5** | 0 |
| deep3 | 24 | 0 | 8 | 0 |
| chains | 8 | 0 | 4 | 4 |
| stress | 363 | 0 | 91 | 14 |
| pathological | 8 | 0 | 4 | 37 |
| **total** | **418** | **0** | 112 | 55 |

`nested-in-branch` went 0 → 5/5 resolved: a fan-out from inside a branch (`f-1 -> {f-2a, f-2b}` while
`p-1 -> p-2`) now draws as a real DAG, which no previous model could express.

**D24 — two rendering fixes from review.**
- **Dangling junctions.** A junction was emitted eagerly after any wide level, before we knew whether
  the next hop would draw direct edges. When it did, the junction was left with edges in and none
  out, reading as if every branch flowed through it. Resolution is now precomputed for all hops so a
  level can look ahead.
- **Lane alignment.** Nodes are pulled onto the average lane of their children, right to left, so a
  1:1 branch draws as a straight horizontal line and a fan-out sits centred on its children. Lanes
  are fractional as a result, so the renderer centres each level on its own lane range rather than a
  count.

### Open

**Dead-end shape is unattributed.** `const pa = step.run("a"); const pb = step.run("b"); await pa;
step.run("c")` — `c` arrives with no parent, so it falls back to a junction and both `a` and `b`
appear to flow into `c`. A sticky-assignment guard in `pushStepToReport` did not fix it. Not yet
root-caused.

**D25 — the ceiling of the resume-window signal.** Consider:

```js
// A: a2 waits for both              // B: a2 waits for a only
const [av] = await Promise.all([a, b]);   const av = await a;
await step.run("a2");                     await step.run("a2");  await b;
```

If `b` finishes first and does nothing, then `a` finishes and `a2` is found in `a`'s window, both
cases report `discoveredAfter = a`. **Indistinguishable.** The window records *when* a step became
reachable, not *what it was waiting on*.

Next idea to prototype: **co-await groups.** `Promise.all([pa, pb])` calls `.then` on both
synchronously in the same turn, so step promises whose `.then` is registered together could be
recorded as one await-set, giving `{a,b} -> a2` in case A and `{a} -> a2` in case B. Requires
`step.run` to return a real thenable — note that patching `.then` on a native promise does **not**
work, since `await` takes V8's fast path and bypasses it (prototyped and confirmed).

**D26 — how to observe the await-set, and how scary it is.**

Getting from "when did this step become reachable" to "what was it waiting on" needs the SDK to see
`.then` registrations. Three mechanisms, measured (`tasks/canvas/proto-promise-interception.mjs`):

| mechanism | intercepts `await` | `instanceof Promise` | `.catch`/`.finally` | `Promise.all` sees it |
|---|---|---|---|---|
| patch `.then` on an instance | **no** | yes | yes | — |
| `class extends Promise` | **yes** | **yes** | **yes** | yes, both members |
| plain object thenable | yes | **no** | **no** | yes |

Patching an instance is a dead end: `await` takes V8's fast path and skips it. A plain thenable works
but loses `instanceof Promise`, `.catch` and `.finally` — a real interop hazard for a library that
has to run everywhere. **A `Promise` subclass keeps all native semantics and still intercepts**, so
the scary option is not required.

Residual risks of a subclass, worth checking before committing: ES5 downlevelling breaks
`class extends Promise` (classic TypeScript gotcha); `Symbol.species` means `.then()` returns another
subclass instance, so registration must be idempotent; patched-Promise environments (Zone.js, some
edge runtimes) need verifying; one extra allocation per step.

**Co-await grouping is observable** (`tasks/canvas/proto-coawait-timing.mjs`): `Promise.all` registers
`.then` on every member in the *same* microtask tick, so registrations sharing a tick are one
await-set. `await Promise.all([a,b])` would yield `{a,b}`; `await a` yields `{a}` — which is exactly
the distinction D25 says the resume window cannot make.

**Not yet working:** carrying that set to the continuation. `AsyncLocalStorage` fails, for the same
reason it failed in the first investigation — context is captured when the reaction is *registered*
(at the await), not when it is *resolved*. A synchronous module-level marker is cleared before the
awaiting code resumes, because `await` on a thenable adds extra microtask hops. The existing
`currentlyResuming` works only because it is long-lived; the same trick should carry a *group*
instead of a single id, which is the next thing to build.

---

## Round 10 — co-await grouping: measured and rejected. Attribution centralised.

**D27 — co-await grouping does not work, and the reason is observable.**
`tasks/canvas/proto-*.mjs` hold the experiments.

The idea was to recover the *await-set* rather than the resume window: `Promise.all([a, b])` calls
`.then` on both members, so registrations sharing a turn would be one set — distinguishing
`await Promise.all([a,b]); step.run(c)` (parents `{a,b}`) from `await a; step.run(c)` (parent `{a}`),
which D25 showed the resume window cannot do.

It half worked. Grouping by microtask tick gets **A and B right** — and also the ragged case — but
**over-groups independent chains** (`L2 <= L1+R1`), because both branches start in the same tick.

The discriminator would be whether a step creation happens *between* two registrations. It does not
exist, because `await` defers `.then` registration to a microtask job. Both shapes produce identical
orderings:

```
independent chains:  CREATE L1, CREATE R1, [tick], REGISTER L1, REGISTER R1
Promise.all([a,b]):  CREATE a,  CREATE b,  [tick], REGISTER a,  REGISTER b
```

`executionAsyncId()` at registration does not separate them either — two distinct contexts in both
cases. Only full promise-lineage instrumentation might, which is the Node-only, per-promise-overhead
route rejected in the original investigation.

**So D25 stands: `discoveredAfter` records when a step became reachable, not what it was waiting on.**

**D28 — how to intercept, if this is ever revisited.**
Measured in `tasks/canvas/proto-promise-interception.mjs`:

| mechanism | intercepts `await` | `instanceof Promise` | `.catch`/`.finally` |
|---|---|---|---|
| patch `.then` on an instance | **no** (V8 fast path) | yes | yes |
| `class extends Promise` | yes | yes | yes |
| plain object thenable | yes | **no** | **no** |

**If any of this is ever pursued, it needs a real compatibility investigation first.** Returning
anything other than a plain native promise from `step.run` touches every runtime, bundler and
library the SDK supports. A `Promise` subclass is far safer than a raw thenable, but still carries:
ES5 downlevelling breaking `class extends Promise`; `Symbol.species` making `.then()` return another
subclass instance; patched-Promise environments (Zone.js, edge runtimes); an extra allocation per
step; and — ironically — extra microtask hops that could push continuations out of the engine's own
`resolveAfterPending(100)` budget. **None of that has been investigated. Do not ship on the strength
of these prototypes.**

**D29 — attribution moved into `tracing.generatorAttrs`, and the executor plumbing deleted.**
`discoveredAfter` rides the opcode, so stamping it where attrs are built from an opcode covers every
producer at once: the executor's five step handlers, finalization, **and the checkpoint API**. The
`OpcodeGroup.ParentStepIDs` field, the `parentStepIDs` helper and the `addParentStepAttr` helper are
all gone — net deletion.

That mattered: a lone new step is executed inline and reported via the checkpoint path, which never
reaches `HandleGeneratorResponse`. Instrumenting the executor showed only 5 opcodes for a run whose
trace had 3 steps — the third never arrived there at all.

### Still open

**Lone inline-executed steps are still unattributed.** `c<=a`, `other-2`, `shallow-2` come back with
no parent. The SDK stamps them correctly — instrumenting the installed build shows
`push 7e5ebe9a resuming=86f7e437` — so the loss is between `pushStepToReport` and the span. The
checkpoint request body does forward `opts` (`checkpointStepsAsync` sends `steps: args.steps`
wholesale), so the next thing to check is whether the ops handed to it are rebuilt without `opts` —
`stepRanHandler` is the likely culprit. Note `executeStep`'s `outgoingOp` was already patched to
carry it and that alone did not fix it.

Multi-step responses (chains, nested fan-out) are unaffected and remain correct.

---

## Round 11 — exact lineage, by probing rather than observing

**D30 — no promise-level interception can carry lineage across `await`.** Third independent
confirmation, this time the strongest form: a `Promise` subclass whose `then` wraps V8's own resumer
in an `AsyncLocalStorage` context. Every discovered step still sees `ROOT`
(`tasks/canvas/proto-als-subclass.mjs`). V8 routes generator resumption through a separate internal
reaction, so the context does not reach the resumed body. Combined with D27, the observational
approach is closed: registration timing, async ids, and context propagation all fail.

**D31 — deduce it instead. The engine controls when memoised steps resolve, which makes it an
oracle.**

> A step appears in a replay **iff every step it awaits has been resumed.**
> So "resume everything except `m`" is a membership test:
> `c depends on m  ⇔  c does not appear when all but m are resumed`.

One replay per candidate, and each replay answers the question for *every* step at once. No promise
interception, no ordering assumptions, no timing — **exact by construction.**

`tasks/canvas/proto-lineage-elimination.mjs`, 11/11 correct:

| shape | result |
|---|---|
| independent chains | `L2<=L1`, `R2<=R1` |
| join `Promise.all([a,b]); c` | `c<=a+b` |
| dead end (`await a; c; then b`) | `c<=a` |
| nested fan-out in a branch | `F2a<=F1`, `F2b<=F1`, `P2<=P1` |
| ragged (branch ends) | `A2<=A1` |
| three-way join | `w<=x+y+z` |
| **gap: deps on x and z, y idle** | **`w<=x+z`** |

Every shape that defeated the previous four approaches, including the `fork`/`bystander` join and the
dead end — and it returns **sets**, so the DAG is expressible rather than being forced into a tree.

A cheaper two-probe variant (forward + reverse order, intersect the prefixes) is exact for
contiguous dependency sets but **wrong for the gap case** — it reports `w<=x+y+z`, because the
intersection is "everything positioned between the first and last dependency"
(`proto-lineage-two-order.mjs`). Kept as a record of why the cheap version is not enough.

### Cost, honestly

`n+1` replays of the function body per discovery request, where `n` is the number of candidate
parents. Group testing was tried and does **not** help — 2n replays, because when steps have
different parents the recursion visits everything.

What makes it plausible anyway:
- **Zero cost for sequential runs.** One candidate means nothing to disambiguate; skip probing.
- **Step callbacks are not re-run** — memoised steps resolve from state. Only the function body
  between steps re-executes.
- **That body already re-runs on every request**, so the determinism contract users are held to is
  unchanged in kind — but probing amplifies it from once per request to `n+1` times. Anything
  side-effecting outside a step (counters, logging, metrics) would fire repeatedly. This is the real
  hazard and it is why this must not be on by default.
- **Candidates can be narrowed** to steps completed since the last discovery, at the cost of
  exactness for late-awaited dead ends.

**Fits the feature.** The canvas is Dev Server only, so exact lineage can be a dev/trace-mode
behaviour and stay off in production entirely. That reframes the cost question from "is this fast
enough for everyone" to "is this fast enough while debugging".

### Not yet done

Prototypes model the resume loop; they are not the engine. Implementing this for real means the
engine performing several full replays per discovery request and diffing the discovered sets, which
is a materially bigger change than `discoveredAfter` — and it needs the same compatibility scrutiny
as D28 before going anywhere near production.

---

## Round 12 — nine theories, and one that works at O(1)

Probing (D31) was rejected on cost: `n+1` replays is O(n²) for a 1000-step run and impossible on
Cloud. Logged and moved on. The theories tried, in order:

| # | theory | result |
|---|---|---|
| 1 | client-side pairing on span `endedAt` | ~6% wrong |
| 2 | executor pairing on `md.Stack` position | wrong on ragged shapes and joins |
| 3 | SDK resume window (`discoveredAfter`) | cannot separate "awaited a" from "awaited a and b" |
| 4 | co-await grouping by microtask tick | over-groups independent chains |
| 5 | ALS through a `Promise` subclass's `then` | `ROOT` everywhere — context never crosses `await` |
| 6 | two-order prefix intersection | wrong when a non-dependency sits between two dependencies |
| 7 | subset-elimination probing | **exact**, but `n+1` replays — too expensive |
| 8 | group testing / bisection | 2n replays, no better than 7 |
| 9 | **combinator spy + composites + await chains** | **15/15, zero extra replays** |

**D32 — theory 9. Watch the joins being built, rather than inferring them afterwards.**

A join only ever comes from a promise combinator. So wrap `Promise.all` / `allSettled` / `race` /
`any` while the function body runs and read the group off the call site, synchronously and exactly.
Three parts:

1. **Combinator spy.** `Promise.all([pa, pb])` over step promises records the co-awaited set. Note
   `Promise.all([chainA(), chainB()])` records *nothing*, because those are not step promises —
   which is exactly right, since the branches are independent.
2. **Composites.** Tag the promise a combinator returns with the ids it stands for, so
   `Promise.all([Promise.all([a,b]), c])` expands to `{a,b,c}`.
3. **Await chains.** Record the window each step promise's `.then` was registered in. In
   `await a; await b; step(c)`, b is awaited *while resuming a*, so the continuation depended on a
   too; expanding transitively turns `{b}` into `{a,b}`.

A step's parents are then the current window's group, transitively expanded. `tasks/canvas/proto-lineage-combinator.mjs`:

```
independent chains   L2<=L1  R2<=R1        gap: x,z dep, y idle   w<=x+z
join Promise.all     c<=a+b                three-way join         w<=x+y+z
dead end             c<=a                  hand-rolled a;b;c      c<=a+b
nested fan-out       F2a<=F1 F2b<=F1       nested Promise.all     d<=a+b+c
ragged               A2<=A1                race / allSettled      correct
```

**Cost: O(1).** No extra replays, no probing. A `WeakMap`, a few small sets per run, and a wrapper
on four static methods for the duration of the body.

### Risks, none of them investigated yet

- **Monkey-patching global combinators.** The prototype installs and restores around each replay,
  which would race as soon as two runs interleave in one process. The fix is to install once for the
  process lifetime and key everything off promise identity rather than install/restore — untested.
- **Combinators captured before the patch.** A module-level `const all = Promise.all` at import time
  bypasses the spy. (Capturing inside the body is fine — tested.)
- **Third-party combinators.** `p-map`, Bluebird, `async` and friends build joins without touching
  `Promise.all`; those degrade to a single parent rather than a set.
- Still needs the `Promise` subclass for `.then` observation, so **every caveat in D28 still
  applies** — ES5 downlevelling, `Symbol.species`, patched-Promise environments, extra microtask
  hops against the `resolveAfterPending(100)` budget.
- `awaitedIn` records only the first registration per step; a step awaited from two places would
  keep the first.

This is a prototype of the mechanism, not an implementation in the engine. It looks like the right
shape of answer, and it is cheap enough for Cloud, but the compatibility work is entirely ahead of it.

---

## Round 13 — theory 9 retracted; five more theories

**D33 — theory 9 (patching `Promise.all` et al globally) is disqualified, not merely risky.**
It scored 15/15, but overwriting built-ins in a library that ships to every JS runtime breaks: other
libraries in the same process, two copies of the SDK patching over each other, frozen or locked-down
realms, Zone.js and similar, and a crash mid-run leaving the patch installed. Recorded as a
measurement of what the *signal* is worth, not as a proposal.

Worth separating clearly: **`discoveredAfter` as shipped touches none of this.** It is a variable set
around `.handle()` inside the engine. No subclass, no globals, nothing user-visible changes.

| # | theory | result |
|---|---|---|
| 10 | identify the combinator from the call stack at `.then` | dead — registration runs in a fresh microtask job with no caller frames |
| 11 | group id on `group.parallel()`'s existing ALS scope | works exactly, but only when users opt into that API |
| 12 | group by the handler identity a combinator passes to `.then` | dead as written — `Promise.resolve` wraps our subclass, so we only see adoption handlers |
| 13 | make the subclass **adoptable** so combinators call `.then` directly | combinator groups visible; plain `await` visibility lost to V8's fast path |
| 14 | **theory 13 + the engine's resume window** | **10/13, zero extra replays, nothing global patched** |

**D34 — theory 13's mechanism.** `Promise.resolve(x)` returns `x` unchanged iff
`x.constructor === Promise`. Defining that on *our own prototype* — nothing global — makes
`Promise.all` skip the wrapper and call `.then` on our promise directly, where every element of one
call shares an onRejected identity. That identity is the group.

Notably this makes the promise *more* interoperable, not less: `Promise.resolve(p) === p` now holds,
alongside `instanceof Promise`, `.catch` and `.finally`. The cost is that `await` then takes V8's
fast path and stops calling our `.then` at all — which is fine, because the engine's resume window
already covers plain awaits.

**D35 — theory 14 is the best safe option found.** `tasks/canvas/proto-lineage-safe.mjs`:

```
independent chains  L2<=L1 R2<=R1     gap x,z dep; y idle  w<=x+z
join Promise.all    c<=a+b            three-way join       w<=x+y+z
dead end            c<=a              ragged               A2<=A1
nested fan-out      F2a<=F1 F2b<=F1   race                 correct
```

Still wrong on three shapes, all for understood reasons:
- **`allSettled`** — passes per-element handlers, so there is no shared identity to group on.
- **Nested combinators** — `Promise.all([Promise.all([a,b]), c])` gives `d<=c`; the inner result is
  not one of our promises, so it cannot be tagged without seeing the combinator call itself.
- **Hand-rolled sequential joins** — `await a; await b; step(c)` gives `c<=b`; fixing it needs the
  await-chain observation that theory 13 trades away.

It also still returns a `Promise` subclass from `step.run`, so the D28 compatibility questions
remain open — ES5 downlevelling of `class extends Promise`, `Symbol.species`, patched-Promise
environments, and the extra microtask hops against `resolveAfterPending(100)`.

**No downside-free theory found in five attempts.** The honest ranking today: what is shipped
(`discoveredAfter`, safe, chains and nested fan-outs correct, joins declined) < theory 14 (three more
shapes, at the price of a Promise subclass) < theory 9 (everything, at a price not worth paying).

---

## Round 14 — five more theories, and two of the old answers were wrong

Prototypes: `proto-lineage-plain.mjs` (T15/18/19/20 + T14 re-scored),
`proto-lineage-hooks.mjs` (T16), `bench-hooks.mjs` (cost), `probe-*.mjs` (the
mechanism checks each theory rests on). All share `shapes.mjs`, so every number
below is against one table of **26 expectations over 20 shapes**, graded the
same way. Re-runnable: `node tasks/canvas/proto-lineage-plain.mjs`.

**D36 — the ground truth was wrong in two places, and fixing it changes the
ranking.** Raised by the user looking at `await a; await b; step.run('c')` and
asking why c's parent was not just b.

1. **An edge is the transitive reduction of "must complete before".** Not the
   full dependency set. That is the entire difference between the two shapes
   that look identical at the point of discovery:

   ```
   const pa = step.run('a')      await step.run('a')
   const pb = step.run('b')      await step.run('b')
   await pa; await pb            step.run('c')
   step.run('c')
   => c <= a + b  (fork/join)    => c <= b  (chain; a -> c is implied)
   ```

   No new signal is needed to separate them: b's own parents are already
   recorded from when b was discovered. `reduce()` in `shapes.mjs` applies it,
   equally, to every theory. The old table never tested the sequential form, so
   it never noticed.

2. **`race` is not a join.** `await Promise.race([r1,r2]); step.run('r3')` — r3
   starts on the *winner*. Every group theory was drawing `r3 <= r1 + r2`,
   claiming it waited for both. The old table scored that as correct.

The old shape table is a subset of the new one, so the earlier scores are still
readable, but **theory 14 re-scores from "10/13" to 20/26** under the corrected
grading and the added shapes.

**D37 — the resumed-prefix rule. A group contributes only the members resumed so
far, and that alone separates `all` from `race` with no combinator identity.**

At discovery, take the group of the step being resumed and split it:

- every member already resumed => **conjunctive** (`all`, `allSettled`, a
  hand-rolled counter). All of them are real edges.
- some member not yet resumed => **disjunctive** (`race`, `any`). The engine is
  resuming in completion order, so the one in hand is the winner; the rest
  **could have unblocked this and did not**.

That second set is worth rendering. The canvas can draw the winner solid and the
losers dashed — `r3<=r1 ~r2`, `q4<=q1 ~q2+q3` in the prototypes — which is a
truthful picture of a race rather than a wrong picture of a join. It costs
nothing; it is the group we already had, minus the winner.

Known ambiguity: `Promise.any` where the first member *rejects* fires on the last
member, so it classifies as conjunctive. Step status could disambiguate; not done.

| # | theory | score |
|---|---|---|
| 3 | resume window only (what ships today) | 15/26 |
| 14 | Promise **subclass** + onRejected identity | 20/26 |
| 18 | **plain native promise**, own `then` + handler identity | 20/26 |
| 19 | plain promise, own `then` + **synchronous registration batch** | **25/26** |
| 20 | 19 + barren-resume carryover, filtered by 15's branch path | **25/26** |
| 16 | **`node:v8` promiseHooks** — nothing touched at all | **26/26** |

**D38 — theory 18. The Promise subclass was never necessary. An own `then`
property on a plain native promise gets the same signal.**

`Promise.all` reaches an element through `PromiseResolve` (which returns it
unchanged, because a native promise's `constructor` *is* `Promise`) and then
**invokes `.then` on it** — an own property included. Verified in
`probe-own-then.mjs`:

```
instanceof Promise          true     Promise.all calls own then   true
constructor === Promise     true       shared onRejected identity true
Promise.resolve(p) === p    true     await calls own then         false
prototype === Promise.proto true     allSettled/race/any          all call it
own properties              then     globals patched              none
```

Scores identically to theory 14 (20/26), and **every D28 compatibility question
disappears**: no `class extends Promise` to downlevel to ES5, no
`Symbol.species`, no subclass to survive a patched-Promise environment, no
change to the TypeScript type. `await` still ignores it (spec: `await` uses the
internal PerformPromiseThen, never a `then` lookup) — which is fine, because the
engine's resume window already covers plain awaits.

This is strictly better than theory 14 and should replace it as the reference
"if we must touch the promise" option.

**D39 — theory 19. Group by the synchronous block a registration happened in,
not by handler identity. That is the better signal and it is simpler.**

Because `await` never calls `then`, essentially the only thing that *does* call
it is a combinator — so "registrations in one synchronous block" is very nearly
a pure combinator detector. It fixes three shapes handler-identity gets wrong:

- **`allSettled`** — per-element handlers, no shared identity, but the same tick.
- **nested combinators** — `Promise.all([Promise.all([a,b]), c])` is one
  expression, so all four registrations land in one block => `d <= a+b+c`.
- **third-party joins** — `p-map`, Bluebird and hand-rolled counters call `.then`
  like everyone else, so they group. Theory 9 listed these as a permanent loss.

One real false positive, and its fix: two unrelated `.catch` calls in the same
tick would group. A combinator always passes an `onFulfilled`; `p.catch(f)` is
`then(undefined, f)`. Skipping registrations with no `onFulfilled` removes the
common case at no cost (`user .catch on two` in the table). `.finally` on two
step promises in one block would still group — untested, low frequency.

Also unfixed: the SDK's own internal `.then` calls on step promises would
pollute batches. It would need a flag around them.

Misses `parallel then joined` (`c<=b`, want `a+b`).

**D40 — theory 20. Carry a barren resume forward. Fixes hand-rolled sequential
joins, breaks fire-and-forget.** A resume that moves *nothing* forward — no new
`step.run` call at all, not merely no discovery — means the continuation is
waiting for something else, so it carries onto the next discovery. That is
exactly `await pa; await pb; step.run('c')` => `c <= a+b`.

Its cost is `fire and forget`: `step.run('i')` started and never awaited is
barren too, and gets carried onto the next step. So **19 and 20 each get 25/26
and miss different shapes**; taking both needs an "was this promise ever
awaited" oracle, which only theory 16 has.

**D41 — theory 15. Branch identity is free in the async stack trace, and is
worth exactly one shape.** V8 stitches combinator frames into async stacks and
exposes them structurally (`CallSite.isPromiseAll()`, `.getPromiseIndex()`), so
no string formatting is needed — ~1.9 µs formatted, less structurally. A step
discovered inside `Promise.all([branchA(), branchB()])` knows it is in `all#0`
or `all#1`. Measured contribution: as a filter on theory 20's carryover it is
worth `ragged flipped` — 24/26 without it, 25/26 with. `Error.stackTraceLimit`
(default 10) truncates deep nesting. Not a lineage source on its own: after
`await Promise.all(...)` completes you are back in the parent frame, so it can
never see a join.

**D42 — theory 16 is the only one that is exact, and it touches nothing at all.**
`node:v8`'s `promiseHooks` hand over promise **objects** — `init(promise,
parent)` plus `before`/`after` around every reaction job. `async_hooks` cannot do
this any more: `resource.promise` is gone in Node 22 (`probe-async-hooks.mjs`),
so ids cannot be tied back to promises.

Two edges are enough for everything:

- `await p` derives a promise from p, created while the previous await's job was
  running. So one async function instance's await chain is a walk up
  `createdIn`, each hop naming what it awaited. **This is the signal every
  `.then`-based theory loses to V8's await fast path** — which is why it is the
  one that gets `parallel then joined` right.
- a combinator creates its result capability, then derives one promise per
  element, all in one block. Walking the tick's inits in order and starting a
  new segment at each non-derived init recovers the group *and* the promise it
  feeds — including nested and hand-rolled combinators.
- a step promise that never appears as anyone's `parent` was never awaited, so
  `fire and forget` is right for free.

`step.run` returns a bare `new Promise(...)`. No subclass, no own property, no
global patched, nothing user-visible. **26/26.**

The price is real, and it is not correctness:

- **Node-only.** Not in Bun, Deno, workerd or browsers. Must degrade to the
  shipped `discoveredAfter` everywhere else.
- **Process-wide cost**, not per-run — every promise in the process, including
  other tenants' (`bench-hooks.mjs`):

  | | tight await loop | nested async + `Promise.all` |
  |---|---|---|
  | empty hook | 1.6x | 1.2x |
  | theory 16's bookkeeping | 15.4x | **5.2x** |

- **Retention.** The edge map must be a per-request `Map` dropped at request end;
  a `WeakMap` leaks, because each value holds its parent promise strongly and so
  pins the whole ancestor chain. Size is O(promises created in the request), not
  O(steps), so it wants a cap that degrades to the resume window.

But unlike theory 9 it is **additive, not a mutation**: two SDK copies installing
hooks compose, a frozen realm is unaffected, and a crash mid-run leaves nothing
patched. That is the difference between "disqualified" and "a dev-mode switch".

**Where this leaves it.** The honest ranking:

- ship as-is (`discoveredAfter`, 15/26) — safe, chains and fan-outs right, joins declined
- **+ theory 19** (25/26) — one own `then` on a plain promise, zero runtime cost,
  works everywhere. Replaces theory 14 outright: better score, none of the
  subclass risk. The compatibility work left is small and specific.
- **+ theory 16 in dev mode** (26/26) — exact, nothing touched, Node-only,
  ~5x on promise-heavy bodies. Fits the canvas exactly, since it is Dev Server only.

19 and 16 are complementary rather than competing: 19 is the ships-everywhere
signal, 16 is the exact one you can afford while debugging.

**Not prototyped: theory 21, static analysis of `fn.toString()`.** The joins are
plainly visible in the source, and runtime call sites could anchor steps to AST
nodes. Skipped because it cannot follow steps called inside helper functions,
loops or higher-order code without real dataflow analysis, it needs a parser
dependency, and it degrades on transpiled or minified bundles — while theory 16
is exact on all of those. Recorded so it is not re-proposed as new.

**D43 — theory 22 (make `await` visible from a plain promise) is dead, and it
explains why D34's trade-off is forced rather than incidental.**

T19's only miss is the hand-rolled join, which it misses because `await` never
calls `then`. It never calls it because `PromiseResolve` returns a promise
unchanged when `p.constructor === %Promise%`. Shadowing `constructor` **on the
instance** (with a `@@species` of `%Promise%`, so `.then()` still yields a normal
promise) does make `await` call our own `then` — verified — with no subclass and
nothing global patched.

But it drops to **17/26** (`proto-lineage-await.mjs`), fusing every pair of
branches that merely started in the same tick. The cause is structural: making
`await` wrap routes *every* registration through `PromiseResolveThenableJob`, so

- registrations no longer happen synchronously inside the combinator call — the
  sync-block signal T19 depends on is gone, and
- they run in a fresh microtask job with **no caller frames at all**, so the call
  site cannot rescue them either (this is theory 10's finding, reached again from
  the other direction).

Confirmed directly: with `constructor` intact, `Promise.all([a,b])` produces two
registrations synchronously with the user's call site on the stack; with it
shadowed, zero synchronous registrations and the deferred ones have a two-frame
stack ending at our own wrapper.

So D34's "combinator groups OR plain awaits, never both" is a property of
`PromiseResolve`, not of the grouping rule. **No further theory should try to get
both signals out of `.then`.** Only theory 16 sees awaits, because promise hooks
observe the derivation instead of the registration.

That leaves T19's miss standing, and the choice between 19 and 20 is a real one:

- **19 under-declares** on `await pa; await pb; step(c)` — draws `c<=b`, leaving
  `a` as a dangling leaf. Same as what ships today; no new wrong edges.
- **20 over-declares** on a never-awaited `step.run` — draws an edge that does
  not exist.

For a debugging surface a missing edge is a dangling leaf; a false edge is a lie.
Prefer 19 unless fire-and-forget steps turn out to be rare enough to ignore.

---

## Round 15 — theory 19 built, end to end, and two engine bugs it exposed

Implemented across both repos and verified against the running stack. Plan and
slice checklist in `tasks/canvas/todo-t19.md`.

**D44 — the wire format is a set.** `opts.discoveredAfter` is now `string[]`,
with `opts.discoveredAfterAlternates` for the losing side of a race. The Go side
reads both a list and a bare string, so a run from an older SDK keeps the
lineage it does report.

**D45 — the grouping is observed with one own `then` on a plain native promise.**
`packages/inngest/src/components/execution/stepLineage.ts`. `Promise.all` reaches
an element through `PromiseResolve`, which returns a native promise unchanged
and then invokes `.then` on it — an own property included. `await` never calls
`then`, so in practice almost the only thing that registers is a combinator, and
"registered in the same synchronous block" is very nearly a pure combinator
detector. Registrations with no `onFulfilled` are skipped, since `p.catch(f)` is
`then(undefined, f)` and two unrelated catches would otherwise look like a join.

Nothing global is patched and the promise is unchanged by every observable test.

**D46 — lineage is stamped onto the op's opts where the step is found, not
merged in at each report site.** This is the fix for the open issue that had been
top of the list since round 9. A step reaches the Executor by three routes —
planned in a batch, executed inline, or checkpointed after running — and the
third serialises `opts` directly, so merging at the other two lost it. The step
that always takes the third route is *a lone step discovered after a resumption*,
which is exactly the step after a join. `tests/v4.deadend` now reports `c<=a`
instead of `(root)`.

**D47 — `resumeStepWithResult` is a resumption too, and the marker has to
outlive `handle()`.** When a step runs inline and the function continues in the
same request, the memoised-replay loop never runs, so nothing marked the step as
being resumed and every inline continuation reported no lineage at all. Marking
it there was half the fix; the other half is that resolving a step's promise only
*schedules* its continuation, so restoring the marker straight after `handle()`
cleared it before the discovery it was meant to attribute. It is now held across
the microtask drain, as the replay loop gets for free by awaiting between
resumptions. `tests/v4.sequential` went from no lineage to a full chain.

**D48 — the exported `step` singleton was hiding the promise it should hand
over.** `step.run(...)` was `getDeferredStepTooling().then((tools) =>
tools.run(...))`, so user code held a *derived* promise: a `Promise.all` over
step calls registered on those wrappers rather than on the steps' own promises,
and the wrappers adopted the real ones in microtasks of their own, which looked
like combinator activity the user never performed. Both directions were wrong,
and it showed as `tests/v4.chains` reporting `left-2 <= [left-1] dashed=[right-1]`
and `right-2 <= [left-1,right-1]` — a fabricated race and a fabricated join.

Inside a running function ALS is already resolved, so the tools are available
synchronously; `step.*` now calls them directly and only falls back to the
deferred path before ALS has loaded. That removes a microtask hop per step call
as well.

### Measured on the running stack

`tests/v4.*`, read back through the GraphQL API:

| shape | reported |
|---|---|
| `sequential` | `for 2s <= [first step]`, `second step <= [for 2s]` |
| `deadend` | `c<=a <= [a]` — was `(root)` |
| `parallel` | `d <= [a, b, c]` |
| `nested` | `collect <= [a1, a2, b]` (nested combinator, flattened) |
| `race` | `after-race <= [fast] dashed=[slow]` |
| `chains` | `left-2 <= [left-1]`, `right-2 <= [right-1]`, `join <= [right-2]` |

**D49 — the one shape it gets wrong, confirmed in the wild.** `chains` is

```js
const chain = async (name) => { await step.run(`${name}-1`, …);
                                return step.run(`${name}-2`, …); };
await Promise.all([chain("left"), chain("right")]);
await step.run("join", …);
```

The combinator is over the *branch functions*, not over step promises, so there
is no registration to group and `join` falls back to the resume window. It
under-declares — `right-2` is named and `left-2` dangles — which is the
conservative direction, and the same picture that ships today. Added to
`shapes.mjs` as `join over branches`; **every** theory misses it, theory 16
included, because the branch promise is only linked to its step by an adoption
that none of them track. Updated scores over 29 expectations:

| theory | score |
|---|---|
| 3 — resume window (was shipping) | 17/29 |
| 14 — Promise subclass + handler identity | 22/29 |
| 18 — plain promise, own `then` + handler identity | 22/29 |
| **19 — plain promise, own `then` + sync batch (built)** | **27/29** |
| 20 — 19 + barren carryover | 27/29 |
| 16 — `node:v8` promiseHooks | 28/29 |

The UI keeps declining a hop when the SDK has not proved it can report a set —
`reportsJoins`, satisfied by any step naming two parents or carrying alternates.
On `chains` that fallback is not a compromise but the right answer: it draws
`left-2` and `right-2` into a junction and then `join`, which is what happened.

---

## Round 16 — confidence becomes part of the drawing

Two bugs reported from looking at real runs, and a redesign that came out of them.

**D50 — `Promise.race` drew both branches on top of each other.** The
lane-straightening pass pulls each node onto the average lane of its children.
A race's winner has a child and its loser does not (an alternate edge is not a
parent), so the winner was pulled onto the loser's lane and the two overlapped —
which also explains why selecting one in the trace "brought it to the top".
Straightening now assigns desired lanes and then walks the level left to right
giving each node its desired lane *or* one clear of its neighbour, whichever is
further right. Order preserved, nothing overlaps, 1:1 chains still straighten.

**D51 — `tests/v4.deadend` drew `b-deadend` into `c`, which never happened.**
`b` is a step that is started, and awaited only *after* `c` is discovered. The
hop was falling back to a junction, and a junction asserts that everything in
the level completed before the next step ran.

The cause is worth stating precisely, because it defeated two attempted fixes.
These two shapes are **indistinguishable downstream**:

```js
// dead end: c waits on a alone            // join the SDK cannot see:
const pa = step.run("a"), pb = step.run("b")   await Promise.all([chain(), chain()])
await pa; await step.run("c"); await pb        await step.run("join")
```

In both the SDK reports one parent, and in both a sibling finished in time. The
first is complete, the second is half a join — and they need opposite drawings.

- Inferring capability from the payload (does any step name two parents?) fails:
  a run can contain no join at all.
- A `lineageVersion` flag from the SDK fails too, and was built and then removed:
  the SDK cannot mark the second case as incomplete, because it does not know it
  missed anything. It saw a resume window and reported it truthfully.

**D52 — so the drawing carries the confidence instead.** Suggested by the user:
broken lines for connections we are not sure of. Three kinds now:

| edge | claim |
|---|---|
| solid | a dependency the SDK observed |
| dashed | the losing side of a race: could have unblocked it, did not |
| dotted, faint | finished in time to have gated it; nothing says it did |

The ambiguous case is now drawn rather than resolved, and both readings are
compatible with what is on screen. `deadend` draws `a → c` solid with
`b-deadend ⋯> c` dotted; `chains` draws `left-2 → join` solid with
`right-2 ⋯> join` dotted. No junction fallback is needed for either.

Two rules keep it honest and quiet:

- **Timing rules edges out, never in.** A step still running when the next level
  started cannot have gated it, so it gets no edge at all. (Earlier rounds tried
  to infer edges *from* timestamps and were ~6% wrong. Ruling out is sound where
  ruling in is not.)
- **Only where the level converges** — one step, or several sharing parents. On
  a ragged fan-out each step continues its own branch, so an unnamed step is a
  branch that ended, and drawing it into every sibling would suggest it gated
  four unrelated chains. `v4branches-ragged` emits none; `v4pathological` keeps
  `fork ⋯> fork-kid-*`, which is its encoded ground truth, as unproven.

**D53 — every solid edge is true, including a partial join's.** Worth recording,
because an earlier test asserted the opposite. On `Promise.all([fork, bystander])`
an older SDK names one of them; that one *is* a real dependency, so drawing it
solid is correct. It is the *set* that is incomplete, and the missing member is
what the dotted edge is for. The rule is now: solid means observed, and never
asserts something that did not happen.

Fixtures `t19-{deadend,race,parallel,chains,nested}.json` are fresh captures
from the current SDK, replacing pre-lineage ones for these shapes.

**D54 — junctions mark every change of width, not just unresolved hops.** A
circle wherever the run goes one-to-many or many-to-one: the points where the
SDK actually made a discovery and the shape of the run changed. Inserted as a
post-pass over the finished edge set (`insertJunctions`), so levels and lanes
stay as derived; a junction takes the half-level between the two it joins, and
`toFlowElements` interpolates its lane midpoint from either side.

Only proven edges route through it. A race's loser and an unconfirmed ordering
stay as direct broken lines — routing them through the junction would assert
they took part in the convergence.

Still to do: back each junction with its `executor.step.discovery` span, so it
carries real timing and is selectable. Blocked on `loaders/trace.go`, which
drops those spans (`Omit`) after mining them for plans, metadata and timings.
Placement does not need them — `plannedSteps` already says what fanned out — so
this is additive when it lands.

**D55 — canvas chrome.** Legend top-left (collapsed by default, scrolls), zoom /
fit / expand top-right, both in our own tokens. React Flow's `<Controls>` was
replaced rather than restyled: its buttons are hard-coded white with a
white-filled icon, invisible on our canvas in dark mode, and the Tailwind
arbitrary variants needed to override them are not generated from this package.
Expand opens the same canvas in a modal at 92vw × 85vh, since a wide fan-out
only fits in 300px by zooming out past readability.

Broken edges are now drawn *heavier* than solid ones (1.5px, ~0.8 opacity), not
lighter: they carry the least obvious claim and were the easiest to lose against
the dot grid. Dash length, not weight, separates them.

---

## Round 17 — junctions are real, and there is only one kind

**D56 — one rule, both directions.** A junction was two cases (fan-out, fan-in)
resolved separately, which produced a second circle wherever several steps fed
several steps. It is now one rule: group the proven edges by *the exact set of
sources feeding a step*, and draw a junction for any bundle that is not
one-to-one.

```
{a}       -> {b}        1:1, no junction
{a}       -> {b, c, d}  single -> parallel
{a, b, c} -> {d}        parallel -> single
{a, b}    -> {c, d}     both at once, and still one circle
```

So the answer to "do we need any others?" is no — and it turns out those two
were never separate kinds, just the two common shapes of the same thing.

**D57 — lines meeting a junction are drawn straight.** The stepped router turns
every approach into a pair of right angles with a 24px radius; half a dozen of
those converging on a 22px circle is a knot. The circle exists precisely so the
lines meet at a point, so they now take the direct path to it.

**D58 — a junction is a real request, and clicking one says so.**
`executor.step.discovery` spans are gathered onto the run and exposed as
`RunTraceSpan.discoveries` — spanID, status, timings and the steps that response
planned. They are still kept out of the trace tree, which is a tree of steps;
this is a separate field, so the timeline is untouched.

Matching a junction to its request takes two rules, because a discovery reports
what it *planned* and not everything it did:

- **A fan-out planned its targets**, so the plan names them exactly.
- **A step discovered on its own is executed inline and checkpointed**, never
  planned, so it appears in no plan at all. There the request is the one whose
  window the step ran inside — `v4.parallel`'s `d` runs at 338–339ms inside a
  discovery spanning 325–344ms.

Measured on live runs: `parallel` gets `[a, b, c]` in 13ms widening and `[d]` in
21ms narrowing; `nested` gets `[a1, a2, b]` and `[collect]`.

The popover lists **what ran next**, not the raw plan: the plan for an inline
step is the run-complete op, whose hashed id is meaningless on screen, while the
junction's own targets are correct under both matches.

**Unmatched junctions stay plain circles and are not clickable.** `v4.race` is
the honest case: `ParallelMode.Race` schedules a discovery per branch, so no
single request planned both sides and there is nothing truthful to attach.

Also fixed: the fallback junctions carried `spanID: root.spanID`, so selecting
the run highlighted every one of them at once. They carry no span id now, which
is what they are.

**D59 — the circle was in the wrong place, and that is what made the routing
wander.** A junction's position was derived from a lane number averaged across
its neighbours, then offset by a lane midpoint interpolated between the two
levels either side. Both halves were wrong together: each level is centred on
its own lane range, so the same lane number is a different height on each side
of the junction, and averaging across the two mixes incompatible coordinates.
The circle therefore sat off the line its own edges converged on, and the router
bent every one of them around it.

Junctions are now placed *after* the steps, at the mean screen centre of the
nodes they connect. With the circle on the line, the original router is fine
again — the straight-line override put in to hide the misalignment is gone.

**D60 — the narrowing junction is the interesting one, so it says what it is.**
`shouldEnqueueDiscovery(hasPendingSteps, mode)` returns `!hasPendingSteps || mode
== Race`: as each parallel branch finishes the executor *deliberately does not*
ask the function what to do next, and branches that finish together dedupe onto
a shared `ParallelCoalesceKey` to the same discovery JobID. N completions become
one request, and that is worth showing rather than leaving as a dot.

Each junction carries its shape (`diverge` / `converge` / `both`, with counts),
so it is clickable whether or not a request could be matched to it:

- **Fanned out** — "Inngest planned 3 steps here and ran them in parallel."
- **Coalesced** — "All 3 finished, and Inngest continued the run with one
  request rather than 3."

Timing comes from the matched discovery span where there is one. `v4.race` has
none — per-branch discovery is exactly what `ParallelMode.Race` opts out of — so
it shows the shape without a duration.

**Tooling note.** `npx tsc` in `ui/packages/components` resolves to an unrelated
package called `tsc`, which prints a banner and exits 0. Several "typecheck
clean" claims this session were that banner. Use `./node_modules/.bin/tsc`, or
the apps' own `pnpm type-check`, which is what caught the leftover reference the
banner hid.

**D61 — the loops around the outer branches were the router overshooting.** A
junction sits in the gap between two node columns, which is `columnGap -
nodeWidth` wide — 66px, so ~22px either side of a 22px circle. React Flow's
smoothstep runs 20px straight out of each end before turning, so it wanted 40px
of approach in 22px of space: it overshot past the node and doubled back, which
is what drew a bracket around the top and bottom branches. Edges touching a
junction now turn after 8px with a 6px radius, which keeps the whole path inside
the gap. Two bends each, which is the minimum for an orthogonal fan.

**D62 — the popover says what the engine did, and nothing the chart already
says.** The steps are named on the canvas, so listing them again was noise:

- **Running in parallel** — "Inngest started 3 steps in parallel."
- **Back to sequential** — "All 3 steps finished, so Inngest carried on with a
  single request rather than 3."

**D63 — junction edges are beziers, because a stepped corner cannot fit there.**
The rounded right angles everywhere else come from a radius the router caps at
half the straight run before the turn, and that run has to stay under half the
gap or the two ends cross and the path doubles back. In the 22px beside a
junction that caps the corner at ~5px, against ~16px in the 66px between two
node columns — which is why the fan-out read as hard right angles while
everything else looked curved.

A curve needs no straight run at all, so those edges are drawn as beziers. It is
the only place on the canvas where a line has to change lane over a short
distance, and it now matches the softness of the rest rather than fighting the
geometry.

**D64 — an unproven contributor joins the convergence; a race loser does not.**
Junctions were built from proven edges only, which drew a circle where a run
fanned out and nothing at all where the same run came back together — the two
halves of `v4.chains` looked like different diagrams.

The exclusion was too blunt. An `unconfirmed` edge only exists where the source
had *already finished* before the target started (that is the timing rule that
creates it), so the convergence is true as an ordering however the dependency
turns out; the line stays broken to carry that doubt. A race loser is different
in kind — it is known *not* to have contributed — so routing it through would
say the opposite of what happened, and it stays a direct broken line.

Where a bundle is mixed, the junction drops the claim about what the engine
waited for: "2 steps finished here, and the run carried on as one" rather than
"…so Inngest carried on with a single request rather than 2". The count of
unproven contributors rides on the junction's shape.

**D65 — junction columns are wider, so every line is routed the same way.**
Reverting D63: beziers were treating the symptom. The circle sits in the middle
of the 66px gap between two node columns and halves it, and the stepped router
needs its straight run to be under half the span or the two ends cross — so 22px
allowed an 11px run and a ~5px corner, against 10px on every other hop.

Gaps that contain a junction are now `columnGap + 60` wide (`LAYOUT.junctionGap`),
which leaves 52px either side: the default 20px run fits, and the corner comes
out at 10px — identical to a plain node-to-node hop. Column positions are laid
out cumulatively rather than as `level * columnGap`, so only the gaps that hold a
junction pay for it and the rest of the graph keeps its pitch. Every edge on the
canvas is now a single routing rule again, with no special case.
