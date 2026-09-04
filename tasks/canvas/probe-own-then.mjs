/**
 * PROBE for theory 18: can an *own* `then` property on a plain native promise
 * see what a Promise subclass sees, without being a subclass at all?
 *
 * Checks, in order:
 *   1. is it still a plain native promise by every observable test
 *   2. does Promise.all call our own `then` (shared onRejected identity)
 *   3. does `await` call it (expected: no — spec uses PerformPromiseThen)
 *   4. do the other combinators call it
 */
const log = [];
function tagged(id) {
  let resolve;
  const p = new Promise((r) => { resolve = r; });
  const real = Promise.prototype.then;
  Object.defineProperty(p, 'then', {
    value: function (f, r) {
      log.push({ id, f: f?.name || 'anon', r: r?.name || 'anon', rRef: r });
      return real.call(this, f, r);
    },
    writable: true, configurable: true, enumerable: false,
  });
  p.__id = id;
  p.__resolve = resolve;
  return p;
}

// 1. identity checks
const p = tagged('id');
console.log('instanceof Promise         ', p instanceof Promise);
console.log('constructor === Promise    ', p.constructor === Promise);
console.log('Promise.resolve(p) === p   ', Promise.resolve(p) === p);
console.log('toString tag               ', Object.prototype.toString.call(p));
console.log('is a subclass instance     ', Object.getPrototypeOf(p) !== Promise.prototype);
p.__resolve(1);

// 2. Promise.all
{
  log.length = 0;
  const a = tagged('a'), b = tagged('b');
  const all = Promise.all([a, b]);
  a.__resolve(1); b.__resolve(2); await all;
  console.log('\nPromise.all calls own then ', log.length === 2);
  console.log('  shared onRejected identity', log.length === 2 && log[0].rRef === log[1].rRef);
  console.log('  entries                   ', log.map((l) => `${l.id}:${l.f}/${l.r}`).join(' '));
}

// 3. await
{
  log.length = 0;
  const c = tagged('c');
  queueMicrotask(() => c.__resolve(3));
  await c;
  console.log('\nawait calls own then       ', log.length > 0, `(${log.length} calls)`);
}

// 4. other combinators
for (const name of ['allSettled', 'race', 'any']) {
  log.length = 0;
  const x = tagged('x'), y = tagged('y');
  const pr = Promise[name]([x, y]);
  x.__resolve(1); y.__resolve(2);
  await pr.catch(() => {});
  const shared = log.length === 2 && log[0].rRef === log[1].rRef;
  console.log(`${name.padEnd(11)} calls own then `, log.length > 0,
    ` shared handler: ${shared}`, ` [${log.map((l) => `${l.f}/${l.r}`).join(' ')}]`);
}

// 5. does the added own property cost a microtask hop?
{
  const plain = Promise.resolve(1);
  const own = tagged('t'); own.__resolve(1);
  let order = [];
  plain.then(() => order.push('plain'));
  own.then(() => order.push('own'));
  await new Promise((r) => setTimeout(r, 0));
  console.log('\nmicrotask order plain/own  ', order.join(','));
}
