#!/usr/bin/env node
/**
 * Build `failcluster.json` — 500 steps with a burst of failures two-thirds of
 * the way through.
 *
 *   node make-failcluster.mjs
 *
 * THIS IS NOT A CAPTURE. It is `tall500.json` with the status of a contiguous
 * run of steps rewritten, and it is marked `synthetic` in `index.ts`.
 *
 * It exists for one specific acceptance test, stated in the brief: *a failure
 * cluster two-thirds through a 500-step run should be a red smear visible in
 * the first screenful with no interaction.* That is a claim about a view, and
 * it cannot be checked without a run that has such a cluster. Producing one for
 * real would mean a fixture app that fails on demand at a chosen iteration —
 * worth doing eventually, but the view can be judged now.
 *
 * Only `status` changes. Timestamps, span shapes, names and structure are the
 * real capture's.
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';

/** Where the burst sits, and how wide it is, as fractions of the run. */
const CLUSTER_AT = 2 / 3;
const CLUSTER_WIDTH = 0.04;

const dir = import.meta.dirname;
const source = JSON.parse(readFileSync(resolve(dir, 'tall500.json'), 'utf8'));

const steps = (source.run.trace.childrenSpans ?? []).filter((s) => s.stepOp === 'RUN');
if (steps.length < 100) {
  console.error(`tall500.json has ${steps.length} RUN steps — the source fixture changed shape`);
  process.exit(1);
}

const first = Math.floor(steps.length * CLUSTER_AT);
const last = Math.min(steps.length - 1, first + Math.ceil(steps.length * CLUSTER_WIDTH));

let marked = 0;
for (let i = first; i <= last; i++) {
  const step = steps[i];
  step.status = 'FAILED';
  // Attempts too, so the run reads as something that was retried and still
  // failed rather than as a single unexplained red mark.
  step.attempts = 2;
  for (const child of step.childrenSpans ?? []) child.status = 'FAILED';
  marked += 1;
}

const out = resolve(dir, 'failcluster.json');
writeFileSync(out, `${JSON.stringify(source, null, 2)}\n`);

console.log(
  `failcluster.json  ${marked} of ${steps.length} steps failed, from index ${first} to ${last}`
);
