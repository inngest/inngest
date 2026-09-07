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

- [ ] **A code example on every figure.** Trimmed pseudo-code, enough to say
  what this trace is — `await step.run('a'); await step.waitForEvent('b')`.
  A figure you cannot tie back to the code that produced it is asking the
  reader to reverse-engineer the point.
- [ ] **Surrounding spans, present but obscured.** Every figure should look like
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
- [ ] **"Fan-out inside a fan-out"** — good figure; apply the same per-element
  hover rule as above.
- [?] **"Width is decided at runtime"** — delete, probably. Its one unique claim
  is that the cable runs from `decide` *into* the request rather than out to its
  members — but "one resumption, two steps" states the same rule with a more
  common shape. Proposal: delete it, and fold "width is decided at runtime" into
  the uneven fan-out caption, which already demonstrates it.
- [?] **"One resumption reports two steps"** — keep or delete. Its claim (one
  ribbon over two steps implies no order) is already made twice above. Proposal:
  keep, because with a code example it becomes the clearest `Promise.all` shape
  in the section — but it has to earn that.
- [ ] **"Two branches, two requests, no ribbon"** — should show the discovery
  requests. Under parallelism we always preplan steps, so two requests exist and
  should be visible. Drawn with none today, which contradicts the rule above.

## Attribution

- [ ] **Cut the section back to what earns its place.** It is inert until
  **"Promise.race()"**, **"a step started but never awaited"**, and
  **"grouping inferred rather than reported"**. Those three are the section.
  The run-up — "pairings stay inside their branch", "plan order is not row
  order", "discovery after unrelated awaits" — is not carrying its weight.
- [ ] **"Grouping inferred rather than reported" needs to say when that
  happens.** The dashed ribbon and dashed cable are right, but nothing tells the
  reader *what causes* a grouping to be inferred — an SDK reporting one opcode
  per response, so the grouping is recovered from overlap rather than read off
  the request.

## Time and the axis

- [ ] **"A compressed idle gap" reads as absence, and it is not.** The `sleep 7d`
  row draws its bar straight *through* the break band. So: no, we do not expect
  gaps — there is essentially always something there, usually a suspended row
  spanning the whole stretch. The break compresses the **axis**; it is not a
  hole in the data, and drawing it as a hatched band that looks like nothing
  happened is the defect.
- [ ] **The elastic axis needs a better design.** Same root cause: the break band
  and the bar running through it are fighting, and the band wins. Note the React
  gallery already has the derivation (`buildTimeScale`, breaks at >=5s and >=15%
  of the run) — the presentation is the problem, not the maths.

## Judged good, no action

- **Outcomes.**
- **Waiting.**
- **Fan-out inside a fan-out**, other than the hover rule.
