---
TITLE: Sandbox Opcodes
AUTHOR: Jack Williams
STATUS: Draft
---

# Sandbox Opcodes

## Overview

Make `step.sandbox` operations first-class executor opcodes instead of
ordinary `step.run` calls. The executor, not the user's app, talks to the
Cloud sandbox API and waits for long operations. This removes long-held
executor-to-app requests, gives the executor ownership of idempotency and
cleanup, produces meaningful traces, and is the foundation for opt-in
"durable" sandboxes that rewind machine state on retry.

Scope: Inngest Cloud and the dev server. Self-hosted executors do not
support the opcode; SDKs fall back to today's `step.run` implementation.

## Problem

References are to inngest-js `main` (`packages/inngest/src/components/sandbox/`)
unless noted.

1. **Long executor-to-app requests.** `SandboxMiddleware` wraps every
   operation in `step.run` (`middleware.ts` ~77). The app then calls Cloud
   and blocks:
   - `exec`: up to 5 min (`validation.ts` `maxSandboxProcessTimeoutMs`);
     Cloud `exec` is itself a held-open stream (monorepo
     `api/rest/v2/api_sandbox_exec.go`).
   - create + wait running: default 120 s, max 5 min (`validation.ts`).
   - pause/resume: `pollSandboxStatus` (`client.ts` ~747), 5 min default
     (`lifecycleTimeout`, `client.ts` ~564).
   - snapshot ready: up to 5 min.

   This exceeds typical serverless limits, bills the user's compute while
   idle, and drops connections (observed: a pause held the request ~3m43s
   until it dropped).
2. **Retries double-execute.** No idempotency key is sent. Only sandbox
   create is name-idempotent in Cloud (monorepo `api_sandboxes.go` ~93);
   process start and snapshot create are not. A dropped request is retried
   while the first attempt may still be running in both the app and the
   sandbox. The dev bridge already has to answer `operation_ambiguous`
   (`cmd/internal/cloudsandboxes/cloudsandboxes.go` ~239).
3. **Messy traces.** One logical action becomes several steps (snapshot =
   create + `:wait-until-ready` step, `durable.ts` ~447; process = start +
   wait + output). Spans are generic `step.run` spans containing wire JSON.
4. **Output ceiling.** Step output is capped at 4 MiB, so exec output is
   tail-truncated in the SDK (`durable.ts` ~88).
5. **No cleanup.** Sandbox lifetime is not tied to the run. Failed or
   cancelled runs leave sandboxes running until Cloud's runtime timeout
   pauses them (monorepo `api/fns/sandbox_timeout.go`); paused sandboxes and
   snapshots still hold storage. Only the dev bridge keeps a journal.
6. **Per-SDK cost.** Each SDK must port the REST client, polling and error
   mapping (`client.ts` is ~1.4k lines).

## Proposal

### Opcode

One new opcode, `Sandbox` (`enums.OpcodeSandbox`, SDK `StepOpCode.Sandbox`),
emitted like `Gateway` (inngest-js `InngestStepTools.ts` ~1053):
`mode: Async`, `displayName`, and `opts`:

```json
{
  "protocolVersion": 1,
  "action": "exec",
  "sandbox": { "handle": "sbx_logical_id" },
  "target": {},
  "input": [],
  "durable": false,
  "keepAlive": false
}
```

`action`/`target`/`input` reuse today's wire schema (`protocol.ts`), and the
step result is today's wire result. The durable facade in `durable.ts` stays;
only `rawTool` changes from `step.run` to emitting the opcode. Composite
actions become one opcode: `snapshot` (create + ready), `exec` (start + wait
+ output), `create` (create + running).

### Executor

`handleGeneratorSandbox` is dispatched alongside `OpcodeGateway`
(`pkg/execution/executor/executor.go` ~4114). Two classes of action:

- **Short** (get, list, signal, destroy, snapshot get/list/delete): a single
  Cloud call inside the handler, then `SaveStep`, as
  `handleGeneratorGateway` does (~4953).
- **Long** (create, exec, pause, resume, snapshot): the handler issues the
  start call with an idempotency key, records a pending-op entry, creates a
  pause with the user's timeout (as `handleGeneratorWaitForSignal` does,
  ~5259), and returns. Completion is detected asynchronously (below) and
  resumes the run with the step result. No queue worker or app connection is
  held, so timeouts can be hours instead of 5 min.

`exec` is implemented as process start + completion wait + output fetch, not
the held-open Cloud `exec` stream.

### Async wait

Phase 1 is polling; callbacks come later.

- **Polling (first):** a new queue job kind, `KindSandboxPoll`, carries
  `{runID, stepID, action, sandboxID, processID|snapshotID, deadline}`. Each
  run of it does one cheap Cloud GET. If the op is terminal, it saves the step
  and resumes the pause. Otherwise it re-enqueues with capped backoff
  (1 s growing to 15 s). If the deadline passes, it saves a timeout error.
- **Callbacks (later):** Cloud already emits Inngest events for sandbox
  lifecycle (`compute/sandbox.runtime-started`, monorepo
  `pkg/compute/computeevts`). If Cloud emits `compute/sandbox.status-changed`,
  `compute/sandbox.process-exited` and `compute/sandbox.snapshot-ready`, the
  executor can resume the pause through event matching and cut polling back
  to a slow safety net. Nothing in the executor's pause/step model changes.

### State

- Step output: the same wire result as today. Output larger than the step
  limit is truncated as today in phase 2. Offloading to blob storage is a
  follow-up.
