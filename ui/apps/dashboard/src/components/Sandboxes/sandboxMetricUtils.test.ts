import { describe, expect, it } from 'vitest';

import { SandboxStatus } from '@/gql/graphql';

import {
  sandboxLifetimeStats,
  sandboxMetricStats,
  sandboxMetricsWindow,
} from './sandboxMetricUtils';

const first = '2026-08-25T12:00:00Z';
const second = '2026-08-25T12:01:00Z';
const third = '2026-08-25T12:02:00Z';
const live = { until: new Date('2026-08-25T12:05:00Z'), bucketMS: 60_000 };

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
      live,
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
      live,
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
      live,
    );

    expect(stats.cpu?.current).toBeCloseTo(0.00001);
    expect(stats.cpu?.percent).toBeCloseTo(0.001);
  });

  it('ignores the bucket that is still filling', () => {
    const stats = sandboxMetricStats(
      {
        cpuUserTime: {
          data: [
            { bucket: first, value: 0 },
            { bucket: second, value: 30_000_000 },
            { bucket: third, value: 35_000_000 },
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
      { until: new Date('2026-08-25T12:02:10Z'), bucketMS: 60_000 },
    );

    expect(stats.cpu).toEqual({ current: 0.5, percent: 50 });
  });

  it('returns null stats when no series are present', () => {
    expect(sandboxMetricStats({}, 1, 400, live)).toEqual({
      cpu: null,
      memory: null,
    });
  });
});

describe('sandboxMetricsWindow', () => {
  const now = new Date('2026-10-13T12:00:00Z');
  const startedAt = '2026-10-13T11:00:00Z';

  it.each([
    [{ status: SandboxStatus.Running, startedAt }, 'live'],
    [{ status: SandboxStatus.Stopping, startedAt }, 'live'],
    [{ status: SandboxStatus.Pending }, 'pending'],
    [{ status: SandboxStatus.Failed, failedAt: startedAt }, 'neverStarted'],
    [{ status: SandboxStatus.LaunchUnknown }, 'neverStarted'],
    [
      {
        status: SandboxStatus.Stopped,
        startedAt: '2026-09-01T11:00:00Z',
        endedAt: '2026-09-01T12:00:00Z',
      },
      'expired',
    ],
  ])('classifies %o as %s', (sandbox, kind) => {
    expect(sandboxMetricsWindow(sandbox, now).kind).toBe(kind);
  });

  it('queries a stopped sandbox from start to just past its end', () => {
    expect(
      sandboxMetricsWindow(
        {
          status: SandboxStatus.Stopped,
          startedAt,
          endedAt: '2026-10-13T11:30:00Z',
        },
        now,
      ),
    ).toEqual({
      kind: 'lifetime',
      startedAt: new Date(startedAt),
      endedAt: new Date('2026-10-13T11:30:00Z'),
      from: new Date(startedAt),
      until: new Date('2026-10-13T11:31:00Z'),
    });
  });

  it('ends a failed sandbox at its failure time, capped at now', () => {
    expect(
      sandboxMetricsWindow(
        {
          status: SandboxStatus.Failed,
          startedAt,
          failedAt: '2026-10-13T11:59:30Z',
        },
        now,
      ),
    ).toMatchObject({
      kind: 'lifetime',
      endedAt: new Date('2026-10-13T11:59:30Z'),
      until: now,
    });
  });
});

describe('sandboxLifetimeStats', () => {
  it('averages cumulative CPU over the runtime and uses the highest peak', () => {
    const stats = sandboxLifetimeStats(
      {
        cpuUserTime: {
          data: [
            { bucket: first, value: 10_000_000 },
            { bucket: second, value: 40_000_000 },
          ],
        },
        cpuSystemTime: {
          data: [
            { bucket: first, value: 5_000_000 },
            { bucket: second, value: 20_000_000 },
          ],
        },
        memoryCurrent: { data: [{ bucket: second, value: 100 }] },
        memoryPeak: {
          data: [
            { bucket: first, value: 300 },
            { bucket: second, value: 250 },
          ],
        },
      },
      2,
      400,
      120,
    );

    expect(stats).toEqual({
      cpu: { average: 0.5, percent: 25 },
      memory: { peak: 300, percent: 75 },
    });
  });

  it('falls back to current memory when no peak was recorded', () => {
    const stats = sandboxLifetimeStats(
      {
        memoryCurrent: {
          data: [
            { bucket: first, value: 100 },
            { bucket: second, value: 200 },
          ],
        },
      },
      1,
      400,
      60,
    );

    expect(stats).toEqual({ cpu: null, memory: { peak: 200, percent: 50 } });
  });

  it('returns null stats when nothing was recorded', () => {
    expect(sandboxLifetimeStats({}, 1, 400, 60)).toEqual({
      cpu: null,
      memory: null,
    });
  });
});
