/**
 * THEORIES 15, 18, 19, 20, plus theory 14 re-scored against the corrected
 * ground truth in shapes.mjs (transitive reduction; race is not a join).
 *
 * The promise `step.run` returns in modes T18/T19/T20 is a PLAIN NATIVE PROMISE.
 * No subclass. `p instanceof Promise`, `p.constructor === Promise`,
 * `Promise.resolve(p) === p` and the toString tag are all natural, not simulated.
 * The only change is one own, non-enumerable `then` property on the instance.
 *
 *  T3   window    resume window only (what ships today)
 *  T14  subclass  Promise subclass + onRejected identity (the old best option)
 *  T18  handler   same signal, plain promise, own `then` instead of a subclass
 *  T19  batch     group = `then` registrations in one synchronous block
 *  T20  carry     T19 + barren-resume carryover, filtered by T15's branch path
 *
 * All modes use the resumed-prefix rule: a group contributes only the members
 * resumed so far, which is what makes race/any come out as "the winner" and
 * all/allSettled as "everyone", with no combinator identity needed.
 */
import { grade, report } from './shapes.mjs';

Error.stackTraceLimit = 60;
const MODES = ['window', 'subclass', 'handler', 'batch', 'carry-nopath', 'carry'];
let MODE;

// ------------------------------------------------------- T15: branch path
// V8 exposes combinator frames structurally; no string formatting needed.
function branchPath() {
  const prev = Error.prepareStackTrace;
  Error.prepareStackTrace = (_, frames) => frames;
  const e = {};
  Error.captureStackTrace(e, branchPath);
  const frames = e.stack;
  Error.prepareStackTrace = prev;
  const out = [];
  for (const f of frames) {
    if (typeof f.isPromiseAll === 'function' && f.isPromiseAll()) out.push(`all#${f.getPromiseIndex()}`);
    else if (typeof f.isPromiseAny === 'function' && f.isPromiseAny()) out.push(`any#${f.getPromiseIndex()}`);
  }
  return out.reverse().join('/');
}

// ------------------------------------------------------------------ state
let resumers, found, resuming, barren, resumedSoFar, stepCalls, pathOf;
let handlerGroup, batchGroup, groupOf, batchId, batchScheduled;

function resetRequest() {
  resumers = new Map(); found = []; resuming = undefined; barren = [];
  resumedSoFar = new Set(); stepCalls = 0; pathOf = new Map();
  handlerGroup = new Map(); batchGroup = new Map(); groupOf = new Map();
  batchId = 0; batchScheduled = false;
}
resetRequest();

function noteGroup(map, key, id) {
  let g = map.get(key);
  if (!g) { g = new Set(); map.set(key, g); }
  g.add(id);
  if (g.size > 1) for (const m of g) groupOf.set(m, g);
}

function observeThen(id, f, r) {
  if (MODE === 'handler' || MODE === 'subclass') { if (r ?? f) noteGroup(handlerGroup, r ?? f, id); }
  if (MODE === 'batch' || MODE === 'carry' || MODE === 'carry-nopath') {
    // A combinator always passes an onFulfilled. `p.catch(f)` is then(undefined, f),
    // so skipping those removes the commonest false grouping at no cost.
    if (f === undefined) return;
    if (!batchScheduled) {
      batchScheduled = true;
      queueMicrotask(() => { batchId++; batchScheduled = false; });
    }
    noteGroup(batchGroup, batchId, id);
  }
}

// --- T14: the subclass, kept verbatim so the comparison is like for like
const realThen = Promise.prototype.then;
class StepPromise extends Promise {
  static get [Symbol.species]() { return Promise; }
  then(f, r) { if (this.__id) observeThen(this.__id, f, r); return super.then(f, r); }
}
Object.defineProperty(StepPromise.prototype, 'constructor',
  { value: Promise, writable: true, configurable: true });

