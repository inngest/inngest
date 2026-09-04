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
