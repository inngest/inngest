# Scenarios — the bible

The scenarios enumerate every behaviour we support, so they are the
specification. This is the working list against them.

Figures are named by the caption printed under them in the artifact, because
that is the only name a reader can see. Status: `[ ]` open · `[x]` done ·
`[?]` needs a decision.

---

## Applies to every figure

The three big ones. Each touches every scenario, so they come before further
per-figure polish.

- [x] **A code example on every figure.** Trimmed pseudo-code, enough to say
  what this trace is — `await step.run('a'); await step.waitForEvent('b')`.
  A figure you cannot tie back to the code that produced it is asking the
  reader to reverse-engineer the point.
- [x] **Surrounding spans, present but obscured.** Every figure should look like
  a *real* trace — the run row, the trigger, the other steps, finalization, the
  minimap — with everything irrelevant dimmed **and blurred**, not absent.
  Two reasons, the second being the real one:
  1. the figure reads as a trace rather than a diagram;
  2. **it forces us to practise the full trace.** Where does the minimap sit
     relative to the run row and the steps? Does finalization make sense here?
     Today the minimap is described but never drawn in place, so nobody can
     tell whether any of it holds together.
- [ ] **Show the requests, in every scenario.** Which requests were made and what
  each was for is most of the complexity in this view, and *Sequential*
  establishes it as load-bearing. Every scenario gets checked against "does this
  show the requests honestly" — not just the sequential one.

## Sequential

**"One request, one step."**

- [x] **The gap between `a` and `b` should not have been there.** `a`'s work
  ended and `b`'s queue began four units later — unnamed time, in the figure
  whose entire point is *one HTTP call, not two*. `b`'s queue now opens the
  instant `a` resolves.
- Keep: "reports and executes in the same request" is the most important fact
  in the section.

## Fan-out and coalesce

- [x] **"One request reports three steps"** — hovering `b` was fading and
  lighting whole rows. Now attention is per element: hovering `b` lights `b`
  entire, the ribbon, and on `a`'s row *only* the discovery bar, the queued mark
  and the planned mark. `a`'s `ok` bar stays faded, because it had nothing to do
  with `b` starting.
- [x] **"Three steps resolve and cause the next request"** — the `d selected`
  frame drew no selection on `d`. It does now.
- [x] **"An uneven fan-out is countable without interacting"** — real bug. `b`
  and `c` had no queue interval at all, so no planned marks, and the ribbon was
  threading rows with no circles to thread. Every member of a fan-out is
  enqueued by the request and waits its turn.
- [x] **"Fan-out inside a fan-out"** — good figure; apply the same per-element
  hover rule as above.
- [x] **"Width is decided at runtime"** — deleted. Its one unique claim
  is that the cable runs from `decide` *into* the request rather than out to its
  members — but "one resumption, two steps" states the same rule with a more
  common shape. Proposal: delete it, and fold "width is decided at runtime" into
  the uneven fan-out caption, which already demonstrates it.
- [?] **"One resumption reports two steps"** — keep or delete. Its claim (one
  ribbon over two steps implies no order) is already made twice above. Proposal:
  keep, because with a code example it becomes the clearest `Promise.all` shape
  in the section — but it has to earn that.
- [x] **"Two branches, two requests, no ribbon"** — should show the discovery
  requests. Under parallelism we always preplan steps, so two requests exist and
  should be visible. Drawn with none today, which contradicts the rule above.

## Attribution

- [x] **Cut the section back to what earns its place.** It is inert until
  **"Promise.race()"**, **"a step started but never awaited"**, and
  **"grouping inferred rather than reported"**. Those three are the section.
  The run-up — "pairings stay inside their branch", "plan order is not row
  order", "discovery after unrelated awaits" — is not carrying its weight.
- [x] **"Grouping inferred rather than reported" now says when that
  happens.** The dashed ribbon and dashed cable are right, but nothing tells the
  reader *what causes* a grouping to be inferred — an SDK reporting one opcode
  per response, so the grouping is recovered from overlap rather than read off
  the request.

## Time and the axis

