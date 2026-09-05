# Run canvas — running log

**Testing bar for this work (set by the user, item A onwards).** This is a proof of concept: enough
tests to keep iteration from silently breaking things, not a complete suite. Where a test would be
complex and the risk is low, a comment is preferred. Model-level tests are cheap and stay — the
collapse tests caught a real bug before anything was rendered. Render-level tests stay light, with
the fixture gallery and the two review agents (`canvas-ux-review`, `canvas-correctness`) doing that
job instead. If a class of breakage starts recurring, that is the signal to test it properly.

Terse, append-only, newest at the bottom. One entry per item from `PROMPT.md`: what changed, how it
was verified, what is uncertain, what was deferred.

`HANDOVER.md` is the cold-start doc and `todo.md` is the decision log D1–D65 from the lineage work.
This file is the log of the visualisation work that followed.

---

## Item 0 — Orientation

**Dev server.** `pnpm run --filter @inngest/dev-server-ui dev:vite` running in the background for
the whole session. Ports 5173 and 5174 were already taken, so it is on **:5175**. `dev:vite` rather
than `dev` deliberately: `dev` also spawns a graphql-codegen watcher that needs the Go package tree.

**Read first**, before writing anything: `graph.ts` + `graph.types.ts`, `Timeline.tsx` /
`TimelineBar.tsx` / `TimelineBar.types.ts`, `TimeBrush.tsx` / `TimelineHeader.tsx`,
`RunDetailsV4.tsx`, `runDetailsUtils.ts`, `utils/timing.ts`, `__fixtures__/README.md`,
`HANDOVER.md`, `lessons.md`.

### Findings that changed the plan

- **`calculateBarPosition` and `calculateDuration` are already separate** (`utils/timing.ts:37,64`),
  and `calculateBarPosition` has exactly one caller (`Timeline.tsx:499`). So item B's invariant —
  only the axis compresses, every reported duration stays wall-clock — can be made structural rather
  than a rule someone has to remember: only the position function ever learns about the scale.

- **Item C is half-built already.** `TimeBrush.tsx` (364 lines, 32 assertions) is a general
  drag-range control, and `TimelineHeader.tsx:199` renders a flat status-coloured bar inside its
  `children` slot. The density strip is a histogram dropped into that slot, not a new control, and
  drag-to-set-viewport comes free.

- **Item E's span-links shortcut is dead.** Checked because the brief flagged it as unconfirmed and
  cheap. `FollowsFrom` *is* written — `executor.go:3419`, `:3601`, `:4029` and ~10 more — but only
  for pause spans and lifecycle queue items, i.e. **within** a single run. Nothing writes a link
  between a parent's invoke span and a child's root span. And it would not reach the client anyway:
  `links` is not in the GraphQL schema at all, and `gql.schema.graphql:641` carries the literal
  comment `# links should be here`. Exposing it is a Go change, which is a shared surface and a
  stop-and-ask, so item E uses only the read-only route the brief lists as its fallback
  (`childRunID`, `trigger` event ids, `sendEvent` `output.ids`, `/v2/events/{id}/runs`).

- **The gallery needs no platform work.** dev-server-ui aliases `@inngest/components` straight to
  source (`vite.config.ts`) and JSON imports already work in the canvas tests, so a route can render
  the real `Canvas` and `Timeline` from a committed fixture with no GraphQL. `Canvas` takes only
  `{trace, runID, getTrigger?}`. Being in dev-server-ui it is Cloud-excluded by construction —
  Cloud is a separate app (`ui/apps/dashboard`).

- **No new dependency is needed** for any item. Sparklines and the density strip are inline SVG.

### Smaller things noticed, not yet acted on

- `traceRollup` runs **twice** per render — `RunDetailsV4.tsx:84` and `:136` — each
  `structuredClone`ing the whole trace. Hoist when item C adds a third consumer.
- `TimelineBar.tsx:871` sets row height twice, as `h-7` and as an inline
  `height: ${ROW_HEIGHT_PX}px`. They agree today; the class is the stale one. Fix in item B2.
- `__fixtures__/README.md` documents 19 of the 38 fixtures. `v4deep3.json` has no importer.
- `useStepSelection` / `stepSelectionEmitter` have no test coverage. Item D adds the first.
- The Dev Server's `function_runs` table has no primary key and no indexes (`schema.sql:38-51`), so
  event→runs is a scan locally. Recorded as a follow-up; not changed, because Go stays read-only.

### Decisions taken up front

Recorded here rather than asked, per the brief's §5.

- **Batched runs: one start node, not N.** The event node carries `batchSize` and draws as the
  existing offset-card stack; it becomes expandable to the N events. One run is one beginning.
- **Cancelled is a third state.** `status.cancelled` tokens, distinct from both success and failure.
  A step still running when the run was cancelled reports `(interrupted)` where a duration would
  be — Terraform's `(known after apply)` rule: a token in place of the value, never a silent gap
  that reads as "nothing happened".
- **Restyle the timeline in place**, with no Cloud-gated second presentation. The branch has no
  upstream and is never pushed, so there is no production exposure, and two presentations of the
  same component is real ongoing cost.
- **Elastic axis on by default**, but only for gaps clearing both a relative and an absolute floor,
  and always drawn as a visible break.

Nothing deferred. Nothing uncertain beyond the two checks item E still owes (whether `sendEvent`
`ids` are internal ULIDs).

---

## Item A0 — The fixture gallery, and the shapes the set was missing

Two commits: the gallery, then the captures.

**The gallery** is `/canvas-gallery` in dev-server-ui — every committed fixture rendered through the
real `Canvas` and `Timeline`, with the facts that matter beside it. It needs no GraphQL: the package
alias resolves to source, so the route imports the fixture and runs the same
`traceRollup → toCanvasGraph / traceToTimelineData` the real view runs.

