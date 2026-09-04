/**
 * Crux experiment. `await p` compiles to `p.then(resumeFn)`, and resumeFn
 * resumes the async function body SYNCHRONOUSLY. So if a Promise subclass wraps
 * resumeFn in an ALS context, does the resumed body see that context?
 *
 * If yes, direct awaits give exact lineage. Promise.all is expected to lose it,
 * because the aggregate promise is native and unwrapped.
 */
import { AsyncLocalStorage } from 'node:async_hooks';
const CTX = new AsyncLocalStorage();

const seen = [];
class StepPromise extends Promise {
  static get [Symbol.species]() { return Promise; }
  then(onF, onR) {
    const id = this.__id;
    return super.then(
      onF && ((v) => {
        const next = new Set([...(CTX.getStore() ?? []), id]);
        return CTX.run(next, () => onF(v));
      }),
      onR
    );
  }
}
const mk = (id, v) => { const p = StepPromise.resolve(v); p.__id = id; return p; };
const step = (id) => { seen.push([id, [...(CTX.getStore() ?? [])].sort().join('+') || 'ROOT']); };

console.log('1. direct await');
await (async () => { await mk('a', 1); step('b'); })();

console.log('2. chained awaits');
await (async () => { await mk('x', 1); await mk('y', 1); step('z'); })();

console.log('3. two independent chains');
await Promise.all([
  (async () => { await mk('L1', 1); step('L2'); })(),
  (async () => { await mk('R1', 1); step('R2'); })(),
]);

console.log('4. Promise.all over steps');
await (async () => { await Promise.all([mk('p', 1), mk('q', 1)]); step('r'); })();

for (const [id, ctx] of seen) console.log(`   ${id.padEnd(4)} sees ${ctx}`);
