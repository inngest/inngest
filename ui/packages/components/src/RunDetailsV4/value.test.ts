/**
 * The risk in this file is overclaiming, so most of these assert what is NOT
 * said. A value line that flatters the platform is worse than no value line,
 * because it is the one part of the page whose only job is to be believed.
 */
import { describe, expect, it } from 'vitest';

import blocked from './canvas/__fixtures__/blocked.json';
import emit from './canvas/__fixtures__/emit.json';
import failure from './canvas/__fixtures__/failure.json';
import longgap from './canvas/__fixtures__/longgap.json';
import retry from './canvas/__fixtures__/retry.json';
import simple from './canvas/__fixtures__/simple.json';
import step from './canvas/__fixtures__/step.json';
import type { Trace } from './types';
import { traceRollup } from './utils/traceConversion';
import { runValue, valueLines } from './value';

// Always rolled up, as the views use it: the raw payload lists a retried step
// three times and would report one recovery as three.
const traceOf = (fixture: unknown) => traceRollup((fixture as { run: { trace: Trace } }).run.trace);

describe('runValue', () => {
  it('reports a retry that actually recovered', () => {
    const value = runValue(traceOf(retry));
    expect(value.recoveredSteps).toBeGreaterThan(0);
    expect(value.recoveredAttempts).toBeGreaterThan(0);
    expect(valueLines(value)[0]).toMatch(/^Recovered after \d+ failed attempt/);
  });

  it('does not call a step that stayed failed a recovery', () => {
    // `failure` is a run that went down. Nothing was saved, so nothing is
    // claimed — this is the assertion that keeps the line honest.
    const value = runValue(traceOf(failure));
    expect(value.recoveredSteps).toBe(0);
    expect(valueLines(value).join(' ')).not.toMatch(/Recovered/);
  });

  it('reports flow control only when it really held the run', () => {
    // `blocked` waited 6.3s behind a concurrency limit of one.
    expect(runValue(traceOf(blocked)).heldMs).toBeGreaterThan(5000);
    // `step` was scheduled immediately; a few hundred ms is ordinary and must
    // not be dressed up as flow control doing something for you.
    expect(runValue(traceOf(step)).heldMs).toBe(0);
  });

  it('reports time suspended, and only for waits', () => {
    expect(runValue(traceOf(longgap)).suspendedMs).toBeGreaterThan(6 * 86_400_000);
    expect(runValue(traceOf(simple)).suspendedMs).toBe(0);
  });

  it('counts steps that sent events onward', () => {
    expect(runValue(traceOf(emit)).emittedSteps).toBe(1);
    expect(runValue(traceOf(step)).emittedSteps).toBe(0);
  });

  it('says nothing at all about an unremarkable run', () => {
    // The most important case. A plain run gets a plain page.
    expect(valueLines(runValue(traceOf(simple)))).toEqual([]);
  });
});

describe('valueLines', () => {
  it('is factual and never celebratory', () => {
    const lines = valueLines(runValue(traceOf(retry))).join(' ');
    // No exclamation marks, and no emoji — matched with the \u flag so the
    // surrogate pairs are treated as single characters.
    expect(lines).not.toMatch(/[🎉✨🚀!]/u);
  });

  it('quantifies everything it claims', () => {
    for (const fixture of [retry, blocked, longgap]) {
      for (const line of valueLines(runValue(traceOf(fixture)))) {
        expect(line, line).toMatch(/\d/);
      }
    }
  });
});
