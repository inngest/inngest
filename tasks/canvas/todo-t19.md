# Implementing theory 19 — plan

Goal: replace the single `discoveredAfter` with a **set** of parents derived from
combinator grouping, so joins are drawn instead of declined. 25/26 on the shape
table; the one miss (`await pa; await pb; step(c)`) under-declares, leaving a
dangling leaf, which is what ships today anyway.

## Design decisions

- **Tag the promise the user actually holds.** `createTool` returns an async
  wrapper, and `stepHandler` is itself async, so the engine's deferred promise
  never reaches user code. Tag `createTool`'s returned promise instead.
- **Keep the async wrapper.** `(async () => stepHandler(...))()` gives us the
  promise object to tag while preserving throw semantics and microtask count
  exactly. Dropping `async` would change both.
- **The step id is not known synchronously** — `hashedId` comes from
  `await applyMiddlewareToStep(...)`. So groups are keyed by a per-call **token**
  object, and tokens are resolved to hashed ids later, at discovery time, when
  they are known. Registration order is unaffected.
- **Group = one synchronous block.** `await` never calls `then`, so essentially
  only combinators do. Skip registrations with no `onFulfilled` (`p.catch(f)` is
  `then(undefined, f)`) — the one common false positive.
- **Resumed-prefix rule.** Parents are the group members already resumed. If any
  member is still unresumed the group is disjunctive (`race`/`any`): the step
  being resumed is the real parent and the rest are *alternates*, drawn dashed.
- **Wire format becomes plural.** `opts.discoveredAfter: string[]` plus
  `opts.discoveredAfterAlternates?: string[]`. Nothing is released, so no
  back-compat shim; the Go side reads both shapes for safety.

## Slices

- [x] S1 SDK: lineage registry, promise tagging, plural `discoveredAfter`
- [x] S1t SDK tests: one per shape that changes behaviour
- [x] S2 Go: plural accessor, `step.parent_ids` + alternates attrs, GraphQL
- [x] S2t Go tests
- [x] S3 UI: types, `graph.ts` multi-parent edges, dashed alternates
- [x] S3t UI tests against captured fixtures
- [x] S4 end-to-end against the running stack with `tests/v4.*` shapes

## Acceptance

- `tests/v4.chains`, `.parallel`, `.nested`, `.race` draw the joins they should
- no false edges on any existing fixture (regressions in `graph.test.ts` etc.)
- older SDKs still render, with the existing `grouping: inferred` degradation

## Results

Built and verified. All slices done.

**inngest-js** (`~/repo/inngest/inngest-js`, uncommitted on `main`):
- `components/execution/stepLineage.ts` — new; the registry and the own-`then` tag
- `components/execution/stepLineage.test.ts` — new; 10 tests driving real executions
- `components/execution/engine.ts` — lineage stamped onto `opts` at discovery;
  `resumeStepWithResult` marks inline resumptions and holds the marker across
  the microtask drain
- `components/InngestStepTools.ts` — tags the promise handed to user code;
  the exported `step` singleton calls the execution's tools directly

**inngest** (this repo, uncommitted):
- `pkg/execution/state/opcode.go` — `DiscoveredAfter() []string` +
  `DiscoveredAfterAlternates()`, tolerant of a bare string from an older SDK
- `pkg/tracing/meta/attributes.go` + regenerated `extracted_values_gen.go` —
  `step.parent_ids`, `step.parent_alternate_ids`
- `pkg/tracing/util.go` — stamps both
- `pkg/coreapi/gql.schema.graphql`, `graph/models/augmented.go`,
  `graph/loaders/trace.go`, regenerated `generated.go` — `parentStepIDs`,
  `parentAlternateStepIDs`
- `pkg/execution/executor/parent_step_test.go` — 10 subtests
- `ui/.../canvas/graph.ts` — parent *sets*, dashed alternates, `reportsJoins`
- `ui/.../canvas/joins.test.ts` — new; 4 tests
- `ui/.../canvas/__fixtures__/*.json` — 32 files migrated to `parentStepIDs`
- `ui/apps/dev-server-ui/src/coreapi.ts` + regenerated store types

**Verification**
- SDK: 4129 tests pass (81 files), typecheck clean
- Go: `go build ./pkg/...`, tests green across executor/state/tracing/coreapi
- UI: 385 tests pass, eslint clean, dev-server-ui and dashboard typecheck clean
- End to end against the running stack: see the table in `todo.md` round 15

**Known limitation**: a join built over branch *functions* rather than step
promises (`Promise.all([chain("left"), chain("right")])`) under-declares — see
D49. It leaves a dangling leaf rather than drawing a false edge, and the UI's
`reportsJoins` gate keeps the junction fallback for exactly that case.
