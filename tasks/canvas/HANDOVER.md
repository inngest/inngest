# Run Canvas — handover

Written at the end of a long session so the next one can pick up cold. Start here, then read
`todo.md` (the running log of decisions D1–D49, most recent at the bottom) and `00-recon.md` (the
Phase 0 recon of how the data layer works).

Nothing is committed. Nothing has been pushed. Two repos are dirty.

---

## 1. What this is

A left-to-right flow graph of a single run in the Dev Server, sitting above the existing trace.
Clicking a node opens the **existing** `StepInfo` panel — the canvas owns no step state of its own.
Long-term this goes to Cloud too; it is Dev-Server-gated only while it is being built.

Live at `http://localhost:5174/run?runID=<id>` once the stack below is running.

---

## 2. Getting the stack running

Four things, in this order. Ports matter: the Go server is fixed at 8288, Vite picks the first free
port from 5173.

```bash
# 1. dev server (CGO_ENABLED=0 is required — no gcc on this box for the sqlite path)
cd ~/repo/inngest/inngest
CGO_ENABLED=0 go build -o /tmp/inngest-dev ./cmd
INNGEST_DEV=1 /tmp/inngest-dev dev --no-poll --retry-interval 1 -u http://127.0.0.1:3000/api/inngest

# 2. fixture app (Next.js; serves BOTH SDK versions)
cd ~/repo/inngest/inngest/tests/js && pnpm install && INNGEST_DEV=1 pnpm dev

# 3. UI
cd ~/repo/inngest/inngest/ui/apps/dev-server-ui && pnpm install && pnpm codegen && pnpm dev:vite

# 4. register both apps (the dev server does not always autodiscover the second one)
curl -X PUT http://127.0.0.1:3000/api/inngest       # v3 app: "test-suite"
curl -X PUT http://127.0.0.1:3000/api/inngest-v4    # v4 app: "canvas-v4"
```

**Gotchas that cost real time:**

- `pgrep -f "inngest-dev"` **matches the shell running it** and kills your own session. Use a bracket
  trick: `pgrep -f "inngest-de[v]"`.
- There is no `python3` on this box. Use `node`.
- Playwright's downloaded Chromium **does not run on NixOS**. Use the Nix one:
  `CHROME_BIN=/nix/store/3qgx41z8882ff85y9prdc5zgbb2id6y8-chromium-*/bin/chromium` plus
  `--no-sandbox`. Screenshot helpers live in the job scratchpad, not the repo.
- The dev server uses **in-memory SQLite** unless `--persist`, so there is no DB file to inspect.
  To debug span data, add a temporary `fmt.Printf` gated on an env var and rebuild — see
  lessons.md #1, that advice is correct and repeatedly paid off.
- After changing the SDK, `rm -rf tests/js/.next` and remove/re-add the package as two separate
  commands; a single `pnpm add` over the same path can serve stale content.

### Triggering run shapes

```bash
curl -XPOST http://127.0.0.1:8288/e/test -H 'content-type: application/json' \
  -d '{"name":"tests/v4.chains","data":{}}'
```

- **v3 SDK** (`tests/js/src/inngest/canvas_shapes.ts`): `tests/parallel.test`, `tests/step.test`,
  `tests/retry.test`, `tests/wait.test`, `tests/canvas.{invoke,failure,wait-timeout,loop,wide,chains,slow}`
- **v4 SDK** (`tests/js/src/inngest/canvas_shapes_v4.ts`): `tests/v4.{parallel,chains,sequential,slow,
  unbalanced,nested,race,mixed,sleepbranch,dupes,foreign,dynamic,nestedbranch,pathological,deadend,stress,deep3}`

`tests/v4.pathological` encodes ground truth in step names — a step called `kid<=fork` declares the
parent it *should* have, which makes grading automatic. `tests/v4.stress` takes
`{branches, depth, jitter, ragged}` in event data.

---

## 3. State of the two repos

### `inngest` (this repo) — all uncommitted

**UI, the actual feature** — `ui/packages/components/src/RunDetailsV4/canvas/` (new directory):
`graph.ts` (the pure derivation — read the two long comment blocks before touching it),
`graph.types.ts`, `toFlowElements.ts`, `Canvas.tsx`, `CanvasNode.tsx`, `__fixtures__/` (22 real
captured payloads + README), and three test files (`graph.test.ts`, `gnarly.test.ts`,
`branches.test.ts`).

Modified: `RunDetailsV4.tsx` (canvas section above the trace, `!cloud` gated), `Timeline.tsx`
(reads the shared selection so highlighting is bidirectional), `types.ts` (`plannedSteps`,
`userlandStepID/Index`, `parentStepID`), `package.json` (+`@xyflow/react`).

