import { describe, expect, test } from 'vitest';

import { mergeBacklogMetrics } from './mergeBacklogMetrics';

describe('mergeBacklogMetrics', () => {
  test('joins sparse series by timestamp without inventing zeroes', () => {
    expect(
      mergeBacklogMetrics(
        [
          { bucket: '2026-08-23T03:16:00Z', value: 144650 },
          { bucket: '2026-08-23T03:18:00Z', value: 144607 },
          { bucket: '2026-08-23T03:20:00Z', value: 0 },
        ],
        [
          { bucket: '2026-08-23T03:17:00Z', value: 12 },
          { bucket: '2026-08-23T03:18:00Z', value: 10 },
        ],
        '1m',
      ),
    ).toEqual([
      {
        name: '2026-08-23T03:16:00Z',
        values: { scheduled: 144650 },
      },
      {
        name: '2026-08-23T03:17:00Z',
        values: { sleeping: 12 },
      },
      {
        name: '2026-08-23T03:18:00Z',
        values: { scheduled: 144607, sleeping: 10 },
      },
      {
        name: '2026-08-23T03:19:00.000Z',
        values: {},
      },
      {
        name: '2026-08-23T03:20:00Z',
        values: { scheduled: 0 },
      },
    ]);
  });
});
