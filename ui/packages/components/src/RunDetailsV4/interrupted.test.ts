/**
 * A step that never finished, in a run that did.
 *
 * `calculateDuration` measures an unfinished span against *now*, which is right
 * for a run still in flight and a lie for one that ended. On `cancelled` the
 * open `waitForEvent` reported 29 minutes on a run cancelled after 35 seconds,
 * and the number grew every time the page was opened — the kind of bug that
 * returns silently, so it is pinned here.
 */
import { describe, expect, it } from 'vitest';

import cancelled from './canvas/__fixtures__/cancelled.json';
import inflight from './canvas/__fixtures__/inflight.json';
import step from './canvas/__fixtures__/step.json';
import type { Trace } from './types';
import { traceRollup, traceToTimelineData } from './utils/traceConversion';

const barsOf = (fixture: unknown) =>
  traceToTimelineData(traceRollup((fixture as { run: { trace: Trace } }).run.trace), {
    runID: 'test',
  }).bars;

const walk = (bars: ReturnType<typeof barsOf>): ReturnType<typeof barsOf> =>
  bars.flatMap((bar) => [bar, ...walk(bar.children ?? [])]);

describe('a step cut off by cancellation', () => {
  it('is measured against the run, not against now', () => {
    const run = barsOf(cancelled)[0]!;
    const wait = walk([run]).find((bar) => bar.name === 'never arrives')!;

    expect(wait.interrupted).toBe(true);
    expect(wait.endTime).not.toBeNull();

    // Never past the run. Before the fix this was `now`, so it grew without
    // bound and was already 29 minutes on a 35 second run.
    expect(wait.endTime!.getTime()).toBeLessThanOrEqual(run.endTime!.getTime());
  });

  it('does not touch a run that is still in flight', () => {
    // `inflight` was captured mid-run: there, measuring an open span against
    // now is the correct thing to do, and nothing should be marked cut short.
    expect(walk(barsOf(inflight)).some((bar) => bar.interrupted)).toBe(false);
  });

  it('does not mark anything in a run that finished cleanly', () => {
    expect(walk(barsOf(step)).some((bar) => bar.interrupted)).toBe(false);
  });
});
