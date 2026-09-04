/**
 * PROBING. Stop observing the promise graph; deduce it.
 *
 * The engine controls the order memoised steps resolve in. A step c that
 * depends on set S appears immediately after the LAST member of S is resumed.
 * So in a replay with resume order O, we learn:  S ⊆ prefix(O, trigger)
 * and  trigger ∈ S.
 *
 * Replay again with the order REVERSED and intersect the prefixes:
 *   S ⊆ prefix(forward) ∩ prefix(reverse)
 *
 * chains  : L2 after L1 forward (prefix {L1}) -> exact
 * join a,b: c after b forward (prefix {a,b}), after a reverse (prefix {b,a})
 *           -> intersection {a,b}, correct
 * deadend : c after a in both -> prefix {a} ∩ {b,a} = {a}, correct
 *
 * Cost: one extra replay of the function body per discovery request.
 */
let memo = new Map(), resumers = new Map(), resumedSoFar = [], found = [];

function stepRun(id) {
  if (memo.has(id)) {
    const v = memo.get(id);
    return new Promise((res) => resumers.set(id, () => res(v)));
  }
  found.push({ id, prefix: [...resumedSoFar] });
  return new Promise(() => {});
}

const drain = () => new Promise((r) => setImmediate(r));

async function replay(body, order) {
  found = []; resumers = new Map(); resumedSoFar = [];
  const done = (async () => body())().catch(() => {});
  await drain();
  for (const id of order) {
    resumedSoFar.push(id);
    resumers.get(id)?.();
    await drain();
  }
  await Promise.race([done, drain()]);
  return found;
}

async function deduce(body, stack) {
  memo = new Map(stack.map((id) => [id, id]));
  const fwd = await replay(body, stack);
  const rev = await replay(body, [...stack].reverse());
  const byId = new Map(rev.map((f) => [f.id, new Set(f.prefix)]));
  return fwd.map((f) => {
    const other = byId.get(f.id) ?? new Set();
    const parents = f.prefix.filter((p) => other.has(p));
    return { id: f.id, parents: parents.sort() };
  });
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
};

let pass = 0, fail = 0;
for (const [name, [body, stack, expected]] of Object.entries(shapes)) {
  const res = await deduce(body, stack);
  const parts = res.map((r) => {
    const got = r.parents.join('+') || 'ROOT';
    const want = expected[r.id];
    if (want === undefined) return `${r.id}<=${got}`;
    const ok = got === want; ok ? pass++ : fail++;
    return `${r.id}<=${got}${ok ? '' : `(WANT ${want})`}`;
  });
  console.log(`${name} ${parts.join('  ')}`);
}
console.log(`\n${pass} correct, ${fail} wrong`);
