# Canvas fixtures

Real `GetRun` payloads captured from a local Dev Server running against the
`tests/js` fixture app. They are committed rather than hand-written because the
span shape has artifacts no one would invent — see the PARALLELISM note in
`../graph.ts`.

Each file is the raw GraphQL `data` for the same query the UI issues:
`{ run: { status, trace } }`, with the `TraceDetails` fragment and five levels of
`childrenSpans`.

## What each fixture is

| File                | Run shape                                                                           |
| ------------------- | ----------------------------------------------------------------------------------- |
| `simple.json`       | No steps at all                                                                     |
| `step.json`         | `step.run` → `step.sleep("2s")` → `step.run`                                        |
| `parallel.json`     | `Promise.all([a, b, c])` then a sequential `d`                                      |
| `chains.json`       | `Promise.all([chain, chain])` where each branch is 2 steps, then a join step        |
| `wide.json`         | 12-wide fan-out then a collect step                                                 |
| `retry.json`        | A step that fails once then succeeds                                                |
| `noretry.json`      | `NonRetriableError` caught by userland — step fails, run succeeds                   |
| `failure.json`      | A successful step then a step that fails the run outright                           |
| `wait.json`         | `step.waitForEvent` that timed out                                                  |
| `waitmatched.json`  | `step.waitForEvent` satisfied by a matching event                                   |
| `waittimeout.json`  | `step.waitForEvent` with a short timeout, expired                                   |
| `invoke.json`       | `step.invoke` of a child function, between two steps                                |
| `child.json`        | The child run `invoke.json` started                                                 |
| `loop.json`         | An 8-iteration agent-shaped loop (16 steps)                                         |
| `loop40.json`       | A 40-iteration agent-shaped loop (80 steps)                                         |
| `inflight.json`     | Captured mid-run: one parallel branch finished, the other still going               |
| `v4parallel.json`   | **SDK v4**: `Promise.all([a,b,c])` then `d` — carries `plannedSteps`                |
| `v4chains.json`     | **SDK v4**: parallel chains — two plans, `[left-1,right-1]` then `[left-2,right-2]` |
| `v4sequential.json` | **SDK v4**: sequential with a sleep — no plans, so no false positives               |

Not yet captured: a **cancelled** run. Two attempts raced the run to completion;
worth adding when someone has a reliable way to cancel mid-flight.

## The two SDK versions matter

Fixtures without a `v4` prefix come from the v3 client (`inngest@3.11.1-pr-411.4`), whose
`PREFERRED_EXECUTION_VERSION` is `ExecutionVersion.V1`. A V1 SDK reports **one opcode per
response**, so `plannedSteps` is always absent and the canvas has to infer parallel groups from
execution overlap.

The `v4*` fixtures come from `inngest-v4` (4.3.0), which prefers `ExecutionVersion.V2` and reports
a whole batch in one response. Those carry `plannedSteps`, so the grouping is exact. Keep both:
they are the only way to test the two code paths in `graph.ts`.

The v4 functions are in `tests/js/src/inngest/canvas_shapes_v4.ts`, served separately at
`/api/inngest-v4` so the v3 app is untouched.

## Recapturing

```bash
# 1. dev server + fixture app
go run ./cmd dev --no-discovery -u http://127.0.0.1:3000/api/inngest
cd tests/js && pnpm install && pnpm dev

# 2. fire the shape you want, e.g.
curl -XPOST http://127.0.0.1:8288/e/test -H 'content-type: application/json' \
  -d '{"name":"tests/canvas.chains","data":{}}'

# 3. find the run id
curl -s http://127.0.0.1:8288/v0/gql -H 'content-type: application/json' \
  -d '{"query":"{ runs(orderBy:[{field:QUEUED_AT,direction:DESC}],filter:{from:\"2025-01-01T00:00:00Z\"}){edges{node{id status function{slug}}}}}"}'
```

Then issue the `GetRun` query from `ui/apps/dev-server-ui/src/coreapi.ts` for that
run id and save the `data` object. The canvas-specific run shapes
(`tests/canvas.*`) are defined in `tests/js/src/inngest/canvas_shapes.ts`.
