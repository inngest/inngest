# The design language, as rules

A flat, greppable reading of `trace-design-system.html` — every rule it asserts,
with the encoding that carries it and the data the engine has to report for it
to be drawable at all.

**The artifact is authoritative; this file is a reading of it.** It exists
because the artifact is 942KB of HTML you have to look at, which makes it
impossible to diff, grep, or hand to an agent. When the two disagree, the
artifact wins and this file is stale.

Read at commit `4c5b78836`. Regenerate the artifact with `cd src && node ds.mjs`;
serve it with the static server on :5199.

The `DEPENDS` lines are the important ones for planning: they are the rules that
cannot be implemented in React without the engine reporting something it may not
report today.

---

## Moments
- The trace only has what the spans record.
- The three timestamps a step reports are `queuedAt`, `startedAt`, `endedAt`.
- ENCODING: three marks on one row — queued (hollow, `--queued`), started (hollow, muted), ok (filled, green).
- DEPENDS: per-span `queuedAt` / `startedAt` / `endedAt`.

## Durations
- The interval between two timestamps is the readable unit, not the timestamp.
- ENCODING: `queuedAt→startedAt` drawn `idle` (hatched grey); `startedAt→endedAt` drawn `good` (solid).

## A step's lifecycle
- Every step type reaches the same states in the same order: enqueued → executing → resolved.
- A throw adds an attempt bar plus a backoff bar and repeats the cycle. **Attempts stay in one row, never separate rows.**
- ENCODING: idle → bad → backoff → idle → good, one row, marks at each boundary.
- DEPENDS: per-attempt start/end, backoff intervals, attempt count.

## Two rules
- Marks report resolution: **hollow = not resolved, filled = resolved.**
- **Only a resolution mark is ever green or red.**
- Bars report SDK execution: **solid = your function was running, hatched = it was not.**
- Colour says *kind*; fill says *executing* (bars) or *resolved* (marks).
- ENCODING: hollow = queued, ribbon, disc, hollow(started), hollow-bad(retry). Solid = good, bad, child, running, disc, stopped. Hatched = wait, waitok, waitout, idle, backoff, waitstop, hold.

## Structure
- A discovery request is the SDK reporting what to run next.
- A discovery bar appears **only** where the request reported more than one step (fan-out, or the run's first step).
- **A `planned` mark is the claim that a request reported several steps, and the ribbon is what backs it — so it appears only where a ribbon threads it.** After a discovery bar that reported one step the row was simply enqueued, and the mark is `queued`. Emitting `planned` after every discovery bar put dangling blue marks on rows with no ribbon to belong to.
- A request reporting exactly one step is not a row — it rolls into that step's row as a leading blue interval.
- The ribbon covers exactly the steps one request reported; cables appear only on hover/select.
- ENCODING: `disc` blue bar; ribbon = solid blue vertical rect (w 3.4) threaded through members' queue circles, drawn *under* rows so circles sit on it; cables = pale curved wires with dark casing + drop shadow, no arrowhead, ending at the circle's edge (socket metaphor).
- DEPENDS: **which steps a given discovery request reported** (request→step attribution).

## The Run row
- The Run row reports **where elapsed time went**, not the run status — the header, badge and axis already carry status.
- Grey track = elapsed time with no SDK execution; coloured slices = SDK executing, discovery included.
- **Failure wins.** A slice holding any failure draws red. Mixed successes-and-waits still fall back to blue/`mix`, but a failure is never averaged away — the old "only if every step agrees" rule drew a failure cluster among successes as neutral blue, losing the signal at exactly the scale an overview exists for.
- **The Run row is derived from every row in the trace, finalization included**, and reaches the end of the run rather than the end of the last user step. Leaving finalization out showed the run doing nothing while it executed, and stopped the profile short of where the trace actually ended — the overview disagreeing with the rows beneath it. Asserted in `validate.mjs`.
- ENCODING: `TRACK_H` 5 grey ground, `RUN_H` 8 slices; failed slices floored at `MIN_FAIL_W` 3.2 vs `MIN_W` 1.4 so failure stays findable; queued mark left, resolution mark right.
- DEPENDS: per-step compute intervals with outcome, to slice the axis at every start/end edge.

## OpenTelemetry

- A step instrumented with `@inngest/otel` reports **userland spans** — HTTP calls, database queries, third-party APIs.
- **They are your code at finer grain**, distinguished by **nesting and weight, not a new colour**: indented under the step, drawn thinner, because each is a subdivision of the bar above rather than a peer of it.
- **A span carries the three states OpenTelemetry gives it — OK, ERROR, Unset — and Unset is the common case.** Most instrumentation sets no status, so Unset must not read as an outcome: grey means *it ran and nobody said*, green means something said OK, red means it failed. Unset resolves on `done`, a filled muted mark, because `ok` would claim something nobody said.
- **A span's status never tints its step.** A step that returned is green whatever happened in a span inside it; those are different facts, and merging them would make the step's own outcome unreadable.
- **At rest a step with spans says so in the gutter and nothing else** — three stacked strokes, meaning there is something layered under this row. **Selecting the step expands them.** Anything drawn on the bar itself either assumes the spans are sequential (notches did) or compresses a span tree into one bar's width, and neither survives a `Promise.all` inside a step.
- **A span is a plain bar on a tighter pitch: no queue mark, no start, no resolution circle.** The event vocabulary belongs to the trace, and a span is something that happened inside one row of it. Packed until the bars nearly touch, they read as a block belonging to the step above rather than as more of the run — which is the differentiation, without spending a colour on it.
- Overlapping spans need no ribbon and no ordering claim. The trace did not schedule them, so it has none to make.
- **A span sits inside its step's execution.** One that starts before its step or outruns it is a clock disagreement between the two processes, not a slow query, and the row says so rather than clamping it quietly.
- The step is the thing that retried; a span inside it is not separately retryable.
- DEPENDS: userland spans reaching the client at all — span name, kind, service name and attributes.

## The selected row
- Selecting a row replots that row alone across the full width — same bars, same marks, **not a second drawing language.**
- **The timing breakdown lives here, not in the trace.** Today's product expands a step into `Inngest` and `Your server` sub-rows inline, and that is most of what makes the trace tall. A breakdown is detail about *one* row, and it only matters once you have chosen that row — at which point selecting it already gives the full width to name every interval, which is exactly what a breakdown is. (Stakeholder's call.)
- Every interval gets a name and a duration; every mark gets what happened.
- ENCODING: interval name + duration centred under each bar; mark names on a second line with alternating high/low ticks.