Dev-only twice: this app is not the Cloud dashboard, and the route redirects away unless
`import.meta.env.DEV`. The fixture payloads sit behind a dynamic import guarded on the same
constant, so Rollup drops the branch and the JSON never reaches the shipped binary.

`__fixtures__/index.ts` is hand-maintained rather than globbed, because this package is also
consumed by Next.js in Cloud and `import.meta.glob` is not a feature every consumer has.
`index.test.ts` makes an omission fail rather than go unnoticed, and builds a graph from every
listed fixture, since the gallery renders them unguarded.

**It immediately did its job.** The node/row counters against the ~40 budget read, at rest:

| fixture   | nodes  | rows   |
| --------- | ------ | ------ |
| `loop40`  | 82/40  | 82/40  |
| `tall500` | 502/40 | 502/40 |
| `wide`    | 17/40  | 15/40  |

`tall500` renders a completely blank canvas — fitted to 502 levels, there is nothing left to see.

**The four missing shapes**, all captured for real except where stated:

- `cancelled.json` — the README's known gap. Earlier attempts raced the run to completion, so
  `tests/v4.cancel` parks on a ten-minute `waitForEvent`. The capture is a `CANCELLED` run with a
  step still `WAITING`: the third state, exactly as §3.3 describes.
- `tall500.json` — 500 sequential steps, 502 spans, 726KB. Under the 1MB line, so no pruning was
  needed and it is an unmodified capture.
- `blocked.json` — the losing half of two runs on `concurrency: { limit: 1 }`. 6.3s queued before
  executing, visible as run-level queue delay. Capturing the *winner* gives an ordinary run with
  nothing to see; the README now says so.
- `longgap.json` — **synthetic, and labelled as such** in `index.ts`, in the README, and in orange
  at the top of the gallery. `make-longgap.mjs` takes the real `step.json` and pushes every
  timestamp at or after the sleep forward by seven days. Only the clock moves; span shapes and work
  durations are untouched. Result: 7 days elapsed, ~0.2s executing.

`capture.mjs` replaces the README's "issue the GetRun query and save the data object" — accurate,
and fifteen minutes by hand. Its query must be kept in step with `coreapi.ts`.

**Found while looking, not yet fixed** — `cancelled` reports its open wait as `3m 58s` in the
timeline, because an unfinished span is measured against `now`. The run was cancelled at 35s. That
is a lie of exactly the kind §3.2 warns about, and it wants `(interrupted)` rather than a number.
Fixing it belongs with the cancelled-state work, not here.

**Other changes**: the README table covered 19 of 41 fixtures and now covers all of them, grouped as
the gallery groups them. `eslint.config.mjs` gained a Node-globals override for standalone `.mjs`
scripts under `src/`.

**A reviewer agent** — `.claude/agents/canvas-ux-review.md` — screenshots the gallery and reviews it
as UX/DX QA. Added at the user's suggestion, and run per item from here on so the visual result is
judged rather than assumed.

**Verified**: 440 component tests pass (25 files); eslint clean on `src/RunDetailsV4`; dev-server-ui
and dashboard both `tsc --noEmit` clean. `tests/js` has one pre-existing type error in
`sdk_retry_test.ts`, untouched by this work and last changed in `07bb954a6`.

---

## Item A — Collapse repetition

**Result.** The acceptance criterion is a number, so here it is, at rest:

| fixture   | nodes before | after | rows before | after |
| --------- | ------------ | ----- | ----------- | ----- |
| `tall500` | 502          | **3** | 502         | **3** |
| `loop40`  | 82           | **4** | 82          | **4** |
| `wide`    | 17           | **6** | 15          | **4** |

Detection is one pure module, `canvas/collapse.ts`, over the finished `CanvasGraph`. Deliberately
not inside `graph.ts`: that file is about what the trace *says*, this one about how much of it to
draw. Nothing in `graph.ts` changed, so all five existing test files stayed green throughout.

**Two group kinds.** `iteration` finds the shortest repeating period in the level signatures, so a
think/act loop collapses to its *body* — two nodes of 40 — rather than 80 unrelated things, and 500
sequential steps to one. `siblings` collapses lanes within a level: the fan-out.

**Carrying the variance.** `variants` gives "38 × search, 2 × write_file" when names genuinely
differ. When every label is distinct it is *numbering*, not variance — `think-0 … think-39` is forty
labels and no variation — so it renders as the range `think-[0…39]` instead. Calling that "40
variants" would be noise dressed as signal. `exceptions` names failed / retried / cancelled / still
running / slow members; slow uses median absolute deviation, because one 30s outlier in a loop of
50ms steps drags a mean until it stops flagging anything. A group takes its **worst** member's
status, so a group holding a failure is red at rest without anyone expanding it.

**Envelope, not percentiles** for fan-out: first start, last start, stagger. The stagger *is* the
concurrency limit made visible, and it is the one thing a p50/p99 discards.

**Both views, one plan.** In the timeline a group is a row whose `children` are its members, so it
reuses the existing expansion, selection, tooltip and keyboard behaviour rather than adding a second
idiom. Group rows deliberately carry **no** phase breakdown: those belong to the members, and
claiming one member's discovery timing as the group's would be a fabrication.

**Defaults.** `shouldAggregate` budgets each dimension separately. Node count alone was wrong — a
12-wide fan-out is only 17 nodes and still unreadable, because the problem is twelve lanes in a
300px pane. Levels > 12, widest level > 6, or nodes > 40.

**Two bugs the tests caught before anything was rendered:**

