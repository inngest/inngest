/**
 * The shared shape table and the ground truth behind it.
 *
 * ---------------------------------------------------------------------------
 * WHAT AN EDGE MEANS  (stated, because two of the old expectations were wrong)
 * ---------------------------------------------------------------------------
 * Draw x -> c iff:
 *   (1) c cannot begin until x has completed, AND
 *   (2) there is no z with x -> z -> c.
 * That is: the *transitive reduction* of "must complete before".
 *
 * (2) is what stops a chain a -> b -> c from being drawn as a triangle. It is
 * also the whole difference between the two shapes people write that look alike:
 *
 *   parallel, then joined            sequential
 *   const pa = step.run('a')         await step.run('a')
 *   const pb = step.run('b')         await step.run('b')
 *   await pa; await pb               step.run('c')
 *   step.run('c')
 *
 *   a and b start together, and c    b does not exist until a is done, so
 *   waits for both => c <= a + b     a -> b -> c and the a -> c edge is
 *   (a fork/join diamond)            implied => c <= b (a chain)
 *
 * Both are `await a; await b; step c` at the point of discovery. Only the
 * reduction separates them, and it needs no new signal: b's own parents are
 * already known from when b was discovered.
 *
 * So every theory below reports the raw "must complete before" set, and
 * `reduce()` is applied to all of them equally before grading.
 *
 * ---------------------------------------------------------------------------
 * RACE IS NOT A JOIN
 * ---------------------------------------------------------------------------
 * `await Promise.race([r1, r2]); step.run('r3')` — r3 starts as soon as the
 * FIRST settles. Drawing r3 <= r1 + r2 would claim it waited for both, which is
 * false. The right answer is the winner alone. The engine can tell: a race
 * continuation advances on the first resume, an `all` only on the last. Hence
 * the resumed-prefix rule used by the group theories — parents are the members
 * of the group resumed *so far*, not the whole group. That gets race, any, all,
 * allSettled and the "gap" shape right with one rule and no combinator identity.
 */

// Each shape: [ body(stepRun), completedStack, expected, note ]
export const shapes = {
  'independent chains     ': [(s) => () => Promise.all([
      (async () => { await s('L1'); await s('L2'); })(),
      (async () => { await s('R1'); await s('R2'); })()]),
    ['L1','R1'], { L2:'L1', R2:'R1' }],

  'join Promise.all([a,b])': [(s) => async () => {
      await Promise.all([s('a'), s('b')]); await s('c'); },
    ['a','b'], { c:'a+b' }],

  'dead end               ': [(s) => async () => {
      const pa=s('a'), pb=s('b'); await pa; await s('c'); await pb; },
    ['a','b'], { c:'a' },
    'c only waits on a; b is still running when c starts'],

  'nested fan-out         ': [(s) => () => Promise.all([
      (async () => { await s('F1'); await Promise.all([s('F2a'), s('F2b')]); })(),
      (async () => { await s('P1'); await s('P2'); })()]),
    ['F1','P1'], { F2a:'F1', F2b:'F1', P2:'P1' }],

  'ragged                 ': [(s) => () => Promise.all([
      (async () => { await s('A1'); await s('A2'); })(),
      (async () => { await s('B1'); })()]),
    ['A1','B1'], { A2:'A1' }],

  'gap x,z dep; y idle    ': [(s) => async () => {
      const px=s('x'), py=s('y'), pz=s('z');
      await Promise.all([px,pz]); await s('w'); await py; },
    ['x','y','z'], { w:'x+z' }],

  'three-way join         ': [(s) => async () => {
      await Promise.all([s('x'), s('y'), s('z')]); await s('w'); },
    ['x','y','z'], { w:'x+y+z' }],

  'nested Promise.all     ': [(s) => async () => {
      await Promise.all([Promise.all([s('a'), s('b')]), s('c')]); await s('d'); },
    ['a','b','c'], { d:'a+b+c' }],

  // The pair that the reduction has to separate. Same awaits, different graphs.
  'parallel then joined   ': [(s) => async () => {
      const pa=s('a'), pb=s('b'); await pa; await pb; await s('c'); },
    ['a','b'], { c:'a+b' },
    'a and b start together, c waits for both'],

  'sequential 3           ': [(s) => async () => {
      await s('s1'); await s('s2'); await s('s3'); },
    ['s1','s2'], { s2:'s1', s3:'s2' },
    's3 <= s2 only; the s1 -> s3 edge is implied by s1 -> s2 -> s3'],

  'allSettled             ': [(s) => async () => {
      await Promise.allSettled([s('m'), s('n')]); await s('o'); },
    ['m','n'], { o:'m+n' }],

  'ragged flipped         ': [(s) => () => Promise.all([
      (async () => { await s('A1'); })(),
      (async () => { await s('B1'); await s('B2'); })()]),
    ['A1','B1'], { B2:'B1' },
    'the barren branch is the FIRST one; carryover must not pick up A1'],

  // `a ~b` means: solid edge from a, dashed edge from b. Dashed = "could have
  // unblocked this, but did not" — the losing side of a race.
  'race                   ': [(s) => async () => {
      await Promise.race([s('r1'), s('r2')]); await s('r3'); },
    ['r1','r2'], { r3:'r1 ~r2' },
    'r3 waits for the winner only; r2 is drawn dashed'],

  'any                    ': [(s) => async () => {
      await Promise.any([s('q1'), s('q2'), s('q3')]); await s('q4'); },
    ['q1','q2','q3'], { q4:'q1 ~q2+q3' },
    'same as race: one real edge, two dashed'],

  'join then dead end     ': [(s) => async () => {
      const pa=s('a'), pb=s('b'), pc=s('c');
      await Promise.all([pa,pb]); await s('d'); await pc; },
    ['a','b','c'], { d:'a+b' }],

  'branch join inside all ': [(s) => () => Promise.all([
      (async () => { await Promise.all([s('J1'), s('J2')]); await s('J3'); })(),
      (async () => { await s('K1'); await s('K2'); })()]),
    ['J1','J2','K1'], { J3:'J1+J2', K2:'K1' }],

  'third-party join       ': [(s) => async () => {
      const ps = [s('t1'), s('t2')];      // what p-map / Bluebird / async do
      await new Promise((res) => { let n=0; for (const p of ps) p.then(() => { if (++n===2) res(); }); });
      await s('t3'); },
    ['t1','t2'], { t3:'t1+t2' },
    'a join built by hand, without any Promise combinator'],

  'user .catch on two    ': [(s) => async () => {
      const noop = () => {};
      const pa=s('a'), pb=s('b');
      pa.catch(noop); pb.catch(noop);          // two registrations, one tick, NOT a join
      await pa; await s('c'); await pb; },
    ['a','b'], { c:'a' },
    'sync-batch grouping must not treat two independent .catch calls as a join'],

  'fire and forget        ': [(s) => async () => {
      s('i');                              // started, never awaited
      await s('a'); await s('c'); },
    ['i','a'], { c:'a' },
    'i is never awaited, so nothing downstream depends on it'],

  // The shape tests/v4.chains actually uses, and the one T19 gets wrong in
  // practice: the combinator is over the *branch functions*, not over step
  // promises, so there is no registration to group and J falls back to the
  // resume window. Under-declares — R2 is named, L2 dangles.
  'join over branches     ': [(s) => async () => {
      const chain = async (n) => { await s(`${n}1`); return s(`${n}2`); };
      await Promise.all([chain('L'), chain('R')]);
      await s('J'); },
    ['L1','R1','L2','R2'], { L2:'L1', R2:'R1', J:'L2+R2' },
    'a join whose members are branch promises, not step promises'],

  'loop of 3              ': [(s) => async () => {
      for (const n of ['g1','g2','g3']) await s(n); },
    ['g1','g2'], { g2:'g1', g3:'g2' }],
};

