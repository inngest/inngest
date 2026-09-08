# Lessons Learned

## Debugging Approach

### 1. Practical repro beats static analysis
Build, run, query GQL, look at actual data. The span pipeline has too many layers (tracer → exporter → CQRS tree builder → GQL) for static analysis to be reliable. A 15-minute repro with targeted debug logging beats hours of code reading.

**Best debug points:** `ExportSpans` in `tracer_sqlc.go` (what gets written to SQLite) and `mapRootSpansFromRows` in `cqrs.go` (how the trace tree is built). Log `name`, `spanID`, `parentSpanID`, `dynamicSpanID`. Remove after diagnosis.

## Span & Trace Architecture

### 2. Orphaned spans corrupt the root
`mapRootSpansFromRows` sets `root` to the last span with no parent. A span with `parentSpanID == "0000000000000000"` (zero value) becomes a root candidate, **replacing** the real Run span and collapsing the hierarchy.

### 3. Span parent propagation requires queue metadata
Parent span references flow through `queue.Item.Metadata` via `meta.PropagationKey`. The `Carriers` field in `CreateSpanOptions` handles injection. When creating queue items outside the executor (e.g., checkpoint code), you must manually inject the parent span reference or execution spans will be orphaned.

### 4. Sync checkpoint must replicate async span hierarchy
In async mode, the executor creates all spans. In sync/checkpoint mode, `checkpoint.go` must create the same hierarchy:
- **Step span**: parent = Run span
- **Execution span**: parent = Step span (one per attempt)
- **Retry queue metadata**: must carry the step span reference

### 5. Two data stores: `trace_runs` (legacy) vs `spans` (preview)
- **`trace_runs`**: Populated by OTLP lifecycle hooks. Sync functions that complete without going async never fire `OnFunctionFinished`, leaving status stuck at QUEUED.
- **`spans`**: Populated by the internal tracer's `dbExporter`. Always up-to-date.

GQL `runs` list: `preview=false` → `trace_runs`, `preview=true` → `spans`. Always prefer spans-based data. The UI's `updateDynamicRunData` can overwrite correct span status with stale `trace_runs` status — use `runData.trace.status` not `runData.status`.

## SDK (inngest-js)

### 6. JS Error serialization for checkpoints
`JSON.stringify(new Error("x"))` produces `{}` because `name`/`message`/`stack` are non-enumerable. Always use `serializeError()` from `helpers/errors.ts`. The canonical pattern for transforming step results (success or error) is `stepRanHandler`, which calls `transformOutput` → runs middleware → serializes errors. Use it in any handler that produces step results before checkpointing.

### 7. Checkpoint request bodies can silently drop fields
If a field exists in the TypeScript type but isn't in the `JSON.stringify`/fetch body, the server receives zero/empty. Verify the actual `body:` object in the `fetch` call in `api.ts`.

### 8. `retries` = retry count, not total attempts
SDK sends `retries: N` (default 3) meaning N retries after the initial attempt = N+1 total. Checkpoint retry queue items start at `Attempt: 1`.

### 9. React `useState` + `useEffect` race
```typescript
const { value: flag } = booleanFlag('feature', true);  // async, defaults true
const [preview, setPreview] = useState(false);          // starts false
useEffect(() => setPreview(flag), [flag]);              // sets true after render
const query = useQuery({ enabled: flagReady, ... });    // fires with preview=false
```
Fix: initialize state to the default value, or compute directly without intermediate state.

## Go Server

### 10. httpv2 sync driver drops error info on non-opcode responses
When `httpv2.sync()` gets a non-opcode response (e.g., `function-rejected` at 400), `parseOpcodes` fails and returns a `UserError`. Returning `nil` for `DriverResponse` causes the executor to see `StatusCode: 0` and no error, finalizing as **Completed** instead of **Failed**. Fix: construct a proper `DriverResponse` with HTTP response data when `parseOpcodes` fails.

### 11. Sync runs should start as Running, not Queued
`executor.go`'s `Schedule()` defaults `DynamicStatus` to `StepStatusQueued`. For `RunModeSync`, the function is already executing — set `StepStatusRunning` instead.

## Dev Workflow