- Junction nodes live in `graph.nodes` but **not** in `graph.levels` — `insertJunctions` adds them
  on half-levels after levelling. Rebuilding the collapsed graph from `levels` alone dropped them
  and severed the fan-out from its collect step. They are now carried separately and repositioned
  from their edges.
- A group's range label was taken from the first and last member in *run* order. A fan-out's members
  are in lane order, so `wide` would have read `w[8…3]`. Ranges now come from a numeric-aware sort.

**One bug the tests did not catch, because it was not in the code** — see `lessons.md` #16. The
gallery reported `1 group, aggregated=true` while the `Canvas` on the same page drew all 502
uncollapsed nodes: Vite was serving a stale `collapse.ts` to one importer. Restarting Vite fixed it
instantly. Detection signal: **the UI contradicts a passing unit test on the same input.**

**Deferred / uncertain**

- The sparkline on `tall500` is nearly flat, because 500 steps of 1–4ms genuinely are flat. Correct,
  but not informative. Worth revisiting once the density strip (item C) exists, which may be the
  better home for that signal.
- `MIN_MEMBERS` is 3 and the slow threshold is 3×MAD. Both are judgement calls, not measured.
- The review agents live in `.claude/agents/`, which this repo **gitignores**, so they are not
  committed. Left as-is rather than forcing past the ignore rule — worth a decision.

**Verified**: 492 component tests pass (26 files, up from 440); eslint clean; dev-server-ui and
dashboard both type-check; the four numbers in the table above read off the running gallery.

---

## Item B — Elastic time axis, then the restyle

Two commits, because they are two different kinds of change: one is a model, one is presentation.

### B1 — the axis

`utils/timeScale.ts` maps time to width piecewise, giving each qualifying idle stretch a fixed
narrow band instead of its true width. Elided time renders as a filled band bounded by dashed rules,
with its real duration set vertically inside — visibly a break, not a gap.

**The invariant is structural, not a convention.** `calculateBarPosition` is the only function that
takes a scale; `calculateDuration` reports wall-clock and cannot see one. There is no path by which
axis compression could change a number the user is shown, and a test asserts that rather than
trusting it.

**Deciding what counts as idle took two attempts.** The first counted every leaf bar as busy and
produced no breaks at all on `longgap` — because the seven-day *sleep* was being counted as work.
A sleep is precisely the suspended stretch worth eliding. `step.invoke` stays busy: a child run
really is executing. Parent bars are excluded too, since a parent spans its children and would fill
every gap they left.

Two floors so a break earns itself: 5s absolute **and** 15% of the run. Across all 41 fixtures only
two break — `longgap`, and `blocked` on its 6.3s concurrency hold, which is exactly the queued phase
that fixture was captured for. `step` (2s sleep) and `waittimeout` (3s) stay linear, correctly.

Axis markers now come from the scale. Under compression they are evenly spaced in width but not in
time, and a marker landing inside an elided stretch is dropped: labelling a time from a stretch the
axis is explicitly not drawing is worse than no label.

### B2 — the presentation

Rows 28px → 18px (and the stale duplicate `h-7` removed, so height has one source again). The
per-row centre line and per-row vertical guide are gone; **one** vertical rule separates the label
column from the plot, drawn once by the container. Label column recedes — mono, 11px, muted.
Durations mono with tabular numerals. Bars thin with a 1.5px radius.

**The encoding change is the one that matters.** Waiting is now a different *substance* from
working: neutral and hollow, against solid colour for execution. Previously both were green and the
distinction rode on a fill pattern nobody would read. On `step` the 2s sleep is now unmistakably not
work. No new colour values — every class was already in the preset.

All 502 tests passed throughout, which is the evidence this was presentation and not behaviour:
nothing asserted the old row height or the centre line, and no affordance was removed.

**Deferred, honestly**

- The brief asks for annotations *immediately right of the bar* ("890ms — longest in the fan-out").
  Durations still sit at the right edge of the label column, adjacent to the divider and so next to
  the bars, but not inline with them. Moving them is a larger change to the row layout than the rest
  of this pass and is not done.
- The legend is still in the tab label rather than hairline-separated at the bottom.
- `blocked` eliding its 6.3s concurrency hold is arguably wrong for that one case — the queue delay
  is the thing you want to *see* there. It is marked rather than hidden, so it is not lost, but item
  G may want to treat flow-control holds differently from idle sleeps.

### Idea parked during item C — a packed ("swimlane") timeline

From the user: instead of one row per step, pack steps into as few rows as possible. Sequential work
shares a single row laid out left to right; parallelism is what forces a second, third, fourth row.
Greedy interval packing — a step goes in the first row whose previous bar has already ended.

Why it is worth doing: **the row count becomes the concurrency profile**. A run that is 40 steps but
never more than one at a time is one row, and a 12-wide fan-out is visibly twelve. That is a
property the current view cannot show at all, and it collapses the tall case for free.

The honest cost is the label column: with several steps per row there is nowhere to put per-row
names. Mitigations, in order of preference: the label inline inside the bar when it is wide enough,
an annotation immediately right of the bar when it is not (which item B2 deferred and wants anyway),
and the existing hover card for the rest.

Planned as a third view mode alongside the collapsed/expanded toggle, after item C.

---

## Item C — The density strip

A thin full-width histogram sharing the timeline's axis, inside the existing `TimeBrush` rather than
beside it — so dragging to set the viewport is the gesture that was already there. `TimeBrush` gained
a `trackClassName` because at its `h-4` track the strip was a two-pixel line.

**The acceptance test passes.** The brief states it as a sentence: a failure cluster two-thirds
through a 500-step run should be a red smear visible in the first screenful with no interaction. It
is. No captured run had such a cluster, so `failcluster.json` is generated from the real `tall500`
capture with the status of a contiguous block rewritten — marked synthetic in `index.ts`, in the
README and in orange on the page, same as `longgap`. A test asserts the checkable half (the red
lands in the right third and stays contiguous); whether it is *visible* is for the eye.

