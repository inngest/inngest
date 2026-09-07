---
title: Understanding traces
description: Read a function run's trace — what every bar and circle means, and what each one tells you to do next.
---

Every function run produces a trace: one row per thing that happened, on one
time axis. It shows what your code did, what Inngest did around it, and where
the time actually went.

Two rules cover almost all of it.

## The two rules

**Bars are intervals. Colour says what kind of work, fill says whether your code
was running.**

| | |
|---|---|
| **Solid** | Your function was executing. This is the part you pay for. |
| **Hatched** | Nothing of yours was running — queued, sleeping, waiting, backing off. |

**Circles are moments. Hollow means unresolved, filled means resolved.**

Only a resolution circle is ever green or red. Everything before it is neutral,
so the outcome of a row is always the last thing on it.

That is the whole encoding. A row you can read at a glance:

```
○────────  ●━━━━━━━━━━  ●
queued     started      ok
```

## Reading a row

A step's row runs left to right through its own life:

1. **`queued`** — grey hollow circle. Enqueued, waiting for a worker.
2. **A hatched bar** — the wait. Costs nothing.
3. **`started`** — white hollow circle. Your code began.
4. **A solid bar** — your code running.
5. **`ok`, `failed` or `timeout`** — filled. The row resolved.

A row with no final circle has not resolved. That missing circle *is* the
signal — a step still running is drawn blue with nothing closing it.

## Requests, and why some rows start blue

Inngest does not know your whole function up front. It calls your app, your code
runs until it hits something it has to hand back, and the SDK reports what comes
next. That call is a **discovery request**, and it is drawn in blue.

A discovery request is a request like any other, so it has the same lifecycle:
queued, then a wait, then it starts, then it runs.

What happens at the end of it is the interesting part.

**Sequential code — the request finds one step and runs it.** No handoff, no
second call. The blue turns green when the step returns, and the row ends on one
`ok`.

```ts
await step.run('a', …)
await step.run('b', …)
```

**Parallel code — the request finds several steps.** It cannot run them all
here, so it reports them and stops. The next circle is **`planned`**, the only
blue hollow circle in the trace, and a ribbon drops from it to every step that
request reported.

```ts
await Promise.all([
  step.run('a', …),
  step.run('b', …),
  step.run('c', …),
])
```

Each of those steps then starts its own cycle: queued, wait, started, run.

### The ribbon

The ribbon is the vertical line threading the `planned` circles of steps that
came from one request. It is drawn at rest, not on hover, so:

- **Count the ribbon** to see how wide a fan-out was, without clicking anything.
- **Steps under one ribbon have no order between them.** They were reported
  together.
- **Two ribbons means two requests.** Two branches that each scheduled their own
  work do not share one.

Ribbons nest, so depth reads without indentation.

## Waiting

`step.sleep()` and `step.waitForEvent()` are hatched, always. Your code is not
running and you are not billed for the wait.

| | |
|---|---|
| **Blue hatched** | Still open. Undecided. |
| **Green circle** | The sleep elapsed, or the event arrived. |
| **Grey circle** | The wait expired with no match. |

**A timeout is not a failure.** It is a result your function can act on, so the
row goes grey and the run carries on.

## Failure and recovery

- **One red step in a green run** means your code caught the error. The step
  failed; the run did not.
- **A retry** shows every attempt on one row — red attempt, neutral gap, green
  attempt. The gap between attempts is backoff: suspended, costing nothing, and
  drawn neutral because nothing happened in it.
- **A filled red circle** appears only when every attempt failed.
- **Failure never spreads.** A red row does not tint its neighbours.

**Cancelled is its own state.** A step running when a run was cancelled is
neither success nor failure, so it ends in a square rather than a circle.

## Long runs

A run can be nine days elapsed and five seconds executing. The axis compresses
the idle stretches and marks each one with a break.

**Only the axis compresses. Every duration stays wall clock.** A step that took
890ms says 890ms whatever the axis is doing.

A step too short to draw still gets drawn, at a minimum width, with its real
duration beside it. The drawing rounds; the number does not.

## Big runs

At a few hundred steps, a full list stops being readable. Repeated steps that
share a shape collapse into one row that reports the count and draws where each
member ran.

A collapsed group tells you what varied:

> `38 × search, 2 × write_file`

That is usually what you opened the trace to find. Iteration 7 calling a
different tool is the interesting one, and collapsing by shape is what surfaces
it. Expand any group in place.

## The minimap

The strip above the run is the same trace, in the same order and the same
colours — one hairline per row. A cluster of failures two thirds through a long
run is a red smear you can see without scrolling.

Drag it to set the viewport. Click a cluster to jump to it.

## What the trace will not tell you

Worth knowing, so you do not go looking:

- **Why a branch was not taken.** A run records what happened, not what didn't.
- **What caused a request, when nothing reported it.** Where the trace cannot
  say, it declines rather than guessing — you will see a note, not a line.
- **Anything outside a step.** A bare `inngest.send()` outside `step.sendEvent()`
  has no span, so nothing ties it to what it triggered.

Where the view is inferring rather than reading a reported fact, it says so.
Inferred grouping is drawn dashed, and it never looks like something that was
reported.
