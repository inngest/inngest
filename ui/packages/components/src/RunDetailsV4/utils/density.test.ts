/**
 * The brief states this item's acceptance test as a sentence: a failure cluster
 * two-thirds through a 500-step run should be a red smear visible in the first
 * screenful with no interaction. The `failcluster` fixture exists for it, and
 * this asserts the half that is checkable in a test — that the red lands in the
 * right third, and that it is not drowned out. Whether it is *visible* is a
 * question for the eye and the review agent.
 */
import { describe, expect, it } from 'vitest';

import failcluster from '../canvas/__fixtures__/failcluster.json';
import step from '../canvas/__fixtures__/step.json';
import tall500 from '../canvas/__fixtures__/tall500.json';
import type { Trace } from '../types';
import { densityBuckets, densityPeak } from './density';
import { linearScale } from './timeScale';
import { traceRollup, traceToTimelineData } from './traceConversion';

function bucketsFor(fixture: unknown, count = 120) {
  const raw = (fixture as { run: { trace: unknown } }).run.trace as Trace;
  const data = traceToTimelineData(traceRollup(raw), { runID: 'test' });
  const scale = linearScale(data.minTime.getTime(), data.maxTime.getTime());
  return densityBuckets(data.bars, scale, count);
}

describe('densityBuckets', () => {
  it('covers the run without gaps or overlaps', () => {
    const buckets = bucketsFor(tall500, 50);
    expect(buckets).toHaveLength(50);
    expect(buckets[0]!.startPercent).toBe(0);
    expect(buckets[49]!.endPercent).toBe(100);
    for (let i = 1; i < buckets.length; i++) {
      expect(buckets[i]!.startPercent).toBeCloseTo(buckets[i - 1]!.endPercent);
    }
  });

  it('counts something for a run with steps in it', () => {
    expect(densityPeak(bucketsFor(tall500))).toBeGreaterThan(0);
    expect(densityPeak(bucketsFor(step))).toBeGreaterThan(0);
  });

  it('puts the failure cluster in the third of the run it happened in', () => {
    const buckets = bucketsFor(failcluster);
    const failing = buckets.filter((b) => b.counts.failed > 0);

    expect(failing.length).toBeGreaterThan(0);

    // Every failing bucket should sit in the last half and none in the first
    // third — the cluster is at two thirds.
    for (const bucket of failing) {
      expect(bucket.startPercent).toBeGreaterThan(33);
    }

    // And the cluster must be contiguous rather than smeared over the whole run,
    // or "where is the problem" has no answer.
    const first = buckets.indexOf(failing[0]!);
    const last = buckets.indexOf(failing[failing.length - 1]!);
    expect(last - first).toBeLessThan(buckets.length / 4);
  });

  it('shows the whole run even when the views below are collapsing', () => {
    // The strip is built from the uncollapsed bars, so a 500-step run has far
    // more than the handful of rows the timeline draws for it.
    const total = bucketsFor(tall500).reduce((n, b) => n + b.total, 0);
    expect(total).toBeGreaterThan(400);
  });
});
