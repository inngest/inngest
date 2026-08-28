import { describe, expect, test } from 'vitest';

import { mergeStepsRunningMetrics } from './mergeStepsRunningMetrics';

describe('mergeStepsRunningMetrics', () => {
  test('shows running gaps lasting at least three minutes as inferred zeroes', () => {
    expect(
      mergeStepsRunningMetrics(
        [
          { bucket: '2026-08-23T03:16:00Z', value: 12 },
          { bucket: '2026-08-23T03:20:00Z', value: 10 },
        ],
        [{ bucket: '2026-08-23T03:18:00Z', value: 1 }],
        '1m',
        '2026-08-23T03:16:00Z',
        '2026-08-23T03:21:00Z',
      ),
    ).toEqual([
      {
        name: '2026-08-23T03:16:00Z',
        values: { running: 12 },
      },
      {
        name: '2026-08-23T03:17:00.000Z',
        values: { running: 0 },
        inferred: ['running'],
      },
      {
        name: '2026-08-23T03:18:00Z',
        values: { concurrencyLimit: true, running: 0 },
        inferred: ['running'],
      },
      {
        name: '2026-08-23T03:19:00.000Z',
        values: { running: 0 },
        inferred: ['running'],
      },
      {
        name: '2026-08-23T03:20:00Z',
        values: { running: 10 },
      },
    ]);
  });
});
