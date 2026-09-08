/**
 * The code that produced each group of figures.
 *
 * The page is organised by what you WROTE, not by what the drawing is called.
 * One snippet, then everything worth saying about it — because the old
 * arrangement grouped by concept and so reprinted the same three lines under
 * seven different headings: `await step.run('a')` was the code for queue time,
 * flow control, system latency, finalization, the hatching rule, the interval
 * rule and the hover panel, each with its own copy.
 *
 * A snippet has to name every row its figures draw, and every step it names has
 * to be drawn by one of them. `validate.mjs` fails otherwise, which is what
 * stops a group from quietly collecting a figure that no longer belongs to it.
 *
 * Kept under ~36 columns on purpose: these sit in a narrow sticky column beside
 * the figures, and a snippet that wraps stops looking like code.
 */
export const CODE={
 oneStep:`await step.run('a', …)`,

 chain:`await step.run('a', …)\nawait step.run('b', …)`,

 caught:`try {\n  await step.run('a', …)\n} catch {}\n\nawait step.run('b', …)`,

 parallel:`await Promise.all([\n  step.run('a', …),\n  step.run('b', …),\n  step.run('c', …),\n])\nawait step.run('d', …)`,

 discovery:`await step.run('a', …)\n\nawait Promise.all([\n  step.run('b', …),\n  step.run('c', …),\n])`,

 nested:`await Promise.all([\n  (async () => {\n    await step.run('a', …)\n    await step.run('a1', …)\n    await step.run('a2', …)\n  })(),\n  (async () => {\n    await step.run('b', …)\n    await step.run('b1', …)\n    await step.run('b2', …)\n  })(),\n])`,

 orphan:`await step.run('a', …)\n\n// started, never awaited\nstep.run('orphan', …)\n\nawait step.run('b', …)`,

 emit:`await step.sendEvent('notify', [\n  { name: 'app/thing' },\n])`,

 sleep:`await step.run('a', …)\nawait step.sleep('nap', '7d')\nawait step.run('b', …)`,

 wait:`await step.run('a', …)\n\nconst ev = await step.waitForEvent('w', {\n  event: 'app/paid',\n  timeout: '1h',\n})\n\nif (ev) await step.run('b', …)\nelse     await step.run('fallback', …)`,

 otel:`await step.run('charge', async () => {\n  await fetch('/pay', …)\n  await db.query('SELECT …')\n  await db.query('UPDATE …')\n})\n\nawait step.run('fanout', async () => {\n  await Promise.all([\n    fetch('/a'),\n    fetch('/b'),\n    fetch('/c'),\n  ])\n})`,

 loop:`await step.run('setup', …)\n\nfor (const i of items) {        // 500\n  await step.run('fetch', …)\n}\n\nawait step.run('teardown', …)`,

 wideFanout:`await Promise.all(\n  items.map(() =>                // 12\n    step.run('worker', …)),\n)\nawait step.run('collect', …)`,

 failureCluster:`for (const i of items) {\n  await step.run('act', …)\n}`,

 expandGroup:`await step.run('first', …)\n\nfor (const i of items) {        // 40\n  await step.run('batch', …)\n}\n\nawait step.run('last', …)`,

 manyCompressions:`for (const i of items) {          // 9\n  await step.run('poll', …)\n  await step.sleep('nap', '2m')\n}`,
};
