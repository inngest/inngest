# Canvas fixtures

Real `GetRun` payloads captured from a local Dev Server running against the
`tests/js` fixture app. They are committed rather than hand-written because the
span shape has artifacts no one would invent — see the PARALLELISM note in
`../graph.ts`.

Each file is the raw GraphQL `data` for the same query the UI issues:
`{ run: { status, trace } }`, with the `TraceDetails` fragment and five levels of
`childrenSpans`.

**Two things read this directory.** `index.ts` lists every fixture with a
description, and is what the dev-only fixture gallery in `dev-server-ui` renders
(`/canvas-gallery`). `index.test.ts` fails if a committed fixture is missing from
that list, so adding a `.json` here without adding it there is a test failure
rather than a silence.

## What each fixture is

### Basics

| File                | Run shape                                                         |
| ------------------- | ----------------------------------------------------------------- |
| `simple.json`       | No steps at all                                                   |
| `step.json`         | `step.run` → `step.sleep("2s")` → `step.run`                      |
| `v4sequential.json` | **v4**: sequential with a sleep — no plans, so no false positives |
| `invoke.json`       | `step.invoke` of a child function, between two steps              |
| `child.json`        | The child run `invoke.json` started                               |

### Waiting

| File               | Run shape                                                              |
| ------------------ | ---------------------------------------------------------------------- |
| `wait.json`        | `step.waitForEvent` that timed out                                     |
| `waitmatched.json` | `step.waitForEvent` satisfied by a matching event                      |
| `waittimeout.json` | `step.waitForEvent` with a short timeout, expired                      |
| `blocked.json`     | **v4**: held 6.3s by `concurrency: { limit: 1 }` — a real queued phase |
| `longgap.json`     | **Synthetic.** Seven days asleep — see "The synthetic one" below       |

### Failure

| File                | Run shape                                                         |
| ------------------- | ----------------------------------------------------------------- |
| `retry.json`        | A step that fails once then succeeds                              |
| `noretry.json`      | `NonRetriableError` caught by userland — step fails, run succeeds |
| `failure.json`      | A successful step then a step that fails the run outright         |
| `gnarly-mixed.json` | **v4**: a failing branch beside a succeeding one, caught          |
| `cancelled.json`    | **v4**: cancelled mid-flight, leaving a step `WAITING`            |

### Parallel

| File              | Run shape                                                            |
| ----------------- | -------------------------------------------------------------------- |
| `parallel.json`   | `Promise.all([a, b, c])` then a sequential `d`                       |
| `v4parallel.json` | **v4**: the same shape, carrying `plannedSteps`                      |
| `chains.json`     | `Promise.all([chain, chain])`, each branch 2 steps, then a join step |
| `v4chains.json`   | **v4**: two plans, `[left-1,right-1]` then `[left-2,right-2]`        |
| `inflight.json`   | Captured mid-run: one branch finished, the other still going         |

### Adversarial

Deliberately horrible shapes, from the `tests/v4.*` set. These exist to find out
where the canvas lies. None of them carries reported lineage — they predate it.

| File                           | Run shape                                          |
| ------------------------------ | -------------------------------------------------- |
| `gnarly-unbalanced.json`       | Branches of different depths                       |
| `gnarly-nested.json`           | Nested `Promise.all` — a fan-out of fan-outs       |
| `gnarly-race.json`             | `Promise.race`; every branch schedules a discovery |
| `gnarly-sleep-branch.json`     | A sleep inside one branch, work in the other       |
| `gnarly-dupe-names.json`       | The same step name in both branches                |
| `gnarly-foreign-async.json`    | Steps discovered after non-Inngest async work      |
| `gnarly-dynamic.json`          | Fan-out width decided by a previous step's output  |
| `gnarly-nested-in-branch.json` | A branch that itself fans out                      |

### Reported lineage

Captured after the SDK began reporting which steps each new step waited for, so
these carry `parentStepIDs` and exercise `resolveBranches` rather than the
fallback.

| File                  | Run shape                                               |
| --------------------- | ------------------------------------------------------- |
| `t19-parallel.json`   | Fan-out, with parent sets reported                      |
| `t19-chains.json`     | Parallel chains, every hop stated                       |
| `t19-nested.json`     | Nested combinators                                      |
| `t19-race.json`       | A race: one real dependency, the rest known alternates  |
| `t19-deadend.json`    | Two steps started, one never awaited                    |
| `v4pathological.json` | Ground truth in the step names — `c<=a,b` grades itself |
| `v4deep3.json`        | Three-deep symmetric chains, jittered completion order  |

