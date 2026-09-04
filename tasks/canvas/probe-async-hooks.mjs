/**
 * PROBE for theory 16: what edges does node:async_hooks actually give us for
 * promises, and can `await` chains be reconstructed from them?
 *
 * Two edges per promise are recorded:
 *   trigger   = triggerAsyncId  (V8's "derived from" link, per spec)
 *   createdIn = executionAsyncId() captured inside init (who built it)
 */
import { createHook, executionAsyncId } from 'node:async_hooks';
import { writeSync } from 'node:fs';

const meta = new Map();
const idOf = new WeakMap();
let resourceHasPromise = null;

createHook({
  init(asyncId, type, triggerAsyncId, resource) {
    if (type !== 'PROMISE') return;
    meta.set(asyncId, { trigger: triggerAsyncId, createdIn: executionAsyncId() });
    if (resourceHasPromise === null) resourceHasPromise = !!(resource && resource.promise);
    if (resource && resource.promise) idOf.set(resource.promise, asyncId);
  },
}).enable();

const out = (s) => writeSync(1, s + '\n');

const named = new Map();  // asyncId -> label
function label(p, name) { const id = idOf.get(p); if (id) named.set(id, name); return p; }
function chain(from) {
  const path = []; const seen = new Set(); let cur = from;
  while (cur && !seen.has(cur)) {
    seen.add(cur);
    const m = meta.get(cur);
    const t = m ? m.trigger : null;
    path.push(`${named.get(cur) ?? cur}${t != null ? `~>${named.get(t) ?? t}` : ''}`);
    if (!m) break;
    cur = m.createdIn;
    if (path.length > 8) break;
  }
  return path.join('  <<  ');
}

out(`resource.promise available: ${'pending'}`);

// A. sequential awaits inside one async function
{
  const a = label(new Promise((r) => setTimeout(r, 1)), 'A');
  const b = label(new Promise((r) => setTimeout(r, 2)), 'B');
  await (async function seqBody() {
    await a;
    await b;
    out(`\nA. after \`await a; await b\`  exec=${executionAsyncId()}`);
    out(`   chain: ${chain(executionAsyncId())}`);
  })();
}

// B. one await only
{
  const a = label(new Promise((r) => setTimeout(r, 1)), 'A2');
  label(new Promise((r) => setTimeout(r, 2)), 'B2');
  await (async function oneBody() {
    await a;
    out(`\nB. after \`await a\` only`);
    out(`   chain: ${chain(executionAsyncId())}`);
  })();
}

// C. await Promise.all([x, z]) — can the elements be recovered?
{
  const x = label(new Promise((r) => setTimeout(r, 1)), 'X');
  const y = label(new Promise((r) => setTimeout(r, 1)), 'Y');
  const z = label(new Promise((r) => setTimeout(r, 1)), 'Z');
  const initsDuringAll = [];
  const before = new Set(meta.keys());
  const all = Promise.all([x, z]);
  for (const [id, m] of meta) {
    if (before.has(id)) continue;
    initsDuringAll.push(`${id}(trig=${named.get(m.trigger) ?? m.trigger},in=${m.createdIn})`);
  }
  label(all, 'ALL');
  out(`\nC. promises inited by Promise.all([X,Z]): ${initsDuringAll.join(' ')}`);
  await (async function allBody() {
    await all;
    out(`   chain after await all: ${chain(executionAsyncId())}`);
  })();
  await y;
}

// D. two independent async branches
{
  const p = label(new Promise((r) => setTimeout(r, 1)), 'P');
  const q = label(new Promise((r) => setTimeout(r, 1)), 'Q');
  await Promise.all([
    (async function br0() { await p; out(`\nD. branch0 chain: ${chain(executionAsyncId())}`); })(),
    (async function br1() { await q; out(`   branch1 chain: ${chain(executionAsyncId())}`); })(),
  ]);
}

out(`\nresource.promise present in init: ${resourceHasPromise}`);
out(`promises tracked: ${meta.size}`);

// E. cost
{
  const N = 200000;
  let t = process.hrtime.bigint();
  for (let i = 0; i < N; i++) await Promise.resolve(i);
  const withHook = Number(process.hrtime.bigint() - t) / N;
  out(`\nE. ${withHook.toFixed(1)} ns per awaited promise WITH the hook enabled`);
}