// --- T18: the same signal from a plain native promise
function tagPlain(p, id) {
  Object.defineProperty(p, 'then', {
    value: function (f, r) { observeThen(id, f, r); return realThen.call(this, f, r); },
    writable: true, configurable: true, enumerable: false,
  });
  return p;
}

const makeStepRun = (memoIds) => {
  const memo = new Set(memoIds);
  return function stepRun(id) {
    stepCalls++;
    const path = branchPath();
    pathOf.set(id, path);
    const Ctor = MODE === 'subclass' ? StepPromise : Promise;
    let p;
    if (memo.has(id)) {
      p = new Ctor((res) => resumers.set(id, () => res(id)));
    } else {
      let parents = [], alternates = [];
      if (resuming) {
        const g = groupOf.get(resuming);
        if (g) {
          const pending = [...g].filter((m) => !resumedSoFar.has(m));
          if (pending.length) {
            // The continuation advanced before every member of the group had
            // completed => the group is disjunctive (race / any). One member
            // actually unblocked it; the rest could have, and did not.
            parents = [resuming];
            alternates = pending;
          } else {
            // Advanced only once the last member landed => conjunctive
            // (all / allSettled / a hand-rolled counter).
            parents = [...g].filter((m) => resumedSoFar.has(m));
          }
        } else {
          parents = [resuming];
          if (MODE === 'carry') {
            for (const b of barren) if (pathOf.get(b) === path) parents.push(b);
          } else if (MODE === 'carry-nopath') {
            for (const b of barren) parents.push(b);
          }
        }
      }
      found.push({ id, parents, alternates });
      p = new Ctor(() => {});
    }
    if (MODE === 'subclass') p.__id = id; else tagPlain(p, id);
    return p;
  };
};

const drain = () => new Promise((r) => setImmediate(r));
async function replay(mk, prefix) {
  resetRequest();
  const body = mk(makeStepRun(prefix));
  const done = (async () => body())().catch(() => {});
  await drain();
  for (const id of prefix) {
    const before = stepCalls;
    resuming = id;
    resumedSoFar.add(id);
    resumers.get(id)?.();
    await drain();
    // "barren" = the resume moved nothing forward at all, not merely
    // "discovered nothing" — a memoised next step still counts as progress.
    if (stepCalls === before) barren.push(id); else barren = [];
  }
  resuming = undefined;
  await Promise.race([done, drain()]);
  return found;
}

const LABEL = {
  window:  'T3   resume window only                      (what ships today)',
  subclass:'T14  Promise SUBCLASS + handler identity      (previous best; re-scored)',
  handler: 'T18  plain promise, own `then` + handler id   (T14 without the subclass)',
  batch:   'T19  plain promise, own `then` + sync batch   (grouping by registration block)',
  'carry-nopath': 'T19+carryover, NO branch-path filter        (shows what T15 is worth)',
  carry:   'T20  T19 + barren carryover, path-filtered    (adds hand-rolled joins)',
};

const results = [];
for (const mode of MODES) {
  MODE = mode;
  results.push(await grade(LABEL[mode], replay));
}
for (const r of results) report(r);

console.log('\n--- what user code receives from step.run ---');
MODE = 'carry'; resetRequest();
const p = makeStepRun([])('probe');
console.log('  instanceof Promise         ', p instanceof Promise);
console.log('  constructor === Promise    ', p.constructor === Promise);
console.log('  Promise.resolve(p) === p   ', Promise.resolve(p) === p);
console.log('  prototype === Promise.proto', Object.getPrototypeOf(p) === Promise.prototype);
console.log('  toString                   ', Object.prototype.toString.call(p));
console.log('  own properties             ', Object.getOwnPropertyNames(p).join(','));
console.log('  globals patched            ', Promise.prototype.then === realThen ? 'none' : 'YES');
console.log('\nscores:', results.map((r) => `${r.label.slice(0, 4).trim()} ${r.pass}/${r.pass + r.fail}`).join('   '));
