/**
 * The brief states this item's acceptance test as a sentence: a failure cluster
 * two-thirds through a 500-step run should be findable in the first screenful
 * with no interaction. `failcluster` exists for it.
 *
 * These assert the half a test can: that the red lands in the third it happened
 * in, that it stays contiguous, and that the map shows the whole run rather than
 * whatever the views below have collapsed. Whether it is *visible* is for the
 * eye and the review agents.
 */
import { describe, expect, it } from 'vitest';

import failcluster from '../canvas/__fixtures__/failcluster.json';
import parallel from '../canvas/__fixtures__/parallel.json';
import step from '../canvas/__fixtures__/step.json';
import tall500 from '../canvas/__fixtures__/tall500.json';
import wide from '../canvas/__fixtures__/wide.json';
import type { Trace } from '../types';
import { packMinimap } from './density';
import { linearScale } from './timeScale';
import { traceRollup, traceToTimelineData } from './traceConversion';

function minimapFor(fixture: unknown) {
  const raw = (fixture as { run: { trace: unknown } }).run.trace as Trace;
  const data = traceToTimelineData(traceRollup(raw), { runID: 'test' });
  return packMinimap(data.bars, linearScale(data.minTime.getTime(), data.maxTime.getTime()));
}

describe('packMinimap', () => {
  it('puts the failure cluster in the third of the run it happened in', () => {
    const failing = minimapFor(failcluster).marks.filter((m) => m.state === 'failed');

    expect(failing.length).toBeGreaterThan(0);
    for (const mark of failing) {
      expect(mark.startPercent).toBeGreaterThan(33);
    }

    // Contiguous rather than smeared across the run, or "where is the problem"
    // has no answer.
    const spread =
      Math.max(...failing.map((m) => m.startPercent)) -
      Math.min(...failing.map((m) => m.startPercent));
    expect(spread).toBeLessThan(25);
  });

  it('shows the whole run, not what the views below collapsed to', () => {
    // tall500 draws three rows in the timeline; the map still has all 500 steps.
    expect(minimapFor(tall500).marks.length).toBeGreaterThan(400);
  });

  it('collapses sequential work to a single lane', () => {
    // The point of packing: a run that never did two things at once is one line,
    // however many steps it had.
    expect(minimapFor(tall500).rows).toBe(1);
    expect(minimapFor(step).rows).toBe(1);
  });

  it('needs a lane per concurrent step, so the row count is the concurrency', () => {
    // `wide` is a 12-wide fan-out; `parallel` is three at once. The row count is
    // a property neither the trace nor the canvas shows.
    expect(minimapFor(wide).rows).toBeGreaterThan(5);
    expect(minimapFor(parallel).rows).toBeGreaterThan(1);
  });

  it('never leaves a step too small to see', () => {
    for (const fixture of [tall500, wide, step]) {
      for (const mark of minimapFor(fixture).marks) {
        expect(mark.widthPercent).toBeGreaterThan(0);
      }
    }
  });
});