### 12. E2E test recipe
```bash
# Build (CGO_ENABLED=0 required without gcc)
CGO_ENABLED=0 go build -o /tmp/inngest-dev ./cmd/*.go

# Start dev server
/tmp/inngest-dev dev --no-poll --retry-interval 1

# Install SDK changes in bun-sync example
cd inngest-js/examples/bun-sync
bun remove inngest && (cd ../../packages/inngest && pnpm local:pack) && bun add inngest@../../packages/inngest/inngest.tgz

# Start example
INNGEST_DEV=1 bun run index.ts

# Query runs
curl -s 'http://localhost:8288/v0/gql' -H 'Content-Type: application/json' \
  -d '{"query":"{ runs(orderBy: [{field: QUEUED_AT, direction: DESC}], filter: {from: \"2025-01-01T00:00:00Z\"}) { edges { node { id status } } } }"}'

# Query trace tree (substitute RUN_ID)
curl -s 'http://localhost:8288/v0/gql' -H 'Content-Type: application/json' \
  -d '{"query":"{ runTrace(runID: \"<RUN_ID>\") { spanID name status attempts outputID childrenSpans { spanID name status attempts outputID childrenSpans { spanID name status outputID } } } }"}'
```

### 13. EXTEND spans with zero-value status overwrite real status
`UpdateSpan` always wrote `DynamicStatus: opts.Status` to EXTEND spans, even when Status was the zero value (`StepStatusUnknown`). In `GetSpanRuns`, spans are grouped by `run_id + dynamic_span_id` and processed in start_time order — the EXTEND span (later) overwrites the run span's real status (e.g., Running → Unknown). Then `ToFunctionRunStatus(RunStatusUnknown)` returns an error, and the resolver silently skips the run with `if err != nil { continue }`.

**Fix**: (1) Only write `DynamicStatus` in `UpdateSpan` when status is explicitly set (non-zero). (2) Skip `StepStatusUnknown` when merging statuses in `GetSpanRuns` as defense-in-depth.

**Detection signal**: Run appears in `preview: false` (trace_runs) but not `preview: true` (spans). Direct SQLite query shows all spans exist with correct parents.

### 14. Checkpoint retry enqueue must use the configured backoff
The checkpoint path (`checkpoint.go`) was enqueuing step retries at `time.Now()` (immediate), while the normal async path (`queue/process.go`) uses `q.backoffFunc(attempt)` which respects the configured backoff (table-based by default, linear if `--retry-interval` is set). Fix: plumb the backoff function from devserver → `CheckpointAPIOpts` → `checkpoint.Opts.BackoffFunc`, defaulting to `backoff.DefaultBackoff` when nil.

### 15. Stop dev server before running SDK tests
SDK unit tests use nock mocks. A running dev server on :8288 causes the SDK to auto-discover it and bypass mocks, failing all framework registration tests.

### 16. Vite can serve a stale module to one importer and a fresh one to another
Symptom: the fixture gallery's facts row reported `1 group, aggregated=true` from
`planCollapse`, while the `Canvas` *rendered on the same page* drew all 502
uncollapsed nodes. Same input, same function, two answers — which is impossible
in the model, and the vitest suite agreed with the gallery.

Cause: HMR had updated `Canvas.tsx` and the route module, but `collapse.ts` was
being served from a stale transform to one of them. `pkill -f "vite dev"`,
`rm -rf apps/dev-server-ui/node_modules/.vite`, restart — and the render matched
the model immediately.

**Detection signal**: the UI contradicts a passing unit test on the same input,
or React Flow warns `Node type "X" not found` for a type you just added.
**Prevention**: after adding a new *module* (not just editing one) that is
imported from more than one place, restart Vite before debugging the logic.
Screenshots are fresh browser instances, so they do not rule this out — the
staleness is server-side, not in the page.

### 17. A passing assertion about a model is not evidence about a view
Symptom: segment tooltips were written, populated by every generator, asserted
by a test over all 44 fixtures, and **invisible to every user**. They shipped as
a native `title=` attribute, and the row's Radix HoverCard opens on the same
pointer move — immediately, and styled — so the browser tooltip never won.

The test asserted `segment.tooltip` was truthy. That was true. The strings were
correct. The reviewer found it by hovering.

**Failure mode**: writing the test at the layer the change was easiest to test
at, rather than at the layer the change was supposed to have an effect. The
model was never the risky part; the delivery was, and the delivery had no test
and no manual check either.