**Go** — `gql.schema.graphql` + `models/augmented.go` + `loaders/trace.go` expose `plannedSteps`,
`userlandStepID`, `userlandStepIndex`, `parentStepIDs` and `parentAlternateStepIDs`;
`tracing/meta/attributes.go` adds `StepParentIDs` + `StepParentAlternateIDs`; `tracing/util.go`
stamps both in `generatorAttrs`; `state/opcode.go` adds `DiscoveredAfter() []string` and
`DiscoveredAfterAlternates()`. Plus `pkg/execution/executor/parent_step_test.go`.

Both `pkg/coreapi/generated/generated.go` and `pkg/tracing/meta/extracted_values_gen.go` are
generated — after touching the schema or the attrs, run
`go run github.com/99designs/gqlgen --config ./pkg/coreapi/gqlgen.yml` and
`go generate ./pkg/tracing/meta/...`, then `pnpm codegen` in `ui/apps/dev-server-ui`.

`tasks/canvas/planned-steps.patch` is a stale snapshot from an earlier round — the change it holds is
now applied and evolved. Delete it or ignore it; do not re-apply.

### `inngest-js` (`~/repo/inngest/inngest-js`)

On `main` @ `947a244e`. **Local changes were discarded and it was pulled at the user's instruction**,
including someone's unrelated `typescript@7.0.1-rc` pin. Untracked files were left alone.

Three files modified plus two new, no user-facing API, no hashing change:
- `components/execution/stepLineage.ts` — **new**. The registry: one own `then` on the promise handed
  to user code, grouping by synchronous registration block, and the resumed-prefix rule.
- `components/execution/stepLineage.test.ts` — **new**. 10 tests driving real executions.
- `components/execution/engine.ts` — `currentlyResuming` and the registry live on the execution;
  lineage is stamped onto the op's `opts` in `pushStepToReport`; `resumeStepWithResult` marks inline
  resumptions and holds the marker across the microtask drain.
- `components/InngestStepTools.ts` — `FoundStep.discoveredAfter?: string[]` plus alternates and a
  token; `createTool` tags the promise it returns; the exported `step` singleton calls the
  execution's tools directly instead of deriving a promise from `getDeferredStepTooling().then(…)`.

**`tests/js` currently points at a machine-local tarball** —
`"inngest-v4": "file:/home/nixos/repo/inngest/inngest-js/packages/inngest/inngest.tgz"`. Swap it back
to `npm:inngest@4.x` before anyone else runs the fixture app.

Rebuild after editing the SDK:
```bash
cd ~/repo/inngest/inngest-js/packages/inngest && pnpm local:pack
cd ~/repo/inngest/inngest/tests/js && pnpm remove inngest-v4 && rm -rf .next
pnpm add "inngest-v4@file:/home/nixos/repo/inngest/inngest-js/packages/inngest/inngest.tgz"
```

---

## 4. What works

- Graph derivation from real captured payloads; 381 UI tests green.
- Canvas above the trace, collapsible; selection bidirectional with the trace.
- Per-step-type icons, monospace `step.run` / `step.waitForEvent` labels, waits as nodes, duplicate
  step names disambiguated with the SDK's collision index, filled result node, lane alignment so 1:1
  branches draw straight.
- **Parallel grouping is exact on SDK execution v2+** via `plannedSteps` (`response.step.ops`).
  Falls back to inferred execution-overlap on v3 and says so with a `grouping: inferred` badge; the
  canvas starts collapsed in that case.
- **Branch membership** via `discoveredAfter`, now a **set**: chains, fan-outs from inside a branch,
  joins (`Promise.all`, `allSettled`, nested combinators, hand-rolled ones), and races drawn as one
  solid edge plus dashed alternates. Verified end to end against `tests/v4.*` — see round 15 in
  `todo.md`. Still declines a hop when the SDK has not proved it can report a set (`reportsJoins`).

**Verify everything:**
```bash
cd ui/packages/components && pnpm vitest run && npx eslint src/RunDetailsV4
cd ../../apps/dev-server-ui && pnpm type-check
cd ../dashboard && pnpm type-check
cd ~/repo/inngest/inngest && go build ./pkg/... && go test ./pkg/execution/executor/... ./pkg/coreapi/... ./pkg/tracing/...
```

---

## 5. Open, in priority order

Items 1 and 2 of the previous list are **done** — see round 15 in `todo.md` (D44–D49). The
checkpoint path was losing lineage because it serialises `opts` directly, and joins are now drawn.

1. **A join over branch functions under-declares.** `Promise.all([chain("left"), chain("right")])`
   then a step: the combinator is over the branch promises, not over step promises, so there is
   nothing to group and the step falls back to the resume window. It leaves a dangling leaf rather
   than drawing a false edge, and `reportsJoins` keeps the junction fallback for that shape. Every
   theory misses it, theory 16 included — the branch promise is only tied to its step by an adoption
   none of them track.
