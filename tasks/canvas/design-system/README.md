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
