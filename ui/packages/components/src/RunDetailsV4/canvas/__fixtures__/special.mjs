#!/usr/bin/env node
/**
 * The two shapes a single event cannot produce.
 *
 *   blocked   two runs contend on `concurrency: { limit: 1 }`; the LOSER is the
 *             fixture, because the queued phase is the whole point of it.
 *   cancelled a run cancelled while a `waitForEvent` is still open, so a step is
 *             left WAITING inside a CANCELLED run.
 */
import { execFileSync } from 'node:child_process';
import { setTimeout as sleep } from 'node:timers/promises';

const DEV = process.env.INNGEST_DEV ?? 'http://localhost:8288';

async function gql(query, variables) {
  const r = await fetch(`${DEV}/v0/gql`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  });
  const b = await r.json();
  if (b.errors?.length) throw new Error(JSON.stringify(b.errors));
  return b.data;
}

const send = async (name) => {
  const r = await fetch(`${DEV}/e/canvas-recapture`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ name, data: {}, ts: Date.now() }),
  });
  return (await r.json()).ids[0];
};

const runsFor = async (id) => {
  const q = `query($id: ID!) { event(query: { eventId: $id }) { functionRuns { id status } } }`;
  for (let i = 0; i < 40; i++) {
    const d = await gql(q, { id }).catch(() => null);
    if (d?.event?.functionRuns?.length) return d.event.functionRuns;
    await sleep(400);
  }
  return [];
};

const statusOf = async (id) => {
  const q = `query($id: ID!) { functionRun(query: { functionRunId: $id }) { status } }`;
  return (await gql(q, { id }).catch(() => null))?.functionRun?.status ?? 'UNKNOWN';
};

const settle = async (id, tries = 90) => {
  for (let i = 0; i < tries; i++) {
    const s = await statusOf(id);
    if (['COMPLETED', 'FAILED', 'CANCELLED'].includes(s)) return s;
    await sleep(700);
  }
  return 'TIMEOUT';
};

// blocked: fire twice, take the run that had to wait.
{
  const first = await send('tests/v4.contended');
  await sleep(150);
  const second = await send('tests/v4.contended');
  const a = (await runsFor(first))[0];
  const b = (await runsFor(second))[0];
  const loser = b ?? a;
  const status = await settle(loser.id);
  execFileSync('node', ['capture.mjs', loser.id, 'blocked'], { stdio: 'pipe' });
  console.error(`blocked   ${status} ${loser.id}`);
}

// cancelled: start it, let the wait open, then cancel it.
{
  const id = await send('tests/v4.cancel');
  const run = (await runsFor(id))[0];
  await sleep(2500);
  await gql(`mutation($id: ULID!) { cancelRun(runID: $id) { id } }`, { id: run.id }).catch(
    async () => {
      // Older dev servers expose this as a REST call instead.
      await fetch(`${DEV}/v1/runs/${run.id}`, { method: 'DELETE' }).catch(() => {});
    }
  );
  const status = await settle(run.id, 30);
  execFileSync('node', ['capture.mjs', run.id, 'cancelled'], { stdio: 'pipe' });
  console.error(`cancelled ${status} ${run.id}`);
}

// waitmatched: start a run that waits, then send the event it is waiting for.
{
  const id = await send('tests/canvas.wait-timeout');
  const run = (await runsFor(id))[0];
  await sleep(900);
  await send('tests/canvas.never');
  const status = await settle(run.id, 30);
  execFileSync('node', ['capture.mjs', run.id, 'waitmatched'], { stdio: 'pipe' });
  console.error(`waitmatched ${status} ${run.id}`);
}
