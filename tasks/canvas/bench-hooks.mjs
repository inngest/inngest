/**
 * What does theory 16 actually cost? promiseHooks are process-wide, so the tax
 * lands on every promise in the process, not only on step promises.
 */
import { promiseHooks } from 'node:v8';

const N = 200000;

async function tight() {
  for (let i = 0; i < N; i++) await Promise.resolve(i);
}
// Something closer to a handler body: nested async calls, a combinator, a chain.
async function realistic() {
  const leaf = async (i) => i * 2;
  for (let i = 0; i < N / 20; i++) {
    const [a, b] = await Promise.all([leaf(i), leaf(i + 1)]);
    await leaf(a + b);
  }
}

async function time(label, fn) {
  await fn();                                   // warm
  const t = process.hrtime.bigint();
  await fn();
  const ns = Number(process.hrtime.bigint() - t);
  console.log(`  ${label.padEnd(34)} ${(ns / 1e6).toFixed(1)} ms`);
  return ns;
}

console.log('no hooks installed:');
const baseTight = await time('tight await loop', tight);
const baseReal = await time('nested async + Promise.all', realistic);

const empty = promiseHooks.createHook({ init() {}, before() {}, after() {} });
console.log('\nempty hook installed:');
const emptyTight = await time('tight await loop', tight);
const emptyReal = await time('nested async + Promise.all', realistic);
empty();

// the bookkeeping theory 16 actually does
const info = new Map();
let jobStack = [], batch = [];
const full = promiseHooks.createHook({
  init(p, parent) { info.set(p, { parent, createdIn: jobStack[jobStack.length - 1] }); batch.push(p); },
  before(p) { jobStack.push(p); },
  after() { jobStack.pop(); },
});
console.log('\ntheory 16 bookkeeping installed:');
const fullTight = await time('tight await loop', tight);
const fullReal = await time('nested async + Promise.all', realistic);
full();

const x = (a, b) => `${(a / b).toFixed(1)}x`;
console.log('\nslowdown vs no hooks:');
console.log(`  empty hook      tight ${x(emptyTight, baseTight)}   realistic ${x(emptyReal, baseReal)}`);
console.log(`  theory 16       tight ${x(fullTight, baseTight)}   realistic ${x(fullReal, baseReal)}`);
console.log(`\n  promises retained by the request-scoped map: ${info.size}`);
console.log('  (a per-request Map, dropped when the request ends, is the intended');
console.log('   lifetime — a WeakMap would leak, because each value holds its');
console.log('   parent promise strongly and so pins the whole ancestor chain.)');
