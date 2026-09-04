/**
 * THEORY 14: engine resume window (already shipped, engine-internal) PLUS
 * combinator groups detected by handler identity (theory 13).
 *
 * Nothing global is patched. `step.run` returns a Promise subclass that is
 * adopted as-is by `Promise.resolve`, so a combinator calls `.then` on it
 * directly and every element of one call shares an onRejected identity.
 *
 *   parents(step) = the group containing the resume window, or {window}
 */
let memo = new Map(), resumers = new Map(), found = [], resuming;
const groupByHandler = new Map();   // handler fn -> Set<stepId>
const groupOfStep = new Map();      // stepId -> Set<stepId>

class StepPromise extends Promise {
  static get [Symbol.species]() { return Promise; }
  then(f, r) {
    const key = r ?? f;
    if (this.__id && key) {
      let g = groupByHandler.get(key);
      if (!g) { g = new Set(); groupByHandler.set(key, g); }
      g.add(this.__id);
      if (g.size > 1) for (const m of g) groupOfStep.set(m, g);
    }
    return super.then(f, r);
  }
}
Object.defineProperty(StepPromise.prototype, 'constructor', {
  value: Promise, writable: true, configurable: true,
});

function stepRun(id) {
  let p;
  if (memo.has(id)) {
    const v = memo.get(id);
    p = new StepPromise((res) => resumers.set(id, () => res(v)));
  } else {
    const g = resuming ? groupOfStep.get(resuming) : null;
    found.push({ id, parents: g ? [...g].sort() : resuming ? [resuming] : [] });
    p = new StepPromise(() => {});
  }
  p.__id = id;
  return p;
}

const drain = () => new Promise((r) => setImmediate(r));
async function replay(body, stack) {
  found = []; resumers = new Map(); groupByHandler.clear(); groupOfStep.clear(); resuming = undefined;
  const done = (async () => body())().catch(() => {});
  await drain();
  for (const id of stack) { resuming = id; resumers.get(id)?.(); await drain(); }
  resuming = undefined;
  await Promise.race([done, drain()]);
  return found;
}

const shapes = {
  'independent chains     ': [() => Promise.all([
      (async () => { await stepRun('L1'); await stepRun('L2'); })(),
      (async () => { await stepRun('R1'); await stepRun('R2'); })()]),
    ['L1','R1'], { L2:'L1', R2:'R1' }],
  'join Promise.all([a,b])': [async () => {
      await Promise.all([stepRun('a'), stepRun('b')]); await stepRun('c'); },
    ['a','b'], { c:'a+b' }],
  'dead end               ': [async () => {
      const pa=stepRun('a'), pb=stepRun('b'); await pa; await stepRun('c'); await pb; },
    ['a','b'], { c:'a' }],
  'nested fan-out         ': [() => Promise.all([
      (async () => { await stepRun('F1'); await Promise.all([stepRun('F2a'), stepRun('F2b')]); })(),
      (async () => { await stepRun('P1'); await stepRun('P2'); })()]),
    ['F1','P1'], { F2a:'F1', F2b:'F1', P2:'P1' }],
  'ragged                 ': [() => Promise.all([
      (async () => { await stepRun('A1'); await stepRun('A2'); })(),
      (async () => { await stepRun('B1'); })()]),
    ['A1','B1'], { A2:'A1' }],
  'gap x,z dep; y idle    ': [async () => {
      const px=stepRun('x'), py=stepRun('y'), pz=stepRun('z');
      await Promise.all([px,pz]); await stepRun('w'); await py; },
    ['x','y','z'], { w:'x+z' }],
  'three-way join         ': [async () => {
      await Promise.all([stepRun('x'), stepRun('y'), stepRun('z')]); await stepRun('w'); },
    ['x','y','z'], { w:'x+y+z' }],
  'nested Promise.all     ': [async () => {
      await Promise.all([Promise.all([stepRun('a'), stepRun('b')]), stepRun('c')]);
      await stepRun('d'); },
    ['a','b','c'], { d:'a+b+c' }],
  'hand-rolled a; b; c    ': [async () => {
      const pa=stepRun('a'), pb=stepRun('b'); await pa; await pb; await stepRun('c'); },
    ['a','b'], { c:'a+b' }],
  'allSettled             ': [async () => {
      await Promise.allSettled([stepRun('m'), stepRun('n')]); await stepRun('o'); },
    ['m','n'], { o:'m+n' }],
};

let pass=0, fail=0; const misses=[];
for (const [name,[body,stack,expected]] of Object.entries(shapes)) {
  memo = new Map(stack.map((id)=>[id,id]));
  await replay(body, []);
  const res = await replay(body, stack);
  const parts = res.map((r)=>{
    const got = r.parents.join('+')||'ROOT';
    const want = expected[r.id];
    if (want===undefined) return `${r.id}<=${got}`;
    const ok = got===want; ok?pass++:(fail++,misses.push(`${name.trim()}: ${r.id}<=${got} want ${want}`));
    return `${r.id}<=${got}${ok?'':'*'}`;
  });
  console.log(`${name} ${parts.join('  ')}`);
}
console.log(`\n${pass} correct, ${fail} wrong  (0 extra replays, no globals patched)`);
if (misses.length) console.log(misses.map((m)=>'  '+m).join('\n'));
