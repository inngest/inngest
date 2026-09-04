/**
 * EXACT LINEAGE BY ELIMINATION.
 *
 * A step c appears in a replay iff every step it awaits has been resumed. So
 * "resume everything except m" is a membership test:
 *
 *     c depends on m   <=>   c does NOT appear when all but m are resumed
 *
 * One replay per candidate parent, and each replay answers the question for
 * EVERY step at once. No promise interception, no ordering assumptions, no
 * timing. Exact by construction.
 */
let memo = new Map(), resumers = new Map(), found = [];

function stepRun(id) {
  if (memo.has(id)) {
    const v = memo.get(id);
    return new Promise((res) => resumers.set(id, () => res(v)));
  }
  found.push(id);
  return new Promise(() => {});
}

const drain = () => new Promise((r) => setImmediate(r));

/** Replay the body resuming only `subset`; return the ids discovered. */
async function probe(body, subset) {
  found = []; resumers = new Map();
  const done = (async () => body())().catch(() => {});
  await drain();
  for (const id of subset) { resumers.get(id)?.(); await drain(); }
  await Promise.race([done, drain()]);
  return new Set(found);
}

async function deduce(body, stack) {
  memo = new Map(stack.map((id) => [id, id]));
  const all = await probe(body, stack);          // baseline: everything resumed
  const parents = new Map([...all].map((id) => [id, []]));
  for (const m of stack) {                       // one probe per candidate
    const without = await probe(body, stack.filter((s) => s !== m));
    for (const id of all) if (!without.has(id)) parents.get(id).push(m);
  }
  return { parents, probes: stack.length + 1 };
}

const shapes = {
  'independent chains          ': [
    () => Promise.all([
      (async () => { await stepRun('L1'); await stepRun('L2'); })(),
      (async () => { await stepRun('R1'); await stepRun('R2'); })()]),
    ['L1', 'R1'], { L2: 'L1', R2: 'R1' }],
  'join: Promise.all([a,b]); c ': [
    async () => { await Promise.all([stepRun('a'), stepRun('b')]); await stepRun('c'); },
    ['a', 'b'], { c: 'a+b' }],
  'dead end: await a; c; then b': [
    async () => { const pa = stepRun('a'), pb = stepRun('b'); await pa; await stepRun('c'); await pb; },
    ['a', 'b'], { c: 'a' }],
  'nested fan-out in a branch  ': [
    () => Promise.all([
      (async () => { await stepRun('F1'); await Promise.all([stepRun('F2a'), stepRun('F2b')]); })(),
      (async () => { await stepRun('P1'); await stepRun('P2'); })()]),
    ['F1', 'P1'], { F2a: 'F1', F2b: 'F1', P2: 'P1' }],
  'ragged: one branch ends     ': [
    () => Promise.all([
      (async () => { await stepRun('A1'); await stepRun('A2'); })(),
      (async () => { await stepRun('B1'); })()]),
    ['A1', 'B1'], { A2: 'A1' }],
  'gap: dep on x and z, y idle ': [
    async () => {
      const px = stepRun('x'), py = stepRun('y'), pz = stepRun('z');
      await Promise.all([px, pz]); await stepRun('w'); await py;
    },
    ['x', 'y', 'z'], { w: 'x+z' }],
  'three-way join              ': [
    async () => { await Promise.all([stepRun('x'), stepRun('y'), stepRun('z')]); await stepRun('w'); },
    ['x', 'y', 'z'], { w: 'x+y+z' }],
  'diamond: c needs a; d needs c': [
    async () => {
      const [av] = await Promise.all([stepRun('a'), stepRun('b')]);
      await stepRun('c'); void av;
    },
    ['a', 'b'], { c: 'a+b' }],
};

let pass = 0, fail = 0, totalProbes = 0;
for (const [name, [body, stack, expected]] of Object.entries(shapes)) {
  const { parents, probes } = await deduce(body, stack);
  totalProbes += probes;
  const parts = [];
  for (const [id, ps] of parents) {
    const got = ps.sort().join('+') || 'ROOT';
    const want = expected[id];
    if (want === undefined) continue;
    const ok = got === want; ok ? pass++ : fail++;
    parts.push(`${id}<=${got}${ok ? '' : `(WANT ${want})`}`);
  }
  console.log(`${name} probes=${probes}  ${parts.join('  ')}`);
}
console.log(`\n${pass} correct, ${fail} wrong   (${totalProbes} replays total)`);