**Detection signal**: a change whose whole purpose is "the user can now see X",
verified only by a unit test. Ask "what would I look at to know a user sees
this?" — if the answer is not a screenshot or a DOM read, the change is
unverified however green the suite is.

**Prevention**: for anything user-visible, read it back out of the DOM before
committing. `page.hover()` + reading the popper's `innerText` took under a
minute and would have caught it immediately. The same check later caught a
second real bug on the first try — a fan-out whose members had all collapsed to
0% because the segment positions were measured from `queuedAt` while the
envelope used execution start.

### 18. Check the rendered thing against something outside the change
The general form of #17, and the rule that actually catches this class. Three
times in one feature I shipped something that passed its own check and did not
work, and the check passed each time for a *different* reason:

1. **Tooltips invisible to every user.** They were in the file, wired correctly,
   and asserted by a test over 44 fixtures — but shipped as `title=`, which the
   row's HoverCard beat to the pointer. The model was right; the delivery had no
   test and no manual check.
2. **A rendered label wrong against its own ruler.** The Run row reported the
   run's whole life while the axis still spanned execution only, on 43 of 45
   fixtures. The coherence test compared the bar's *geometry* to the axis and
   passed throughout, because the regression was in the printed label.
3. **A fix whose code was not there.** A scripted `.replace()` whose target no
   longer matched — a formatter had re-wrapped the expression — returns the
   input unchanged and reports nothing. Build passed, tests passed, three of
   four fixes shipped.

No single narrower rule covers all three. A grep for each replacement's token
catches only (3); (1) and (2) would both sail past it, because the code was
present and correct-looking in each. What caught all three was the same act:
**open the rendered artefact, read the number it shows, and compare it against
something the change did not produce** — another view, the axis, the payload.

**Prevention, in order of cost:**
- After a scripted multi-edit, `grep -c` a distinctive token from each
  replacement. Cheap pre-flight, catches (3) only. `Edit` errors on a missed
  match; `.replace()` does not, so this is the price of batching.
- For anything user-visible, read it back out of the DOM. `page.hover()` plus
  the popper's `innerText` takes under a minute.
- Assert the *printed* value, not the value that feeds it. The same number
  computed twice is not a check; the number on screen against the number in
  another view is.

**Detection signal**: "the change didn't take", or a green suite for a change
whose whole purpose is that someone can now see something.

### 19. A defect that looks like a decision stops being questioned
`wide`'s collapsed fan-out drew twelve steps as twelve evenly-pitched, equal
ticks — perfectly regular, so it read as a deliberate design. It survived two
full review rounds before anyone noticed the run was *concurrent* and the row
was drawing it as a sequence, contradicting the minimap directly above it.

Irregular output invites scrutiny; regular output gets taken for intent. When
something looks designed, the question "what would this look like if it were
wrong?" is the one that doesn't get asked.

**Detection signal**: a rendering that is suspiciously uniform — equal widths,
even spacing, identical rows — where the underlying data has no reason to be.
Check it against a second view of the same data before accepting it.

### 20. A verification tool pinned to an absolute path is a verification tool that will stop existing
All three review agents (`canvas-ux-review`, `canvas-fixture-critic`,
`canvas-correctness`) carried the literal string
`/nix/store/3qgx41z8882ff85y9prdc5zgbb2id6y8-chromium-152.0.7977.64/bin/chromium`.
Nix garbage-collected it. Every agent's screenshot step was dead, and nothing
reported that — an agent asked to review a screen it cannot see writes a report
about nothing, and the report still arrives looking like a report.

The same file also hardcoded the Vite port (`5175`, and `5177` in `LOG.md`),
which moves whenever another dev server holds 5173.

**Detection signal**: none, which is the point. The failure is silent from the
outside and only visible if you run the tool yourself.

**Prevention**: one script, `tasks/canvas/shot.sh`, that every agent and every
session calls.
- It resolves Chromium through a **Nix GC root** (`nix build --out-link
  ~/.cache/canvas-shot/chromium nixpkgs#ungoogled-chromium`), which both creates
  the symlink and registers it as an indirect root, so the store path cannot be
  collected while the link exists — and rebuilds it from cache if it is.