## Annotations
- Annotations are optional and usually a duration.
- The annotation sits **after the row's last bar, not in a reserved column** — it costs no horizontal space and disappears when empty.
- The same slot may carry attempt count, flow-control reason, or the number of runs an event started.
- ENCODING: mono 6.5px muted, 9px after the last bar's end.

## Step types
- One fragment per step type, states in the order reached; cancelled last.
- `step.run` — attempts, backoff and result are intervals in one row.
- `step.sleep` — blue while open; resolves **green because sleep cannot fail** (only cancel).
- `step.waitForEvent` / `waitForSignal` — **a timeout is a result, not a failure**; same states, only the resolver differs.
- `step.invoke` — the bar is the child run's duration on *this* run's axis, its own substance (neither green nor blue).
- `step.sendEvent` — the row reports how many runs the events started, and that count is the way into them.
- An open row draws no closing mark; it stops at the start of its last bar.
- ENCODING: `child` purple; `child!` for a failed child; `stopped`/`waitstop` grey ending in a square; lineage spur = 7px line + hollow ring + centre dot + "N runs ↗" in accent.
- DEPENDS: **child run duration**; **count of runs an emitted event started**.

## Scenarios
- Ordered from one step to the cases that are hard to draw; each shows at rest and, where it differs, on hover/select.

## Sequential
- In a single sequential thread the SDK answers and executes in the same request, so **there is no discovery bar**.
- The execution still began not knowing what it would run — the blue start circle says so; a step planned by an earlier request opens on the grey mark instead.

## Fan-out and coalesce
- A ribbon at rest means hovering a member needs no cable — the relationship is already on screen.
- Cables are drawn only for the hop the ribbon cannot span (coalesce into the next request).
- Ribbons nest; one ribbon per discovery request, so **depth reads without indentation**.
- Ribbon width is decided at runtime — it reports what happened and predicts nothing.
- An uneven fan-out is countable at rest without interacting.
- Two steps under one ribbon **imply no order between them**.
- Two branches that each caused their own request get two requests and no ribbon.
- A ragged queue still gets one ribbon snapped to a single queue instant.
- ENCODING: members' queue marks drawn without halo so they read as sitting on the ribbon; dimmed ribbon (o .35) in the hover frame.
- DEPENDS: **request→member set**; per-member `queuedAt`.