- A per-run sandbox ledger in run state:
  `handle -> {sandboxID, createdByRun, keepAlive, durable, snapshots: [{stepID, snapshotID}]}`.
  It drives idempotency, cleanup and rewind.

### Idempotency

The key is `hash(runID, stepID, attempt)` for mutating calls, and
`hash(runID, stepID)` for create, which is already name-idempotent: the
name is derived from the key when the user gives none. On retry, the
executor first consults the ledger and the pending-op entry. If the prior
attempt's process or snapshot exists, it adopts it instead of starting
another. This needs idempotency-key support on Cloud process start and
snapshot create (open question).

### Auto-cleanup

Default on. When a run ends (completed, failed or cancelled), a lifecycle
listener destroys every ledger sandbox with `createdByRun && !keepAlive`,
and deletes the run's rewind snapshots (not user-requested ones). Opt out per
sandbox with `keepAlive: true`. Sandboxes attached by ID, not created by
the run, are never destroyed.

### Concurrency

Mutating ops on one sandbox run one at a time: parallel steps targeting the
same handle are queued in step order by the executor. Reads are not
serialized.

### Local dev

The dev server executor calls Cloud in-process, reusing the CLI-login token
and journal from `cmd/internal/cloudsandboxes`. Callbacks cannot reach
localhost, so dev always polls.

### Capability and fallback

The SDK emits the opcode only when the executor advertises support (a
request header or feature flag on the execution request). Otherwise it uses
today's `step.run` path. Behaviour is identical apart from durability,
cleanup and timeouts.

### Traces (for N1)

One span per logical action (`sandbox.create`, `sandbox.exec`,
`sandbox.snapshot`, ...) emitted by the executor, with attributes
`sandbox.id`, `sandbox.handle`, `sandbox.action`, `process.id`,
`command`, `exit_code`, `snapshot.id`, `duration`, `timed_out`, and
`rewound_to` for retries. Polling is invisible or appears as child spans.
The executor can set N1's planned `inngest.sandbox` metadata kind directly,
so the SDK needs no trace code.

## Durable sandboxes (opt-in rewind)

`step.sandbox.create(id, { durable: true })`. After each successful
mutating op (exec, process wait, file writes), the executor takes a snapshot
before saving the step, and records `{stepID, snapshotID}` in the ledger.
On retry of step N, or a "rerun from step", the executor:

1. kills any orphaned process from the previous attempt;
2. restores the sandbox to the snapshot recorded for the last completed
   mutating step before N (or to the creation snapshot);
3. re-runs N with a new-attempt idempotency key.

Restore should be in place so the sandbox ID stays stable and earlier
memoized refs remain valid. Cloud resume already restores a snapshot into
the same workload ID (monorepo `pkg/compute/sandbox_lifecycle.go` ~146,
`CloneSnapshot` into the existing workload), so "restore workload to
snapshot X" looks like a generalisation of that path. If Cloud cannot do
this, the ledger maps the logical handle to a new physical ID.

Why this needs the opcode design: under `step.run`, each SDK would snapshot
inside the user's request (added latency), infer "previous snapshot" from
memoized state (ambiguous with parallel steps), and cannot stop an orphaned
earlier attempt. The executor already knows step order, attempts and run end.

Cost: Cloud snapshots appear to be full, not diff. Snapshot-per-command is
only viable with cheap incremental snapshots. Until then, `durable` should
snapshot only on ops that changed the machine, and GC all rewind snapshots
at run end.

Limits: direct `inngest.sandboxes` calls inside a plain `step.run` bypass the
ledger and are outside rewind guarantees. External side effects of commands
(network calls) are not rewound.

## Phased plan

| Phase | Work | Size |
| --- | --- | --- |
| 0: interim, SDK only | Send idempotency keys where Cloud accepts them. Split long waits into short poll steps with `step.sleep` between them. Opt-in `autoDestroy` via run finish/`onFailure`. Lower default timeouts. | ~1 wk |
| 1: opcode, short ops | `OpcodeSandbox` enum and opts parsing (`pkg/enums`, `pkg/execution/state/opcode.go`, `driver_response.go`). Go sandbox client. Sync handler. Tracing plumbing (`pkg/tracing`, `pkg/run/trace*.go`, telemetry, cqrs, coreapi loaders). SDK emitter and capability fallback. Dev server wiring. | ~2 wk |
| 2: async long ops | `KindSandboxPoll`, pause/resume, composite actions, ledger, idempotent adopt-on-retry, auto-cleanup listener, one-at-a-time mutations per sandbox. | ~2 wk |
| 3: callbacks | Consume Cloud status events and reduce polling. | depends on Cloud |
| 4: durable rewind | Snapshot-per-mutation, restore-on-retry, snapshot GC. | ~2 wk + Cloud work |

Main components: executor, queue job kinds, run state (ledger), tracing,
dev server, inngest-js sandbox middleware. Cloud work: idempotency keys,
status events, in-place restore, cheap snapshots.

## Open questions for the sandbox team

1. Can Cloud emit status events (sandbox status changed, process exited,
   snapshot ready), like the existing `compute/sandbox.runtime-started`?
   Until then the executor polls.
2. Can process start and snapshot create accept an idempotency key, and can
   a process be looked up by it?
3. Can a running or paused sandbox be restored in place to an arbitrary
   ready snapshot, keeping its sandbox ID (generalising resume)?
4. Are incremental (diff) snapshots feasible, and at what latency and
   storage cost per snapshot?
5. What is the right internal auth for the executor to call the sandbox
   API on behalf of an account/environment, rather than using the user's
   API key?
6. Rate limits for executor polling: is one GET per second per active op
   acceptable?
