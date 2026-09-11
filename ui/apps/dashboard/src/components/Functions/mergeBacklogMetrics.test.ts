import { describe, expect, test } from 'vitest';

import { mergeBacklogMetrics } from './mergeBacklogMetrics';

const buckets = [
  '2026-08-23T03:16:00Z',
  '2026-08-23T03:17:00Z',
  '2026-08-23T03:18:00Z',
  '2026-08-23T03:19:00Z',
  '2026-08-23T03:20:00Z',
];

const series = (values: Array<number | null>) => {
  return buckets.map((bucket, index) => ({ bucket, value: values[index]! }));
};

describe('mergeBacklogMetrics', () => {
  test('combines dense series and preserves nulls and observed zeroes', () => {
    expect(
      mergeBacklogMetrics(
        series([144650, null, 144607, null, 0]),
        series([null, 12, 10, null, null]),
        '1m',
        buckets[0]!,
        '2026-08-23T03:21:00Z',
      ),
    ).toEqual([
      {
        name: buckets[0],
        values: { scheduled: 144650, sleeping: null },
      },
      {
        name: buckets[1],
        values: { scheduled: null, sleeping: 12 },
      },
      {
        name: buckets[2],
        values: { scheduled: 144607, sleeping: 10 },
      },
      {
        name: buckets[3],
        values: { scheduled: null, sleeping: null },
      },
      {
        name: buckets[4],
        values: { scheduled: 0, sleeping: null },
      },
    ]);
  });

  test('shows null runs lasting at least three minutes as inferred zeroes', () => {
    expect(
      mergeBacklogMetrics(
        series([42, null, null, null, 0]),
        series([1, 1, 1, 1, 1]),
        '1m',
        buckets[0]!,
        '2026-08-23T03:21:00Z',
      ),
    ).toEqual([
      { name: buckets[0], values: { scheduled: 42, sleeping: 1 } },
      {
        name: buckets[1],
        values: { scheduled: 0, sleeping: 1 },
        inferred: ['scheduled'],
      },
      {
        name: buckets[2],
        values: { scheduled: 0, sleeping: 1 },
        inferred: ['scheduled'],
      },
      {
        name: buckets[3],
        values: { scheduled: 0, sleeping: 1 },
        inferred: ['scheduled'],
      },
      { name: buckets[4], values: { scheduled: 0, sleeping: 1 } },
    ]);
  });

  test('does not infer nulls in partial edge buckets', () => {
    expect(
      mergeBacklogMetrics(
        series([null, null, null, null, null]),
        series([1, 1, 1, 1, 1]),
        '1m',
        '2026-08-23T03:16:30Z',
        '2026-08-23T03:20:30Z',
      ),
    ).toEqual([
      { name: buckets[0], values: { scheduled: null, sleeping: 1 } },
      {
        name: buckets[1],
        values: { scheduled: 0, sleeping: 1 },
        inferred: ['scheduled'],
      },
      {
        name: buckets[2],
        values: { scheduled: 0, sleeping: 1 },
        inferred: ['scheduled'],
      },
      {
        name: buckets[3],
        values: { scheduled: 0, sleeping: 1 },
        inferred: ['scheduled'],
      },
      { name: buckets[4], values: { scheduled: null, sleeping: 1 } },
    ]);
  });

  test('leaves short null runs and empty responses blank', () => {
    expect(
      mergeBacklogMetrics(
        series([4, null, 3, null, 2]),
        series([null, null, 1, 1, 1]),
        '1m',
        buckets[0]!,
        '2026-08-23T03:21:00Z',
      ),
    ).toEqual([
      { name: buckets[0], values: { scheduled: 4, sleeping: null } },
      { name: buckets[1], values: { scheduled: null, sleeping: null } },
      { name: buckets[2], values: { scheduled: 3, sleeping: 1 } },
      { name: buckets[3], values: { scheduled: null, sleeping: 1 } },
      { name: buckets[4], values: { scheduled: 2, sleeping: 1 } },
    ]);
    expect(
      mergeBacklogMetrics(
        [],
        [],
        '1m',
        '2026-08-23T03:00:00Z',
        '2026-08-23T04:00:00Z',
      ),
    ).toEqual([]);
  });

  test('uses elapsed time rather than bucket count for the threshold', () => {
    const fiveMinuteBuckets = [
      '2026-08-23T03:10:00Z',
      '2026-08-23T03:15:00Z',
      '2026-08-23T03:20:00Z',
    ];
    const fiveMinuteSeries = (values: Array<number | null>) => {
      return fiveMinuteBuckets.map((bucket, index) => ({
        bucket,
        value: values[index]!,
      }));
    };

    expect(
      mergeBacklogMetrics(
        fiveMinuteSeries([4, null, 2]),
        fiveMinuteSeries([1, 1, 1]),
        '5m',
        '2026-08-23T03:10:00Z',
        '2026-08-23T03:25:00Z',
      ),
    ).toEqual([
      {
        name: fiveMinuteBuckets[0],
        values: { scheduled: 4, sleeping: 1 },
      },
      {
        name: fiveMinuteBuckets[1],
        values: { scheduled: 0, sleeping: 1 },
        inferred: ['scheduled'],
      },
      {
        name: fiveMinuteBuckets[2],
        values: { scheduled: 2, sleeping: 1 },
      },
    ]);
  });
});
