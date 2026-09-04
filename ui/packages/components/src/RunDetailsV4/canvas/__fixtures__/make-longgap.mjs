#!/usr/bin/env node
/**
 * Build `longgap.json` — a run that sleeps for seven days.
 *
 *   node make-longgap.mjs
 *
 * THIS IS NOT A CAPTURE. Every other fixture in this directory came off a real
 * Dev Server; this one is `step.json` with its timestamps rewritten, and it is
 * marked `synthetic` in `index.ts` so nothing mistakes it for evidence.
 *
 * The reason is simple: a genuinely long sleep cannot be captured by waiting for
 * it. The alternative — hand-writing a payload — would invent span shapes nobody
 * would think to invent, which is the exact thing the README warns about. So
 * this takes a real capture and moves only the clock.
 *
 * What it changes: the sleep span's end, and every timestamp at or after it, are
 * pushed forward by seven days. Durations of the actual work are untouched, so
 * the run really is ~9 days elapsed and a fraction of a second executing — which
 * is the case the elastic axis exists for. Nothing else in the payload is
 * altered.
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';

const GAP_MS = 7 * 24 * 60 * 60 * 1000;

const dir = import.meta.dirname;
const source = JSON.parse(readFileSync(resolve(dir, 'step.json'), 'utf8'));

/** The sleep is the gap. Everything at or after its start slides forward. */
function findSleepStart(trace) {
  let at = null;
  const walk = (span) => {
    if (span.stepOp === 'SLEEP' && span.startedAt) {
      const t = Date.parse(span.startedAt);
      if (at === null || t < at) at = t;
    }
    (span.childrenSpans ?? []).forEach(walk);
  };
  walk(trace);
  return at;
}

const sleepStart = findSleepStart(source.run.trace);
if (sleepStart === null) {
  console.error('step.json has no SLEEP span — the source fixture changed shape');
  process.exit(1);
}

const FIELDS = ['queuedAt', 'scheduledAt', 'startedAt', 'endedAt'];

function shift(span) {
  for (const field of FIELDS) {
    const raw = span[field];
    if (typeof raw !== 'string') continue;
    const t = Date.parse(raw);
    if (Number.isNaN(t)) continue;
    // Strictly after the sleep's start: the sleep keeps its own start and gains
    // the seven days on its end, which is what makes it the gap rather than
    // shifting the whole run sideways.
    if (t > sleepStart) span[field] = new Date(t + GAP_MS).toISOString();
  }
  (span.childrenSpans ?? []).forEach(shift);
  (span.discoveries ?? []).forEach(shift);
}

shift(source.run.trace);

const out = resolve(dir, 'longgap.json');
writeFileSync(out, `${JSON.stringify(source, null, 2)}\n`);

const root = source.run.trace;
const elapsed = Date.parse(root.endedAt) - Date.parse(root.queuedAt);
console.log(
  `longgap.json  elapsed=${(elapsed / 86_400_000).toFixed(2)}d  sleep=+${GAP_MS / 86_400_000}d`
);
