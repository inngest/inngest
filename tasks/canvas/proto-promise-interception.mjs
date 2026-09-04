/**
 * Is there a way to observe `.then` registration WITHOUT giving up a native
 * promise? Three candidates, in increasing order of scariness.
 */
console.log('--- 1. instance-level .then patch on a native promise ---');
{
  const p = Promise.resolve(1);
  const orig = p.then.bind(p);
  let seen = false;
  p.then = (f, r) => { seen = true; return orig(f, r); };
  await p;
  console.log(`   await saw our then? ${seen}   (V8 fast path bypasses it)`);
}

console.log('--- 2. Promise subclass overriding then ---');
{
  let seen = 0;
  class StepPromise extends Promise {
    then(f, r) { seen++; return super.then(f, r); }
  }
  const p = StepPromise.resolve(1);
  await p;
  console.log(`   await saw our then? ${seen > 0}  (calls: ${seen})`);
  console.log(`   instanceof Promise: ${p instanceof Promise}`);
  console.log(`   has .catch/.finally: ${typeof p.catch === 'function'}/${typeof p.finally === 'function'}`);

  let both = 0;
  class P2 extends Promise { then(f, r) { both++; return super.then(f, r); } }
  let ra, rb;
  const a = new P2((r) => (ra = r)), b = new P2((r) => (rb = r));
  const all = Promise.all([a, b]);
  ra(1); rb(2); await all;
  console.log(`   Promise.all registered on both subclass members? ${both >= 2} (calls: ${both})`);
}

console.log('--- 3. plain object thenable ---');
{
  let seen = false;
  const t = { then(f) { seen = true; return Promise.resolve(1).then(f); } };
  const v = await t;
  console.log(`   await saw our then? ${seen}, value ${v}`);
  console.log(`   instanceof Promise: ${t instanceof Promise}`);
  console.log(`   has .catch: ${typeof t.catch === 'function'}`);
}