- It **finds the gallery port** rather than assuming it.
- It `curl`s the URL first and **refuses to screenshot a page that did not
  answer** — chromium happily renders its own error page into a valid PNG, and
  that image read back as evidence is worse than no image at all.
- It **deletes the output file before writing**, so a failure cannot leave the
  previous run's image behind to be read as the current one. This is #18 again,
  in a new place.

Not Playwright or Puppeteer: both download a dynamically-linked Chrome-for-
Testing build that will not start on NixOS (`libglib-2.0.so.0` missing).

### 22. An injection anchored on a tag the document does not have fails silently
To screenshot one tab of the artifact I appended a script by replacing
`</body>`. The generated document has **no `</body>` tag** — it ends at
`</script>` — so `String.replace` found nothing, returned the input unchanged,
and every screenshot came back showing the default tab while I read it as the
tab I had asked for. The same bug was in the earlier "scroll to this section"
helper, which had therefore never scrolled.

This is the `.replace()` failure from #18 in a new place: **a replace whose
target is absent is not an error, it is a no-op that looks like success.**

**Detection signal**: a screenshot that looks plausible but shows the default
state of whatever you tried to drive.
**Prevention**: append rather than anchor when you control the output, and when
you must anchor, assert the anchor exists first. `Edit` errors on a missed
match; `String.replace` does not, and that difference has now cost three
separate sessions.

**Then it happened again, one layer up.** With the injection working, the helper
set the tab by writing `hidden` on each section directly. That *bypassed the
page's own click handler* — which had a hardcoded list of three tab ids and had
never been updated for the fourth. So the screenshot showed a perfect Docs tab
and the served page showed nothing at all, and the person looking at
`localhost:5199` found it before I did.

**The rule this settles: a harness must drive the same path a user does.**
Setting the end state directly proves the end state can render; it proves
nothing about the thing that produces it. The helper clicks the real button now,
and throws if the click leaves every section hidden — so the failure that
shipped cannot ship silently again.

And the fix in the page itself is the general one: **the handler derives its
sections from the tab buttons instead of a hardcoded list**, so adding a tab
cannot leave it stale.

### 21. A harness that slices HTML produces bugs the page does not have
To screenshot one section of the design artifact I sliced the built HTML by
string offset and wrapped the fragment in a new document. The slice cut through
an element, so the code block escaped its container and every syntax-highlight
span rendered on its own line — a page that looked catastrophically broken and
was not. I nearly "fixed" a layout that was already correct.

**Detection signal**: a rendering failure far larger than the change that
supposedly caused it, in a region the change did not touch.

**Prevention**: screenshot the *real* document. To reach a section, drive the
page — flip the tab attribute and inject a `scrollIntoView` — rather than
cutting the markup up. `tasks/canvas/shot.sh` plus a scroll target is the whole
technique, and it cannot produce invalid HTML because it never edits structure.

This is #18 from the other side: there the artefact was real and the check was
weak; here the check invented a defect. Both are the same rule — the thing you
read has to be the thing that ships.

### 23. A greedy regex over a whole document will eventually eat the document
The figure exporter harvested CSS variables with `/--[a-z0-9-]+: *[^;]+;/g`.
That worked for a month, because every declaration it met had a trailing
semicolon. Then rows started carrying `style="--i:3;--s:0"` — and the last
declaration in a style **attribute** has no trailing `;`, so `[^;]+` ran on
across newlines, through the rest of the SVG and into the page, until it found a
semicolon somewhere else entirely.

Every exported figure came out at **1.1MB with `</main>` and `<script>` inside
it**. The docs tab inlined them, the page ended up with sixteen `</main>` against
one `<main>` and a script that never closed, and the browser parsed the rest of
the document as JavaScript. Every tab stopped working.

**What makes this one worth recording is what did not notice.** The build was
clean. The validator was green — 109 figures, 296 rows, code and figures agree.
Every check I had asked about the *figures*, and the figures were perfect. The
artefact they were written into was rubble.

**Detection signal**: none from the build. It was found by opening the page,
again.

**Prevention, in order:**
- Bound the match: `[^;{}<>\n]{1,60};` cannot leave the declaration it is in.
- **Assert the size of anything generated.** A figure is a few KB; the exporter
  now refuses to write one over 60KB, because a silently enormous asset is
  exactly the sort of thing that ships.
