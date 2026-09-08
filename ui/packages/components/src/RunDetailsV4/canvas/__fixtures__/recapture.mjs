#!/usr/bin/env node
/**
 * Re-capture fixtures from a running Dev Server, end to end.
 *
 *   node recapture.mjs                # every mapped fixture
 *   node recapture.mjs v4chains loop40
 *
 * `capture.mjs` takes a run id you already have. This finds one: it sends the
 * event that produces a shape, waits for the run to reach a terminal state, and
 * captures it under the fixture's name.
 *
 * Why it exists: 24 of the 45 committed fixtures predate the server exposing
 * `discoveries`, so they carry none — which is why the trace can say which
 * request planned which steps on some runs and not others. Nothing is wrong with
 * those payloads; they are just old, and this is how they stop being old.
 */
import { execFileSync } from 'node:child_process';
import { setTimeout as sleep } from 'node:timers/promises';

const DEV = process.env.INNGEST_DEV ?? 'http://localhost:8288';

/**
 * fixture -> the event that produces it.
 *
 * Deliberately not exhaustive. Left out on purpose:
 *   longgap, failcluster  synthetic, built by their own make-*.mjs
 *   parallel-inferred     preserved to keep the inference path covered
 *   inflight              a run captured mid-flight, not a terminal one
 *   child                 captured as the run `invoke` starts, below
 */
const SHAPES = {
  simple: 'tests/function.test',
  step: 'tests/step.test',
  invoke: 'tests/canvas.invoke',
  failure: 'tests/canvas.failure',
  retry: 'tests/retry.test',
  noretry: 'tests/no-retry.test',
  wait: 'tests/wait.test',
  waittimeout: 'tests/canvas.wait-timeout',
  parallel: 'tests/parallel.test',
  chains: 'tests/canvas.chains',
  wide: 'tests/canvas.wide',
  loop: 'tests/canvas.loop',
  v4sequential: 'tests/v4.sequential',
  v4parallel: 'tests/v4.parallel',
  v4chains: 'tests/v4.chains',
  v4deep3: 'tests/v4.deep3',
  v4pathological: 'tests/v4.pathological',
  'v4branches-balanced': ['tests/v4.stress', { branches: 6, depth: 4 }],
  'v4branches-ragged': ['tests/v4.stress', { branches: 5, depth: 3, ragged: true }],
  'gnarly-mixed': 'tests/v4.mixed',
  'gnarly-nested': 'tests/v4.nested',
  'gnarly-nested-in-branch': 'tests/v4.nestedbranch',
  'gnarly-sleep-branch': 'tests/v4.sleepbranch',
  'gnarly-dupe-names': 'tests/v4.dupes',
  'gnarly-dynamic': 'tests/v4.dynamic',
  'gnarly-unbalanced': 'tests/v4.unbalanced',
  'gnarly-race': 'tests/v4.race',
  'gnarly-foreign-async': 'tests/v4.foreign',
  't19-race': 'tests/v4.race',
  't19-chains': 'tests/v4.chains',
  't19-parallel': 'tests/v4.parallel',
  't19-nested': 'tests/v4.nested',
  't19-deadend': 'tests/v4.deadend',
  loop40: ['tests/canvas.loop', { iterations: 40 }],
  tall500: ['tests/v4.tall', { steps: 500 }],
  emit: 'tests/v4.emit',

  // The paired shapes: one function, one event, two apps that differ only in
  // whether checkpointing is on. Both runs start from the same send, so the
  // pair has to be told apart by which app produced it.
  'cp-simple': { event: 'tests/pair.simple', fn: 'canvas-cp-' },
  'nocp-simple': { event: 'tests/pair.simple', fn: 'canvas-nocp-' },
  'cp-sequential': { event: 'tests/pair.sequential', fn: 'canvas-cp-' },
  'nocp-sequential': { event: 'tests/pair.sequential', fn: 'canvas-nocp-' },
  'cp-emit': { event: 'tests/pair.emit', fn: 'canvas-cp-' },
  'nocp-emit': { event: 'tests/pair.emit', fn: 'canvas-nocp-' },
  'cp-parallel': { event: 'tests/pair.parallel', fn: 'canvas-cp-' },
  'nocp-parallel': { event: 'tests/pair.parallel', fn: 'canvas-nocp-' },
  'cp-chains': { event: 'tests/pair.chains', fn: 'canvas-cp-' },
  'nocp-chains': { event: 'tests/pair.chains', fn: 'canvas-nocp-' },
  'cp-invoke': { event: 'tests/pair.invoke', fn: 'canvas-cp-' },
  'nocp-invoke': { event: 'tests/pair.invoke', fn: 'canvas-nocp-' },
  'cp-unbalanced': { event: 'tests/pair.unbalanced', fn: 'canvas-cp-' },
  'nocp-unbalanced': { event: 'tests/pair.unbalanced', fn: 'canvas-nocp-' },
  'cp-nested': { event: 'tests/pair.nested', fn: 'canvas-cp-' },
  'nocp-nested': { event: 'tests/pair.nested', fn: 'canvas-nocp-' },
  'cp-nested-in-branch': { event: 'tests/pair.nested-in-branch', fn: 'canvas-cp-' },
  'nocp-nested-in-branch': { event: 'tests/pair.nested-in-branch', fn: 'canvas-nocp-' },
  'cp-race': { event: 'tests/pair.race', fn: 'canvas-cp-' },
  'nocp-race': { event: 'tests/pair.race', fn: 'canvas-nocp-' },
  'cp-mixed': { event: 'tests/pair.mixed', fn: 'canvas-cp-' },
  'nocp-mixed': { event: 'tests/pair.mixed', fn: 'canvas-nocp-' },
  'cp-sleep-in-branch': { event: 'tests/pair.sleep-in-branch', fn: 'canvas-cp-' },
  'nocp-sleep-in-branch': { event: 'tests/pair.sleep-in-branch', fn: 'canvas-nocp-' },
  'cp-dupe-names': { event: 'tests/pair.dupe-names', fn: 'canvas-cp-' },
  'nocp-dupe-names': { event: 'tests/pair.dupe-names', fn: 'canvas-nocp-' },
  'cp-foreign-async': { event: 'tests/pair.foreign-async', fn: 'canvas-cp-' },
  'nocp-foreign-async': { event: 'tests/pair.foreign-async', fn: 'canvas-nocp-' },
  'cp-dynamic': { event: 'tests/pair.dynamic', fn: 'canvas-cp-' },
  'nocp-dynamic': { event: 'tests/pair.dynamic', fn: 'canvas-nocp-' },
  'cp-deep3': { event: 'tests/pair.deep3', fn: 'canvas-cp-' },
  'nocp-deep3': { event: 'tests/pair.deep3', fn: 'canvas-nocp-' },
  'cp-dead-end': { event: 'tests/pair.dead-end', fn: 'canvas-cp-' },
  'nocp-dead-end': { event: 'tests/pair.dead-end', fn: 'canvas-nocp-' },
  'cp-loop': { event: 'tests/pair.loop', fn: 'canvas-cp-' },
  'nocp-loop': { event: 'tests/pair.loop', fn: 'canvas-nocp-' },
  'cp-wide': { event: 'tests/pair.wide', fn: 'canvas-cp-' },
  'nocp-wide': { event: 'tests/pair.wide', fn: 'canvas-nocp-' },
  'cp-wait-timeout': { event: 'tests/pair.wait-timeout', fn: 'canvas-cp-' },
  'nocp-wait-timeout': { event: 'tests/pair.wait-timeout', fn: 'canvas-nocp-' },
  blocked: 'tests/v4.contended',
  cancelled: 'tests/v4.cancel',
};