## Attribution
- Pairings stay inside their branch; a crossed pairing would show as a crossed cable.
- Rows sort by start time, so **plan order is not row order** — the ribbon carries the plan, the axis carries the time.
- Response order stops matching branch structure after unrelated waits; the cable is then the only structural claim, and only on demand.
- A race winner is a dependency and gets a solid cable; losers that resolved after the request started get a dashed one or none.
- A step started but never awaited has no outgoing cable — **a dead end is drawn by absence.**
- **A guess must never look like a reported fact.** Where the trace cannot say what caused something it declines rather than drawing a confident line.
- ENCODING: dashed accent stroke (2.5/2.5); causal ribbon variant is orthogonal + dashed (4 3) at the last contributor's end.
- DEPENDS: **reported parent step per request**.
- **Inferred grouping is no longer drawn.** The artifact had a dashed ribbon and cable for grouping recovered from overlap rather than reported; it was cut as not worth its place. React still surfaces the distinction (`parallelismSource`, the `grouping: exact | inferred` badge), so this is a deliberate gap rather than agreement — see `DELTA.md`.

## Platform time
- Queue time is the step waiting for the executor; thin, quiet, ends at the circle where your code starts.
- **Flow-control holds (concurrency, throttle, rate limit, debounce) are one fact and get one treatment**: amber hatched, still not compute.
- System latency is the same substance as queueing and named the same way — three shades of grey would be three things to learn.
- Platform rows keep their own row and the platform colour.
- Everything Inngest did is hatched and quieter; only SDK execution is solid, so the eye lands on green and red first.
- **No unexplained gaps** — every millisecond between a row's start and its resolution belongs to a named interval.
- `step.sendEvent()` is the only row that points out of the run; the ring is hollow because those runs are not in this trace.
- The gap between a step resolving and the next request starting is drawn once — the cable and the interval are the same object.
- ENCODING: `hold` = amber (`--hold`) hatched at line .62; `idle` = hatched at bg .30 / line 1 (the densest hatch).
- DEPENDS: **hold reason** (which flow-control control fired); latency vs queue attribution; count of runs started by emitted events.

## Outcomes
- Green bar + green resolution is the baseline everything else is read against.
- A run-ending failure is the last thing on the row, so nothing after it implies recovery.
- A caught failure is a red step inside a green run — the row is red at its own resolution, the Run row stays green.
- On retry, only the resolution is green; the mark between attempts is neutral and means "retrying".
- A filled red circle appears **only** when every attempt failed.
- Failure is per-row and per-slice, **never inherited** by neighbours or the level.
- Still executing = blue bar with **no** terminal mark; the missing mark is what says unresolved.
- **Cancelled is its own state, neither green nor red** — a square, stopped from outside, never resolved.
- ENCODING: `bad` = `--warn`; backoff drawn faded (bg .12 / line .42) so a recovered run doesn't carry more red than a broken one; cancelled = square mark with a square halo.
- DEPENDS: attempt outcomes; **whether userland caught the error**; cancellation timestamp.

## Waiting
- A wait is blue while open (undecided) and green at the mark where the event arrived.
- A timeout is a result the function can act on — the row goes grey and the run carries on.
- A wait inside a fan-out holds the level open; the next request cannot start until the slowest member resolves, reported as queue time on the next row.
- **Waiting and failing must not read alike** — waiting is hatched and never red; failing is solid and red.
- ENCODING: `wait` blue hatched, `waitok` green hatched, `waitout` muted hatched, `waitstop` muted hatched + square.

## Time and the axis
- **Everything that survives compression shares ONE scale.** The width left over
  is divided between the live stretches in proportion to their real duration —
  30s on one side of a gap and 10s on the other get 75% and 25% of it. Which is
  the same thing as saying **a second is worth the same number of pixels wherever
  it lands**, so two spans in different parts of the run stay comparable. Without
  it, compressing a gap silently rescales one half of the trace against the
  other. It is also why N compressions need no special case: the arithmetic is
  total live width over total live time, applied everywhere.
- **The bands share a budget.** All of them together get at most ~16% of the
  plot, so a trace with twenty idle stretches does not spend its width on the
  parts where nothing happened — each band thins instead, down to a single
  marked line. **A band never grows to fit its content**, because its content is
  precisely the thing not worth space. Below a width that can hold them a band
  drops its label, then its tear; the rules and the blur never go, because those
  are what says *not to scale*.