- **Check the container, not just the contents.** `validate.mjs` now counts
  `<main>` and `<script>` pairs in the generated page and the size of every doc
  image. Deliberately appending one `</main>` makes it exit 1.

The general rule, which is #18 once more: every check I had verified the thing I
was making, and none verified the thing I was making it *into*.

### 24. Code emitted through a template literal loses one level of escaping
The page's own script is built inside a template literal in `ds.mjs`, so every
backslash in it is consumed once before it ever reaches the browser. A viewBox
parser written as `split(/[\s,]+/)` shipped as `split(/[s,]+/)` — a regex that
splits on the letter **s** — so every `"0 0 470 89"` parsed as a single token,
the length guard rejected it, and the function that resized every figure ran
three times and touched nothing. No error, no warning, and the feature simply
did not happen.

The same file had already eaten backticks three times: a backtick anywhere in an
injected comment ends the template literal and the build dies with a syntax
error pointing at the comment.

**Detection signal**: a function that demonstrably runs (a counter proves it) and
demonstrably does nothing. Instrumenting the *call* is not enough; instrument the
first line of the body that has an effect.

**Prevention**: in code destined for a template literal, avoid characters that
carry meaning to the outer layer. Use `[ ,]` instead of `[\s,]`, `\\d` where you
mean `\d`, and no backticks in comments at all. When in doubt, grep the
generated artefact for the line you wrote and check it survived:

```bash
grep -o 'split([^)]*)' trace-design-system.html
```

This is lesson 18 again — check the rendered artefact, not the source.

---

## 25. A byte-identical gate proves nothing moved, not that it was right

**Failure mode**: converting 300 hand-authored rows of bars into derived rows, I
gated the whole conversion on the built artifact being byte-identical. It was —
and I reported that as success. But I had built the bars-to-moments reader by
tuning it until it reproduced the authoring, so the gate could only ever pass.
The stakeholder saw it immediately: "it could be that byte equivalence isn't
true btw, as doing this may genuinely fix some of the figures haha".

It did. Two rows drew the same event two different ways: a *step* that threw
with another attempt coming was followed by `backoff` (red hatch), while a
*discovery request* that threw with another attempt coming was followed by plain
`idle` queue — in a figure whose own annotations read "backoff 1s" and "backoff
2s" over the grey.

**Detection signal**: a verification that cannot fail. If the oracle was derived
from the thing under test, it is a consistency check, not a correctness one.

**Prevention**: when converting authored data to derived data, byte-identity is
the *regression* check. Add a separate **disagreement scan** for the correctness
one: enumerate every (event, what followed it) pair actually drawn and group
them. One event drawn two ways is a defect in one of them. Do that scan before
declaring the conversion done.

---

## 26. `display:none` on one figure kills the fills on all of them

**Failure mode**: to screenshot a single figure I hid its siblings with
`display:none`. The screenshot came back with every bar missing and only the
event circles left, which looked like the change under test had broken the
renderer.

Each `<svg>` emits its own copy of `HATCH`, so `#hx-good`, `#hx-disc` and the
rest are duplicated across ~145 figures. A `url(#hx-good)` fill resolves to the
*first* one in the document — and hiding the earlier figures took it out of the
render tree, so every fill on the page silently resolved to nothing.

**Detection signal**: a screenshot that contradicts the artifact's own markup.
Grep the built HTML for the element before believing the picture:

```bash
grep -o '<rect class="bar[^>]*>' trace-design-system.html | head
```

**Prevention**: never isolate a figure by hiding its siblings. Shift the page
instead — set a negative `margin-top` on `body` so the figure lands at the top of
the viewport, leaving the whole document in the render tree.

---

## 27. Grouping figures by code invites drawing the same picture four times

**Failure mode**: reorganising the Scenarios page around the code that produced
each figure, I gave every sentence its own figure. Under `await step.run('a')`
that produced four identical drawings in a row — queue time, system latency, the
hatching rule and "the step returned" are four things to say about **one**
picture, not four pictures. The stakeholder spotted it immediately: "figure 1,
2, 3, 4 are all the same 😃".

**Detection signal**: two figures in one group whose rows and sequence of bar
kinds match. Not byte-equality — they differed in where the bars sat, which is
invisible to a reader and was enough to hide the repetition from a diff.