/** Transitive reduction against parents already recorded for earlier steps. */
export function reduce(parents, known) {
  const ancestors = (id, seen = new Set()) => {
    for (const p of known.get(id) ?? []) {
      if (seen.has(p)) continue;
      seen.add(p);
      ancestors(p, seen);
    }
    return seen;
  };
  return parents.filter((p) => !parents.some((q) => q !== p && ancestors(q).has(p)));
}

/**
 * Drive one shape the way the engine actually does it: one request per
 * completed step, memoising the prefix, recording a step's parents the first
 * time it is discovered.
 */
export async function runShape(replay, mk, stack) {
  const known = new Map(), alts = new Map();
  for (let k = 0; k <= stack.length; k++) {
    // Request k has exactly the first k steps in state: they are memoised AND
    // resumed. Everything after them is still undiscovered.
    const prefix = stack.slice(0, k);
    const found = await replay(mk, prefix);
    for (const f of found) {
      if (known.has(f.id)) continue;
      known.set(f.id, reduce([...new Set(f.parents)], known).sort());
      if (f.alternates?.length) alts.set(f.id, [...new Set(f.alternates)].sort());
    }
  }
  return { known, alts };
}

const show = (known, alts, id) => {
  const solid = (known.get(id) ?? []).join('+') || 'ROOT';
  const dashed = (alts.get(id) ?? []).join('+');
  return dashed ? `${solid} ~${dashed}` : solid;
};

export function grade(label, replay) {
  let pass = 0, fail = 0; const misses = [], lines = [];
  return (async () => {
    for (const [name, [mk, stack, expected]] of Object.entries(shapes)) {
      const { known, alts } = await runShape(replay, mk, stack);
      const parts = [];
      for (const [id, want] of Object.entries(expected)) {
        const got = show(known, alts, id);
        const ok = got === want;
        ok ? pass++ : (fail++, misses.push(`${name.trim()}: ${id}<=${got} want ${want}`));
        parts.push(`${id}<=${got}${ok ? '' : '*'}`);
      }
      lines.push(`${name} ${parts.join('  ')}`);
    }
    return { label, pass, fail, misses, lines };
  })();
}

export function report(r) {
  console.log(`\n=== ${r.label} ===`);
  console.log(r.lines.join('\n'));
  console.log(`  ${r.pass}/${r.pass + r.fail} correct`);
  if (r.misses.length) console.log(r.misses.map((m) => '    ' + m).join('\n'));
}
