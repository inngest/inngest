# Trace design system

A proposed design language for drawing an Inngest function run, with a worked
example for every shape the trace has to survive. It is a document to argue
from, not shipped UI.

Published as an artifact; `trace-design-system.html` here is the same file and
is regenerated from `src/`.

## Build

```sh
cd src
node ds.mjs          # writes ../trace-design-system.html
node validate.mjs    # checks every drawn row for a legal sequence of marks
```

`ds.mjs` runs each generator first, so there is no separate step.

## Layout

`src/vocabulary.mjs` is the single source of truth. It holds the palette, the
bar kinds and their substances, the event marks, the row geometry, and the two
drawing primitives (`barSvg`, `markSvg`). Everything else reads from it:

- `micro.mjs` — the row/figure renderer used by every static figure.
- `render.mjs` — a second, wider geometry used only by the older captured
  fixture drawings in `batch1.mjs`.
- `items-*.mjs`, `primer.mjs`, `steptypes.mjs`, `detail.mjs`, `runbar.mjs`,
  `connect.mjs`, `batch1.mjs` — the figures, grouped by section.
- `fixtures.mjs` — the scrubbable fixtures, emitted as data rather than as
  drawings. The page computes the state at whatever moment the handle is on,
  using `barSvg`/`markSvg` shipped into it as source, so the browser draws with
  the same code the static figures were built with.
- `ds.mjs` — assembles the page: the vocabulary panel, the three tabs, the row
  popover, and the fixture scrubbers.

## Why the validator exists

The recurring defect in this work is the same interval drawn twice under two
names, or a row whose marks do not describe a state a run can be in.
`validate.mjs` reads the generated SVGs back, classifies every mark, and
asserts each row opens on a queued or planned mark, never repeats a mark, and
resolves at most once at the end. It has caught more real errors than looking
at the page has.

## How this relates to the React implementation

**This artifact decides what a mark *means*. The gallery decides whether the
data supports it.** The split, and why, is in `../HOW-WE-WORK.md`.

The artifact's own limit, stated so nobody forgets it: its fixtures are
**hand-authored proportions**, not captured payloads — three of them (`step`,
`v4sequential`, `invoke`) in `src/fixtures.mjs`. It cannot catch a number that
does not reconcile, which is the defect class this view actually has. Anything
decided here is a hypothesis until a real payload goes through it in the
gallery, against `longgap`, `v4pathological`, `blocked` and `loop40`.

### The ledger

**The two vocabularies are not the same taxonomy, and nobody has reconciled
them.** That is the state, recorded rather than glossed over.

- The artifact's `SEMANTIC` in `src/vocabulary.mjs` has **13 bar kinds** keyed by
  *what substance the interval is* — `good` `bad` `running` `stopped` `child`
  `disc` `wait` `waitok` `waitout` `waitstop` `backoff` `hold` `idle` — crossed
  with one binary, `COMPUTE` vs `NOCOMPUTE`, which decides solid or hatched.
- React's `BarStyleKey` in `RunDetailsV4/TimelineBar.types.ts` has **22 keys**
  organised by *what produced the interval* — step type (`step.run`,
  `step.sleep`, …), timing category (`timing.waiting`, `timing.backoff`,
  `timing.inngest.concurrency`, …), and HTTP phase (`timing.http.dns` …).

Concrete divergences, verified by reading both lists:

| Artifact has | React has | Note |
|---|---|---|
| `hold` — concurrency, throttle, rate limit **or** debounce, as one substance | `timing.inngest.concurrency` only | Three of the four flow-control reasons have no encoding. Item G ("flow control did its job") depends on this. |
| `stopped`, `waitstop` — executing/waiting when the run was cancelled | *nothing* | Constraint 3 in `PROMPT.md`: cancelled needs a third state. The artifact has one; React does not. |
| `waitok` vs `waitout` — a wait that matched vs one that expired | `step.waitForEvent`, outcome not in the style key | The outcome is carried elsewhere, not by the substance. |
| *nothing* | `timing.http.dns` / `.tcp` / `.tls` / `.server` / `.transfer` | The HTTP phase breakdown predates the artifact and the artifact has no opinion on it. |
| 9 event marks (`EV`) — `queued` `ribbon` `disc` `hollow` `hollow-bad` `ok` `failed` `timeout` `cancelled` | not audited | Compare against `RunMinimap.tsx` before relying on either. |

**Not audited:** whether the artifact's colours resolve to the same design
tokens React uses, the mark vocabulary above, and the geometry (`GEOM` here vs
the React row heights). Do not assume agreement on any of them.

### The rule that keeps this honest

When something decided here lands in React, **add a row to the ledger** saying
so. When something here is tried and rejected, add a row saying that and why —
that is the half that gets lost, and it is why the same idea comes back.