**Prevention**: `validate.mjs` now compares figures by shape within each group
and fails on a match, telling you to make them bullets under one of them. A note
in the scenario list may be a string or a list of strings; the list renders as
bullets above a single figure.

---

## 28. Never `git checkout --` a file you have been editing this session

**Failure mode**: to undo a two-line probe I had just injected into `ds.mjs`, I
ran `git checkout -- ds.mjs`. It reverted the file to HEAD, silently discarding
about forty lines of edits from earlier in the same turn that had not been
committed yet. The probe was removed; so was the work.

**Detection signal**: the next `grep` for something added minutes earlier
returned zero. Nothing errored — the build still passed, because the reverted
file was internally consistent.

**Prevention**: undo an edit with the inverse edit, not with the version control
system. If a probe needs reverting, write it so removing it is a string
replacement — or make the probe in a copy under `/tmp` and never touch the real
file. Reach for `git checkout --` only on a file this session has not written to.

---

## 29. Lesson 24, a third time: no backslashes in a template literal

**Failure mode**: panel sections were given a fold control whose handler
identified its section by parsing the heading text:

```js
var name = h.textContent.replace(/›/,'').trim().split(/\s{2,}|\n/)[0];
```

That line lives inside a JS template literal in `ds.mjs`, so `\s` shipped as
`s` and `\n` shipped as a real newline — producing an unterminated regex, a
`SyntaxError`, and **the entire page script failing to run**. The page still
looked fine: the markup was all server-rendered, so the only symptom was that
nothing in the panel did anything.

**Detection signal**: a feature that does nothing, with no visible error. The
way to see it is to ask the page:

```js
window.addEventListener('error', e => window.__errs.push(e.message + ' @' + e.lineno))
```

then read `__errs` out of the DOM dump. Worth reaching for whenever injected
behaviour is silently inert.

