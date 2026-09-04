/**
 * THEORY 22 — theory 19, plus the one signal it was missing, still from a plain
 * native promise and still with nothing global patched.
 *
 * T19's only miss is the hand-rolled join `await pa; await pb; step.run('c')`,
 * because `await` never calls `then`. It never calls it because `PromiseResolve`
 * returns a promise unchanged when `p.constructor === %Promise%` — so the fix is
 * to shadow `constructor` ON THE INSTANCE. `await` then wraps, and wrapping does
 * a `.then` lookup, which is ours.
 *
 * D34 recorded this trade as "combinator groups OR plain awaits, not both",
 * because theory 13 grouped by handler identity and wrapping destroys it. That
 * conclusion was tied to the grouping rule, not to the mechanism: T19 groups by
 * *where and when* the registration happened, which survives wrapping intact.
 *
 * Three rules, all on data we already have:
 *   group    registrations sharing one synchronous block AND one call site
 *   await    `awaitedIn[p]` = the step being resumed when p's `then` was
 *            registered, walked transitively
 *   window   fall back to the step being resumed
 *
 * Cost: one extra microtask hop per awaited step promise (~0.4 µs, measured
 * below) — O(steps), not O(promises). The regression is `Promise.resolve(p)`
 * no longer returning p.
 */
import { grade, report } from './shapes.mjs';

const ROOT = Symbol('root');

// A constructor that is not %Promise%, so PromiseResolve wraps, but whose
// @@species is %Promise%, so `p.then()` still produces a normal promise.
function StepCtor() {}
Object.defineProperty(StepCtor, Symbol.species, { value: Promise });

const realThen = Promise.prototype.then;

let resumers, found, resuming, resumedSoFar, awaitedIn, groupOf;
let batchId, batchScheduled, batchKeys;

function resetRequest() {
  resumers = new Map(); found = []; resuming = undefined; resumedSoFar = new Set();
  awaitedIn = new Map(); groupOf = new Map();
  batchId = 0; batchScheduled = false; batchKeys = new Map();
}
resetRequest();

/** First user frame above the `then` wrapper: `file:line:col`. */
function callSite(boundary) {
  const prev = Error.prepareStackTrace;
  Error.prepareStackTrace = (_, f) => f;
  const e = {};
  Error.captureStackTrace(e, boundary);
  const frames = e.stack;
  Error.prepareStackTrace = prev;
  for (const f of frames) {
    const ln = f.getLineNumber();
    if (ln != null && !f.isNative()) return `${f.getFileName()}:${ln}:${f.getColumnNumber()}`;
  }
  return '?';
}

function observeThen(id, f) {
  // A combinator always passes an onFulfilled; `p.catch(g)` is then(undefined, g).
  if (f === undefined) return;

  // Now that awaits register too, "same tick" alone would fuse two branches that
  // merely started in the same tick. Same tick AND same call site is a combinator.
  if (!batchScheduled) {
    batchScheduled = true;
    queueMicrotask(() => { batchId++; batchScheduled = false; batchKeys = new Map(); });
  }
  const key = `${batchId}@${callSite(observeThen)}`;
  let g = batchKeys.get(key);
  if (!g) { g = new Set(); batchKeys.set(key, g); }
  g.add(id);
  if (g.size > 1) for (const m of g) groupOf.set(m, g);

  // The resume window in force when this promise was awaited. `await pa; await pb`
  // registers pb while a is being resumed, so b's continuation depended on a too.
  if (!awaitedIn.has(id)) awaitedIn.set(id, resuming ?? ROOT);
}

function tag(p, id) {
  Object.defineProperty(p, 'then', {
    value: function (f, r) { observeThen(id, f); return realThen.call(this, f, r); },
    writable: true, configurable: true, enumerable: false,
  });
  Object.defineProperty(p, 'constructor', {
    value: StepCtor, writable: true, configurable: true, enumerable: false,
  });
  return p;
}

function awaitChain(from) {
  const out = [];
  let cur = from, guard = 0;
  while (cur && cur !== ROOT && guard++ < 64) {
    out.push(cur);
    cur = awaitedIn.get(cur);
    if (out.includes(cur)) break;
  }
  return out;
}

function makeStepRun(memoIds) {
  const memo = new Set(memoIds);
  return function stepRun(id) {
    let p;
    if (memo.has(id)) {
      p = new Promise((res) => resumers.set(id, () => res(id)));
    } else {
      let parents = [], alternates = [];
      if (resuming) {
        const g = groupOf.get(resuming);
        if (g) {
          const pending = [...g].filter((m) => !resumedSoFar.has(m));
          if (pending.length) { parents = [resuming]; alternates = pending; }
          else parents = [...g].filter((m) => resumedSoFar.has(m));
        } else {
          parents = awaitChain(resuming);
        }
      }
      found.push({ id, parents, alternates });
      p = new Promise(() => {});
    }
    return tag(p, id);
  };
}

const drain = () => new Promise((r) => setImmediate(r));
async function replay(mk, prefix) {
  resetRequest();
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

report(await grade('T22  plain promise, own `then` + shadowed `constructor`', replay));

console.log('\n--- what user code receives from step.run ---');
resetRequest();
const p = makeStepRun([])('probe');
console.log('  instanceof Promise         ', p instanceof Promise);
console.log('  prototype === Promise.proto', Object.getPrototypeOf(p) === Promise.prototype);
console.log('  toString                   ', Object.prototype.toString.call(p));
console.log('  p.then() is a real Promise ', p.then(() => {}) instanceof Promise);
console.log('  globals patched            ', Promise.prototype.then === realThen ? 'none' : 'YES');
console.log('  Promise.resolve(p) === p   ', Promise.resolve(p) === p, ' <- the one regression');
console.log('  p.constructor              ', p.constructor === Promise ? 'Promise' : 'StepCtor (shadowed)');

const N = 100000;
const bench = async (make) => {
  const t = process.hrtime.bigint();
  for (let i = 0; i < N; i++) { const q = make(); q.__res(i); await q; }
  return Number(process.hrtime.bigint() - t) / N;
};
const plainP = () => { let res; const q = new Promise((r) => { res = r; }); q.__res = res; return q; };
const taggedP = () => { const q = plainP(); return tag(q, 'x'); };
const a = await bench(plainP), b = await bench(taggedP);
console.log(`\ncost per awaited STEP promise: ${a.toFixed(0)} ns plain -> ${b.toFixed(0)} ns tagged` +
  `  (+${(b - a).toFixed(0)} ns, and only on step promises)`);
