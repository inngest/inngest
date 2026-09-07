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

## A figure is a list of events. The drawing follows from the rules.

**This is the governing rule of the artifact and it outranks convenience.** A
figure declares what happened — rows of events with whatever small extra data
they need — and the whole visual result is derived. Change a rule and every
figure changes with it, without anyone remembering which ones to revisit.

Where this was not true, it bit: compression was written as `elastic()` and then
*called by three figures*, so the fixtures went on drawing a 2s sleep across 96%
of their width long after the rule said dead time is worth almost none of it.
The rule existed and reached almost nothing.

The threshold is what makes this work without every figure inventing durations
it never measured. A flat "compress anything idle for more than 3% of the run"
compresses ordinary queue intervals — short, meaningful, and exactly what the
trace exists to show. The test that holds is **scale-free**: a stretch is
compressed when it is longer than all the compute in the run put together,
several times over. Seven days beside 100ms of work passes by a factor of
millions; six units of queue beside a hundred of work does not pass at all.

So:

- **A row is a list of moments, and its bars are derived.** `derive(kind, at,
  end)` in `rules.mjs` is the whole mapping: the interval between two moments is
  a bar, and which bar depends on what the row is — `started > ok` is compute on
  a step, the SDK reporting on a discovery request, a child run on an invoke,
  elapsed sleep on a wait, a userland span on a span. A `started` moment may
  name a third thing, the kind of work it opens, for the row that changes
  substance partway. `moments()` is the inverse, so a figure still authored as
  `segs` is read back into the moments it implies and re-derived through the
  same rule — every row in the artifact goes through it. Verified by
  `audit-derive.mjs` (336/336 round-trip) and by the artifact being
  byte-identical across the change; verified to REACH the figures by changing
  what a wait resolves to and watching 75 bars change with it.
- **`src/rules.mjs` is where the rules live.** Geometry, the elastic thresholds,
  how a band is drawn, the frame, attention, the panel's controls, and the
  interval a pair of moments implies. If you are typing a number into
  `vocabulary.mjs`, `micro.mjs` or `ds.mjs`, it belongs there instead — the rules
  were spread across 2,142 lines of drawing code, so "change the rule" meant
  "find every place that encoded it".
- **There is one renderer.** `render.mjs` was a second one, with its own width,
  row pitch and label gutter, and the captured fixtures went through it — which
  is why they had no frame, no compression, no live geometry and a Run row they
  declared themselves. Deleted. A second renderer is the same interval drawn
  twice under two names, at the level of the code.
- `layout(total, rows)` in `micro.mjs` is **the one place the elastic rule
  lives.** It derives the dead stretches from where nothing was executing across
  every row, collapses what is over the threshold, and shares the remaining
  width by real duration. Pass its result to `fig`.
- **New figures declare milliseconds and go through it.** A figure that
  hand-places positions is a figure the rules cannot reach.
- `opts.linear` opts out, and needs a reason. Only two figures use it, and both
  exist to show the *uncompressed* proportion — they are the argument for
  compression, so compressing them would delete their point.
- **The Run row is derived from the trace, never authored.** A drawn Run row is
  a second account of the same run, kept in step by hand until it is not — and
  an overview disagreeing with the rows beneath it is the one thing it must
  never do. `validate.mjs` fails on a hand-written one.
- **A figure may declare work its rows do not carry** with `busy`, for the case
  where the drawing is done by something else — a collapsed group draws its
  members through `groupRow`, and without `busy` the rule reads the whole
  envelope as idle and compresses the work away.
- **`validate.mjs` counts what the rule cannot reach** and prints the worst
  offenders. That number is a backlog, and it should go down, never up.

## A change is not done until every tab agrees

The artifact has four tabs and they all describe the same system:

| Tab | What it is |
|---|---|
| **Concepts** | The vocabulary and the rules, in the abstract |
| **Scenarios** | Every behaviour we support, with the code that produced it |
| **Fixtures** | Real captured runs at measured proportions |
| **Docs** | The page we would ship, rendered from `docs/understanding-traces.md` |

**Change one and you have probably invalidated the others.** Making the backoff
neutral changes Concepts (the swatch), Scenarios (every retry figure), and Docs
(a sentence describing it). Deleting the minimap changed the Interaction
scenario, a rule in `RULES.md`, and a whole section of the docs. Every one of
those was missed the first time and caught later by reading, which is the slow
way.

So after any change to the vocabulary, a rule, or a figure:

1. **`node ds.mjs && node validate.mjs`.** The validator is not just a row
   checker — it also asserts **every figure's code example names the steps the
   figure actually draws**. That check exists because 34 of 51 snippets were
   written from captions rather than rows and named steps the drawings never
   had. Nothing else would have caught it.
2. **Re-run `DS_FRAME=0 node ds.mjs && node export-docs.mjs && node ds.mjs`** if
   any figure the docs use has changed. The Docs tab renders the *exported*
   files, so a stale image there is the drift made visible.
3. **Look at all four tabs**, not just the one you changed —
   `node src/open-tab.mjs <tab>` then `tasks/canvas/shot.sh --url file:///tmp/ds-tab.html --size 1400,2200`.
4. **Grep the prose.** `RULES.md`, `DELTA.md` and the docs all state rules in
   words, and words do not fail a build. Search for the thing you changed.

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
  **The dump is one unwrapped ~130KB line**, so `Read`ing it fails on token
  limits however you set `offset`/`limit`. Never read it directly — `grep` the
  fragment you want out of it:
  ```bash
  tasks/canvas/shot.sh step --dom > /tmp/d.html
  grep -o '.\{0,120\}first step.\{0,120\}' /tmp/d.html
  ```
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