**Prevention**: this is the third time (see #24). The real fix is not "escape it
correctly" but **do not parse in injected code**. The handler needed a name; the
markup should hand it one — `data-sec="${title}"` — and the regex disappears
rather than being made to survive a round trip through a template literal.

---

## 30. A CSS transition needs a previous computed value; `display:none` has none

**Failure mode**: morphing between two pre-rendered variants of a figure. Both
were in the DOM with one `display:none`. The approach was: set the incoming
elements' geometry to the outgoing values, swap which variant is shown, force a
reflow, then remove the inline values so CSS transitions carry them home.

Nothing animated. The transition was correctly registered — `transition-property`
computed to `x, cx, width, --w, opacity` — and the start values were applied.
But an element that was `display:none` until this frame has no *previous*
computed style, so the first style it gets is a starting state, not a change,
and there is nothing to transition from.

**Detection signal**: sample the computed value on a timer after the toggle. It
sat at the final value from `t=0`, which distinguishes "the transition never
started" from "the transition is too fast" or "the wrong property".

**Prevention**: use `element.animate()` for anything crossing a visibility
change. It states both endpoints explicitly and does not care what the element's
style history is. Keep CSS transitions for elements that were already visible.

Two things that bit alongside it, both worth keeping:

- **A custom property must be registered with `@property` to interpolate.** An
  unregistered `--w` is a string to the animation engine and jumps.
- **Match elements between two renders per KIND, not across one mixed list.** A
  compressed figure draws itself twice — once more, clipped, for the blur — so
  the variants hold different numbers of elements. A single `querySelectorAll`
  of everything diverges at the first extra and every pair after it is a bar
  against a circle.

---

## 31. Fix it in one place, then check the other places that draw the same thing

**Failure mode**: a compression band was inflating the height a figure was
fitted to. Fixed it in `fig()`, measured it, reported it fixed. The stakeholder
came back: "Still happening on the Fixtures tab". The scrubber draws its own
band, in the browser, in a different function — and that one was never wrapped.

**Detection signal**: I verified on the tab where I had made the change. The
measurement was sound and the conclusion was still wrong, because the sample was
chosen to match the fix.

**Prevention**: after fixing a drawing rule, grep for every place that emits the
thing, not every place you edited: `grep -rn 'cmpband' src/`. The artifact draws
its figures twice on purpose — once at build time and once in the browser for
the scrubbers — so "the renderer" is two renderers for anything the scrubber
also draws.

And verify across all four tabs, which `HOW-WE-WORK.md` already says and I did
not do.

---

## 32. Measure after the webfont lands, or measure twice

**Failure mode**: with the band excluded, figure heights across a feature toggle
matched to within 0.5px — close enough that I reported it as sub-pixel residue
and moved on. It was not residue. The bounding boxes were IDENTICAL
(`y=10.36 h=81.14` both ways); only the viewBox differed, because the first
`refit()` ran before the display font arrived and measured text against the
fallback, while every later one measured against the real font.

**Detection signal**: two numbers that differ while the thing they are derived
from does not. That is not rounding, it is two different measurements.

**Prevention**: anything that sizes a box from `getBBox()` over text has to run
again once the fonts are ready:

```js
if (document.fonts && document.fonts.ready) document.fonts.ready.then(refit);
```

---

## 33. Duplicate SVG ids, again — this time defining different things

**Failure mode**: with four builds of each figure on one page, each build
numbered its clip paths and blur filters from one. Two variants of the same
figure both defined `cmp-7`. A `url(#cmp-7)` resolves to whichever came first in
the DOCUMENT, so the second variant's compression blur was clipped to the FIRST
variant's band — drawing a soft bright copy of the rows and their labels
wherever the other layout had put its cut. Reported as "some of the step names
and events have like a halo and shine slightly".

This is lesson 26's problem with the sign flipped. There, identical definitions
shared an id and hiding one broke the others. Here, DIFFERENT definitions shared
an id and the wrong one won. Same root cause: ids are document-global and the
artifact emits the same generator's output many times over.

**Detection signal**: nothing errors. Grep the built page:

```bash
grep -o 'id="cmp-[0-9]*"' trace-design-system.html | sort | uniq -c | awk '$1>1'
```

...but note that a duplicate is only a bug when the definitions DIFFER — the
same figure rendered on two tabs duplicates its ids harmlessly. The check has to
compare the bodies, not count the names.

**Prevention**: `validate.mjs` now asserts that every `url(#…)` inside a figure
variant resolves within that variant, and that no id on the page is defined two
different ways.

**And the fix that looked obvious was wrong**: giving each build its own number
range fixed the collision and grew the page 13%, because ids that differ between
builds make two identical drawings compare as different — so every compressed
figure shipped four times instead of once. Namespace at the point where the
variants are merged, not at the point they are generated.

---

## 34. Conceding a disputed claim is not the same as resolving it

**Failure mode**: I claimed a v4 run ran a step and planned the next in one
request. Jack said that was impossible. I conceded and told him the capture was
missing requests — which was wrong, and sent us both down a diagnosis of a
non-existent data gap.

**Detection signal**: a disagreement about system behaviour where the payload is
consistent with both readings. That is the signal to stop reasoning and go and
look.

**Prevention**: instrument the source. Twenty lines of `fmt.Printf` in
`HandleGeneratorResponse` and `CheckpointAsyncSteps` settled it in one run: two
requests, and each response carrying ONE op, with the completed step arriving
out of band as a checkpoint. Neither of our positions was right. Revert the
probes afterwards; say plainly who was wrong.

Three real bugs this session were found only because a disputed claim got
instrumented rather than argued: the sleep opcode's `startedAt` overwriting the
execution span's, the timed-out wait drawing as matched, and the retry drawing
one green bar over a failure.

---

## 35. When the drawing looks wrong, suspect the reader before the data

Every defect this session was a field the payload already carried and `loadRun`
was not reading:

| symptom | field that knew |
|---|---|
| a timed-out wait drawn green as matched | `stepInfo.timedOut` |
| a retried step drawn as one clean green bar | the per-attempt spans |
| the whole gap between attempts drawn as backoff | `scheduledAt` on the retry |
| a cancelled run drawn as still going | the run's own `status` |
| a request drawn three times, once per row it planned | `plannedStepIDs` order |
| row order not matching the code | `plannedStepIDs` order again |

**Prevention**: before concluding the trace cannot say something, dump every
non-null field on the spans and metadata and look. Twice this session the answer
was there and I proposed inventing a value instead — an earlier `queuedAt` to
close a gap, a backoff derived from `TableBackoff`'s defaults. Both would have
been guesses over measurements: the gap is real executor latency, and the
backoff is 15s plus up to 30s of jitter so the defaults cannot give it.
