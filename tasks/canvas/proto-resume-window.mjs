/**
 * THEORY: the engine resumes ONE memoised step per tick. So "which resume was
 * active when this step was discovered" distinguishes the two shapes that look
 * identical from outside:
 *
 *   independent chains -> N new steps appear across N DISTINCT windows (1:1)
 *   fork/bystander     -> 2 new steps appear in the SAME window (a join)
 *
 * If that holds, the client rule is simply: draw lanes iff the mapping is a
 * bijection. No timing, no ordering inference.
 *
 * This models the engine's resume loop only — no promise wrapping needed.
 */
let memo = new Map(), resumers = new Map(), found = [], activeWindow = null;

function stepRun(id) {
  if (memo.has(id)) {
    const v = memo.get(id);
    return new Promise((res) => resumers.set(id, () => res(v)));
  }
  found.push({ id, window: activeWindow });     // <- the one new datum
  return new Promise(() => {});                 // unresolved this replay
}

const drain = () => new Promise((r) => setImmediate(r));

async function replay(body, stack) {
  found = []; resumers = new Map(); activeWindow = null;
  const done = body().catch(() => {});
  await drain();                                 // initial body run: window=null
  for (const id of stack) {
    activeWindow = id;                           // engine resumes exactly one
    resumers.get(id)?.();
    await drain();
  }
  activeWindow = null;
  await Promise.race([done, drain()]);
  return found;
}

const shapes = {
  'independent chains': [
    () => Promise.all([
      (async () => { await stepRun('L1'); await stepRun('L2'); await stepRun('L3'); })(),
      (async () => { await stepRun('R1'); await stepRun('R2'); await stepRun('R3'); })(),
    ]),
    [[], ['L1','R1'], ['L1','R1','L2','R2']],
  ],
  'fork / bystander': [
    async () => {
      const [x] = await Promise.all([stepRun('fork'), stepRun('bystander')]);
      await Promise.all([stepRun('kid1'), stepRun('kid2')]); void x;
    },
    [[], ['fork','bystander']],
  ],
  'ragged (one branch ends)': [
    () => Promise.all([
      (async () => { await stepRun('A1'); await stepRun('A2'); })(),
      (async () => { await stepRun('B1'); })(),
    ]),
    [[], ['A1','B1']],
  ],
  'io between parent and child': [
    () => Promise.all([
      (async () => { await stepRun('IOP'); await new Promise(r=>setTimeout(r,60)); await stepRun('IOC'); })(),
      (async () => { await stepRun('OTH1'); await stepRun('OTH2'); })(),
    ]),
    [[], ['IOP','OTH1']],
  ],
  'chain deeper than the drain window': [
    () => Promise.all([
      (async () => { await stepRun('DP');
        let p = Promise.resolve(); for (let i=0;i<500;i++) p = p.then(()=>undefined); await p;
        await stepRun('DC'); })(),
      (async () => { await stepRun('SH1'); await stepRun('SH2'); })(),
    ]),
    [[], ['DP','SH1']],
  ],
  'nested fan-out in a branch': [
    () => Promise.all([
      (async () => { await stepRun('F1'); await Promise.all([stepRun('F2a'), stepRun('F2b')]); })(),
      (async () => { await stepRun('P1'); await stepRun('P2'); })(),
    ]),
    [[], ['F1','P1']],
  ],
};

for (const [name, [body, stacks]] of Object.entries(shapes)) {
  console.log(`\n=== ${name} ===`);
  let lastLevelSize = 0;
  for (const stack of stacks) {
    memo = new Map(stack.map((id) => [id, id]));
    const res = await replay(body, stack);
    if (!res.length) continue;
    // RULE: attribute a step to the resume window it was discovered in. Trust
    // it only when every member of the previous level produced at least one new
    // step — otherwise some member contributed nothing, which means the new
    // steps were gated on a join we cannot see.
    const prev = stack.slice(-lastLevelSize || stack.length);
    const producers = new Set(res.map((r) => r.window));
    const allProduced = res.every((r) => r.window !== null) &&
                        prev.every((p) => producers.has(p));
    console.log(`  stack=[${stack}]  prevLevel=[${prev}]`);
    console.log(`     found: ` + res.map((r) => `${r.id} <= ${r.window ?? 'ROOT'}`).join(', '));
    console.log(`     => ${allProduced ? 'DRAW EDGES (' + res.map(r=>r.window+'->'+r.id).join(', ') + ')' : 'draw junction'}`);
    lastLevelSize = res.length;
  }
}
