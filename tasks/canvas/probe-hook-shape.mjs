/**
 * What is the exact promise-hook edge structure around an `await` chain?
 * Needed to find the stop condition for theory 16's walk.
 */
import { promiseHooks } from 'node:v8';

let n = 0;
const id = new WeakMap();
const info = new WeakMap();
let jobStack = [];
const nm = (p) => (p ? (id.get(p) ?? '?') : '-');

promiseHooks.createHook({
  init(promise, parent) {
    id.set(promise, `p${n++}`);
    info.set(promise, { parent, createdIn: jobStack[jobStack.length - 1] });
  },
  before(p) { jobStack.push(p); },
  after() { jobStack.pop(); },
});

function label(p, name) { id.set(p, name); return p; }
function chain(tag) {
  const out = [];
  let cur = jobStack[jobStack.length - 1];
  const seen = new Set();
  while (cur && !seen.has(cur)) {
    seen.add(cur);
    const m = info.get(cur);
    out.push(`${nm(cur)}[parent=${nm(m?.parent)}]`);
    if (!m) break;
    cur = m.createdIn;
  }
  console.log(`  ${tag.padEnd(18)} job=${nm(jobStack[jobStack.length - 1])}  ${out.join(' <- ')}`);
}

const mk = (name) => label(new Promise((r) => setTimeout(r, 1)), name);

console.log('A. await a; await b; <here>');
{
  const a = mk('A'), b = mk('B');
  await (async () => { await a; await b; chain('after a,b'); })();
}

console.log('\nB. await a; <here>   (b never awaited)');
{
  const a = mk('A2'), b = mk('B2');
  await (async () => { await a; chain('after a'); })();
  await b;
}

console.log('\nC. two branches under Promise.all');
{
  const p = mk('P'), q = mk('Q');
  await Promise.all([
    (async () => { await p; chain('branch0'); })(),
    (async () => { await q; chain('branch1'); })(),
  ]);
}

console.log('\nD. await Promise.all([X,Z]); <here>');
{
  const x = mk('X'), y = mk('Y'), z = mk('Z');
  await (async () => { await Promise.all([x, z]); chain('after all'); })();
  await y;
}
