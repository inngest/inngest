# How we work on this

The method, written down because until now it only existed in conversation and
in one paragraph buried in `LOG.md`. `PROMPT.md` is *what* to build. This is
*how*, and it takes precedence over the repo's general working style for
anything under `tasks/canvas/` and `RunDetailsV4/canvas/`.

## The loop

**Messy first, then firm, then tested — in that order, and the order matters.**

1. **A line through the system, fast.** Rough code, no abstraction, no tests.
   The goal is a picture on screen that the human can react to. Getting this
   wrong costs one revert; getting it wrong *after* it has been made robust
   costs a day.
2. **Show it.** The dev server is up for the whole session and the human is
   watching it. Say which fixture to look at and what changed. A screenshot in
   the reply beats a description of a screenshot.
3. **React, together.** Keep, change, or throw away. Most of the value in this
   work has come from this step, and it is cheap only while the code is still
   messy.
4. **Then firm it up.** Once the shape is agreed: proper names, the model
   change made properly, the edge cases.
5. **Tests, much later, and thin.** This is a proof of concept. Enough tests to
   stop iteration silently breaking things — model-level tests are cheap and
   stay, render-level tests stay light. No E2E. Where a test would be complex
   and the risk is low, a comment is preferred. If a class of breakage starts
   recurring, *that* is the signal to test it properly.

## When to stop and ask

**Design forks only.** When two options are both defensible and a user would
see the difference, stop and ask — `AskUserQuestion`, real options, one at a
time, with a recommendation and what changes depending on the answer. Never
prose the human has to compose a reply to.

Everything else is decided and recorded in `LOG.md`. Technical choices get their
justification written down; they are a record, not a gate.

Also stop for the `PROMPT.md` §5 tripwires: a new dependency, a Go or SDK change
beyond read-only, an item that turns out to be invalid as specified, an item
past ~15 changed files, or wanting to weaken or delete a test.

## Where a problem gets solved

Two surfaces, and the question decides which:

| The question is about | Solve it in | Why |
|---|---|---|
| What a mark **means** — solid vs hatched, what colour is allowed to say, what a row is | `tasks/canvas/design-system/` | Fastest possible iteration. Regenerate and look. |
| What the data **is** — timings, structure, whether a number reconciles, scale | the gallery, in React | The artifact's three fixtures are hand-authored proportions. It cannot catch a number that does not add up, and that is the defect class this view actually has. |

**The artifact cannot verify itself.** Its fixtures are drawn, not captured, so
everything decided there is a hypothesis until a real payload goes through it.
An encoding that survives the artifact and then fails on `longgap` or
`v4pathological` has not been validated, it has been sketched.

So: decide meaning in the artifact, prove it in the gallery, and record the
decision in the artifact README's ledger so the two do not drift.

## Verifying

**The rule, from `lessons.md` #18, which is the most expensive thing learned
here:** open the rendered artefact, read the number it shows, and compare it
against something the change did not produce — another view, the axis, the
payload. A passing assertion about a model is not evidence about a view.

- **Screenshots and DOM:** `tasks/canvas/shot.sh <fixture>`, `--dom` for the
  HTML. It finds the Vite port, resolves Chromium through a Nix GC root, and
  refuses to write an image when the page did not load. Never hand-roll a
  chromium invocation — a pinned `/nix/store` path is exactly what broke the
  three review agents once already.
- **`--dom` discovers, screenshots confirm.** Reading the DOM and doing the
  arithmetic found the 86% ghost, the duplicated interval, the axis regression,
  the 1px `Run` row and the 0px note. Screenshots found none of them.
- **The three review agents** — `canvas-ux-review` (does it look right),
  `canvas-fixture-critic` (would the author of the function recognise it),
  `canvas-correctness` (does it match the payload). Run them after a change of
  any size, told what changed and which fixtures to look at.

## Housekeeping

- `LOG.md` — one entry per item: what changed, how it was verified, what is
  uncertain, what was deferred. Keep the **Status / Open / Settled** header at
  the top true; that header is what a cold session reads.
- `lessons.md` — after any correction or discovered mistake: the failure mode,
  the detection signal, the prevention rule.
- Commit per item, message naming it. Branch is local — never push, never add a
  remote.
