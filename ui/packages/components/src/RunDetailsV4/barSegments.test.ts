/**
 * One invariant, and it is the most direct way this view could lie:
 *
 *   THE BAR IS THE SPAN, AND ITS SEGMENTS PARTITION IT.
 *
 * A breakdown that totals more than the span it decorates is describing
 * something outside that span. `blocked`'s `hold` step is the case that found
 * it: the step ran 5999ms and queued for none of it, but carries
 * `inngest.timing` metadata with `queue_delay_ms: 6299` — the RUN-level
 * concurrency hold, stamped onto the step. Drawn from that metadata the bar was
 * ~12s wide with half of it shown as waiting the step never did, while its own
 * label said 5.999s.
 */
import { describe, expect, it } from 'vitest';

import { generateBarSegments } from './Timeline';
import blocked from './canvas/__fixtures__/blocked.json';
import retry from './canvas/__fixtures__/retry.json';
import step from './canvas/__fixtures__/step.json';
import type { Trace } from './types';
import { traceRollup, traceToTimelineData } from './utils/traceConversion';

const barsOf = (fixture: unknown) =>
  traceToTimelineData(traceRollup((fixture as { run: { trace: Trace } }).run.trace), {
    runID: 'test',
  }).bars;

function walk(bars: ReturnType<typeof barsOf>): ReturnType<typeof barsOf> {
  return bars.flatMap((bar) => [bar, ...walk(bar.children ?? [])]);
}

describe('generateBarSegments', () => {
  it('never lets a bar report more time than its own span', () => {
    for (const fixture of [blocked, retry, step]) {
      for (const bar of walk(barsOf(fixture))) {
        const segments = generateBarSegments(bar);
        if (!segments?.length || !bar.endTime) continue;

        const total = segments.reduce((n, s) => n + s.widthPercent, 0);
        expect(total, bar.name).toBeLessThanOrEqual(100.01);
      }
    }
  });

  it('does not attribute the run-level concurrency hold to the step that followed it', () => {
    const hold = walk(barsOf(blocked)).find((bar) => bar.name === 'hold')!;

    // The metadata really does claim more than the span — this is the data
    // quirk, asserted so the fix is not mistaken for defensive noise.
    const spanMs = hold.endTime!.getTime() - hold.startTime.getTime();
    expect(hold.timingBreakdown!.totalMs).toBeGreaterThan(spanMs);

    // ...and the step is nonetheless drawn as overwhelmingly work.
    //
    // Not zero waiting: a step legitimately owns the short discovery gap that
    // led to it, and that is drawn as its ghosted lead-in. What it must never
    // own is the 6.3s hold, which happened before the step existed and belongs
    // to the run. So the assertion is on proportion, which is what the reader
    // actually perceives — a hair of lead-in is fine, half the bar is the bug.
    const segments = generateBarSegments(hold) ?? [];
    const waiting = segments
      .filter((s) => s.style === 'timing.waiting')
      .reduce((n, s) => n + s.widthPercent, 0);

    expect(waiting).toBeLessThan(5);
  });
});