Two properties are load-bearing and easy to break later:

- **It renders the whole run, always**, however much is collapsed below. Built from `data.bars`,
  never the collapsed bars. The strip is the map; collapsing is a property of the territory.
- **It shares the axis, breaks included**, so a column really does sit above the rows it describes.

Fewer states than `CanvasStatus` on purpose — a five-pixel column read at a glance can only carry
distinctions a reader can act on.

**A bonus from item A**, visible on the same fixture: the collapsed group node draws **red** with
"21 failed" on it. The worst-status rule means a group holding a failure cannot present itself as
green, so the cluster is legible on the canvas too, without expanding anything.

**`lessons.md` #16 recurred.** The strip rendered nothing while its own unit tests passed — Vite
serving a stale copy of the two new modules. Restarting fixed it instantly, again. The rule is now
firm: after adding a *new module* imported from more than one place, restart Vite before debugging
anything.

**Not done in this item**: dragging the strip currently sets the timeline viewport (inherited from
the brush) but does not yet drive the canvas — the shared viewport belongs with item D's emitter.
`onSelectRange` exists on the component and is not yet wired.

---

## Item B follow-up — the waiting encoding, twice

Two rounds of user feedback on the same row, worth recording because the second overturned the first.

**Round 1, the problem.** A step that queues for seconds and runs for milliseconds drew a long grey
bar and a green speck about a pixel wide. The eye went to the waiting, which is the least interesting
thing in the row, and a perfectly healthy run looked broken. Cause: waiting was a *different colour*,
so it read as a separate object competing with the work rather than as part of the same step.

**Round 1, the fix that was wrong.** Drew the waiting stretch hollow — the step's own status colour
as a 1px inset ring. Correct as an encoding, and ugly: a thin ring reads as a border artefact rather
than as a quantity.

**Round 2, the fix that stands.** Same fill, a quarter of the weight. A row is now one continuous bar
that is green because the step succeeded and simply becomes solid where the work is. `step.sleep` and
`step.waitForEvent` are ghosted the same way rather than striped — **weight carries this better than
texture**. Bars thickened twice on request: tall 6 → 10 → 12px, rows 18 → 20 → 22px.

**The process lesson.** The UX review agent passed the original grey-slab row, because it was
checking whether the encoding was *correct* and not whether it looked good. It now has an explicit
"does it actually look good" criterion naming that row as the canonical failure — hollow outlines,
bars too thin or too thick, and any element that dominates a row while carrying the least
information.

### What the UX review did catch (item A), all fixed

- **The group expander was a one-way door.** Expanding a group on `loop40` dropped the fit to
  `scale(0.087)` — the exact hairline collapsing exists to prevent — and `openGroups` was never
  cleared, so the toolbar toggle could not undo it. Page reload was the only recovery. The control
  now re-collapses open groups first and only switches mode once there is nothing left to close.
- **The sparkline contradicted its own exception count.** Max-normalised against a zero baseline, so
  `think`'s single 1.762s outlier squashed the ~160ms series into the bottom tenth of a 10px box: a
  node reading "3 slow" drew one spike and looked like it was lying. The reviewer verified the
  *threshold* was right (think-32 1.762s, think-25 433ms, think-29 409ms against a ~160ms median), so
  the fix was the chart, not the statistic. Now min-max normalised, and suppressed entirely when the
  spread is under 15% — on `tall500` it was rendering as a horizontal rule that read as an underline.
- **The exceptions string hard-clipped.** `shrink-0` meant the span never gave ground, so
  `text-overflow` never fired; a full-house string overflowed its box by 169px and the tail vanished
  with nothing to signal it.
- **Icon collision.** The group expander used `RiExpandDiagonalLine` — the canvas *fullscreen* glyph,
  ~50px away in the toolbar — in a 12×12px hit target. Now a chevron at 20×20 with a hover state.

### Still open from that review

- The group node is drawn at fit zoom, which varies 0.745 (`wide`) to 1.20 (`tall500`), so on `wide`
  the 10px meta row renders at ~7.4px and is genuinely not readable. A minimum legible zoom, or a
  size that does not depend on fit, is the real fix.
- A group's only cues are the count pill, the stack and the sparkline — the icon is the same one
  ordinary step nodes use. `for 2s` gets a dashed border and its own icon; a group is at least as
  different a kind of thing.
- In dark mode the stacked cards' `border-subtle` renders brighter than the green status border,
  inverting the hierarchy.
- `variantLine` and `Sparkline` are mutually exclusive, so a mixed-name group — the most interesting
  kind — loses its timing shape.
- The `…` does double duty: range inside brackets, truncation everywhere else.
- Confirmed from the payload: `cancelled` reports its open wait as `29m 47s` on a 35s run. Same
  unfinished-span-measured-against-`now` bug already noted in item A0. Still unfixed; belongs with
  the cancelled-state work.

---

## Item D — Hover sync

`stepHoverEmitter` + `useStepHover` in `runDetailsUtils.ts`, the same idiom as the two emitters
already there, but carrying a span id rather than a whole `Trace` — hover fires constantly and the
views only ever compare identity.

**Subscribed once per view**, at the container, and passed down. Per-row subscription would mean 500
listeners and 500 re-renders per pointer move on `tall500`; this way it is one re-render and a string
comparison. Selection keeps its ring; hover gets a lighter lift, so a decision and a glance do not
look alike. Focus is not filter — nothing is ever removed from another view.