### Scale

| File                       | Run shape                                          |
| -------------------------- | -------------------------------------------------- |
| `loop.json`                | An 8-iteration agent-shaped loop (16 steps)        |
| `loop40.json`              | A 40-iteration agent-shaped loop (80 steps)        |
| `wide.json`                | 12-wide fan-out then a collect step                |
| `tall500.json`             | **v4**: 500 sequential steps, one per level        |
| `v4branches-ragged.json`   | **v4**: the stress shape with uneven branch depths |
| `v4branches-balanced.json` | **v4**: 48 steps, all with reported parents        |

## The synthetic one

`longgap.json` is the only file here that did **not** come off a real run, and it
says so in `index.ts` so the gallery labels it.

A genuinely long sleep cannot be captured by waiting for it, and hand-writing the
payload would invent span artifacts nobody would think to invent — the exact
failure mode the rest of this directory exists to avoid. So it is generated:

```bash
node make-longgap.mjs
```

That takes the real `step.json` capture and pushes every timestamp at or after
the sleep forward by seven days. Only the clock changes; span shapes and the
durations of the actual work are untouched. The result is ~7 days elapsed and a
fraction of a second executing, which is the case an elastic time axis exists
for.

## The two SDK versions matter

Fixtures without a `v4`/`t19` prefix come from the v3 client (`inngest@3.11.1-pr-411.4`),
whose `PREFERRED_EXECUTION_VERSION` is `ExecutionVersion.V1`. A V1 SDK reports **one opcode
per response**, so `plannedSteps` is always absent and the canvas has to infer parallel
groups from execution overlap.

The v4 fixtures come from `inngest-v4`, which prefers `ExecutionVersion.V2` and reports a
whole batch in one response. Those carry `plannedSteps`, so the grouping is exact. Keep
both: they are the only way to test the two code paths in `graph.ts`.

The v4 functions are in `tests/js/src/inngest/canvas_shapes_v4.ts`, served separately at
`/api/inngest-v4` so the v3 app is untouched.

## Recapturing

```bash
# 1. dev server + fixture app
CGO_ENABLED=0 go build -o /tmp/inngest-dev ./cmd
INNGEST_DEV=1 /tmp/inngest-dev dev --no-poll --retry-interval 1 \
  -u http://127.0.0.1:3000/api/inngest -u http://127.0.0.1:3000/api/inngest-v4
cd tests/js && pnpm install && INNGEST_DEV=1 pnpm dev

# 2. register both apps — autodiscovery does not reliably find the second
curl -X PUT http://127.0.0.1:3000/api/inngest
curl -X PUT http://127.0.0.1:3000/api/inngest-v4

# 3. fire the shape you want
curl -XPOST http://127.0.0.1:8288/e/test -H 'content-type: application/json' \
  -d '{"name":"tests/v4.chains","data":{}}'

# 4. find the run id
curl -s http://127.0.0.1:8288/v0/gql -H 'content-type: application/json' \
  -d '{"query":"{ runs(orderBy:[{field:QUEUED_AT,direction:DESC}],filter:{from:\"2025-01-01T00:00:00Z\"}){edges{node{id status function{slug}}}}}"}'

# 5. capture it
node capture.mjs <runID> <name>
```

`capture.mjs` issues the same `GetRun` query the UI does and writes the
`{ run: { status, trace } }` shape every other fixture is in. Keep its query in
step with `ui/apps/dev-server-ui/src/coreapi.ts` if that one changes.

Then add the new file to `index.ts` with a note saying why it is worth keeping.
`index.test.ts` will fail until you do.

### Capturing a cancelled run

Earlier attempts lost the race — the run finished before the cancel landed.
`tests/v4.cancel` parks on a ten-minute `waitForEvent` so the window is generous:

```bash
curl -XPOST http://127.0.0.1:8288/e/test -H 'content-type: application/json' \
  -d '{"name":"tests/v4.cancel","data":{}}'
# then, with the run id:
curl -s http://127.0.0.1:8288/v0/gql -H 'content-type: application/json' \
  -d '{"query":"mutation($runID: ULID!){ cancelRun(runID:$runID){ id } }","variables":{"runID":"<RUN_ID>"}}'
```

### Capturing a concurrency-blocked run

`tests/v4.contended` is limited to one concurrent run and holds for six seconds.
Send two events at once and capture the **second** run to finish — the one whose
root span shows a large `queuedAt` → `startedAt` gap. Capturing the winner gets
you an ordinary run with nothing to see.