async function gql(query, variables) {
  const res = await fetch(`${DEV}/v0/gql`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  });
  const body = await res.json();
  if (body.errors?.length) throw new Error(JSON.stringify(body.errors));
  return body.data;
}

async function send(spec) {
  const [name, data] = Array.isArray(spec) ? spec : [spec, {}];
  const res = await fetch(`${DEV}/e/canvas-recapture`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ name, data, ts: Date.now() }),
  });
  const body = await res.json();
  const id = body.ids?.[0];
  if (!id) throw new Error(`no event id for ${name}: ${JSON.stringify(body)}`);
  return id;
}

/** The runs an event triggered, once the executor has picked it up. */
async function runsFor(eventID, { tries = 40, waitMs = 500 } = {}) {
  const q = `query($id: ID!) { event(query: { eventId: $id }) { functionRuns { id status function { slug } } } }`;
  for (let i = 0; i < tries; i++) {
    const data = await gql(q, { id: eventID }).catch(() => null);
    const runs = data?.event?.functionRuns ?? [];
    if (runs.length) return runs;
    await sleep(waitMs);
  }
  return [];
}

const TERMINAL = new Set(['COMPLETED', 'FAILED', 'CANCELLED']);

async function settle(runID, { tries = 90, waitMs = 700 } = {}) {
  const q = `query($id: ID!) { functionRun(query: { functionRunId: $id }) { status } }`;
  let last = 'UNKNOWN';
  for (let i = 0; i < tries; i++) {
    const data = await gql(q, { id: runID }).catch(() => null);
    last = data?.functionRun?.status ?? last;
    if (TERMINAL.has(last)) return last;
    await sleep(waitMs);
  }
  return last;
}

const wanted = process.argv.slice(2);
const targets = Object.entries(SHAPES).filter(([id]) => !wanted.length || wanted.includes(id));

const done = [];
const failed = [];

for (const [fixture, spec] of targets) {
  try {
    // A shape is an event, optionally with data, optionally naming which app's
    // run to keep -- one send starts a run in every app subscribed to it.
    const s =
      typeof spec === 'string'
        ? { event: spec }
        : Array.isArray(spec)
        ? { event: spec[0], data: spec[1] }
        : spec;
    const eventID = await send(s.data ? [s.event, s.data] : s.event);
    const runs = await runsFor(eventID);
    if (!runs.length) throw new Error('no run started');

    const pick = s.fn ? runs.find((r) => (r.function?.slug ?? '').startsWith(s.fn)) : runs[0];
    if (!pick) throw new Error(`no run from an app matching ${s.fn}`);
    const runID = pick.id;
    const status = await settle(runID);
    execFileSync('node', ['capture.mjs', runID, fixture], { stdio: 'pipe' });
    done.push(`${fixture.padEnd(24)} ${status.padEnd(10)} ${runID}`);
    console.error(`ok   ${fixture}`);
  } catch (err) {
    failed.push(`${fixture.padEnd(24)} ${String(err.message).slice(0, 120)}`);
    console.error(`FAIL ${fixture}: ${String(err.message).slice(0, 120)}`);
  }
}

console.log(`\n=== captured ${done.length} ===\n${done.join('\n')}`);
if (failed.length) console.log(`\n=== failed ${failed.length} ===\n${failed.join('\n')}`);