**Also, from the user, and better than what was there**: a collapsed group's timeline row now draws
its *members* rather than one continuous block. A solid bar said only "something happened here for
1.9s"; a run of marks says **where in the sequence** the failures were. On `failcluster` the row is
green ticks with a red cluster two thirds along, agreeing with the density strip above it, with
nothing expanded.

Two refinements after seeing it: marks are uniform with an even gap (duration-proportional widths
read as ragged — the marks answer "where", and the row label already says "how long"), and the number
of marks comes from the **measured** plot width rather than a constant, so a narrow pane gets fewer,
larger marks instead of a smear. Past that budget members are bucketed, and a bucket takes its worst
member's status.

---

## Item E — Lineage

**Both cheap checks were run first, and both answers changed the item.**

1. **Span links are dead.** `FollowsFrom` really is written — `executor.go:3419` and ~10 more — but
   only between pause and lifecycle spans *within* a run, never parent-invoke → child-root. And it
   would not reach the client anyway: `links` is absent from the GraphQL schema, where
   `gql.schema.graphql:641` still carries the comment `# links should be here`. Exposing it is a Go
   change, i.e. a shared surface. Nothing here walks span links.
2. **Sent-event ids ARE internal ULIDs.** Verified end to end with a new `tests/v4.emit` shape:
   output is `{"ids":["01M1Q00QZAD5YFMY4TP9PH0YZ3", …]}`, and
   `GET /v2/events/01M1Q00QZAD5YFMY4TP9PH0YZ3/runs` returns the run it caused, whose own trigger
   echoes the same id. The loop closes.

**A correction to the brief, and it is the load-bearing one.** A `step.sendEvent` is **not**
`stepOp == SEND_EVENT` — it is reported as an ordinary `RUN`, and keying on `stepOp` finds *nothing*.
What identifies it is **`stepType == 'step.sendEvent'`** (mirrored in `stepInfo.type`). That is the
difference between the feature being affordable and not: with it, emitting steps are found for free
from the trace already fetched; without it, the only way to find them is to fetch every step's output
and see which happen to contain an `ids` array — one request per step of the run.

The correction also fixed a bug the canvas had **by its own contract**: a node is meant to name the
SDK call the user wrote, and it was calling every `sendEvent` `step.run`, because it labelled from
`stepOp`. Nodes now prefer `stepType`.

`lineage.ts` is the model — emitting steps, invoked child runs, and a parser that returns `null`
rather than `[]` when it cannot read a payload, so "sent nothing" and "cannot tell" stay
distinguishable.

**Deferred**: the recursive lineage *UI* (walking up to the source event and down to everything
caused) is not built. The mechanism is verified and the model exists; assembling it needs
`useGetRunLinkage`'s shape extended in both host apps plus a REST call for event→runs, which is a
larger surface than the rest of this item. Also still true and stated in the module: a bare
`inngest.send()` outside a step has no span and no output, and batch fan-in is a join, not an edge.

---

## Item G — What Inngest did

One line beside the canvas. `value.ts`'s design is mostly what it refuses to say: factual and
quantified or absent, never celebratory, and never a claim the trace cannot support. `simple`
produces no lines at all, and a test asserts that — a plain run gets a plain page.

Three things the brief asked for are **not** reported, with the reason recorded rather than
approximated:

- **Memoized steps on resume.** Nothing in the payload distinguishes a step served from state from
  one that ran; both have spans and timings. Needs a signal the SDK/executor does not emit.
- **Survived a deploy.** A run's trace does not record deploys.
- **Deduplicated / debounced N into 1.** The run that was kept has no record of the ones that
  were not.

A retry counts as a recovery only when it *recovered* — `failure` asserts a step that stayed failed
claims nothing. Flow control is only claimed above a second, since a few hundred milliseconds is
ordinary scheduling.

**A bug the tests caught**: `runValue` must take the rolled-up trace and read only its direct
children. The raw payload lists a retried step three times — an attempt group plus two flat copies —
so a recursive walk reported one recovery as three. Overclaiming, in the one file that must not.

---

## Item F — Invoke expands inline (timeline half)

**Done**: a `step.invoke` row can pull the run it started in beneath it, indented, on the same axis.
Loaded **lazily** — nothing fetched until the row is actually expanded, which is the failure the
brief names — with an in-flight guard against a double toggle and a cache so reopening costs nothing.

The child grafts onto the existing bar tree, so it inherits expansion, selection, hover and tooltips
rather than needing its own. The only new data is `childRunID` on a bar, lifted from the invoke's
`stepInfo`, which the payload already carried. The gallery proves it with no server: `invoke` and
`child` were captured as a pair for exactly this.

**Not done**: the canvas does not render the child's graph *inside* the invoke node. That needs a
nested React Flow with its own provider and depth capping — a larger surface than the rest of the
item, and worth doing deliberately rather than at the end of a session. The data and the loader
contract are both in place for it.

**Not done, and needs app wiring either way**: real apps must supply the loader. In dev-server-ui and
dashboard that means fetching another run's trace through `SharedContext`, the same way
`getTrigger` is already supplied to the canvas.

---

## Where this leaves the list

| item | state |
| ---- | ----- |
| A0 fixture gallery + missing shapes | done |
| A collapse repetition | done |
| B1 elastic axis | done |
| B2 density and encoding | done, two follow-up rounds on the waiting bar |
| C density strip | done; strip → canvas viewport not wired |
| D hover sync | done; shared zoom not wired |
| E lineage | verified and modelled; recursive UI not built |
| F invoke inline | timeline half done; canvas half not |
| G what Inngest did | done |