2. **Loop grouping.** A 40-iteration agent loop is 80 levels, ~21,700px, fitted at zoom 0.03 — a grey
   hairline. Needs repeated sub-shapes collapsed into one node with an iteration count and a stepper.
3. **Discovery spans as selectable nodes.** The junction circle is where discovery actually happens.
   Blocked: `loaders/trace.go` sets `Omit` for `executor.step.discovery`, so they never reach the
   client. `plannedSteps` is already lifted off them; their own timing/status is not.
4. Prediction layer for in-flight runs; animation on poll; batch-event stacking is written but
   untested (no batched run); no **cancelled**-run fixture.

---

## 6. The lineage problem — do not restart from scratch

Twenty theories have been tried and measured. Every prototype is in this directory and re-runnable
with `node tasks/canvas/proto-*.mjs`; the mechanism checks each one rests on are `probe-*.mjs`.
**Read D30–D42 in `todo.md` before proposing anything.**

All theories are now graded against one shared table — `shapes.mjs`, 26 expectations over 20 shapes.
Two of the older expectations were wrong and everything has been re-scored (D36):

- **An edge is the transitive reduction of "must complete before"**, not the full dependency set.
  That is what separates `const pa=step.run(a), pb=step.run(b); await pa; await pb; step.run(c)`
  (`c<=a+b`, a real join) from `await step.run(a); await step.run(b); step.run(c)` (`c<=b`, a chain).
  It needs no new signal — b's parents are already recorded — and applies to every theory equally.
- **`race` is not a join.** `r3` waits for the winner. The group theories can tell, because a race
  continuation advances before every member has been resumed, and an `all` only on the last (D37).
  The losers are worth drawing as **dashed** edges: "could have unblocked this, and did not".

Settled negatives, each confirmed by experiment:
- No promise-level interception carries context across `await` — V8's fast path skips a user `then`,
  ALS through a subclass yields `ROOT`, and `async_hooks` can no longer even map an id to a promise
  (`resource.promise` is gone in Node 22).
- Ordering and timing cannot distinguish independent chains from a join.
- Subset-elimination probing is exact but costs `n+1` replays — rejected on cost.
- Patching the global `Promise` combinators is **disqualified** for a library.
- Static analysis of `fn.toString()` (theory 21) is recorded as considered-and-skipped, not new.

**Current scores.** `node tasks/canvas/proto-lineage-plain.mjs`, `…-hooks.mjs`:

| theory | score | what it costs |
|---|---|---|
| 3 — resume window (shipped) | 15/26 | nothing |
| 14 — Promise **subclass** + handler identity | 20/26 | superseded, do not build this |
| 18 — **plain native promise**, own `then` | 20/26 | one own property; no subclass |
| 19 — own `then` + **synchronous registration batch** | **25/26** | same, and simpler |
| 20 — 19 + barren-resume carryover | 25/26 | misses a different shape than 19 |
| 16 — **`node:v8` promiseHooks** | **26/26** | Node-only, ~5x on promise-heavy bodies |

**The subclass is dead (D38).** `Promise.all` invokes an element's own `then` property, because a
native promise's `constructor` already *is* `Promise`. So theory 14's signal is available from a
plain `new Promise` with one own, non-enumerable `then` — `instanceof`, `constructor`,
`Promise.resolve(p) === p`, the prototype and the TS type all stay natural. Every D28 compatibility
question (ES5 downlevelling, `Symbol.species`, patched realms) simply does not arise. The user's
"nothing but a plain native promise" constraint is satisfiable.

**Theory 19 is built.** See round 15 in `todo.md` (D44–D49) and
`todo-t19.md` for the slice list, file inventory and verification. It reports a
parent *set* end to end, draws races with dashed alternates, and fixed two engine
bugs on the way: lineage was lost on the checkpoint path (the top open item since
round 9, `tests/v4.deadend` now correct) and inline continuations were never
marked as resumptions at all. The exported `step` singleton also handed user code
a derived promise, which fabricated joins and races until it was made direct.

The one shape it misses is a join over branch *functions* rather than step
promises — it under-declares, leaving a dangling leaf. Every theory misses it,
theory 16 included.

**Why 19 rather than the others.** Grouping by *which synchronous
block* a `.then` registration happened in beats grouping by handler identity, and picks up
`allSettled`, nested combinators and third-party joins (`p-map`, Bluebird) that theory 9 had written
off. Skip registrations with no `onFulfilled` (`p.catch(f)`) to avoid the one real false positive.

**Theory 16 is exact and touches nothing** — `step.run` returns a bare promise; the whole graph comes
from `promiseHooks`. It is the only theory that sees plain `await` chains, because V8's await fast
path is invisible to any `.then`. Node-only, process-wide cost, and the edge map must be a
per-request `Map` (a `WeakMap` pins ancestor chains). That profile fits a Dev-Server-only canvas.

Constraints still standing: plain native promises only, and do not lean on `group.parallel()`
(theory 11) since it may be removed.
