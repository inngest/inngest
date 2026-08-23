import { describe, expect, test } from 'vitest';

import { mergeStepsRunningMetrics } from './mergeStepsRunningMetrics';

describe('mergeStepsRunningMetrics', () => {
  test('joins sparse series by timestamp without inventing zeroes', () => {
    expect(
      mergeStepsRunningMetrics(
        [
          { bucket: '2026-08-23T03:16:00Z', value: 12 },
          { bucket: '2026-08-23T03:20:00Z', value: 10 },
        ],
        [{ bucket: '2026-08-23T03:18:00Z', value: 1 }],
        '1m',
      ),
    ).toEqual([
      {
        name: '2026-08-23T03:16:00Z',
        values: { running: 12 },
      },
      {
        name: '2026-08-23T03:17:00.000Z',
        values: {},
      },
      {
        name: '2026-08-23T03:18:00Z',
        values: { concurrencyLimit: true },
      },
      {
        name: '2026-08-23T03:19:00.000Z',
        values: {},
      },
      {
        name: '2026-08-23T03:20:00Z',
        values: { running: 10 },
      },
    ]);
  });
});
