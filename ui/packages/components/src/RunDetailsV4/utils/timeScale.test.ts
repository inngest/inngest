/**
 * The invariant is the point of these tests: the axis may compress, a duration
 * may not. Everything else here is arithmetic.
 */
import { describe, expect, it } from 'vitest';

import { buildTimeScale, linearScale, mergeIntervals, scaleTicks } from './timeScale';
import { calculateBarPosition, calculateDuration } from './timing';

const SECOND = 1000;
const DAY = 24 * 60 * 60 * 1000;

describe('mergeIntervals', () => {
  it('merges overlapping and touching runs of work', () => {
    expect(
      mergeIntervals([
        { startMs: 10, endMs: 20 },
        { startMs: 15, endMs: 25 },
        { startMs: 25, endMs: 30 },
        { startMs: 50, endMs: 60 },
      ])
    ).toEqual([
      { startMs: 10, endMs: 30 },
      { startMs: 50, endMs: 60 },
    ]);
  });
});

describe('buildTimeScale', () => {
  it('stays linear when nothing is idle enough to break', () => {
    const scale = buildTimeScale(0, 1000, [
      { startMs: 0, endMs: 400 },
      { startMs: 500, endMs: 1000 },
    ]);
    expect(scale.compressed).toBe(false);
    expect(scale.gaps).toEqual([]);
    expect(scale.toPercent(500)).toBeCloseTo(50);
  });

  it('breaks a long idle stretch and gives it a narrow band', () => {
    // The `longgap` shape: a little work, seven days of nothing, a little work.
    const end = 7 * DAY + 2 * SECOND;
    const scale = buildTimeScale(0, end, [
      { startMs: 0, endMs: SECOND },
      { startMs: 7 * DAY + SECOND, endMs: end },
    ]);

    expect(scale.compressed).toBe(true);
    expect(scale.gaps).toHaveLength(1);

    const [gap] = scale.gaps;
    // The gap reports its real duration — it is only its *width* that shrinks.
    expect(gap!.durationMs).toBe(7 * DAY);
    expect(gap!.endPercent - gap!.startPercent).toBeCloseTo(5, 1);

    // The two seconds of actual work now occupy most of the width, which is the
    // entire purpose. Linearly they would have had 0.0003% between them.
    expect(scale.toPercent(SECOND)).toBeGreaterThan(40);
  });

  it('never moves backwards', () => {
    const end = 7 * DAY;
    const scale = buildTimeScale(0, end, [
      { startMs: 0, endMs: SECOND },
      { startMs: 3 * DAY, endMs: 3 * DAY + SECOND },
      { startMs: end - SECOND, endMs: end },
    ]);

    let previous = -1;
    for (let i = 0; i <= 100; i++) {
      const percent = scale.toPercent((end * i) / 100);
      expect(percent).toBeGreaterThanOrEqual(previous);
      previous = percent;
    }
    expect(scale.toPercent(0)).toBe(0);
    expect(scale.toPercent(end)).toBe(100);
  });

  it('ignores a gap that is long but a small part of the run', () => {
    // 10s of idle inside an hour is not what makes a view unreadable.
    const hour = 60 * 60 * 1000;
    const scale = buildTimeScale(0, hour, [
      { startMs: 0, endMs: hour / 2 },
      { startMs: hour / 2 + 10 * SECOND, endMs: hour },
    ]);
    expect(scale.compressed).toBe(false);
  });

  it('ignores a gap that is a big fraction but momentary', () => {
    const scale = buildTimeScale(0, 100, [
      { startMs: 0, endMs: 10 },
      { startMs: 90, endMs: 100 },
    ]);
    expect(scale.compressed).toBe(false);
  });
});

describe('the invariant: only the axis compresses', () => {
  const end = 7 * DAY + 2 * SECOND;
  const busy = [
    { startMs: 0, endMs: 890 },
    { startMs: 7 * DAY + SECOND, endMs: end },
  ];
  const scale = buildTimeScale(0, end, busy);

  it('reports a duration identically with and without a broken axis', () => {
    const start = new Date(0);
    const finish = new Date(890);

    // The scale is not even an argument to this. That is the point: it cannot
    // affect the answer, rather than merely happening not to.
    expect(calculateDuration(start, finish)).toBe(890);

    const linear = calculateBarPosition(start, finish, new Date(0), new Date(end));
    const broken = calculateBarPosition(start, finish, new Date(0), new Date(end), scale);

    // Positions differ...
    expect(broken.widthPercent).toBeGreaterThan(linear.widthPercent);
    // ...but the duration the user is shown does not depend on either.
    expect(calculateDuration(start, finish)).toBe(890);
  });

  it('gives a sub-pixel step a visible width once the axis breaks', () => {
    const linear = calculateBarPosition(new Date(0), new Date(890), new Date(0), new Date(end));
    const broken = calculateBarPosition(
      new Date(0),
      new Date(890),
      new Date(0),
      new Date(end),
      scale
    );
    expect(linear.widthPercent).toBeLessThan(0.5);
    expect(broken.widthPercent).toBeGreaterThan(10);
  });
});

describe('scaleTicks', () => {
  it('never labels a time inside a compressed gap', () => {
    const end = 7 * DAY;
    const scale = buildTimeScale(0, end, [
      { startMs: 0, endMs: SECOND },
      { startMs: end - SECOND, endMs: end },
    ]);

    for (const tick of scaleTicks(scale, 5)) {
      const inGap = scale.gaps.some(
        (g) => tick.percent > g.startPercent + 0.01 && tick.percent < g.endPercent - 0.01
      );
      expect(inGap).toBe(false);
    }
  });

  it('is evenly spaced in width but not in time', () => {
    const scale = linearScale(0, 1000);
    expect(scaleTicks(scale, 5).map((t) => t.percent)).toEqual([0, 25, 50, 75, 100]);
  });
});
