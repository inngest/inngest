/**
 * PROBE for theory 15: what does V8's zero-cost async stack trace contain at
 * the moment a step is discovered inside a resumed continuation?
 *
 * The hope: `at async Promise.all (index N)` frames, which would name the
 * *branch* a discovery happened in without touching promises at all.
 */
function stackHere(label) {
  const e = {};
  Error.captureStackTrace(e, stackHere);
  const frames = e.stack.split('\n').slice(1)
    .map((l) => l.trim())
    .filter((l) => l.includes('async') || l.includes('at '))
    .map((l) => l.replace(/\(.*\)/, '').replace(/file:.*$/, '').trim());
  console.log(`  ${label.padEnd(22)} ${frames.join(' | ') || '(none)'}`);
}

const tick = () => new Promise((r) => setImmediate(r));

console.log('A. discovery inside Promise.all branches, after an await:');
await Promise.all([
  (async function branchA() { await tick(); stackHere('in branchA'); })(),
  (async function branchB() { await tick(); stackHere('in branchB'); })(),
]);

console.log('\nB. discovery after `await Promise.all([...])` completes:');
await (async function body() {
  await Promise.all([tick(), tick()]);
  stackHere('after all resolves');
})();

console.log('\nC. discovery in a nested branch:');
await Promise.all([
  (async function outerA() {
    await Promise.all([
      (async function innerA1() { await tick(); stackHere('in innerA1'); })(),
      (async function innerA2() { await tick(); stackHere('in innerA2'); })(),
    ]);
  })(),
  (async function outerB() { await tick(); stackHere('in outerB'); })(),
]);

console.log('\nD. allSettled / race branches:');
await Promise.allSettled([
  (async function settledA() { await tick(); stackHere('in allSettled[0]'); })(),
  (async function settledB() { await tick(); stackHere('in allSettled[1]'); })(),
]);

console.log('\nE. same branch, two sequential discoveries:');
await (async function seq() {
  await tick(); stackHere('seq first');
  await tick(); stackHere('seq second');
})();

console.log('\nF. cost of one capture:');
const N = 20000;
let t = process.hrtime.bigint();
for (let i = 0; i < N; i++) { const e = {}; Error.captureStackTrace(e, stackHere); void e.stack; }
console.log(`  ${Number(process.hrtime.bigint() - t) / N / 1000} µs per formatted capture`);
t = process.hrtime.bigint();
const prep = Error.prepareStackTrace;
Error.prepareStackTrace = (_, s) => s;
for (let i = 0; i < N; i++) { const e = {}; Error.captureStackTrace(e, stackHere); void e.stack; }
Error.prepareStackTrace = prep;
console.log(`  ${Number(process.hrtime.bigint() - t) / N / 1000} µs per structured capture`);
console.log('  async stack depth limit:', Error.stackTraceLimit);
