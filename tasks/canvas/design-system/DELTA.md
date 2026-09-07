# Artifact vs React — where the two actually stand

Reconciled from a full audit of both sides at commit `c34027a4e`.
`RULES.md` is the artifact's specification; this is the difference between it
and what `RunDetailsV4/` renders today.

**The headline: they are ahead of each other in different dimensions.**

- **React is ahead on derivation and honesty machinery** — collapsing, the
  elastic axis, gap attribution, inference surfacing. Several of these are more
  developed than anything the artifact specifies.
- **The artifact is ahead on the drawing language** — the marks, the substances,
  the ribbon, and the invariants that make a row self-describing. Most of this
  has no React counterpart at all.

So "port the artifact into React" is the wrong shape of task. The artifact
supplies a *vocabulary* that React's existing derivation would feed.

---

## 1. The structural gap: React has no marks

**This is the largest single difference and everything in §2 depends on it.**

The artifact draws a row as **moments + intervals**: nine marks (`queued`
`ribbon` `disc` `hollow` `hollow-bad` `ok` `failed` `timeout` `cancelled`) at
the boundaries between bars, with hollow = unresolved, filled = resolved, and
only a resolution mark ever green or red. Cancelled is the only square.

**React's timeline draws intervals only.** It has row *icons* in the label
gutter (gear, clock, mail — identity, not state) and hairlines in the minimap.
There are no boundary marks on the bars.

Consequence: artifact invariants 1, 2, 5 and 6 have no React counterpart,
because the primitive they govern does not exist. A React row cannot currently
say "this resolved" separately from "this is how long it took".

## 2. The encoding gap: one binary vs five mechanisms

The artifact's load-bearing rule is **solid = your app executed, hatched = it
did not** — `COMPUTE` vs `NOCOMPUTE`, one pair of opacity numbers per kind,
every kind painted through the same mechanism.

React uses five unrelated mechanisms for the same job:

| React mechanism | Where |
|---|---|
| `ghost` (opacity 0.26) | `step.sleep`, `step.waitForEvent`, `timing.backoff` |
| plain opacity dim (0.25 / 0.45 / 0.7) | `timing.waiting`, `.queue`, `.finalization`, `.concurrency` |
| `dotted` mask | `.concurrency`, `timing.connecting`, HTTP dns/tcp/tls/transfer |
| `barber-pole` mask | `timing.server`, `timing.http.server` |
| `vertical-lines` mask | `step.invoke` |

None of them is "did your app execute". A reader cannot recover the compute /
elapsed ratio from the fill, which is the artifact's stated test.

## 3. Where React is genuinely behind — ranked by what it costs

1. **Suspended steps are one undifferentiated bar.** `step.sleep` and
   `step.waitForEvent` render as a single neutral ghost bar, and `IDLE_STYLES`
   **suppresses segment generation entirely** — so queue time and wait time are
   not separate named intervals. This breaks artifact invariant 3 (*every
   millisecond between a row's start and its resolution belongs to a named
   interval*) on exactly the rows where waiting dominates.
2. **The timeline cannot tell a matched wait from a timed-out one.** The canvas
   can (`waitTimedOut` → grey stripe, neutral border, `timed out` replacing the
   duration). The timeline does not: `timedOut` is read in `graph.ts` and
   `StepInfo.tsx` and never in `traceConversion.ts`, `Timeline.tsx` or
   `TimelineBar.tsx`. The artifact makes these three distinct substances
   (`wait` / `waitok` / `waitout`). **Two views disagreeing about the same run
   is the finding class `canvas-correctness` exists to catch.**
3. **Flow-control holds are wrong, not just missing.** React draws only
   `timing.inngest.concurrency`, fed by `queue_delay_ms` — and labels it
   *"Held Xms by a concurrency limit"* unconditionally, when the field's own
   type comment says it covers "concurrency limits, throttle, or other
   constraints". **That is a guess presented as a reported fact** — artifact
   invariant 10, and the one defect here that actively misinforms. The artifact
   makes all four reasons one amber hatched substance and names the reason in
   the annotation slot.
4. **Cancelled mid-flight has no substance.** React clamps the span
   (`markInterrupted`) and prints `cut short` beside the duration; there is no
   distinct fill, no terminator glyph, and no distinction between *executing*
   when cancelled and *queued* when cancelled. The artifact has `stopped` and
   `waitstop`, both ending in the square that is reserved for cancellation.
   This is `PROMPT.md` constraint 3.
5. **The Run row is a different object in each.** Artifact: a grey track of
   elapsed time with coloured slices where the SDK executed, mixed slices
   falling back to blue — it reports *where the time went*. React: a solid
   status-coloured `root` bar with a queued lead-in — it reports *the status*,
   which the header and badge already carry.
6. **No ribbon.** The artifact's answer to showing structure at rest is a solid
   blue ribbon threaded through the members' queue marks, nested, needing no
   indentation. React has a 1px gutter rail with ticks plus hover-only stepped
   arrows. **`LOG.md`'s Open section already names this as unresolved and
   clunky — the artifact has an answer to it.**
