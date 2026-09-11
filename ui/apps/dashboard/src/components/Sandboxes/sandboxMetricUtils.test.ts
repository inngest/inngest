import { describe, expect, it } from 'vitest';

import { sandboxMetricStats } from './sandboxMetricUtils';

const first = '2026-08-25T12:00:00Z';
const second = '2026-08-25T12:01:00Z';
const third = '2026-08-25T12:02:00Z';

describe('sandboxMetricStats', () => {
  it('derives current vCPU usage from cumulative counters', () => {
    const stats = sandboxMetricStats(
      {
        cpuUserTime: {
          data: [
            { bucket: first, value: 1_000_000 },
            { bucket: second, value: 7_000_000 },
          ],
        },
        cpuSystemTime: {
          data: [
            { bucket: first, value: 500_000 },
            { bucket: second, value: 3_500_000 },
          ],
        },
      },
      2,
      400,
    );

    expect(stats.cpu).toEqual({ current: 0.15, percent: 7.5 });
  });

  it('uses the latest RAM usage and peak', () => {
    const stats = sandboxMetricStats(
      {
        memoryCurrent: {
          data: [
            { bucket: second, value: 200 },
            { bucket: first, value: 100 },
          ],
        },
        memoryPeak: {
          data: [
            { bucket: first, value: 150 },
            { bucket: second, value: 250 },
          ],
        },
      },
      1,
      400,
    );

    expect(stats.memory).toEqual({
      current: 200,
      peak: 250,
      percent: 50,
      peakPercent: 62.5,
    });
  });

  it('uses the latest valid CPU interval after a counter reset', () => {
    const stats = sandboxMetricStats(
      {
        cpuUserTime: {
          data: [
            { bucket: first, value: 10_000 },
            { bucket: second, value: 100 },
            { bucket: third, value: 700 },
          ],
        },
        cpuSystemTime: {
          data: [
            { bucket: first, value: 0 },
            { bucket: second, value: 0 },
            { bucket: third, value: 0 },
          ],
        },
      },
      1,
      400,
    );

    expect(stats.cpu?.current).toBeCloseTo(0.00001);
    expect(stats.cpu?.percent).toBeCloseTo(0.001);
  });

  it('returns null stats when no series are present', () => {
    expect(sandboxMetricStats({}, 1, 400)).toEqual({
      cpu: null,
      memory: null,
    });
  });
});
