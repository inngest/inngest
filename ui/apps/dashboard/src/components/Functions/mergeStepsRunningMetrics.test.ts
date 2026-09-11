import { describe, expect, test } from 'vitest';

import { mergeStepsRunningMetrics } from './mergeStepsRunningMetrics';

describe('mergeStepsRunningMetrics', () => {
  test('infers sustained running nulls without mapping absent limits to false', () => {
    const buckets = [
      '2026-08-23T03:16:00Z',
      '2026-08-23T03:17:00Z',
      '2026-08-23T03:18:00Z',
      '2026-08-23T03:19:00Z',
      '2026-08-23T03:20:00Z',
    ];
    const series = (values: Array<number | null>) => {
      return buckets.map((bucket, index) => ({
        bucket,
        value: values[index]!,
      }));
    };

    expect(
      mergeStepsRunningMetrics(
        series([12, null, null, null, 10]),
        series([null, null, 1, null, null]),
        '1m',
        buckets[0]!,
        '2026-08-23T03:21:00Z',
      ),
    ).toEqual([
      {
        name: buckets[0],
        values: { running: 12, concurrencyLimit: null },
      },
      {
        name: buckets[1],
        values: { running: 0, concurrencyLimit: null },
        inferred: ['running'],
      },
      {
        name: buckets[2],
        values: { running: 0, concurrencyLimit: true },
        inferred: ['running'],
      },
      {
        name: buckets[3],
        values: { running: 0, concurrencyLimit: null },
        inferred: ['running'],
      },
      {
        name: buckets[4],
        values: { running: 10, concurrencyLimit: null },
      },
    ]);
  });
});
