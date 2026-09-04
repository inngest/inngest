/**
 * THEORY 5b: combinator spy + two refinements, still zero extra replays.
 *
 *  (a) COMPOSITES. Tag the promise a combinator returns with its member ids, so
 *      a nested `Promise.all([Promise.all([a,b]), c])` expands to {a,b,c}.
 *
 *  (b) AWAIT CHAINS. Record the window in which each step promise's `.then` was
 *      registered. In `await a; await b; step(c)`, b's `.then` is registered
 *      while resuming a, so the continuation depended on a as well. Expanding
 *      transitively turns {b} into {a,b}.
 */
let memo = new Map(), resumers = new Map(), found = [], resuming;
const groups = [];                 // co-awaited step-id sets
const awaitedIn = new Map();       // stepId -> ids of the window it was awaited in
const composite = new WeakMap();   // combinator result promise -> Set<stepId>

class StepPromise extends Promise {
  static get [Symbol.species]() { return Promise; }
  then(onF, onR) {
    // Registering a `.then` during a resume window means the awaiting
    // continuation was itself unblocked by that window.
    if (this.__id && !awaitedIn.has(this.__id)) {
      awaitedIn.set(this.__id, resuming ? currentWindowIds() : []);
    }
    return super.then(onF, onR);
  }
}

const currentWindowIds = () => {
  const g = groups.find((g) => g.ids.has(resuming));
  return g ? [...g.ids] : resuming ? [resuming] : [];
};

/** Members of a promise: a step id, or the ids a combinator result stands for. */
const membersOf = (p) => {
  if (p instanceof StepPromise && p.__id) return [p.__id];
  const c = composite.get(p);
  return c ? [...c] : [];
};

function installCombinatorSpy() {
  const orig = { all: Promise.all, allSettled: Promise.allSettled, race: Promise.race, any: Promise.any };
  for (const name of Object.keys(orig)) {
    Promise[name] = function (items, ...rest) {
      let arr;
      try { arr = Array.isArray(items) ? items : [...items]; } catch { return orig[name].call(this, items, ...rest); }
      const ids = arr.flatMap(membersOf);
      const result = orig[name].call(this, arr, ...rest);
      if (ids.length > 1) {
        const set = new Set(ids);
        groups.push({ kind: name, ids: set });
        composite.set(result, set);      // so an outer combinator can expand it
      } else if (ids.length === 1) {
        composite.set(result, new Set(ids));
      }
      return result;
    };
  }
  return () => Object.assign(Promise, orig);
}

/** {immediate parents} plus everything those were transitively awaited after. */
function expand(ids) {
  const out = new Set(), queue = [...ids];
  while (queue.length) {
    const id = queue.pop();
    if (out.has(id)) continue;
    out.add(id);
    for (const p of awaitedIn.get(id) ?? []) queue.push(p);
  }
  return [...out].sort();
}

function stepRun(id) {
  let p;
  if (memo.has(id)) {
    const v = memo.get(id);
    p = new StepPromise((res) => resumers.set(id, () => res(v)));
  } else {
    found.push({ id, parents: expand(currentWindowIds()) });
    p = new StepPromise(() => {});
  }
  p.__id = id;
  return p;
}

const drain = () => new Promise((r) => setImmediate(r));

async function replay(body, stack) {
  found = []; resumers = new Map(); groups.length = 0; awaitedIn.clear(); resuming = undefined;
  const restore = installCombinatorSpy();
  const done = (async () => body())().catch(() => {});
  await drain();
  for (const id of stack) { resuming = id; resumers.get(id)?.(); await drain(); }
  resuming = undefined;
  await Promise.race([done, drain()]);
  restore();
  return found;
}

const shapes = {
  'independent chains          ': [() => Promise.all([
      (async () => { await stepRun('L1'); await stepRun('L2'); })(),
      (async () => { await stepRun('R1'); await stepRun('R2'); })()]),
    ['L1','R1'], { L2: 'L1', R2: 'R1' }],
  'join: Promise.all([a,b]); c ': [async () => {
      await Promise.all([stepRun('a'), stepRun('b')]); await stepRun('c'); },
    ['a','b'], { c: 'a+b' }],
  'dead end: await a; c; then b': [async () => {
      const pa = stepRun('a'), pb = stepRun('b'); await pa; await stepRun('c'); await pb; },
    ['a','b'], { c: 'a' }],
  'nested fan-out in a branch  ': [() => Promise.all([
      (async () => { await stepRun('F1'); await Promise.all([stepRun('F2a'), stepRun('F2b')]); })(),
      (async () => { await stepRun('P1'); await stepRun('P2'); })()]),
    ['F1','P1'], { F2a: 'F1', F2b: 'F1', P2: 'P1' }],
  'ragged: one branch ends     ': [() => Promise.all([
      (async () => { await stepRun('A1'); await stepRun('A2'); })(),
      (async () => { await stepRun('B1'); })()]),
    ['A1','B1'], { A2: 'A1' }],
  'gap: x,z dep; y idle        ': [async () => {
      const px = stepRun('x'), py = stepRun('y'), pz = stepRun('z');
      await Promise.all([px, pz]); await stepRun('w'); await py; },
    ['x','y','z'], { w: 'x+z' }],
  'three-way join              ': [async () => {
      await Promise.all([stepRun('x'), stepRun('y'), stepRun('z')]); await stepRun('w'); },
    ['x','y','z'], { w: 'x+y+z' }],
  'hand-rolled: a; then b; c   ': [async () => {
      const pa = stepRun('a'), pb = stepRun('b'); await pa; await pb; await stepRun('c'); },
    ['a','b'], { c: 'a+b' }],
  'nested Promise.all          ': [async () => {
      await Promise.all([Promise.all([stepRun('a'), stepRun('b')]), stepRun('c')]); await stepRun('d'); },
    ['a','b','c'], { d: 'a+b+c' }],
  'combinator captured early   ': [async () => {
      const all = Promise.all.bind(Promise);
      await all([stepRun('a'), stepRun('b')]); await stepRun('c'); },
    ['a','b'], { c: 'a+b' }],
  'race                        ': [async () => {
      await Promise.race([stepRun('f'), stepRun('s')]); await stepRun('after'); },
    ['f','s'], { after: 'f+s' }],
  'allSettled                  ': [async () => {
      await Promise.allSettled([stepRun('m'), stepRun('n')]); await stepRun('o'); },
    ['m','n'], { o: 'm+n' }],
};

let pass = 0, fail = 0;
for (const [name, [body, stack, expected]] of Object.entries(shapes)) {
  memo = new Map(stack.map((id) => [id, id]));
  await replay(body, []);
  const res = await replay(body, stack);
  const parts = res.map((r) => {
    const got = r.parents.join('+') || 'ROOT';
    const want = expected[r.id];
    if (want === undefined) return `${r.id}<=${got}`;
    const ok = got === want; ok ? pass++ : fail++;
    return `${r.id}<=${got}${ok ? '' : `(WANT ${want})`}`;
  });
  console.log(`${name} ${parts.join('  ')}`);
}
console.log(`\n${pass} correct, ${fail} wrong   (0 extra replays)`);