- **The cut sits inside the compressed bar, not on its edges.** You see the bar begin, get torn, and resume before it ends, so it is obvious *which bar* was compressed rather than merely that something happened between two rows.
- **Only a stretch where NOTHING was executing may be compressed, and that is derived, not nominated.** A gap in one row is not dead time — another row may be working through it — so the compute intervals of every row are merged first and the holes in that union are the only candidates. `deadStretches()` does this; nominating a stretch by hand is how you compress a region something was running in.
- **An unresolved wait still compresses.** A run asleep right now has real dead time in it whether or not the sleep has finished. Such a figure has no Finalization row and no resolution mark on the Run row, because neither has happened.
- **Dead time is worth almost none of the width, and the threshold for that is low.** If nothing is executing for more than a few percent of the run, that stretch collapses to a fixed narrow band — an hour and seven days get the same few pixels, because the space belongs to the work. Only the drawing compresses; **every reported duration is still wall clock.**
- **A compressed band carries three cues**, because one cannot beat how strongly a time axis reads as linear: **the Run row track tearing** into two or three zigs — the track itself breaks, rather than running straight under a glyph laid on top, because a mark beside a bar reads as an icon while the bar breaking reads as the thing that happened to it, **full-height rules** at both edges so the cut crosses every row rather than being a mark on the axis that rows ignore, and a **blur** of whatever runs through the band — which says "this width is not to scale" without hiding that a row is running through it.
- Reading the fill alone should report the compute/elapsed ratio before you read a number.
- Two tiers of axis label — coarse tier carries the date/hour crossed, fine tier the offsets inside it.
- Ticks come from the scale that placed the bars, so **a label can only name a position the drawing reaches.**
- A step too short to draw is still drawn at a minimum width, with its real duration beside it: **the drawing rounds, the reported number does not.**
- ENCODING: break = fixed narrow band, `--ground` scrim at .28, solid full-height rules at both edges, the elapsed time named in the middle of the band over a `--ground` halo, and the figure redrawn clipped to the band through a blur that also desaturates; `MIN_W` 1.4px floor applied in screen space.

## Scale
- Repetition with one shape collapses to a single row that reports the count and draws where each member ran.
- A collapsed group draws **the envelope the members occupied plus a tick per member**, so count and spread read without expanding.
- A failure cluster must be findable in the first screenful, without scrolling or interacting — **on the Run row**, which is why failure wins over mixed there.
- About forty rows at rest whatever the step count; any group expands in place.
- ENCODING: group envelope = surface-2 rect with grey stroke, member ticks floored at 1.2px.
- **There is no density strip.** It existed to make a failure cluster findable without interacting; the Run row does that itself now that failure wins over mixed. Its one remaining claim was bar *height* as a count per bucket, which is not worth a third overview object on a page that just deleted its second one. If a per-bucket count is ever needed it belongs in the Run row — slice height — not in a new row.
- DEPENDS: group membership and per-member start/duration **even when collapsed**.

## Naming and identity
- **The step name is not the identity — the request that reported it is**; the ribbon already shows that.
- The SDK's `:1` / `:2` suffixes are not stable under parallelism, so nothing in the drawing depends on them; they are shown as text and used for nothing else.

## Interaction
- Three tiers of attention on hover/select: the row, what caused it, everything else.
- **Nothing is removed, only quietened**, so the run's shape stays readable.
- **The Run row is the overview.** There is no minimap: a second strip above it drew the same run in the same place in the same colours, which is one fact drawn twice and a second overview that can drift out of step with the first.
- **Scrubbing is unresolved.** A window with edge handles was drawn on the Run row and removed: at rest its edges are the run bounds, so it sat on the first and last mark, and it read as confusing rather than obvious. The affordance still needs somewhere to live.
- ENCODING: dim opacity .15–.28 for off-path rows; selected row = accent wash at .13, rx 2.

## Honesty
- Where the trace cannot say what caused a request, **it declines rather than guessing** — no cable, and the row says so.
- A row's label and its drawing must agree about which interval the number names (e.g. "3ms reported · 31ms on the row").
- **Nothing is drawn that cannot be asked what it is** — every interval and mark decomposes into named parts on hover.
- ENCODING: `.rowhit` transparent hit rect carrying `data-parts` built from what was actually drawn.
- DEPENDS: whether a parent was reported vs absent.

## Discovery requests that fail
- A failed discovery request lands in the row of the step it eventually produced, so the row tells the whole story of how that step came to exist.
- The bar stays discovery blue even when it failed — **this was Inngest's failure, not the user's**; hollow red circles carry the outcome.
- If it never succeeded, no step exists to host it — the one case where a request gets a row of its own, and the only filled red circle on something that is not user code.
- ENCODING: `disc!` (failed substance suffix `!`), backoffs labelled between attempts.
- DEPENDS: **discovery-request attempt outcomes and backoff durations.**

