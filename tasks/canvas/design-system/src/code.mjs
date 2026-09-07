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

 fanout:`await Promise.all([\n  step.run('a', …),\n  step.run('b', …),\n  step.run('c', …),\n])\n\n// a step outside the fan-out\nawait step.run('unrelated', …)`,

 coalesce:`await Promise.all([\n  step.run('a', …),\n  step.run('b', …),\n  step.run('c', …),\n])\nawait step.run('d', …)`,

 resumeTwo:`await step.run('a', …)\n\nawait Promise.all([\n  step.run('b', …),\n  step.run('c', …),\n])`,

 nested:`await Promise.all([\n  step.run('a', …),\n  step.run('b', …),\n])\nawait Promise.all([\n  step.run('a1', …),\n  step.run('a2', …),\n])\nawait step.run('b1', …)`,

 branches:`await Promise.all([\n  (async () => {\n    await step.run('a', …)\n    await step.run('a2', …)\n  })(),\n  (async () => {\n    await step.run('b', …)\n    await step.run('b2', …)\n  })(),\n])`,

 race:`await Promise.race([\n  step.run('fast', …),\n  step.run('slow', …),\n])\nawait step.run('next', …)`,

 orphan:`await step.run('a', …)\n\n// started, never awaited\nstep.run('orphan', …)\n\nawait step.run('b', …)`,

 emit:`const ids =\n  await step.sendEvent('notify', [\n    { name: 'app/thing' },\n  ])`,

 threw:`await step.run('doomed', async () => {\n  throw new Error('boom')\n})`,

 retried:`await step.run('flaky', async () => {\n  if (attempt < 2)\n    throw new Error('flaky')\n  return 'ok'\n})`,

 caught:`try {\n  await step.run('caught', async () => {\n    throw new Error('boom')\n  })\n} catch {}\n\nawait step.run('after', …)`,

 oneBranchFails:`await Promise.all([\n  step.run('ok-branch', …),\n  step.run('bad-branch', …),\n])\nawait step.run('after', …)`,

 stillRunning:`await step.run('landed', …)\nawait step.run('still going', …)`,

 cancelled:`await Promise.all([\n  step.run('a', …),\n  step.run('b', …),   // still running\n])\n// the run was cancelled`,

 cancelledWait:`await step.waitForEvent('waiting', {\n  event: 'x',\n  timeout: '1h',\n})\n// still open when the run\n// was cancelled`,

 wait:`await step.run('a', …)\n\nawait step.waitForEvent('w', {\n  event: 'app/paid',\n  timeout: '1h',\n})\n\n// on match\nawait step.run('b', …)\n// on timeout\nawait step.run('fallback', …)`,

 waitInFanout:`await Promise.all([\n  step.run('a', …),\n  step.run('b', …),\n  step.waitForEvent('c', {\n    event: 'x',\n    timeout: '1h',\n  }),\n])\nawait step.run('d', …)`,

 sleepThenThrow:`await step.sleep('nap', '2s')\n\nawait step.run('a', async () => {\n  throw new Error('boom')\n})`,

 sleep:`await step.run('a', …)\nawait step.sleep('nap', '7d')\nawait step.run('b', …)`,

 sleepShares:`await step.run('left', …)     // 30s\nawait step.sleep('gap', '7d')\nawait step.run('right', …)    // 10s`,

 sleepMany:`for (const i of items) {          // 9\n  await step.run('poll', …)\n  await step.sleep('nap', '2m')\n}`,

 sleepParallel:`await Promise.all([\n  (async () => {\n    await step.run('a', …)\n    await step.sleep('a', '18s')\n  })(),\n  step.run('b', …),\n])\nawait step.sleep('c', '15m')\nawait step.run('c', …)\nawait step.run('d', …)`,

 days:`// a run measured in days\n// keeps its resolution`,

 loop:`await step.run('setup', …)\n\nfor (const i of items) {        // 500\n  await step.run('fetch', …)\n}\n\nawait step.run('teardown', …)`,

 wideFanout:`await Promise.all(\n  items.map(() =>                // 12\n    step.run('worker', …)),\n)\nawait step.run('collect', …)`,

 failureCluster:`for (const i of items) {\n  // a cluster of these threw\n  await step.run('act', …)\n}`,

 expandGroup:`await step.run('first', …)\n\nfor (const i of items) {        // 40\n  await step.run('batch', …)\n}\n\nawait step.run('last', …)`,

 sameName:`await Promise.all([\n  step.run('fetch', …),\n  step.run('fetch', …),\n])\n// coalescing after the Promise.all\n// reports one step, so it rolls\n// into that step's row\nawait step.run('save', …)`,

 requestFailed:`await step.run('a', …)\n// the request reporting b\n// threw twice first\nawait step.run('b', …)`,

 requestNeverSucceeded:`await step.run('a', …)\n// the request that would report\n// the next step never succeeded,\n// so no step exists to host it\nawait step.run('b', …)`,

 otel:`await step.run('charge', async () => {\n  await fetch('/pay', …)\n  await db.query('SELECT …')\n  await db.query('UPDATE …')\n})`,

 otelFanout:`await step.run('fanout', async () => {\n  await Promise.all([\n    fetch('/a'),\n    fetch('/b'),\n    fetch('/c'),\n  ])\n  await db.query('SELECT …')\n  await db.query('UPDATE …')\n})`,

 otelPartial:`await step.run('charge', async () => {\n  await fetch('/pay', …)\n  await db.query('SELECT …')\n})`,
};
