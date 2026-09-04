/**
 * THEORY 16 — observe the promise graph itself, and touch nothing.
 *
 * `node:v8` promiseHooks hand us promise OBJECTS, not async ids:
 *   init(promise, parent)   parent = the promise this one derives from
 *   before/after(promise)   brackets a reaction job, so we always know whose
 *                           continuation is running
 *
 * From those two edges alone:
 *   - `await p` derives a promise from p, created while the previous await's
 *     job was running. So one async function instance's await chain is a walk
 *     up `createdIn`, and each hop names the promise it awaited. This is the
 *     signal every `.then`-based theory loses to V8's await fast path.
 *   - a combinator derives one promise per element (parent = each element) and
 *     its result capability, all in one synchronous block. So the join group,
 *     and the promise it feeds, are both visible — including hand-rolled ones.
 *   - a step promise that never appears as anyone's `parent` was never awaited.
 *
 * `step.run` returns a bare `new Promise(...)`. No subclass, no own property,
 * no global patched, nothing user-visible whatsoever. Node-only; costs real time.
 */
import { promiseHooks } from 'node:v8';
import { grade, report } from './shapes.mjs';

const info = new WeakMap();     // promise -> { parent, createdIn, gen }
const stepOf = new WeakMap();   // promise -> stepId
const groupOf = new WeakMap();  // capability promise -> Set<stepId>
const awaited = new Set();      // stepIds that something derived from

let jobStack = [], gen = 0, batchInits = [], batchScheduled = false;
let resumers, found, resuming, resumedSoFar;

const bumpBatch = () => {
  if (batchScheduled) return;
  batchScheduled = true;
  queueMicrotask(() => { closeBatch(); batchScheduled = false; });
};

// A combinator creates its result capability first, then derives one promise per
// element. So walk the tick's inits in order: a promise derived from a step (or
// from an already-grouped capability) belongs to the segment above it; anything
// else closes the segment and starts a new one. The segment leader is the
// promise the join feeds into.
function closeBatch() {
  const inits = batchInits; batchInits = [];
  let leader = null, members = new Set();
  const flush = () => {
    if (leader && members.size > 1) groupOf.set(leader, new Set(members));
    leader = null; members = new Set();
  };
  for (const p of inits) {
    const par = info.get(p)?.parent;
    const contributes = !par ? null
      : stepOf.has(par) ? [stepOf.get(par)]
      : groupOf.has(par) ? [...groupOf.get(par)]
      : null;
    if (contributes) for (const m of contributes) members.add(m);
    else { flush(); leader = p; }
  }
  flush();
}

promiseHooks.createHook({
  init(promise, parent) {
    info.set(promise, { parent, createdIn: jobStack[jobStack.length - 1], gen });
    if (parent && stepOf.has(parent)) awaited.add(stepOf.get(parent));
    batchInits.push(promise);
    bumpBatch();
  },
  before(promise) { jobStack.push(promise); },
  after() { jobStack.pop(); },
});

/** Walk back through this async function instance's awaits. */
function lineage() {
  const solid = [], alternates = [];
  let cur = jobStack[jobStack.length - 1];
  const seen = new Set();
  while (cur && !seen.has(cur)) {
    seen.add(cur);
    const m = info.get(cur);
    if (!m || m.gen !== gen) break;          // left this request
    const par = m.parent;
    if (par && stepOf.has(par)) solid.push(stepOf.get(par));
    else if (par && groupOf.has(par)) {
      const g = groupOf.get(par);
      const pending = [...g].filter((s) => !resumedSoFar.has(s));
      if (pending.length) { solid.push(resuming); alternates.push(...pending); }
      else solid.push(...g);
    } else if (solid.length) break;          // left the await chain
    cur = m.createdIn;
  }
  return { solid, alternates };
}

// ------------------------------------------------------------------ harness
function makeStepRun(memoIds) {
  const memo = new Set(memoIds);
  return function stepRun(id) {
    let p;
    if (memo.has(id)) {
      p = new Promise((res) => resumers.set(id, () => res(id)));
    } else {
      const { solid, alternates } = lineage();
      found.push({
        id,
        parents: solid.length ? solid : resuming ? [resuming] : [],
        alternates,
      });
      p = new Promise(() => {});
    }
    stepOf.set(p, id);
    return p;
  };
}

const drain = () => new Promise((r) => setImmediate(r));
async function replay(mk, prefix) {
  gen++; resumers = new Map(); found = []; resuming = undefined;
  resumedSoFar = new Set(); awaited.clear();
  const body = mk(makeStepRun(prefix));
  const done = (async () => body())().catch(() => {});
  await drain();
  for (const id of prefix) {
    resuming = id; resumedSoFar.add(id);
    resumers.get(id)?.();
    await drain();
  }
  resuming = undefined;
  await Promise.race([done, drain()]);
  return found;
}

report(await grade('T16  v8 promiseHooks — nothing touched at all', replay));

// ------------------------------------------------------------------- cost
const N = 300000;
const bench = async () => {
  const t = process.hrtime.bigint();
  for (let i = 0; i < N; i++) await Promise.resolve(i);
  return Number(process.hrtime.bigint() - t) / N;
};
const withHook = await bench();
console.log(`\ncost: ${withHook.toFixed(0)} ns per awaited promise with hooks installed`);
console.log('      (run tasks/canvas/bench-hooks.mjs for the with/without comparison)');