## Scrub a real run
- Each fixture is a captured run at measured proportions, shown complete, scrubbable back through every intermediate state.
- A row that has not started is **not drawn at all** — its span does not exist yet.
- An interval still running takes the in-progress substance (good/bad → `running`; waitok/waitout → `wait`): **a step that will return and one that will throw are identical until they resolve.**
- The axis rescales so the elapsed part always fills the width, as a live trace does.
- Scrub stops are the trace's own moments (span created / started / resolved), not uniform time samples.
- The scrub axis is elastic — each stretch gets its real share clamped to a floor and ceiling; reported times never change, only handle position.
- ENCODING: dead stretches (>6% gap) marked as hatched `brk` bands; one tick per real moment; time inside a stretch interpolates harmonically.

## Captured fixtures
- Fixtures are drawn at their measured proportions, unchanged — sub-pixel steps stay sub-pixel.
- `simple` — two rows and no relationships; **the design must be quiet here or it is noise everywhere.**
- `v4sequential` — an SDK that *can* batch, on a run where nothing is parallel: **no ribbon anywhere is the correct render**, visibly different from parallel without reading a label.
- `emit` — outbound lineage leaves the run, so it is not a ribbon; a distinct outbound marker keeps the two kinds apart.
- `invoke` — expanded, the child's rows indent under it on the same axis; parent keeps its circles, child rows get theirs, so the run boundary reads without a label.

## Connect
- Connect changes the transport, not the spans — same span names, shape and step ops, so **nothing needs a Connect variant.**
- A Connect run carries no `inngest.http.timing`, so there is no network breakdown to expand — **do not offer an empty one.**
- The connect proxy polls every 2s + rand(0–3s); a missed push lands inside the execution span with nothing marking it, **so the execution figure over-reports.**
- ENCODING: red wash at .16 over the inflated region with a mono caption naming the poll interval.
- DEPENDS: `inngest.http.timing` (absent under Connect); no span marks the poll wait.

---

## Vocabulary primitives

**BAR KINDS** — `good` `bad` `disc` `idle` `child` `mix` `running` `wait` `waitok`
`waitout` `backoff` `stopped` `waitstop` `hold`.
Suffixes: `!` = "this substance, but it failed"; `*` = discovered.
- COMPUTE (solid) = good, bad, child, running, disc, stopped
- NOCOMPUTE (hatched) = wait, waitok, waitout, idle, backoff, waitstop, hold
- OPEN (no resolution mark) = running, wait
- ACTIVE (resolution rule) = good, bad, child, running, disc, stopped

**MARKS** — `queued` `ribbon`(planned) `disc` `hollow`(started) `hollow-bad`(retry)
`ok` `failed` `timeout` `cancelled`. EV_HOLLOW = queued, ribbon, disc, hollow,
hollow-bad. Ring weights: queued 1.8, ribbon 1.7, disc 1.8, hollow 1.7,
hollow-bad 1.9, filled 1. **Cancelled is the only square.**
Non-mark glyphs: ribbon (solid rect w 3.4), wire (curved cable, casing 3.2 + core
1.5), causal ribbon (orthogonal, dashed 4 3, w 2.4), lineage spur (hollow ring + count).

**GEOMETRY** — W 470, LBL 64 (label gutter), RGT 10, PLOT 396 = W−LBL−RGT,
ROW 17 (pitch), TOP 6, BAR_H 7, RUN_H 8, TRACK_H 5, MARK_R 3, HALO 1.3,
MIN_W 1.4, MIN_FAIL_W 3.2. `cyOf(i) = TOP+9+ROW*i`; `pxOf(p,k) = LBL + (p/100)*PLOT*k`.
`render.mjs` uses a larger variant (W 680, LBL 92, ROW 24) for the older captured
figures only. Hatch tile 4×4 rotated 45°, stroke-width 2. Substance opacity pairs:
solid [1,0], hatch [.16,.55], dense [.30,1], faint [.12,.42].

## The invariants

These are stated as inviolable. Breaking one is a regression, not a restyle.

1. Only a resolution mark is ever green or red.
2. Hollow/filled is a rule, not a preference — it reports whether the row resolved, and is not adjustable.
3. Every millisecond between a row's start and its resolution belongs to a named interval (`fillGaps` inserts `idle` for any gap).
4. Two abutting segments of the same kind are one interval, not two — no boundary mark where nothing changed.
5. A duplicate consecutive mark is dropped: a mark means something *changed*.
6. An open row draws no resolution mark.
7. **Only the axis may compress time; every reported duration is wall clock.**
8. A bar is never narrower than `MIN_W` on screen, and its label still names the real duration.
9. Failure is per-row and per-slice, never inherited.
10. **A guess must never look like a reported fact.**
11. Nothing drawn may be un-askable.
12. Every kind is painted with the same pattern mechanism, indirected through a CSS variable, so a kind moves without regenerating a figure.