7. **Failed discovery requests.** A whole artifact section (the request lands in
   the row of the step it eventually produced; the bar stays discovery blue
   because it was Inngest's failure, not the user's; the only filled red circle
   on something that is not user code). No React counterpart found.
8. **`step.sendEvent` outbound lineage.** Artifact: a spur with a hollow ring
   and "N runs ↗". React: `lineage.ts` exists with `findEmittingSteps`, is
   tested, and **is imported by nothing**. What ships is `LinkedRuns.tsx`
   (defer relationships as tables) and the invoke child preview.
9. **Connect.** An artifact section, including the caveat that the proxy's
   2s + rand(0–3s) poll lands inside the execution span with nothing marking
   it, so the execution figure over-reports — drawn as a red wash naming the
   poll interval. No React handling found.

## 3b. Where the two now openly disagree

**Backoff colour.** The artifact draws it **faded red**; React draws it
**neutral** (`timing.backoff` = `bg-surfaceMuted`, commented "deliberately
neutral, not tinted by either attempt"); and `LOG.md` critic round 4 records
neutral as *settled*, on the argument that the colour was being asked to carry
both whose fault it was and how it turned out, and needs to carry neither.

The artifact is the stakeholder's call and wins there. One of the other two
records is stale and it has not been decided which — do not "fix" either side to
match the other without settling it.

## 3c. What the shipped product has that the design language does not

Read off `inngest.com/docs/platform/monitor/traces`. Addressed:

- **OpenTelemetry / extended traces** — now a section of its own. It was the
  largest gap: userland spans were absent from the language entirely.
- **The timing breakdown** — deliberately *moved* rather than added. See
  `RULES.md` § The selected row.

Still absent, in rough order of how much they cost:

- **`step.waitForSignal`** — named in the vocabulary's own description of
  `wait`, but no figure. It resolves differently from `waitForEvent` (a signal,
  not a matched event) and nothing draws that.
- **The `Connecting` bar** — the docs list it as one of eight bar types
  (dotted outline, connection phase). React declares `timing.connecting` and
  never emits it. The artifact has nothing. Worth deciding whether it survives.
- **Trigger types.** The docs distinguish **event**, **cron** and **batch**
  triggers, each with different fields. The artifact draws a trigger but does
  not distinguish them, and a cron run has no event to point at.
- **The attempt badge** — a retry count coloured by status, beside the step
  name. The artifact draws attempts as intervals on the row, which is richer,
  but the badge is what makes a retried step *findable* in a long list.
- **Rerun from step**, **Cancel**, **Invoke**, **Rerun** — actions. Out of scope
  for a drawing language, but they need somewhere to live on the row.
- **Metadata / headers / input / output tabs** — the panel, not the trace.

## 4. Where React is ahead of the artifact

Do not regress these while porting anything above.

- **Gap attribution to discovery requests** is the most developed thing on
  either side. `withPlanning` matches `RunDiscovery.plannedStepIDs` to steps,
  widens the first-planned step back to the request's start, tooltips the
  sibling steps, and **keeps `reportedMs` as the row's number** with a
  reconciling note (`+249ms planning`). The artifact asserts the rule; React
  implements it with the reconciliation.
- **Collapsing** far exceeds the artifact: shape normalisation, iteration vs
  sibling detection, variants, exception classes, MAD-based slow detection
  requiring ≥4× median so a ramp is not reported as N anomalies, a
  `markBudget` derived from measured plot width, edge merging by rank.
- **The elastic axis**, with the guarantee made structural rather than
  remembered: only `calculateBarPosition` takes the scale, `calculateDuration`
  never does. Two-tier labels; ticks anchored to segment boundaries on a broken
  axis so a label can only name a position the drawing reaches.
- **Honesty machinery**: `warnings[]`, the `grouping: exact | inferred` badge,
  a legend naming three line styles ("Waited for it" / "Lost a race" / "Might
  have"), junction popovers stating what was not reported, `detailsAddUp`
  suppressing a breakdown whose parts do not sum, `interrupted` printing
  `cut short` as a floor rather than a measurement.
- **HTTP phase breakdown** (dns/tcp/tls/server/transfer). The artifact has no
  opinion on these.
- **Brush, minimap, shared viewport**; hover and selection shared via
  `useStepHover`/`useStepSelection`, narrowing attention and never filtering.
- **Invoke expanded inline**, lazily fetched, in both views.
- **`valueLines`** — factual one-liners with a documented list of claims
  deliberately not made.

## 5. Reading this for planning

The §3 items split cleanly, and the split is what decides sequencing:

**Pure design work, data already present** — 1, 2, 4, 5, 6 (partly), 7.
The engine reports what these need; the drawing language is what is missing.

**Blocked on the engine reporting something** — see the `DEPENDS` lines in
`RULES.md`. The demanding ones: which steps a discovery request reported
(the ribbon), reported-vs-inferred grouping as a distinguishable fact, the
hold *reason*, whether userland caught an error, discovery-request attempt
outcomes, and the count of runs an emitted event started.

**Already half-built and disconnected** — 8 (`lineage.ts` is written, tested,
and imported by nothing).