**The three things I would do next, in order.** Wire the density strip's `onSelectRange` to the
canvas viewport (item C's leftover, and small). Fix the `(interrupted)` lie on unfinished spans —
`cancelled` still reports its open wait as 29m on a 35s run, which is the oldest known-wrong thing
on this list. Then the canvas half of F.

---

## Review round — what the agents caught

Three reviews landed after the items were built. Roughly half of what they found was already fixed;
the rest was real and is now fixed. Recording the findings rather than just the fixes, because the
*category* is the same in every case: **the view asserting something the run did not do.**

### Overclaims, all now fixed

- **"3 slow" named a trend as three anomalies.** `loop40`'s think durations are a smooth ramp —
  …213, 299, 335, 370, 1675. A MAD threshold alone lands at ~335, so 299 was fine and 335 was
  "slow": an 11% difference deciding it *inside* a continuous distribution. Only 1675 (12× median) is
  a genuine outlier. A member must now also be ≥4× the median, so "slow" means anomalous, which is
  what the word claims. `loop40` now says **1 slow**, and that is true.
  *A monotonic ramp is a real and interesting thing that nothing currently describes — it is simply
  no longer described wrongly.*
- **Parallel branches fused into "one step that ran 24 times".** `v4branches-balanced` is six
  branches of four steps (`br0-1` … `br5-4`); both digit runs normalise, so all 24 shared a shape.
  An iteration repeats over *time*, so each repetition should own its level; several members of one
  shape on a level ran in parallel. It now reads as **four fan-outs of six**, which is the run.
- **A range that was not a range.** That group titled itself `br[0-1…5-4]`. Range labels now require
  both ends to be purely numeric.
- **Group rows began after their own children.** Ordinary bars start at `queuedAt`; the group row
  used the envelope's first *start*, so expanding revealed children beginning before their parent —
  88ms overhang on `wide`, ~7% of the run width — and silently dropped the queued phase every
  sibling row includes.

### Presentation, fixed

- **The waste was horizontal, not vertical.** ~420px per row: a 35% (≈376px) label column for
  two-to-six-character labels, plus a hard-coded 48px badge gutter that was empty on all eight
  fixtures reviewed. Left column now 24%, and the badge gutter collapses when no row in the timeline
  has a badge (decided once, so bars stay aligned).
- **The gallery never rendered `TimelineLegend`** — it is mounted in `RunDetailsV4`, not `Timeline`.
  The tool being used to judge the encoding omitted the key to it.
- **The facts row mixed collapsed and uncollapsed numbers** — `edges 501` beside a picture with two
  lines in it. Structural counts are now labelled `(full)`.

### Already fixed before the reviews arrived

One-way group expansion, sparkline normalisation, flat-sparkline suppression, the expander/fullscreen
icon collision, the hard-clipping exceptions string, and the full-width centre line camouflaging
small bars.

### Confirmed correct, and worth keeping

The correctness agent verified against the payloads, by set equality rather than eyeballing: 500
members are exactly `s0`..`s499`; 80 are exactly `think-0..39` + `act-0..39` with a real period-2
split; `wide` is 12 with `collect` correctly excluded; nothing invented or lost (500+0, 80+0, 12+1
each equal the payload step count); **every member duration byte-identical before and after
collapsing**, across all four groups; and collapsing fires on only 8 of 43 fixtures.

### Still open from the reviews

- **`barber-pole` is height-dependent.** A -45° gradient with an 8px period does not cross a 6px bar.
  Bars are now 12px so it survives, and the waiting/working distinction moved to *weight* (ghosted vs
  solid), which is height-independent — but the executing texture is still a 15%-opacity tint that is
  nearly invisible in light mode.
- ~~Numbers that disagree with their own bar.~~ **Fixed** — see below.
- **The hover card is the best thing in the component** and nothing on the row signals it exists; its
  hover target is `minWidth: 4px`, so on the runs where you most need it, it is unreachable.
- **`tall500`'s group bar** spans 1885ms for 500ms of work. Honest by its stated definition, and the
  per-member marks now show the gaps, but a big bar for a little work.
- Group nodes are drawn at fit zoom (0.745 on `wide`), where 10px meta text renders ~7.4px.
- Dark mode: stacked-card `border-subtle` is louder than the green status border, inverting the
  hierarchy.

### The bar/span mismatch — fixed, and it needed three sources to pin down

Worth writing up because no single reviewer had it, and one of them was reporting on a fixture they
had never looked at.

- The **UX baseline review** saw the symptom: `blocked`'s `hold` row reads 5.999s against a bar
  spanning most of the plot.
- The **correctness agent** later objected that this was not its finding, checked the payload, and
  reported that `hold` has *no queued phase at all* — 0ms queued, 5999ms executing — so nothing in
  the span justifies a wider bar. It asked me to confirm which I was seeing before fixing.
- **Measuring the live DOM** settled it: the bar rendered as two segments, 48.4% and 46.1%, under a
  label saying 5.999s. Both reviewers were right about their half.

The cause was in neither place. `hold` carries `inngest.timing` metadata with
`queue_delay_ms: 6299` — the **run-level** concurrency hold, stamped onto the step that followed it.
`getTimingFromMetadata` prefers metadata, so the breakdown totalled 12298ms against a 5999ms span
and half the bar was drawn as waiting the step never did.

**The rule now**: the bar is the span, and its segments partition it. Where metadata claims more than
the span, timestamps win and the split is re-derived from the span. `hold` draws as one solid bar
matching its label. The 6.3s hold is real and unchanged — it belongs to the run span, one row up,
and item G's value line already reports it.

Pre-existing rather than introduced, but the ghosted-waiting change **made it worse**: as a separate
grey bar the mis-attribution read as platform overhead; as part of the step's own bar it claimed the
step spent half its life waiting. A more honest encoding made a latent wrong number louder, which is
worth remembering.

`failure` was named alongside it in the original report and does **not** have this shape — the whole
run is 653ms with no multi-second period to misplace. Not every fixture named in a review is
implicated; that one was checked and cleared.

---

## Discovery: a wrong turn, reverted, and the design that replaces it

**What I did, and why it was wrong.** Reclaiming the leading run delay made the remaining gaps in the
trace conspicuous, so I extended each step's bar backwards over the gap that preceded it, drawing the
interval as that step's ghosted lead-in. On a sequential run it looked right.

It is a fabrication, and the user caught it: **a discovery is its own request, and one discovery can
lead to many steps, one step, or none.** Extending every step back over the discovery that planned it
draws one request N times. The damage was measurable on `wide`: the collapsed group went 262ms →
322ms, and `collect` claimed **719ms** for a step that ran 144ms. Reverted in full.

**The data says exactly this.** `t19-parallel`'s discoveries:

```
q=  1  s= 74  e= 87   planned 3      <- one request, three steps
q= 88  s=229  e=265   planned 1
q= 88  s=226  e=247   planned 1
q= 88  s=227  e=264   planned 1
q=267  s=372  e=393   planned 1
```

The first is 73ms queued plus 13ms running and produces `a`, `b` and `c`. Bundled into steps it would
have been drawn three times.

**The constraint that shapes the design.** Only **9 of 43** fixtures carry `discoveries` at all —
`loaders/trace.go` sets `Omit` for `executor.step.discovery`, so discovery spans never reach the
client and only the root's `discoveries` array survives, inconsistently. `wide`, the fixture that
exposed the bug, has none. Fixing that properly is a Go change, i.e. a shared surface and a
stop-and-ask.

**The design.** A discovery is never part of a step. It gets **one row, with a mark per discovery**,
each at its real interval and labelled with how many steps it planned:

- 1→N is drawn correctly: one mark, then several steps begin after it.
- 1→0 is drawn correctly: a mark with nothing following, which is a real and interesting case.
- It scales: a 500-step run adds one row, not 500.
- It never duplicates and never inflates a step.

Rejected alternatives: a row *per* discovery (doubles the timeline on long runs); drawing it only on
canvas edges (already done there, but leaves the trace's gaps blank); a vertical band like the axis
break (reads as a structural break in the axis, which it is not).

**Where the canvas already gets this right**: the junction node *is* the discovery, and the edge now
carries the interval between two steps. That is the same fact, shown where it belongs.

---

## Making the views tell the truth (user-driven round)

The user reviewed the fixtures as a reader rather than as their author, and most of what they found
was real. Recorded here because the *pattern* matters more than the individual fixes: in almost every
case the data was right and the view was wrong, and in several the view was drawing one thing twice.

### The recapture, and what it revealed

The missing discovery data was **never a server bug**. `loaders/trace.go` already gathers discovery
spans onto the run; the fixtures simply predated it. Every fixture captured today had the data, every
older one had none. Recapturing against a live Dev Server fixed it — `wide` went 0 → 38 discoveries,
`chains` → 14, `loop` → 17.

**Recapturing changed the model.** Because the loader lifts `plannedSteps` off discovery spans, v3
runs now report exact grouping too: `parallel` went from `inferred` to `sdk`. Better for users, worse
for coverage — the inference path lost its last fixture and two tests failed. Rather than weaken
them, the pre-loader capture is preserved as `parallel-inferred`, the tests point at it, and a new
test asserts it *still* exercises the fallback so the coverage cannot be lost silently again. The
index's claim that the two SDK versions are "the only way to test the two code paths" was true and is
now false; it is corrected in place.

### Fixed

- **A retried step read as a plain green bar.** The clearest thing the platform does, hidden behind a
  disclosure triangle. It now draws its attempts — red, gap, the faded backoff, green — so `retry`
  reads as what happened without expanding anything.
- **Steps appeared to start before the run did.** Spans record a step's `queuedAt` as the moment the
  *run* was queued, so on `failure` the first step claimed +1ms while the run started at +156ms,
  overlapping the run's own row and counting the same delay twice.
- **The run row claimed it spent half its life waiting.** The root's breakdown is "total minus what
  the children executed", so all scattered delay summed into one leading block: 53% of `failure`
  shown as waiting when it was queued for 24%. The run row is now drawn from its own clock.
- **A bar reported more time than its own span.** `blocked`'s `hold` carried run-level
  `queue_delay_ms` in its metadata, so a 5999ms step rendered ~12s wide with half of it shown as
  waiting it never did. Timestamps now beat metadata whenever metadata claims more than the span.
- **The "Inngest" row** was named after the company rather than anything the reader wrote, spanned
  the whole run while showing disconnected fragments, and did not line up with the Run bar above it.
  Deleted; everything it carried lives somewhere better.
- **Expanding a row replaced it** with a different decomposition (Inngest / your server) whose parts
  lined up with nothing. Expansion now unpacks the row the reader was looking at.
- **The leading run delay** cost a quarter of the plot on every trace. Below a second it is text
  (`+156ms queued`); above it stays on the axis, because then it is the story.
- **The scrubber looked like more trace.** Now a packed minimap: sequential work shares a lane,
  parallelism adds one, so the **lane count is the concurrency profile**. Fixed 16px; lanes thin
  rather than the strip growing.
- **Canvas edges carry the inter-step delay**, with arrowheads. The gap between steps is literally
  what an edge is, and the graph could not previously explain its own gaps.
- Segments separated by 4px; square bars; label column 35% → 24%; empty badge gutter collapsed.

### The one I got wrong, twice

Extending each step's bar backwards over the gap that preceded it looked right on a sequential run
and was a fabrication: **a discovery is its own request and can lead to many steps, one, or none**.
`wide` made a 144ms step claim **719ms**. Reverted.

The replacement — one Planning row with a mark per request — then made the *same* mistake in a new
guise, because a discovery span is usually the parent of the step it planned and shares its timing.
Drawing both says the platform spent 97ms planning when that 97ms *is* the step. A discovery is now
dropped when a step covers more than 70% of it, which separates the cases cleanly:

- `failure` (v3): the discovery **is** the step, 100% covered → no Planning row at all.
- `t19-parallel` (v4): a 21ms request containing a 1ms step, 5% covered → the other 20ms is real
  request overhead the step does not account for, and is exactly the gap worth drawing.
- `wide`: 38 discoveries down to the 13 that are not steps.

**The lesson worth keeping**: every one of these bugs was the view drawing the same time twice in two
places. It is the failure mode of this codebase, and "does any other row already account for this
interval?" is the question to ask before drawing anything new.

### Process

`canvas-fixture-critic` reviews one fixture at a time across all three views and treats unexplained
gaps, views that disagree, and "would the author of this function recognise it" as explicit findings.
It should have been running from the start rather than waiting for the user to point at things.

## Critic round 1 — the same interval, drawn twice, three more times

`canvas-fixture-critic` reviewed all eight primary fixtures over the gallery and found the Run row
contradicting its own label on **six of eight**. Every finding below was verified against the model
before anything was changed; two of them were regressions I had introduced myself in the previous
two commits.

### What was wrong

**The Run row drew a "waiting" ghost that was not the queue delay it named.** The label read
`+100ms queued` while the bar drew 86% of `chains`, 97% of `t19-parallel` and 99.997% of `cancelled`
as waiting — directly above steps drawn as *executing* over the very same interval. The number being
drawn was the synthesised "total minus what the children executed" figure I had removed from the
breakdown path but not from the root's. Gating it was not enough: a clamped bar (delay reclaimed
into the label, so `delayMs` is 0) and a still-running bar (no `endTime`) both fell straight through
the guard. The root branch is now **exclusive** — it never reaches the breakdown, whatever it finds.
The invariant is asserted on all 44 fixtures, and it caught `inflight` on the first run, which is
how the still-running path came to light at all.

**`Planning` and `Finalization` were the same span.** The run's trailing discovery *is* the
finalization span, and marking Finalization as a platform row had quietly excluded it from the
coverage check — `collectStepSpans` filtered on `isPlatform`, conflating two different questions.
"Is this a step?" decides lanes and counts. "Does anything already account for this interval?"
decides what not to draw twice, and platform rows occupy time on screen exactly like anything else.
Matching on `spanID` as well is exact where the 70%-overlap test was a guess: `t19-parallel` 5
requests → 2, `chains` 6 → 5, `wide` 14 → 13, and `failure`, `retry` and `step` lose the row
entirely.

**A `waitForEvent` was drawn as empty space.** `cancelled`'s 35.2s wait — the single most important
fact in that run — was decomposed into "waiting, then working", came out ~100% pale lead-in, and sat
underneath an opaque axis-break band. Waiting IS the substance of a sleep or a waitForEvent, so
neither is decomposed now; each gets one solid bar in its own style. The break band lost its fill
too (it was painting out its own content) and its width went 5% → 12%, because a break is regularly
occupied by a row drawn across it and has to stay legible as somewhere the run spent time.

**A drawn queue delay is no longer elidable.** A bar's segments are laid out as percentages *of the
bar* while the bar is placed through the elastic scale, so a break falling inside a bar pushes its
own segments off. On `blocked` the Run row was still drawn "queued" a third of the way past the
point where `hold` below it was drawn executing. Keeping that interval busy removes the break
entirely and the two rows now agree to the pixel.

### The canvas and the trace disagreed on every step, and never said so

Worst case `blocked`: terminal pill `Completed 6.049s` against `Run 12.348s`. The canvas reports how
long a step **ran** and puts the delay on the edge into it — the decomposition asked for — while a
trace row reports the step's whole **span**. Both are true and they are different numbers.

The row now carries the third number that reconciles them, in the `note` slot the Run row already
used: `right-1 · +72ms wait · 104ms`, and 104 − 72 = the 32ms on the canvas node. It appears wherever
a lead-in is wide enough to see (≥5% of the span and ≥5ms), so no visible ghost on screen is
anonymous. `leadInMs` in `traceConversion.ts` is the **only** definition of that quantity and both
the bar and the label come through it — the two disagreeing is precisely how a reader ends up with a
row headlined 104ms next to a node headlined 32ms and no way to tell which is lying.

### Still open

- **`Function error` on `failure` is 272ms of a 469ms run** — 58%, the widest bar after Run, and it
  is platform machinery. Truthful, but someone debugging "why did this fail" clicks the biggest red
  bar and finds finalization; the error is in `doomed step` (43ms). A hierarchy problem, not an
  honesty one.
- **`Planning` on `wide` is 13 requests over 77% of the run**, sitting above 12 user steps drawn as
  ticks. Same shape of complaint.
- **Every segment in a row shows the same tooltip**, so there is no way to ask what a ghost is, and
  the per-mark `Planned 2 steps` tooltip built for the Planning row is never rendered. Some
  breakdown figures also do not add up (`left-2`: `DURATION 166ms / DISCOVERY 229ms`, a component
  larger than the whole).
- **A row's internal segments are linear while the row is placed elastically.** Worked around for
  the run's queue delay above; still latent anywhere else a break falls inside a bar.
- `chains`' canvas draws one junction fanning `left-1` into `right-2`; `branches resolved 0` is the
  honest admission in the facts strip, but the arrows say otherwise.
- Minimap: no singular in the aria-label; every run's finalization tail is blank by construction now
  that it draws only steps.
- Facts strip says `grouping sdk` while the canvas corner says `grouping: exact` — two labels for
  one thing on one screen.
