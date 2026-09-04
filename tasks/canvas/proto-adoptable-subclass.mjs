/**
 * THEORY 13: make `Promise.resolve` adopt our promise as-is, so the combinator
 * calls `.then` on it DIRECTLY and we see the handler identity it shares
 * across every element of one call.
 *
 * `Promise.resolve(x)` returns x unchanged iff `x.constructor === Promise`.
 * Setting that on OUR prototype touches nothing global.
 */
const seen = [];
class P extends Promise {
  static get [Symbol.species]() { return Promise; }
  then(f, r) { seen.push({ id: this.__id, f, r }); return super.then(f, r); }
}
Object.defineProperty(P.prototype, 'constructor', {
  value: Promise, writable: true, configurable: true,
});

const mk = (id) => { const p = new P((res) => res(id)); p.__id = id; return p; };

const report = (label, want) => {
  const groups = new Map();
  for (const s of seen) {
    const key = s.r ?? s.f;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(s.id);
  }
  const got = [...groups.values()].map((g) => `{${g.sort().join(',')}}`).sort().join(' ');
  console.log(`${got === want ? 'OK  ' : 'FAIL'} ${label}\n       got ${got}   want ${want}`);
  seen.length = 0;
  return got === want;
};

let pass = 0, fail = 0;
const t = (ok) => (ok ? pass++ : fail++);

await (async () => { await Promise.all([mk('a'), mk('b')]); })();
t(report('Promise.all([a,b])', '{a,b}'));

await Promise.all([
  (async () => { await mk('L1'); })(),
  (async () => { await mk('R1'); })(),
]);
t(report('two independent awaits', '{L1} {R1}'));

await (async () => { await Promise.all([mk('x'), mk('y'), mk('z')]); })();
t(report('three-way join', '{x,y,z}'));

await (async () => { await Promise.allSettled([mk('m'), mk('n')]); })();
t(report('allSettled', '{m,n}'));

await (async () => { await Promise.race([mk('f'), mk('s')]); })();
t(report('race', '{f,s}'));

await (async () => { const pa = mk('p'), pb = mk('q'); await pa; await pb; })();
t(report('sequential awaits', '{p} {q}'));

// interop sanity
const one = mk('sanity');
console.log(`\ninstanceof Promise: ${one instanceof Promise}`);
console.log(`Promise.resolve returns same object: ${Promise.resolve(one) === one}`);
console.log(`await works: ${await one}`);
console.log(`.catch/.finally: ${typeof one.catch}/${typeof one.finally}`);
console.log(`\n${pass} correct, ${fail} wrong`);