- [x] **"A compressed idle gap" reads as absence, and it is not.** Fixed: The `sleep 7d`
  row draws its bar straight *through* the break band. So: no, we do not expect
  gaps — there is essentially always something there, usually a suspended row
  spanning the whole stretch. The break compresses the **axis**; it is not a
  hole in the data, and drawing it as a hatched band that looks like nothing
  happened is the defect.
- [x] **The elastic axis is redesigned.** Same root cause: the break band
  and the bar running through it are fighting, and the band wins. Note the React
  gallery already has the derivation (`buildTimeScale`, breaks at >=5s and >=15%
  of the run) — the presentation is the problem, not the maths.

## Judged good, no action

- **Outcomes.**
- **Waiting.**
- **Fan-out inside a fan-out**, other than the hover rule.

## Added since — the request lifecycle was wrong

- [x] **A discovery request is a request, and waits like one.** Every figure went
  straight from a queued mark into a blue bar, so the request appeared to begin
  the instant it was enqueued. 23 rows now open on the grey `queued` mark, carry
  a hatched queue interval, and only then start.
- [x] **The `discovery` mark is deleted.** It drew as a blue hollow disc,
  identical to `planned`, and `autoDots` never emitted it. A request starting is
  `started` — the same white hollow disc as any other execution. **`planned` is
  now the only blue hollow disc**, which is what lets it mean "this request
  reported several steps".
- [ ] **11 disc bars 1–2 units wide are still uncarved.** At that scale the queue
  interval plus its two marks overlap into a smudge. Needs either a wider figure
  or a decision that a sub-2% request does not show its own queue.
- [x] **Concepts and Fixtures tabs checked** against the corrected
  lifecycle. Scenarios are done; those two are not.

## User-facing docs

- [x] **First draft** at `docs/understanding-traces.md` — the two rules, reading
  a row, requests and the ribbon, waiting, failure, long runs, collapsing, the
  minimap, and what a trace will not tell you.

## The minimap is gone; the Run row is the overview

Decided with the stakeholder, and the reasoning is the durable part.

- [x] **Merged.** The minimap and the Run row were two drawings of one fact —
  the same run, in the same place, in the same colours. That is the
  characteristic defect of this view (`LOG.md`: *"the same interval drawn twice,
  in two places, under two names"*), and I had already treated the symptom by
  making the minimap deliberately thinner just to tell them apart. Merging also
  removes a surface where two overviews could disagree.
- [x] **The Run row is what you scrub**, with the window drawn at rest and a
  handle at each edge. A scrubber nobody finds is a scrubber nobody uses.
- [x] **Failure wins over mixed.** Forced by the merge: the old *"a slice is
  green or red only if every step live in it agrees"* rule drew a failure
  cluster among successes as neutral blue, losing the signal at exactly the
  scale an overview exists for. The row already bent this way — failed slices
  have always had a wider minimum width so they stay findable.
- [x] **The Run row is drawn sharp inside the frame**, not blurred with the rest
  of the context. It is the overview *and* the control; blurring the thing whose
  job is to be obvious defeats it. Only finalization stays soft.
- [x] **`planned` marks got their halo back.** They were drawn without one so
  they would read as sitting *on* the ribbon, which cost them the ring of
  surface every other mark has — so they stopped separating from the bar behind
  them. The ribbon threads between the halos instead, which says the same thing.

### Open, raised by the merge

- [x] **The density strip is deleted.** `Scale` used to show a failure
  cluster as a red smear in a separate bucketed strip; that figure now draws it
  on the Run row, because failure-wins put it there. The strip's remaining claim
  is bar *height* as a count per bucket, which the Run row does not carry. Either
  that is worth a row of its own or the strip goes the way of the minimap.

### Scrubbing, parked

- [x] **The scrub window is removed.** Drawn on the Run row with a handle at each
  edge, it read as confusing: at rest its edges *are* the run bounds, so it sat
  on the first and last mark of the row it was meant to control, and loosening
  it only made it look like a second object on the row rather than a control of
  it. Reverted in full.
- [ ] **The affordance still needs somewhere to live.** The merge is kept — one
  overview, failure-wins — so the question is narrower than it was: not "which
  strip do we scrub" but "how does the Run row say it is draggable without
  drawing a second thing on top of itself".
