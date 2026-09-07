# Scenarios — the bible

The scenarios are the specification: they enumerate every behaviour we support.
This is the working list against them, from the stakeholder's read of the
artifact.

Status: `[ ]` open · `[x]` done · `[?]` needs a decision

---

## Global — applies to every figure

These are the big ones. Each touches every scenario, so they come before any
further per-figure polish.

- [ ] **G1. A code example on every figure.** Trimmed pseudo-code, enough to say
  what this trace is — `await step.run('a'); await step.waitForEvent('b')`. A
  figure that cannot be tied back to the code that produced it is asking the
  reader to reverse-engineer the point.
- [ ] **G2. Surrounding spans, present but obscured.** Every figure should look
  like a *real* trace — run row, trigger, the other steps, finalization, the
  minimap — with everything irrelevant dimmed **and blurred**, not absent.
  Two reasons, and the second is the real one:
  1. the figure reads as a trace rather than a diagram;
  2. **it forces us to practise the full trace** — where the minimap sits
     relative to the run row and the steps, whether finalization makes sense
     here, whether the whole thing holds together. Today the minimap is
     described but never shown in place, so nobody can tell.
- [ ] **G3. Requests, everywhere.** Which requests were made and what each was
  for is most of the complexity in this view, and `c0` establishes it as
  load-bearing. Every scenario should be checked against "does this show the
  requests honestly", not just the sequential one.

## Sequential

- [x] **`c0` gap between `a` and `b`.** `a`'s work ended at 44, `b`'s queue began
  at 48 — four units of unnamed time in the figure whose entire point is *one
  HTTP call, not two*. `b`'s queue now opens the instant `a` resolves.
- Keep: the "reports and executes in the same request" fact is the most
  important thing in the section.

## Fan-out and coalesce

- [x] **`c1` hover model — attention is per element, not per row.** Hovering `b`
  now lights `b` entire, the ribbon, and on `a`'s row *only* the discovery bar,
  the queued mark and the planned mark. `a`'s `ok` bar is faded: it had nothing
  to do with `b` starting. Mechanism is `lit` / `litDots` in `micro.mjs`.
- [x] **`c2` `d` selected shows no selection background.** `pair()` only applied
  the wash when the caption began "hovering". Now explicit via `focus`.
- [x] **`c4` uneven fan-out had no planned marks or queue bars on `b` and `c`.**
  Real bug — they had no `idle` lead-in, so the ribbon threaded rows with no
  circles to thread. Every member of a fan-out is enqueued by the request and
  waits its turn.
- [ ] **`c3` nested fan-out** — good; apply the same per-element hover rule.
- [?] **`c5` "dynamic width" — delete or merge.** Its unique claim is that the
  cable runs from `decide` *into* the request rather than out to its members —
  but `c6` states the same rule ("the cable shows only what caused the
  resumption") with a more common shape. Proposal: delete `c5`, fold "width is
  decided at runtime" into `c4`'s caption, which already demonstrates it.
- [?] **`c6` "one resumption, two steps" — keep or delete.** Its claim (one
  ribbon over two steps implies no order) is already made by `c1` and `c4`.
  Proposal: keep it, because with a code example (G1) it is the clearest
  `Promise.all` shape in the section — but it needs to earn that.
- [ ] **`c8` two branches should show discovery requests.** Under parallelism we
  always preplan steps, so two requests should be visible. Currently drawn with
  none, which contradicts G3.

## Attribution

- [ ] **Cut back to what earns its place.** The section is inert until the
  `Promise.race` example, the never-awaited step, and inferred grouping. Those
  three are the section; the run-up is not.
- [ ] **Inferred grouping needs to say when it happens.** The dashed ribbon and
  dashed cable are right, but nothing tells the reader *what causes* a grouping
  to be inferred rather than reported (an SDK that reports one opcode per
  response, so the grouping is recovered from overlap).

## Time and the axis

- [ ] **`t1` — the break reads as absence, and it is not.** The `sleep 7d` row
  draws a `waitok` bar straight through the break band. So the answer to "do we
  expect gaps?" is **no — there is essentially always something there**, and
  usually it is a suspended row spanning the whole stretch. The break is a
  compression of the *axis*, not a hole in the data, and drawing it as a hatched
  band that looks like nothing happened is the defect.
- [ ] **X1. The elastic axis needs a better design.** Same root cause as above:
  the break band and the bar running through it are fighting, and the band
  wins. The React gallery has the elastic axis already (`buildTimeScale`,
  breaks at ≥5s and ≥15% of the run) — the *derivation* is not the problem, the
  presentation is.

## Sections judged good, no action

- Outcomes.
- Waiting.
- Fan-out inside a fan-out (`c3`), other than the hover rule above.
