// Two questions that decide whether co-await grouping is even possible:
//   1. Does Promise.all call .then on thenables in the SAME synchronous turn?
//   2. Does a context established around onFulfilled reach the awaiting code?
let tick = 0;
const bump = () => { const t = tick; queueMicrotask(() => { tick++; }); return t; };

const mk = (id, resolveLater) => ({
  then(onF) {
    console.log(`  .then registered for ${id} at microtask-tick ${tick}`);
    return resolveLater.then((v) => {
      console.log(`  ${id} settling; marker set`);
      globalThis.__marker = id;
      const r = onF?.(v);
      globalThis.__marker = undefined;
      return r;
    });
  },
});

console.log('Q1: registration timing under Promise.all');
let ra, rb;
const pa = mk('a', new Promise((r) => (ra = r)));
const pb = mk('b', new Promise((r) => (rb = r)));
bump();
const all = Promise.all([pa, pb]);
await new Promise((r) => setImmediate(r));

console.log('\nQ2: is the marker visible to the code after the await?');
ra(1); rb(2);
await all;
console.log(`  after await Promise.all, __marker = ${globalThis.__marker}`);

console.log('\nQ3: same for a single await');
let rc; const pc = mk('c', new Promise((r) => (rc = r)));
const t = (async () => { await pc; console.log(`  after await pc, __marker = ${globalThis.__marker}`); })();
rc(3); await t;
